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

//go:build linux
// +build linux

package cgroups

import (
	"bufio"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// _cgroupFSType is the Linux CGroup file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupFSType = "cgroup"
	// _cgroupSubsysCPU is the CPU CGroup subsystem.
	_cgroupSubsysCPU = "cpu"
	// _cgroupSubsysCPUAcct is the CPU accounting CGroup subsystem.
	_cgroupSubsysCPUAcct = "cpuacct"
	// _cgroupSubsysCPUSet is the CPUSet CGroup subsystem.
	_cgroupSubsysCPUSet = "cpuset"
	// _cgroupSubsysMemory is the Memory CGroup subsystem.
	_cgroupSubsysMemory = "memory"

	_cgroupMemoryLimitBytes = "memory.limit_in_bytes"

	// _cgroupv2MemoryMax is the file name for the CGroup-V2 Memory max
	// parameter.
	_cgroupv2MemoryMax = "memory.max"
	// _cgroupFSType is the Linux CGroup-V2 file system type used in
	// `/proc/$PID/mountinfo`.
	_cgroupv2FSType = "cgroup2"
)

const (
	_procPathCGroup     = "/proc/self/cgroup"
	_procPathMountInfo  = "/proc/self/mountinfo"
	_cgroupv2MountPoint = "/sys/fs/cgroup"
)

// CGroups is a map that associates each CGroup with its subsystem name.
type CGroups map[string]*CGroup

// NewCGroups returns a new *CGroups from given `mountinfo` and `cgroup` files
// under for some process under `/proc` file system (see also proc(5) for more
// information).
func NewCGroups(procPathMountInfo, procPathCGroup string) (CGroups, error) {
	cgroupSubsystems, err := parseCGroupSubsystems(procPathCGroup)
	if err != nil {
		return nil, err
	}

	cgroups := make(CGroups)
	newMountPoint := func(mp *MountPoint) error {
		if mp.FSType != _cgroupFSType {
			return nil
		}

		for _, opt := range mp.SuperOptions {
			subsys, exists := cgroupSubsystems[opt]
			if !exists {
				continue
			}

			cgroupPath, err := mp.Translate(subsys.Name)
			if err != nil {
				return err
			}
			cgroups[opt] = NewCGroup(cgroupPath)
		}

		return nil
	}

	if err := parseMountInfo(procPathMountInfo, newMountPoint); err != nil {
		return nil, err
	}
	return cgroups, nil
}

// NewCGroupsForCurrentProcess returns a new *CGroups instance for the current
// process.
func NewCGroupsForCurrentProcess() (CGroups, error) {
	return NewCGroups(_procPathMountInfo, _procPathCGroup)
}

// MemoryQuota returns the total memory limit of the process
// It is a result of `memory.limit_in_bytes`. If the value of
// `memory.limit_in_bytes` was not set (-1) or (0 || 9223372036854771712 || 9223372036854775807 || 18446744073709551615),
// the method returns `(0, false, nil)`.
func (cg CGroups) MemoryQuota() (int64, bool, error) {
	memCGroup, exists := cg[_cgroupSubsysMemory]
	if !exists {
		return -1, false, nil
	}
	memLimitBytes, err := memCGroup.readDecimal(_cgroupMemoryLimitBytes)
	if err != nil {
		return -1, false, err
	}
	if memLimitBytes.IsNegative() || memLimitBytes.IsZero() {
		return -1, false, nil
	}
	b := memLimitBytes.BigInt().Uint64()
	// If memory is not defined or is set to unlimitedMemorySize (v1 unset),
	// we fallback to /proc/meminfo.
	// the default value of cgroup memory is set to PAGE_COUNTER_MAX, which is LONG_MAX/PAGE_SIZE on 64-bit platform.
	// when the platform`s PAGE_SIZE is different ,the default value of cgroup memory is different, such as 4k and 64k
	// has the different default values:
	// 0x7ffffffffffff000(pagesize=4k), which is 9223372036854771712
	// 0x7fffffffffff0000(pagesize= 64k), which is 9223372036854775807
	// reference:
	// https://tracker.ceph.com/issues/42059
	if b == 9223372036854771712 || b == 9223372036854775807 || b == math.MaxUint64 {
		return -1, false, nil
	}
	return int64(b), true, nil
}

// IsCGroupV2 returns true if the system supports and uses cgroup2.
// It gets the required information for deciding from mountinfo file.
func IsCGroupV2() (bool, error) {
	return isCGroupV2(_procPathMountInfo)
}

func isCGroupV2(procPathMountInfo string) (bool, error) {
	isV2 := false
	newMountPoint := func(mp *MountPoint) error {
		if mp.FSType == _cgroupv2FSType && mp.MountPoint == _cgroupv2MountPoint {
			isV2 = true
		}
		return nil
	}
	if err := parseMountInfo(procPathMountInfo, newMountPoint); err != nil {
		return false, err
	}
	return isV2, nil
}

// MemoryQuotaV2 returns the total memory limit of the process
// It is a result of cgroupv2 `memory.max`. If the value of
// `memory.max` was not set (max), the method returns `(-1, false, nil)`.
func MemoryQuotaV2() (int64, bool, error) {
	return memoryQuotaV2(_cgroupv2MountPoint, _cgroupv2MemoryMax)
}

func memoryQuotaV2(cgroupv2MountPoint, cgroupv2MemoryMax string) (int64, bool, error) {
	memoryMaxParams, err := os.Open(filepath.Clean(filepath.Join(cgroupv2MountPoint, cgroupv2MemoryMax)))
	if err != nil {
		if os.IsNotExist(err) {
			return -1, false, nil
		}
		return -1, false, err
	}
	scanner := bufio.NewScanner(memoryMaxParams)
	if scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "max" {
			return -1, false, nil
		}
		max, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return -1, false, err
		}
		return max, true, nil
	}
	if err := scanner.Err(); err != nil {
		return -1, false, err
	}
	return -1, false, io.ErrUnexpectedEOF
}
