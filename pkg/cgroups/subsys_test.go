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

// Keep the original Uber license.

// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

//go:build linux
// +build linux

package cgroups

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewCGroupSubsysFromLine
func TestNewCGroupSubsysFromLine(t *testing.T) {
	testTable := []struct {
		name           string
		line           string
		expectedSubsys *CGroupSubsys
	}{
		{
			name: "single-subsys",
			line: "1:cpu:/",
			expectedSubsys: &CGroupSubsys{
				ID:         1,
				Subsystems: []string{"cpu"},
				Name:       "/",
			},
		},
		{
			name: "multi-subsys",
			line: "8:cpu,cpuacct,cpuset:/docker/1234567890abcdef",
			expectedSubsys: &CGroupSubsys{
				ID:         8,
				Subsystems: []string{"cpu", "cpuacct", "cpuset"},
				Name:       "/docker/1234567890abcdef",
			},
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Equal(t, tt.expectedSubsys, subsys, tt.name)
		assert.NoError(t, err, tt.name)
	}
}

// TestNewCGroupSubsysFromLineErr
func TestNewCGroupSubsysFromLineErr(t *testing.T) {
	lines := []string{
		"1:cpu",
		"1:cpu,cpuacct:/:/necessary-field",
		"not-a-number:cpu:/",
	}
	_, parseError := strconv.Atoi("not-a-number")

	testTable := []struct {
		name          string
		line          string
		expectedError error
	}{
		{
			name:          "fewer-fields",
			line:          lines[0],
			expectedError: cgroupSubsysFormatInvalidError{lines[0]},
		},
		{
			name:          "more-fields",
			line:          lines[1],
			expectedError: cgroupSubsysFormatInvalidError{lines[1]},
		},
		{
			name:          "illegal-id",
			line:          lines[2],
			expectedError: parseError,
		},
	}

	for _, tt := range testTable {
		subsys, err := NewCGroupSubsysFromLine(tt.line)
		assert.Nil(t, subsys, tt.name)
		assert.Equal(t, tt.expectedError, err, tt.name)
	}
}
