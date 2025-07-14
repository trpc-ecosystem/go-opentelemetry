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

// Package prometheus metrics
package prometheus

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/metrics"

	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

func TestSink_Report(t *testing.T) {
	t.Run("valid counter", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		metrics.IncrCounter("trpc.counter", 1)
		metrics.IncrCounter("trpc.counter", 1)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, "trpc_counter_total", data[0].GetName())                         // name
		assert.Equal(t, "trpc.counter", data[0].GetMetric()[0].GetLabel()[0].GetValue()) // label: _name
		assert.Equal(t, "counter", data[0].GetMetric()[0].GetLabel()[1].GetValue())      // label: _type
		assert.Equal(t, float64(2), data[0].GetMetric()[0].GetCounter().GetValue())      // label: value
	})
	t.Run("valiad counter with chinese", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		metrics.IncrCounter("help中文指标名", 1)
		metrics.IncrCounter("help中文指标名", 1)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		assert.Equal(t, "trpc_counter_total", data[0].GetName())                      // name
		assert.Equal(t, "help中文指标名", data[0].GetMetric()[0].GetLabel()[0].GetValue()) // label: _name
		assert.Equal(t, "counter", data[0].GetMetric()[0].GetLabel()[1].GetValue())   // label: _type
		assert.Equal(t, float64(2), data[0].GetMetric()[0].GetCounter().GetValue())   // label: value
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
	})
	t.Run("valid gauge", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		metrics.SetGauge("trpc.gauge2", 2)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, "trpc_gauge", data[0].GetName())                                // name
		assert.Equal(t, "trpc.gauge2", data[0].GetMetric()[0].GetLabel()[0].GetValue()) // label: _name
		assert.Equal(t, "gauge", data[0].GetMetric()[0].GetLabel()[1].GetValue())       // label: _type
		assert.Equal(t, float64(2), data[0].GetMetric()[0].GetGauge().GetValue())       // label: value
	})
	t.Run("valid timer histogram", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		metrics.RecordTimer("trpc.timer", time.Second)
		_, ok := s.histograms.Load("trpc.timer")
		assert.True(t, ok)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, "trpc_histogram_trpc_timer", data[0].GetName()) // name
		assert.Equal(t, "trpc.timer",
			data[0].GetMetric()[0].GetLabel()[0].GetValue()) // label: _name
		assert.Equal(t, "histogram",
			data[0].GetMetric()[0].GetLabel()[1].GetValue()) // label: _type
		assert.Equal(t, uint64(0),
			data[0].GetMetric()[0].GetHistogram().GetBucket()[6].GetCumulativeCount()) // label: value
		assert.Equal(t, uint64(1),
			data[0].GetMetric()[0].GetHistogram().GetBucket()[7].GetCumulativeCount()) // label: value
	})
}

func TestSink_ReportHistogram(t *testing.T) {
	t.Run("valid histogram name", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		metrics.AddSample("trpc.Histogram", metrics.NewValueBounds(0, 1, 2), 1)
		_, ok := s.histograms.Load("trpc.Histogram")
		assert.True(t, ok)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, "trpc_histogram_trpc_Histogram", data[0].GetName()) // name
		assert.Equal(t, "trpc.Histogram",
			data[0].GetMetric()[0].GetLabel()[0].GetValue()) // label: _name
		assert.Equal(t, "histogram",
			data[0].GetMetric()[0].GetLabel()[1].GetValue()) // label: _type
		assert.Equal(t, uint64(0),
			data[0].GetMetric()[0].GetHistogram().GetBucket()[0].GetCumulativeCount()) // label: value
		assert.Equal(t, uint64(1),
			data[0].GetMetric()[0].GetHistogram().GetBucket()[1].GetCumulativeCount()) // label: value
		assert.Equal(t, uint64(1),
			data[0].GetMetric()[0].GetHistogram().GetBucket()[2].GetCumulativeCount()) // label: value
	})
}

func TestSink_MultiDimensionReport(t *testing.T) {
	s := NewSink(prometheus.NewRegistry())
	metrics.RegisterMetricsSink(s)
	dimensions := []*metrics.Dimension{{Name: "dim_1", Value: "val_1"}, {Name: "dim_2", Value: "val_2"}}
	t.Run("multi dimension counter", func(t *testing.T) {
		var metric []*metrics.Metrics
		metric = append(metric, metrics.NewMetrics("multi_dim_counter", 0.01, metrics.PolicySUM))
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter", dimensions, metric)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, "trpc_counter_multi_dim_counter_total", data[0].GetName())
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[0].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[0].GetMetric()[0].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[0].GetMetric()[0].GetLabel()[2].GetValue())
		assert.Equal(t, "val_2", data[0].GetMetric()[0].GetLabel()[3].GetValue())
	})
	t.Run("multi dimension gauge", func(t *testing.T) {
		var metric []*metrics.Metrics
		metric = append(metric, metrics.NewMetrics("multi_dim_gauge", 0.01, metrics.PolicySET))
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_gauge", dimensions, metric)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 2, len(data))
		assert.Equal(t, "trpc_gauge_multi_dim_gauge", data[1].GetName())
		assert.Equal(t, "multi_dim_gauge_multi_dim_gauge", data[1].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "gauge", data[1].GetMetric()[0].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[1].GetMetric()[0].GetLabel()[2].GetValue())
		assert.Equal(t, "val_2", data[1].GetMetric()[0].GetLabel()[3].GetValue())
	})
	t.Run("multi dimension histogram", func(t *testing.T) {
		var metric []*metrics.Metrics
		metric = append(metric, metrics.NewMetrics("multi_dim_histogram", 0.01, metrics.PolicyHistogram))
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_histogram", dimensions, metric)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 3, len(data))
		assert.Equal(t, "trpc_histogram_multi_dim_histogram", data[2].GetName())
		assert.Equal(t, "val_1", data[2].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "val_2", data[2].GetMetric()[0].GetLabel()[1].GetValue())
		assert.Equal(t, uint64(1),
			data[2].GetMetric()[0].GetHistogram().GetBucket()[2].GetCumulativeCount()) // value: 0.01
	})
	t.Run("two multi-dimension counter report with different dimensions", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 0.01, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}, {Name: "dim_2", Value: "val_2"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 0.01, metrics.PolicySUM),
			},
		)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, 2, len(data[0].GetMetric()))
		assert.Equal(t, "trpc_counter_multi_dim_counter_total", data[0].GetName())
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[0].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[0].GetMetric()[1].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[0].GetMetric()[1].GetLabel()[2].GetValue())
		assert.Equal(t, "val_2", data[0].GetMetric()[1].GetLabel()[3].GetValue())
	})
	t.Run("multiple multi-dimension counter report with different dimensions", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_2"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}, {Name: "dim_2", Value: "val_2"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_11"}, {Name: "dim_2", Value: "val_22"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter2",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter2", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter2",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_11"}, {Name: "dim_2", Value: "val_22"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter2", 1, metrics.PolicySUM),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("no_meaning",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_11"}, {Name: "dim_2", Value: "val_22"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_counter2", 1, metrics.PolicySUM),
			},
		)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 2, len(data))
		// data 0
		assert.Equal(t, "trpc_counter_multi_dim_counter2_total", data[0].GetName())
		// data 0 metric 0
		assert.Equal(t, "multi_dim_counter2_multi_dim_counter2", data[0].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[0].GetMetric()[0].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[0].GetMetric()[0].GetLabel()[2].GetValue())
		assert.Equal(t, float64(1), data[0].GetMetric()[0].GetCounter().GetValue())
		// data 0 metric 0
		assert.Equal(t, "multi_dim_counter2_multi_dim_counter2", data[0].GetMetric()[1].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[0].GetMetric()[1].GetLabel()[1].GetValue())
		assert.Equal(t, "val_11", data[0].GetMetric()[1].GetLabel()[2].GetValue())
		assert.Equal(t, "val_22", data[0].GetMetric()[1].GetLabel()[3].GetValue())
		assert.Equal(t, float64(2), data[0].GetMetric()[1].GetCounter().GetValue())
		// data 1
		assert.Equal(t, "trpc_counter_multi_dim_counter_total", data[1].GetName())
		// data 1 metric 0
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[1].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[1].GetMetric()[0].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[1].GetMetric()[0].GetLabel()[2].GetValue())
		// data 1 metric 1
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[1].GetMetric()[1].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[1].GetMetric()[1].GetLabel()[1].GetValue())
		assert.Equal(t, "val_2", data[1].GetMetric()[1].GetLabel()[2].GetValue())
		// data 1 metric 2
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[1].GetMetric()[2].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[1].GetMetric()[2].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[1].GetMetric()[2].GetLabel()[2].GetValue())
		assert.Equal(t, "val_2", data[1].GetMetric()[2].GetLabel()[3].GetValue())
		// data 1 metric 3
		assert.Equal(t, "multi_dim_counter_multi_dim_counter", data[1].GetMetric()[3].GetLabel()[0].GetValue())
		assert.Equal(t, "counter", data[1].GetMetric()[3].GetLabel()[1].GetValue())
		assert.Equal(t, "val_11", data[1].GetMetric()[3].GetLabel()[2].GetValue())
		assert.Equal(t, "val_22", data[1].GetMetric()[3].GetLabel()[3].GetValue())
	})
	t.Run("two multi-dimension gauge report with different dimensions", func(t *testing.T) {
		s := NewSink(prometheus.NewRegistry())
		metrics.RegisterMetricsSink(s)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_gauge",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_gauge", 0.01, metrics.PolicySET),
			},
		)
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_gauge",
			[]*metrics.Dimension{{Name: "dim_1", Value: "val_1"}, {Name: "dim_2", Value: "val_2"}},
			[]*metrics.Metrics{
				metrics.NewMetrics("multi_dim_gauge", 0.01, metrics.PolicySET),
			},
		)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 1, len(data))
		assert.Equal(t, 2, len(data[0].GetMetric()))
		assert.Equal(t, "trpc_gauge_multi_dim_gauge", data[0].GetName())
		assert.Equal(t, "multi_dim_gauge_multi_dim_gauge", data[0].GetMetric()[0].GetLabel()[0].GetValue())
		assert.Equal(t, "gauge", data[0].GetMetric()[1].GetLabel()[1].GetValue())
		assert.Equal(t, "val_1", data[0].GetMetric()[1].GetLabel()[2].GetValue())
		assert.Equal(t, "val_2", data[0].GetMetric()[1].GetLabel()[3].GetValue())
	})
}

func TestSink_convertMetricName(t *testing.T) {
	type args struct {
		origin string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				origin: "valid",
			},
			want: "valid",
		},
		{
			name: "中文",
			args: args{
				origin: "中文",
			},
			want: "zhong_wen",
		},
		{
			name: "counter中文",
			args: args{
				origin: "counter中文",
			},
			want: "counterzhong_wen",
		},
		{
			name: "中文counter",
			args: args{
				origin: "中文counter",
			},
			want: "zhong_wen_counter",
		},
		{
			name: "中文counter-《》",
			args: args{
				origin: "中文counter-《》",
			},
			want: "zhong_wen_counter_300A_300B",
		},
		{
			name: "合集卡片没有合集字段",
			args: args{
				origin: "合集卡片没有合集字段",
			},
			want: "he_ji_ka_pian_mei_you_he_ji_zi_duan",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSink(prometheus.DefaultRegisterer)
			if got := s.convertMetricName(tt.args.origin); got != tt.want {
				t.Errorf("convertMetricName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimitCardinalityGatherer_Gather(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics.RegisterMetricsSink(NewSink(reg))
	g := metric.LimitCardinalityGatherer{
		Gatherer:       reg,
		PerMetirclimit: 10,
	}
	for i := 0; i < 20; i++ {
		metrics.IncrCounter(fmt.Sprintf("trpc.counter%d", i), 1)
	}
	metrics, err := g.Gather()
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Len(t, metrics[0].Metric, g.PerMetirclimit)
}

func TestSink_SkipReport(t *testing.T) {
	s := NewSink(prometheus.NewRegistry())
	metrics.RegisterMetricsSink(s)
	dimensions := []*metrics.Dimension{{Name: "dim_1", Value: "val_1"}, {Name: "dim_2", Value: "val_2"}}
	t.Run("skip report", func(t *testing.T) {
		var m []*metrics.Metrics
		m = append(m, metrics.NewMetrics("multi_dim_counter", 0.01, metrics.PolicySUM))
		_ = metrics.ReportMultiDimensionMetricsX("multi_dim_counter", dimensions, m, WithNoSink)
		data, err := s.registerer.(prometheus.Gatherer).Gather()
		assert.NoError(t, err)
		str, _ := json.Marshal(data)
		t.Logf("%s", str)
		assert.Equal(t, 0, len(data))
	})
}

// BenchmarkMetricCounter_New
// BenchmarkMetricCounter_New-12    	  141633	      8068 ns/op	    1933 B/op	      39 allocs/op
// 2021-09-10
// BenchmarkMetricCounter_New-12    	  652230	      2284 ns/op	     719 B/op	      16 allocs/op
func BenchmarkMetricCounter_New(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		metrics.IncrCounter(fmt.Sprintf("trpc.counter%d", i), 1)
	}
}

// BenchmarkMetricCounter_Lookup
// BenchmarkMetricCounter_Lookup-12    	 5630991	       190.8 ns/op	      40 B/op	       2 allocs/op
func BenchmarkMetricCounter_Lookup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		metrics.IncrCounter("trpc.counter", 1)
	}
}

// BenchmarkMetricCounter_Inc
// BenchmarkMetricCounter_Inc-12    	 7308434	       161.4 ns/op	      40 B/op	       2 allocs/op
func BenchmarkMetricCounter_Inc(b *testing.B) {
	c := metrics.Counter("trpc.counter")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Incr()
	}
}

// BenchmarkPrometheusCounter_Inc
// BenchmarkPrometheusCounter_Inc-12    	188227936	         6.429 ns/op	       0 B/op	       0 allocs/op
func BenchmarkPrometheusCounter_Inc(b *testing.B) {
	b.ReportAllocs()
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "trpc_counter",
		Help: "trpc counter",
	})
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

// BenchmarkPrometheusCounterVec_Inc
// BenchmarkPrometheusCounterVec_Inc-12    	 8223290	       122.5 ns/op	      32 B/op	       1 allocs/op
func BenchmarkPrometheusCounterVec_Inc(b *testing.B) {
	b.ReportAllocs()
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trpc_counter",
		Help: "trpc counter",
	},
		[]string{"_name", "_type"})
	for i := 0; i < b.N; i++ {
		c.WithLabelValues("trpc.counter", "counter").Inc()
	}
}

// BenchmarkPrometheusCounterVec_CurreyInc
// BenchmarkPrometheusCounterVec_CurreyInc-12    	16325986	        74.12 ns/op	       0 B/op	       0 allocs/op
func BenchmarkPrometheusCounterVec_CurreyInc(b *testing.B) {
	b.ReportAllocs()
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trpc_counter",
		Help: "trpc counter",
	},
		[]string{"_name", "_type"}).MustCurryWith(
		prometheus.Labels{"_name": "trpc.counter", "_type": "counter"})
	for i := 0; i < b.N; i++ {
		c.WithLabelValues().Inc()
	}
}

// BenchmarkPrometheusCounterVec_WithInc
// BenchmarkPrometheusCounterVec_WithInc-12    	174514418	         6.310 ns/op	       0 B/op	       0 allocs/op
func BenchmarkPrometheusCounterVec_WithInc(b *testing.B) {
	b.ReportAllocs()
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trpc_counter",
		Help: "trpc counter",
	},
		[]string{"_name", "_type"}).WithLabelValues("trpc.counter", "counter")
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

// BenchmarkMetricReport_WithoutSink
// BenchmarkMetricReport_WithoutSink-10    	465907614	         2.548 ns/op	       0 B/op	       0 allocs/op
func BenchmarkMetricReport_WithoutSink(b *testing.B) {
	b.ReportAllocs()
	s := NewSink(prometheus.NewRegistry())
	record := metrics.Record{Name: "skip_sink"}
	for i := 0; i < b.N; i++ {
		_ = s.Report(record, WithNoSink)
	}
}
