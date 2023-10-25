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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"trpc.group/trpc-go/trpc-go"
)

var (
	// opentelemetrySDKPanicTotal metric sdk panic
	opentelemetrySDKPanicTotal = promauto.NewCounter(prometheus.CounterOpts{
		Subsystem: "opentelemetry_sdk",
		Name:      "panic_total",
		Help:      "opentelemetry sdk panic total",
	})

	exportSpansBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "opentelemetry_sdk",
		Name:      "export_spans_bytes",
		Help:      "Export Spans Bytes",
		Buckets:   []float64{128, 256, 1024, 10240, 102400, 1024_000, 10240_000, 102400_000, 1024_000_000},
	})

	exportLogsBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "opentelemetry_sdk",
		Name:      "export_logs_bytes",
		Help:      "Export Logs Bytes",
		Buckets:   []float64{128, 256, 1024, 10240, 102400, 1024_000, 10240_000, 102400_000, 1024_000_000},
	})

	requestBodyBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "rpc",
		Name:      "request_body_bytes",
		Help:      "Request Body Bytes",
		Buckets:   []float64{1024, 10240, 102400, 1024_000, 10240_000},
	})

	requestMetaDataBodyBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "rpc",
		Name:      "request_metadata_bytes",
		Help:      "Request Metadata Bytes",
		Buckets:   []float64{1024, 10240, 102400, 1024_000, 10240_000},
	})

	trpcSDKMetadata = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "opentelemetry_trpc_metadata",
		Help: "opentelemetry trpc metadata version",
	}, []string{
		"trpc_version",
	})
)

// IncrSDKPanicTotal report sdk panic
func IncrSDKPanicTotal() {
	opentelemetrySDKPanicTotal.Inc()
}

// ObserveExportSpansBytes observe span bytes
func ObserveExportSpansBytes(s int) {
	exportSpansBytes.Observe(float64(s))
}

// ObserveExportLogsBytes observe log bytes
func ObserveExportLogsBytes(s int) {
	exportLogsBytes.Observe(float64(s))
}

// ObserveRequestBodyBytes observe request body bytes
func ObserveRequestBodyBytes(s int) {
	requestBodyBytes.Observe(float64(s))
}

// ObserveRequestMetaDataBytes observe request metadata bytes
func ObserveRequestMetaDataBytes(s int) {
	requestMetaDataBodyBytes.Observe(float64(s))
}

// MonitorTRPCSDKMeta monitor trpc sdk meta
func MonitorTRPCSDKMeta() {
	trpcSDKMetadata.WithLabelValues(trpc.Version()).Set(1)
}
