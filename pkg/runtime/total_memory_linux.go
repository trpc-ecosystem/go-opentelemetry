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
	procmeminfo "github.com/guillermo/go.procmeminfo"
	"github.com/shirou/gopsutil/v3/mem"

	cgroups "trpc-system/go-opentelemetry/pkg/cgroups"
	cgroupsv2 "trpc-system/go-opentelemetry/pkg/cgroups/cgroupsv2"
)

// MemoryQuota returns total available memory.
// This implementation is meant for linux
func MemoryQuota() (int64, error) {
	var memoryQuota int64
	var defined bool
	var err error

	isV2, err := cgroupsv2.IsCGroupV2()
	if err != nil {
		return 0, err
	}

	if isV2 {
		memoryQuota, defined, err = cgroupsv2.MemoryQuotaV2()
		if err != nil {
			return 0, err
		}
	} else {
		cgv1, err := cgroupsv2.NewCGroupsForCurrentProcess()
		if err != nil {
			return 0, err
		}
		memoryQuota, defined, err = cgv1.MemoryQuota()
		if err != nil {
			return 0, err
		}
	}

	if !defined {
		totalMem, err := readMemInfo()
		if err != nil {
			return 0, err
		}
		return int64(totalMem), nil
	}

	return memoryQuota, nil
}

// MemoryUsage returns usage memory.
// This implementation is meant for linux
func MemoryUsage() (int64, error) {
	if !ProcessInContainer() {
		memInfo := &procmeminfo.MemInfo{}
		if err := memInfo.Update(); err != nil {
			return 0, err
		}
		return int64(memInfo.Used()), nil
	}
	// uses cgroups to determine available memory.
	cgroups, err := cgroups.NewCGroupsForCurrentProcess()
	if err != nil {
		return 0, err
	}
	memoryUsage, defined, err := cgroups.MemoryUsage()
	if err != nil || !defined {
		return 0, err
	}
	return memoryUsage, nil
}

// readMemInfo returns the total memory
// supports in linux, darwin and windows
func readMemInfo() (uint64, error) {
	vmStat, err := mem.VirtualMemory()
	return vmStat.Total, err
}
