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

package metrics

import (
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// declare server instrument
var (
	serverStartedCounter   metric.Int64Counter
	serverHandledCounter   metric.Int64Counter
	serverHandledHistogram metric.Float64Histogram
)

// declare client instrument
var (
	clientStartedCounter   metric.Int64Counter
	clientHandledCounter   metric.Int64Counter
	clientHandledHistogram metric.Float64Histogram
)

func init() {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("")

	serverStartedCounter, _ = meter.Int64Counter("trpc_server_started_total",
		metric.WithDescription("Total number of RPCs started on the server."))
	serverHandledCounter, _ = meter.Int64Counter("trpc_server_handled_total",
		metric.WithDescription("Total number of RPCs handled on the server."))

	serverHandledHistogram, _ = meter.Float64Histogram("trpc_server_handling_seconds",
		metric.WithDescription("Histogram of response latency (seconds) of tRPC that "+
			"had been application-level handled by the server."))

	clientStartedCounter, _ = meter.Int64Counter("trpc_client_started_total",
		metric.WithDescription("Total number of RPCs started on the client."))
	clientHandledCounter, _ = meter.Int64Counter("trpc_client_handled_total",
		metric.WithDescription("Total number of RPCs handled on the client."))

	clientHandledHistogram, _ = meter.Float64Histogram("trpc_client_handling_seconds",
		metric.WithDescription("Histogram of response latency (seconds) of tRPC that "+
			"had been application-level handled by the client."))
}
