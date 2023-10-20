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

package cgroups

import (
	"os"
	"path/filepath"
)

var (
	pwd                 = mustGetWd()
	testDataPath        = filepath.Join(pwd, "testdata")
	testDataCGroupsPath = filepath.Join(testDataPath, "cgroups")
	testDataProcPath    = filepath.Join(testDataPath, "proc")
)

func mustGetWd() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return pwd
}
