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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCGroups(t *testing.T) {
	cgroupsProcCGroupPath := filepath.Join(testDataProcPath, "cgroups", "cgroup")
	cgroupsProcMountInfoPath := filepath.Join(testDataProcPath, "cgroups", "mountinfo")

	testTable := []struct {
		subsys string
		path   string
	}{
		{_cgroupSubsysCPU, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUAcct, "/sys/fs/cgroup/cpu,cpuacct"},
		{_cgroupSubsysCPUSet, "/sys/fs/cgroup/cpuset"},
		{_cgroupSubsysMemory, "/sys/fs/cgroup/memory/large"},
	}

	cgroups, err := NewCGroups(cgroupsProcMountInfoPath, cgroupsProcCGroupPath)
	assert.Equal(t, len(testTable), len(cgroups))
	assert.NoError(t, err)

	for _, tt := range testTable {
		cgroup, exists := cgroups[tt.subsys]
		assert.True(t, exists, "%q expected to present in `cgroups`", tt.subsys)
		assert.Equal(t, tt.path, cgroup.path, "%q expected for `cgroups[%q].path`, got %q", tt.path, tt.subsys,
			cgroup.path)
	}
}

func TestNewCGroupsWithErrors(t *testing.T) {
	testTable := []struct {
		mountInfoPath string
		cgroupPath    string
	}{
		{"non-existing-file", "/dev/null"},
		{"/dev/null", "non-existing-file"},
		{
			"/dev/null",
			filepath.Join(testDataProcPath, "invalid-cgroup", "cgroup"),
		},
		{
			filepath.Join(testDataProcPath, "invalid-mountinfo", "mountinfo"),
			"/dev/null",
		},
		{
			filepath.Join(testDataProcPath, "untranslatable", "mountinfo"),
			filepath.Join(testDataProcPath, "untranslatable", "cgroup"),
		},
	}

	for _, tt := range testTable {
		cgroups, err := NewCGroups(tt.mountInfoPath, tt.cgroupPath)
		assert.Nil(t, cgroups)
		assert.Error(t, err)
	}
}

func TestCGroupsCPUQuota(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   int64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "undefined",
			expectedQuota:   int64(-1.0),
			expectedDefined: false,
			shouldHaveError: false,
		},
	}

	cgroups := make(CGroups)

	quota, defined, err := cgroups.MemoryQuota()
	assert.Equal(t, int64(-1), quota, "nonexistent")
	assert.False(t, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	for _, tt := range testTable {
		cgroupPath := filepath.Join(testDataCGroupsPath, tt.name)
		cgroups[_cgroupSubsysCPU] = NewCGroup(cgroupPath)

		quota, defined, err := cgroups.MemoryQuota()
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupsIsCGroupV2(t *testing.T) {
	testTable := []struct {
		name            string
		expectedIsV2    bool
		shouldHaveError bool
	}{
		{
			name:            "cgroupv1",
			expectedIsV2:    false,
			shouldHaveError: false,
		},
		{
			name:            "cgroupv1v2",
			expectedIsV2:    false,
			shouldHaveError: false,
		},
		{
			name:            "cgroupv2",
			expectedIsV2:    true,
			shouldHaveError: false,
		},
		{
			name:            "nonexistent",
			expectedIsV2:    false,
			shouldHaveError: true,
		},
	}

	for _, tt := range testTable {
		mountInfoPath := filepath.Join(testDataProcPath, "v2", tt.name, "mountinfo")
		isV2, err := isCGroupV2(mountInfoPath)

		assert.Equal(t, tt.expectedIsV2, isV2, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func TestCGroupsMemoryQuotaV2(t *testing.T) {
	testTable := []struct {
		name            string
		expectedQuota   int64
		expectedDefined bool
		shouldHaveError bool
	}{
		{
			name:            "memory",
			expectedQuota:   int64(250000000),
			expectedDefined: true,
			shouldHaveError: false,
		},
		{
			name:            "undefined",
			expectedQuota:   int64(-1),
			expectedDefined: false,
			shouldHaveError: false,
		},
		{
			name:            "invalid",
			expectedQuota:   int64(-1),
			expectedDefined: false,
			shouldHaveError: true,
		},
		{
			name:            "empty",
			expectedQuota:   int64(-1),
			expectedDefined: false,
			shouldHaveError: true,
		},
	}

	quota, defined, err := memoryQuotaV2("nonexistent", "nonexistent")
	assert.Equal(t, int64(-1), quota, "nonexistent")
	assert.Equal(t, false, defined, "nonexistent")
	assert.NoError(t, err, "nonexistent")

	cgroupBasePath := filepath.Join(testDataCGroupsPath, "v2")
	for _, tt := range testTable {
		cgroupPath := filepath.Join(cgroupBasePath, tt.name)
		quota, defined, err := memoryQuotaV2(cgroupPath, "memory.max")
		assert.Equal(t, tt.expectedQuota, quota, tt.name)
		assert.Equal(t, tt.expectedDefined, defined, tt.name)

		if tt.shouldHaveError {
			assert.Error(t, err, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}
