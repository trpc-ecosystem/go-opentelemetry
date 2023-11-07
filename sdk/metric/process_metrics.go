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

package metric

import (
	"github.com/prometheus/client_golang/prometheus"

	pkgruntime "trpc.group/trpc-go/trpc-opentelemetry/pkg/runtime"
)

var (
	cpuCores = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: "process",
			Name:      "cpu_cores",
			Help:      "Total CPU cores",
		},
	)
	memoryQuota = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: "process",
			Name:      "memory_quota",
			Help:      "Total Memory Quota",
		},
	)
	memoryUsage = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: "process",
			Name:      "memory_usage",
			Help:      "Usage Memory",
		}, func() float64 {
			usageMemory, _ := pkgruntime.MemoryUsage()
			return float64(usageMemory)
		})
)

func init() {
	prometheus.MustRegister(cpuCores)
	cpuQuota, _ := pkgruntime.CPUQuota()
	cpuCores.Set(cpuQuota)

	prometheus.MustRegister(memoryQuota)
	totalMemory, _ := pkgruntime.MemoryQuota()

	prometheus.MustRegister(memoryUsage)
	memoryQuota.Set(float64(totalMemory))
}
