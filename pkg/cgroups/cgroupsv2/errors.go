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

import "fmt"

type cgroupSubsysFormatInvalidError struct {
	line string
}

type mountPointFormatInvalidError struct {
	line string
}

type pathNotExposedFromMountPointError struct {
	mountPoint string
	root       string
	path       string
}

func (err cgroupSubsysFormatInvalidError) Error() string {
	return fmt.Sprintf("invalid format for CGroupSubsys: %q", err.line)
}

func (err mountPointFormatInvalidError) Error() string {
	return fmt.Sprintf("invalid format for MountPoint: %q", err.line)
}

func (err pathNotExposedFromMountPointError) Error() string {
	return fmt.Sprintf("path %q is not a descendant of mount point root %q and cannot be exposed from %q", err.path, err.root, err.mountPoint)
}
