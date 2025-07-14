//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package metric
package metric

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

const (
	rpcMetricsCardinalityLimit = 500
)

// highCardinalityMetrics for alert
var (
	highCardinalityMetrics = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "high_cardinality_metrics",
			Help: "high cardinality metrics",
		},
		[]string{"name"})
)

var (
	// PerMetricCardinalityLimit ...
	// ref: https://prometheus.io/docs/practices/naming/#labels
	// https://prometheus.io/docs/prometheus/latest/querying/basics/#avoiding-slow-queries-and-overloads
	PerMetricCardinalityLimit = 2000
	// TotalMetricCardinalityLimit ...
	TotalMetricCardinalityLimit = 10000
)

// Processor metric processor user-defined hook for metric
type Processor func(metrics []*dto.MetricFamily) []*dto.MetricFamily

var (
	// defaultProcessor
	defaultProcessor = func(metrics []*dto.MetricFamily) []*dto.MetricFamily {
		return metrics
	}
	processor Processor = defaultProcessor
)

// SetProcessor set metric processor
func SetProcessor(p Processor) {
	if p == nil {
		return
	}
	processor = p
}

// LimitMetricsHandler handler for limited metrics avoiding Prometheus server OOM
func LimitMetricsHandler() http.Handler {
	gather := &LimitCardinalityGatherer{
		prometheus.DefaultGatherer,
		PerMetricCardinalityLimit,
		TotalMetricCardinalityLimit}
	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(gather, promhttp.HandlerOpts{
			// OpenMetrics is the only way to transmit exemplars. However, the move to OpenMetrics
			// is not completely transparent. Most notably, the values of "quantile"
			// labels of Summaries and "le" labels of Histograms are formatted with
			// a trailing ".0" if they would otherwise look like integer numbers
			// (which changes the identity of the resulting series on the Prometheus
			// server).
			EnableOpenMetrics: true,
		}),
	)
}

// LimitCardinalityGatherer struct for limit cardinality gatherer
type LimitCardinalityGatherer struct {
	prometheus.Gatherer
	PerMetirclimit   int
	TotalMetricLimit int
}

// Gather implements prometheus.Gatherer
func (l *LimitCardinalityGatherer) Gather() ([]*dto.MetricFamily, error) {
	res, err := l.Gatherer.Gather()
	if err != nil {
		return nil, err
	}
	var (
		total                      int
		resetClientM, resetServerM sync.Once
	)
	for i, v := range res {
		if l.PerMetirclimit > 0 && len(v.GetMetric()) > l.PerMetirclimit {
			log.Printf("opentelemetry: high cardinality metric '%s', value:%d, limit:%d",
				v.GetName(), len(v.GetMetric()), l.PerMetirclimit)
			highCardinalityMetrics.WithLabelValues(v.GetName()).Set(float64(len(v.GetMetric())))
			// get topN
			v.Metric = v.Metric[:l.PerMetirclimit]
			if strings.HasPrefix(v.GetName(), "rpc_client") {
				resetClientM.Do(func() {
					log.Printf("opentelemetry: reset rpc_client metric when high cardinality(>%d)",
						l.PerMetirclimit)
					clientStartedCounter.Reset()
					clientHandledCounter.Reset()
					clientHandledHistogram.Reset()
				})
			}
			if strings.HasPrefix(v.GetName(), "rpc_server") {
				resetServerM.Do(func() {
					log.Printf("opentelemetry: reset rpc_server metric when high cardinality(>%d)",
						l.PerMetirclimit)
					serverStartedCounter.Reset()
					serverHandledCounter.Reset()
					serverHandledHistogram.Reset()
				})
			}
		} else if m, _ := highCardinalityMetrics.GetMetricWithLabelValues(v.GetName()); m != nil {
			m.Set(0)
		}
		total += len(v.GetMetric())
		if l.TotalMetricLimit > 0 && total > l.TotalMetricLimit {
			log.Printf("opentelemetry: high cardinality metric '%s', value:%d %d, limit:%d",
				"all", total, len(res), l.TotalMetricLimit)
			highCardinalityMetrics.WithLabelValues("total").Set(float64(len(res)))
			return res[:i], nil
		} else if m, _ := highCardinalityMetrics.GetMetricWithLabelValues("total"); m != nil {
			m.Set(0)
		}
	}
	return processor(res), nil
}

type metricCollector interface {
	prometheus.Collector
	Delete(labels prometheus.Labels) bool
}

// LimitCardinalityCollector collector for limit cardinality
type LimitCardinalityCollector struct {
	metricCollector
	desc  string
	limit int
}

// Collect called when pull metrics
func (c *LimitCardinalityCollector) Collect(ch chan<- prometheus.Metric) {
	results := c.collect()
	if len(results) <= c.limit {
		for _, v := range results {
			ch <- v
		}
		return
	}
	metrics := map[*dto.Metric]prometheus.Metric{}
	var m []*dto.Metric
	for _, v := range results {
		mm := &dto.Metric{}
		_ = v.Write(mm)
		m = append(m, mm)
		metrics[mm] = v
	}
	if len(m) > c.limit {
		highCardinalityMetrics.WithLabelValues(c.desc).Set(float64(len(m)))
		log.Printf("opentelemetry: metric '%s' high cardinality, limit:%d", c.desc, c.limit)
		sort.Slice(m, func(i, j int) bool {
			return getMetricValue(m[i]) >
				getMetricValue(m[j])
		})
		for i := 0; i < len(m); i++ {
			shouldDelete := getMetricValue(m[i]) <= 1 || i >= c.limit
			if shouldDelete {
				labels := prometheus.Labels{}
				for _, vv := range m[i].GetLabel() {
					labels[vv.GetName()] = vv.GetValue()
				}
				c.Delete(labels)
				delete(metrics, m[i])
			}
		}
	}
	for _, v := range metrics {
		ch <- v
	}
}

func (c *LimitCardinalityCollector) collect() []prometheus.Metric {
	tmpCh := make(chan prometheus.Metric, 10)
	done := make(chan struct{})
	go func() {
		c.metricCollector.Collect(tmpCh)
		close(done)
	}()
	go func() {
		<-done
		close(tmpCh)
	}()
	var results []prometheus.Metric
	for v := range tmpCh {
		results = append(results, v)
	}
	return results
}

func getMetricValue(m *dto.Metric) float64 {
	return m.GetCounter().GetValue() +
		m.GetGauge().GetValue() +
		float64(m.GetHistogram().GetSampleCount()) +
		float64(m.GetSummary().GetSampleCount()) +
		m.GetUntyped().GetValue()
}
