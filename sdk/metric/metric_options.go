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

package metric

import prom "github.com/prometheus/client_golang/prometheus"

// A CounterOption lets you add options to Counter metrics using With* funcs.
type CounterOption func(*prom.CounterOpts)

type counterOptions []CounterOption

func (co counterOptions) apply(o prom.CounterOpts) prom.CounterOpts {
	for _, f := range co {
		f(&o)
	}
	return o
}

// WithConstLabels allows you to add ConstLabels to Counter metrics.
func WithConstLabels(labels prom.Labels) CounterOption {
	return func(o *prom.CounterOpts) {
		o.ConstLabels = labels
	}
}

// A HistogramOption lets you add options to Histogram metrics using With*funcs.
type HistogramOption func(*prom.HistogramOpts)

// WithHistogramBuckets allows you to specify custom bucket ranges for histograms if EnableHandlingTimeHistogram is on.
func WithHistogramBuckets(buckets []float64) HistogramOption {
	return func(o *prom.HistogramOpts) { o.Buckets = buckets }
}

// WithHistogramConstLabels allows you to add custom ConstLabels to
// histograms metrics.
func WithHistogramConstLabels(labels prom.Labels) HistogramOption {
	return func(o *prom.HistogramOpts) {
		o.ConstLabels = labels
	}
}
