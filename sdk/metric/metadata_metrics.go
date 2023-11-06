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

// Package metric
package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"

	opentelemetry "trpc.group/trpc-go/go-opentelemetry"
)

var (
	// sdkMetadata
	sdkMetadata = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "opentelemetry_sdk_metadata",
			Help: "opentelemetry sdk metadata version",
		},
		[]string{"sdk_version", "otel_version"})

	// serverMetadata
	serverMetadata = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: "opentelemetry_sdk",
			Name:      "server_metadata",
			Help:      "server metadata",
		},
		[]string{"server_owner", "cmdb_id"})
)

// updateServerMetadata update server metadata
func updateServerMetadata(serverOwner, cmdbID string) {
	serverMetadata.Reset()
	serverMetadata.WithLabelValues(serverOwner, cmdbID).Set(1)
}

func init() {
	prometheus.MustRegister(sdkMetadata)
	sdkMetadata.WithLabelValues(opentelemetry.Version(), otel.Version()).Set(1)

	prometheus.MustRegister(serverMetadata)
}
