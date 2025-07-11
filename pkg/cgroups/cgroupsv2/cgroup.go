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
	"os"
	"path/filepath"
	"strconv"

	"github.com/shopspring/decimal"
)

// CGroup represents the data structure for a Linux control group.
type CGroup struct {
	path string
}

// NewCGroup returns a new *CGroup from a given path.
func NewCGroup(path string) *CGroup {
	return &CGroup{path: path}
}

// Path returns the path of the CGroup*.
func (cg *CGroup) Path() string {
	return cg.path
}

// ParamPath returns the path of the given cgroup param under itself.
func (cg *CGroup) ParamPath(param string) string {
	return filepath.Join(cg.path, param)
}

// readFirstLine reads the first line from a cgroup param file.
func (cg *CGroup) readFirstLine(param string) (string, error) {
	paramFile, err := os.Open(cg.ParamPath(param))
	if err != nil {
		return "", err
	}
	defer paramFile.Close()

	scanner := bufio.NewScanner(paramFile)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", io.ErrUnexpectedEOF
}

// readInt parses the first line from a cgroup param file as int.
func (cg *CGroup) readInt(param string) (int, error) {
	text, err := cg.readFirstLine(param)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(text)
}

// readDecimal parses the first line from a cgroup param file as decimal number.
func (cg *CGroup) readDecimal(param string) (decimal.Decimal, error) {
	text, err := cg.readFirstLine(param)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return decimal.NewFromString(text)
}
