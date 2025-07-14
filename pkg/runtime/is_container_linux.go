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

package runtime

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	procSelfCgroup = "/proc/self/cgroup"
	dockerEnvPath  = "/.dockerenv"
)

var isContainer bool

var dockerQueryOnce sync.Once

// ProcessInContainer returns if process runs in container
func ProcessInContainer() bool {
	dockerQueryOnce.Do(func() {
		isContainer = hasDockerEnvPath() || hasContainerCgroups()
	})
	return isContainer
}

func hasDockerEnvPath() bool {
	_, err := os.Stat(dockerEnvPath)
	return err == nil
}

func hasContainerCgroups() bool {
	if bdata, err := ioutil.ReadFile(procSelfCgroup); err == nil {
		return strings.Contains(string(bdata), ":/docker/") || strings.Contains(string(bdata), ":/kubepods/")
	}
	return false
}
