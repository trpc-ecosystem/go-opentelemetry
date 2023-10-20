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

//go:build !linux
// +build !linux

package runtime

import (
	"fmt"
)

var (
	errTotalMemoryNotAvailable = fmt.Errorf("reading cgroups total memory is available only on linux")
	errUsageMemoryNotAvailable = fmt.Errorf("reading cgroups usage memory is available only on linux")
)

// MemoryQuota returns total available memory.
// This is non-Linux version that returns -1 and errTotalMemoryNotAvailable.
func MemoryQuota() (int64, error) {
	return -1, errTotalMemoryNotAvailable
}

// MemoryUsage returns usage memory.
// This is non-Linux version that returns -1 and errUsageMemoryNotAvailable.
func MemoryUsage() (int64, error) {
	return -1, errUsageMemoryNotAvailable
}
