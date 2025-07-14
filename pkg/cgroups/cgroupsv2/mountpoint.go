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
	_mountInfoSep               = " "
	_mountInfoOptsSep           = ","
	_mountInfoOptionalFieldsSep = "-"
)

const (
	_miFieldIDMountID = iota
	_miFieldIDParentID
	_miFieldIDDeviceID
	_miFieldIDRoot
	_miFieldIDMountPoint
	_miFieldIDOptions
	_miFieldIDOptionalFields

	_miFieldCountFirstHalf
)

const (
	_miFieldOffsetFSType = iota
	_miFieldOffsetMountSource
	_miFieldOffsetSuperOptions

	_miFieldCountSecondHalf
)

const _miFieldCountMin = _miFieldCountFirstHalf + _miFieldCountSecondHalf

// MountPoint is the data structure for the mount points in
// `/proc/$PID/mountinfo`. See also proc(5) for more information.
type MountPoint struct {
	MountID        int
	ParentID       int
	DeviceID       string
	Root           string
	MountPoint     string
	Options        []string
	OptionalFields []string
	FSType         string
	MountSource    string
	SuperOptions   []string
}

// NewMountPointFromLine parses a line read from `/proc/$PID/mountinfo` and
// returns a new *MountPoint.
func NewMountPointFromLine(line string) (*MountPoint, error) {
	fields := strings.Split(line, _mountInfoSep)

	if len(fields) < _miFieldCountMin {
		return nil, mountPointFormatInvalidError{line}
	}

	mountID, err := strconv.Atoi(fields[_miFieldIDMountID])
	if err != nil {
		return nil, err
	}

	parentID, err := strconv.Atoi(fields[_miFieldIDParentID])
	if err != nil {
		return nil, err
	}

	for i, field := range fields[_miFieldIDOptionalFields:] {
		if field == _mountInfoOptionalFieldsSep {
			fsTypeStart := _miFieldIDOptionalFields + i + 1

			if len(fields) != fsTypeStart+_miFieldCountSecondHalf {
				return nil, mountPointFormatInvalidError{line}
			}

			miFieldIDFSType := _miFieldOffsetFSType + fsTypeStart
			miFieldIDMountSource := _miFieldOffsetMountSource + fsTypeStart
			miFieldIDSuperOptions := _miFieldOffsetSuperOptions + fsTypeStart

			return &MountPoint{
				MountID:        mountID,
				ParentID:       parentID,
				DeviceID:       fields[_miFieldIDDeviceID],
				Root:           fields[_miFieldIDRoot],
				MountPoint:     fields[_miFieldIDMountPoint],
				Options:        strings.Split(fields[_miFieldIDOptions], _mountInfoOptsSep),
				OptionalFields: fields[_miFieldIDOptionalFields:(fsTypeStart - 1)],
				FSType:         fields[miFieldIDFSType],
				MountSource:    fields[miFieldIDMountSource],
				SuperOptions:   strings.Split(fields[miFieldIDSuperOptions], _mountInfoOptsSep),
			}, nil
		}
	}

	return nil, mountPointFormatInvalidError{line}
}

// Translate converts an absolute path inside the *MountPoint's file system to
// the host file system path in the mount namespace the *MountPoint belongs to.
func (mp *MountPoint) Translate(absPath string) (string, error) {
	relPath, err := filepath.Rel(mp.Root, absPath)

	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") {
		return "", pathNotExposedFromMountPointError{
			mountPoint: mp.MountPoint,
			root:       mp.Root,
			path:       absPath,
		}
	}

	return filepath.Join(mp.MountPoint, relPath), nil
}

// parseMountInfo parses procPathMountInfo (usually at `/proc/$PID/mountinfo`)
// and yields parsed *MountPoint into newMountPoint.
func parseMountInfo(procPathMountInfo string, newMountPoint func(*MountPoint) error) error {
	mountInfoFile, err := os.Open(filepath.Clean(procPathMountInfo))
	if err != nil {
		return err
	}
	defer mountInfoFile.Close()

	scanner := bufio.NewScanner(mountInfoFile)

	for scanner.Scan() {
		mountPoint, err := NewMountPointFromLine(scanner.Text())
		if err != nil {
			return err
		}
		if err := newMountPoint(mountPoint); err != nil {
			return err
		}
	}

	return scanner.Err()
}
