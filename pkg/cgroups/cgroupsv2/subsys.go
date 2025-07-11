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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	_cgroupSep       = ":"
	_cgroupSubsysSep = ","
)

const (
	_csFieldIDID = iota
	_csFieldIDSubsystems
	_csFieldIDName
	_csFieldCount
)

// CGroupSubsys represents the data structure for entities in
// `/proc/$PID/cgroup`. See also proc(5) for more information.
type CGroupSubsys struct {
	ID         int
	Subsystems []string
	Name       string
}

// NewCGroupSubsysFromLine returns a new *CGroupSubsys by parsing a string in
// the format of `/proc/$PID/cgroup`
func NewCGroupSubsysFromLine(line string) (*CGroupSubsys, error) {
	fields := strings.Split(line, _cgroupSep)

	if len(fields) != _csFieldCount {
		return nil, cgroupSubsysFormatInvalidError{line}
	}

	id, err := strconv.Atoi(fields[_csFieldIDID])
	if err != nil {
		return nil, err
	}

	cgroup := &CGroupSubsys{
		ID:         id,
		Subsystems: strings.Split(fields[_csFieldIDSubsystems], _cgroupSubsysSep),
		Name:       fields[_csFieldIDName],
	}

	return cgroup, nil
}

// parseCGroupSubsystems parses procPathCGroup (usually at `/proc/$PID/cgroup`)
// and returns a new map[string]*CGroupSubsys.
func parseCGroupSubsystems(procPathCGroup string) (map[string]*CGroupSubsys, error) {
	cgroupFile, err := os.Open(filepath.Clean(procPathCGroup))
	if err != nil {
		return nil, err
	}
	defer cgroupFile.Close()

	scanner := bufio.NewScanner(cgroupFile)
	subsystems := make(map[string]*CGroupSubsys)

	for scanner.Scan() {
		cgroup, err := NewCGroupSubsysFromLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		for _, subsys := range cgroup.Subsystems {
			subsystems[subsys] = cgroup
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return subsystems, nil
}
