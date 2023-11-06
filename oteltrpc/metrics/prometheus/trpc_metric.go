//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package prometheus metrics
package prometheus

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/mozillazg/go-pinyin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"

	"trpc-ecosystem/go-opentelemetry/oteltrpc/consts"
)

func initSink() {
	metrics.RegisterMetricsSink(NewSink(prometheus.DefaultRegisterer))
}

// Sink implements metrics.Sink interface, convert trpc metrics API to Prometheus format.
// convert rules is different from trpc-metrics-prometheus plugin:
// trpc metric name => save original name in `_name` label, name use converted pinyin name
// trpc metric type => prometheus `_type` label
// 2021-09-10 update: as too many trpc metrics name is too long,
// use pinyin name to replace original name will make prometheus metadata api useless.
// Change into use same metric name for each type, use label to distinguish.
// 2021-11-29 update: for multi-dimension metrics, generate a collector for each metric,
// each collector has its own dimension information to prevent dimension explosion.
// Add counters and gauges fields to store multi-dimension counter and gauge
// 2023-03-28 update: the style of trpc metrics api multi-dimension metrics is multi-value, Prometheus is single-value.
// when mapping to Prometheus, use record.metric[*].name as metric name, not unique.
// Therefor, use record.name + record.metric[*].name as metric name, but it will break the panel.
// report case 1:
// metrics.ReportMultiDimensionMetricsX(
//
//	    record_name1,  // record.name
//	    []*metrics.Dimension{ {Name: "dim_1", Value: "val_11"} }, // labels values
//	    []*metrics.Metrics{
//					metrics.NewMetrics("metric_name1", 1, metrics.PolicySUM), // record.metric[*].name
//			},
//		)
//
// report case 2:
// metrics.ReportMultiDimensionMetricsX(
//
//	    record_name2,  // record.name
//	    []*metrics.Dimension{ {Name: "dim_1", Value: "val_11"}, {Name: "dim_2", Value: "val_11"} }, // labels values
//	    []*metrics.Metrics{
//					metrics.NewMetrics("metric_name1", 1, metrics.PolicySUM), // record.metric[*].name
//			},
//		)
//
// The correct way is to use record.name + record.metric[*].name as the Prometheus metric name,
// but it will change the metric name and break the panel.
// So use a compatible way to map without modifying the panel statement:
// 1. Add a label _name=record.name + record.metric[*].name.
// report case 1 mapping to metric_name1{_name="record_name1_metric_name1", dim_1="val_11"}
// report case 2 mapping to metric_name1{_name="record_name2_metric_name1", dim_1="val_11", dim_2="val_11"}
// 2. Implement a special composeMetricVec type, wrap the collectorVec with these two labels inconsistent.
// When Prometheus stores metrics, label value is empty and there is no label are equivalent.
type Sink struct {
	counter    *prometheus.CounterVec
	gauge      *prometheus.GaugeVec
	counters   *composeMetricVec
	gauges     *composeMetricVec
	histograms sync.Map
	registerer prometheus.Registerer
}

var (
	defaultCounterName = "trpc_counter_total" // counter name must end with _total
	defaultGaugeName   = "trpc_gauge"
)

// WithNoSink option to skip reporting to this sink
var WithNoSink metrics.Option = func(opts *metrics.Options) {}

// NewSink create a new Sink
func NewSink(registerer prometheus.Registerer) *Sink {
	s := &Sink{
		registerer: registerer,
		counters:   &composeMetricVec{},
		gauges:     &composeMetricVec{},
	}
	s.counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: defaultCounterName,
		Help: "trpc metrics counter",
	},
		[]string{"_name", "_type"},
	).MustCurryWith(prometheus.Labels{"_type": "counter"})
	s.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: defaultGaugeName,
		Help: "trpc metrics gauge",
	},
		[]string{"_name", "_type"},
	).MustCurryWith(prometheus.Labels{"_type": "gauge"})
	_ = s.registerer.Register(s.counter)
	_ = s.registerer.Register(s.gauge)
	_ = s.registerer.Register(s.counters)
	_ = s.registerer.Register(s.gauges)
	return s
}

// Name implements metrics.Sink.
func (*Sink) Name() string {
	return consts.PluginName
}

// Report implements metrics.Sink, which will be called synchronously on each metrics operation.
func (s *Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
	for _, o := range opts {
		if *((*uintptr)(unsafe.Pointer(&o))) == *((*uintptr)(unsafe.Pointer(&WithNoSink))) {
			return nil
		}
	}

	var labels []string
	var values []string
	for _, dimension := range rec.GetDimensions() {
		labels = append(labels, dimension.Name)
		values = append(values, dimension.Value)
	}

	for _, m := range rec.GetMetrics() {
		switch m.Policy() {
		case metrics.PolicySUM:
			s.incrCounter(rec.Name, m.Name(), m.Value(), labels, values)
		case metrics.PolicySET:
			s.setGauge(rec.Name, m.Name(), m.Value(), labels, values)
		case metrics.PolicyTimer:
			s.observeHistogram(m.Name(), time.Duration(int64(m.Value())).Seconds(), labels, values)
		case metrics.PolicyHistogram:
			s.observeHistogram(m.Name(), m.Value(), labels, values)
		case metrics.PolicyAVG, metrics.PolicyMAX, metrics.PolicyMIN, metrics.PolicyMID:
			s.setGauge(rec.Name, m.Name(), m.Value(), labels, values)
		default:
			s.setGauge(rec.Name, m.Name(), m.Value(), labels, values)
		}
	}
	return nil
}

func (s *Sink) register(metric interface{}, name string) {
	c, ok := metric.(prometheus.Collector)
	if !ok {
		return
	}
	if err := s.registerer.Register(c); err != nil {
		log.Warnf("opentelemetry: register err:%v, metric:%s", err, name)
	}
}

func (s *Sink) incrCounter(recName string, name string, value float64, labels []string, values []string) {
	// fast path single-dimension report
	if len(labels) == 0 {
		s.counter.WithLabelValues(strings.ToValidUTF8(name, "")).Add(value)
		return
	}
	// multi-dimension report
	var c interface{}
	var ok bool
	h := labelsHash(name, labels)
	if c, ok = s.counters.Load(h); !ok {
		counterName := getCounterName(s.convertMetricName(name))
		c = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: counterName,
			Help: name,
			ConstLabels: map[string]string{
				"_name": fmt.Sprintf("%s_%s", recName, name),
				"_type": "counter",
			},
		},
			labels,
		)
		c, _ = s.counters.LoadOrStore(h, c)
	}
	c.(*prometheus.CounterVec).WithLabelValues(values...).Add(value)
}

func (s *Sink) setGauge(recName string, name string, value float64, labels []string, values []string) {
	// fast path single-dimension report
	if len(labels) == 0 {
		s.gauge.WithLabelValues(strings.ToValidUTF8(name, "")).Set(value)
		return
	}
	// multi-dimension report
	var c interface{}
	var ok bool
	h := labelsHash(name, labels)
	if c, ok = s.gauges.Load(h); !ok {
		gaugeName := getGaugeName(s.convertMetricName(name))
		c = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: gaugeName,
			Help: name,
			ConstLabels: map[string]string{
				"_name": fmt.Sprintf("%s_%s", recName, name),
				"_type": "gauge",
			},
		},
			labels,
		)
		c, _ = s.gauges.LoadOrStore(h, c)
	}
	c.(*prometheus.GaugeVec).WithLabelValues(values...).Set(value)
}

func getCounterName(name string) string {
	if !strings.HasPrefix(name, "trpc_counter_") {
		name = "trpc_counter_" + name
	}
	if !strings.HasSuffix(name, "_total") {
		name += "_total"
	}
	return name
}

func getGaugeName(name string) string {
	if !strings.HasPrefix(name, "trpc_gauge_") {
		name = "trpc_gauge_" + name
	}
	return name
}

// observeHistogram different Histogram buckets may be different, so they can only be created dynamically
func (s *Sink) observeHistogram(name string, value float64, labels []string, values []string) {
	histogramLabels := []string{"_name", "_type"}
	labelsValues := []string{strings.ToValidUTF8(name, ""),
		"histogram"} // fixed dimension for non-multi-dimension report
	if len(labels) != 0 && len(labels) == len(values) { // use user-defined dimension
		histogramLabels = labels
		labelsValues = values
	}
	var m interface{}
	var ok bool
	if m, ok = s.histograms.Load(name); !ok {
		var buckets []float64
		if h, ok2 := metrics.GetHistogram(name); ok2 {
			for _, b := range h.GetBuckets() {
				buckets = append(buckets, b.ValueUpperBound)
			}
		} else {
			buckets = prometheus.DefBuckets
		}
		m = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trpc_histogram_" + s.convertMetricName(name),
			Help:    strings.ToValidUTF8(name, ""),
			Buckets: buckets,
		},
			histogramLabels,
		)
		if m, ok = s.histograms.LoadOrStore(name, m); !ok {
			s.register(m, name)
		}
	}
	m.(*prometheus.HistogramVec).WithLabelValues(labelsValues...).Observe(value)
}

// convertMetricName convert metric name to pinyin if not valid, use _ or hex to replace special char
func (s *Sink) convertMetricName(origin string) string {
	if model.LabelName(origin).IsValid() {
		return origin
	}
	var buf strings.Builder
	for i, r := range origin {
		switch {
		case isNormalChar(i, r):
			buf.WriteRune(r)
		case unicode.Is(unicode.Han, r):
			// Chinese to pinyin
			p := pinyin.NewArgs()
			buf.WriteString(strings.Join(pinyin.SinglePinyin(r, p), "_"))
			buf.WriteRune('_')
		case r < utf8.RuneSelf:
			// special char use _
			buf.WriteRune('_')
		default:
			// other special char use hex
			buf.WriteString(fmt.Sprintf("%X", r))
			buf.WriteRune('_')
		}
	}
	result := strings.TrimSuffix(buf.String(), "_")
	return result
}

func isNum(b rune) bool {
	return b >= '0' && b <= '9'
}

func isChar(b rune) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isNormalChar 字母或数字或下划线, 首字符不为数字
func isNormalChar(i int, b rune) bool {
	return isChar(b) || b == '_' || (isNum(b) && i > 0)
}

func labelsHash(name string, labels []string) uint64 {
	h := hashNew()
	h = hashAdd(h, name)
	for _, v := range labels {
		h = hashAdd(h, v)
		h = hashAddByte(h, model.SeparatorByte)
	}
	return h
}

type composeMetricVec struct {
	sync.Map
}

// Describe implements prometheus.Collector.
func (s *composeMetricVec) Describe(ch chan<- *prometheus.Desc) {
	s.Map.Range(func(key, value any) bool {
		value.(prometheus.Collector).Describe(ch)
		return true
	})
}

// Collect implements prometheus.Collector.
func (s *composeMetricVec) Collect(ch chan<- prometheus.Metric) {
	s.Map.Range(func(key, value any) bool {
		value.(prometheus.Collector).Collect(ch)
		return true
	})
}
