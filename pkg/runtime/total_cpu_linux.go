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

//go:build linux
// +build linux

package runtime

import (
	"runtime"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/cgroups"
)

// CPUQuota returns the CPU quota applied with the CPU cgroup controller.
// It is a result of `cpu.cfs_quota_us / cpu.cfs_period_us`.
// If it is not in container env, return the number of cpu on host.
// This implementation is meant for linux
func CPUQuota() (float64, error) {
	if !ProcessInContainer() {
		// not in container
		return float64(runtime.NumCPU()), nil
	}
	// uses cgroups to determine cpu quota.
	cg, err := cgroups.NewCGroupsForCurrentProcess()
	if err != nil {
		return float64(runtime.NumCPU()), err
	}
	cpuQuota, defined, err := cg.CPUQuota()
	if err != nil || !defined {
		return float64(runtime.NumCPU()), err
	}
	return cpuQuota, nil
}
