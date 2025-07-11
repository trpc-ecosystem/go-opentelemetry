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
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMemoryQuota
func TestMemoryQuota(t *testing.T) {
	_, err := MemoryQuota()
	require.ErrorIs(t, err, errTotalMemoryNotAvailable)
}

// TestMemoryUsage
func TestMemoryUsage(t *testing.T) {
	_, err := MemoryUsage()
	require.ErrorIs(t, err, errUsageMemoryNotAvailable)
}
