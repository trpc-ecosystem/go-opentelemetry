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

//go:build !linux
// +build !linux

package runtime

import (
	"fmt"
)

var (
	errTotalCPUNotAvailable = fmt.Errorf("reading cgroups total cpu is available only on linux")
)

// CPUQuota returns total available cpu thread count.
// This is non-Linux version that returns -1 and errTotalCPUNotAvailable.
func CPUQuota() (float64, error) {
	return -1, errTotalCPUNotAvailable
}
