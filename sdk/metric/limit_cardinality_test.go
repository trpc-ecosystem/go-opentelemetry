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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestLimitCardinalityCollector_Collect(t *testing.T) {
	lc := &LimitCardinalityCollector{
		metricCollector: clientStartedCounter,
		desc:            "clientStartedCounter",
		limit:           2,
	}
	clientStartedCounter.Reset()
	reg := prometheus.NewRegistry()
	reg.MustRegister(lc)
	t.Run("under limit", func(t *testing.T) {
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m1")
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m2")
		metrics, err := reg.Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(metrics))
		assert.Equal(t, 2, len(metrics[0].GetMetric()))
	})
	t.Run("over limit", func(t *testing.T) {
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m1").Add(2)
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m2").Add(2)
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m3").Add(2)
		metrics, err := reg.Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(metrics))
		assert.Equal(t, 2, len(metrics[0].GetMetric()))
	})
}

func TestLimitCardinalityGatherer_Gather(t *testing.T) {
	reg := prometheus.NewRegistry()
	gatherer := &LimitCardinalityGatherer{
		Gatherer:         reg,
		PerMetirclimit:   2,
		TotalMetricLimit: 3,
	}
	clientStartedCounter.Reset()
	clientHandledCounter.Reset()
	serverStartedCounter.Reset()
	reg.MustRegister(clientStartedCounter)
	reg.MustRegister(clientHandledCounter)
	reg.MustRegister(serverStartedCounter)
	t.Run("under limit", func(t *testing.T) {
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m1")
		metrics, err := gatherer.Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(metrics))
		var total int
		for _, v := range metrics {
			total += len(v.GetMetric())
		}
		assert.Equal(t, 1, total)
	})
	t.Run("over limit for single metric", func(t *testing.T) {
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m1")
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m2")
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m3")
		metrics, err := gatherer.Gather()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(metrics))
		var total int
		for _, v := range metrics {
			total += len(v.GetMetric())
		}
		assert.Equal(t, gatherer.PerMetirclimit, total)
	})
	t.Run("over limit for all metrics", func(t *testing.T) {
		clientStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m2")
		clientHandledCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m1", "code", "code_type", "code_desc")
		serverStartedCounter.WithLabelValues("trpc", "s1", "m1", "s1", "m2")
		metrics, err := gatherer.Gather()
		assert.NoError(t, err)
		assert.Equal(t, 3, len(metrics))
		var total int
		for _, v := range metrics {
			total += len(v.GetMetric())
		}
		assert.Equal(t, gatherer.TotalMetricLimit, total)
	})
}
