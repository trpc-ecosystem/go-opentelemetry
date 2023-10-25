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

package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(BatchProcessCounter)
	prometheus.MustRegister(DeferredProcessCounter)
	prometheus.MustRegister(LogsLevelTotal)
}

var (
	// BatchProcessCounter batch processor counter
	BatchProcessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "opentelemetry_sdk",
			Name:      "batch_process_counter",
			Help:      "Batch Process Counter",
		},
		[]string{"status", "telemetry"},
	)
	// DeferredProcessCounter deferred processor counter
	DeferredProcessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "opentelemetry_sdk",
			Name:      "defereed_process_counter",
			Help:      "deferred Process Counter",
		},
		[]string{"status", "telemetry"},
	)
	// LogsLevelTotal logs level counter
	LogsLevelTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "opentelemetry_sdk",
			Name:      "logs_level_total",
			Help:      "Logs Level Total",
		},
		[]string{"level"},
	)
)
