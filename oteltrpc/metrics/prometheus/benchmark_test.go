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

package prometheus

import (
	"math/rand"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getLabelsAndValues(n, m int) ([]string, []string) {
	var labels, values []string
	for i := 0; i < n; i++ {
		s := randStr(m)
		labels = append(labels, s)
		values = append(values, s)
	}
	return labels, values
}

// BenchmarkSetGauge1010 10 dimensions
func BenchmarkSetGauge1010(b *testing.B) {
	labels, values := getLabelsAndValues(10, 10)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.setGauge("", "bench_mark", 1, labels, values)
	}
}

// BenchmarkSetGauge2020 20 dimensions
func BenchmarkSetGauge2020(b *testing.B) {
	labels, values := getLabelsAndValues(20, 20)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.setGauge("", "bench_mark", 1, labels, values)
	}
}

// BenchmarkSetGauge100100 100*100 dimensions
func BenchmarkSetGauge100100(b *testing.B) {
	labels, values := getLabelsAndValues(100, 100)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.setGauge("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkSetGauge10001000(b *testing.B) {
	labels, values := getLabelsAndValues(1000, 1000)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.setGauge("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkIncrCounter1010(b *testing.B) {
	labels, values := getLabelsAndValues(10, 10)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.incrCounter("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkIncrCounter2020(b *testing.B) {
	labels, values := getLabelsAndValues(20, 20)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.incrCounter("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkIncrCounter100100(b *testing.B) {
	labels, values := getLabelsAndValues(100, 100)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.incrCounter("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkIncrCounter10001000(b *testing.B) {
	labels, values := getLabelsAndValues(1000, 1000)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.incrCounter("", "bench_mark", 1, labels, values)
	}
}

func BenchmarkObserveHistogram1010(b *testing.B) {
	labels, values := getLabelsAndValues(10, 10)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.observeHistogram("bench_mark", 1, labels, values)
	}
}

func BenchmarkObserveHistogram2020(b *testing.B) {
	labels, values := getLabelsAndValues(20, 20)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.observeHistogram("bench_mark", 1, labels, values)
	}
}

func BenchmarkObserveHistogram100100(b *testing.B) {
	labels, values := getLabelsAndValues(100, 100)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.observeHistogram("bench_mark", 1, labels, values)
	}
}

func BenchmarkObserveHistogram10001000(b *testing.B) {
	labels, values := getLabelsAndValues(1000, 1000)
	reg := prometheus.NewRegistry()
	s := NewSink(reg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.observeHistogram("bench_mark", 1, labels, values)
	}
}
