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

package admin

import (
	"errors"
)

// Option is function for applying an option the admin server
type Option func(o *Options)

// Options represents config options for open telemetry http admin server
type Options struct {
	addr             string
	enablePrometheus bool
	enablePprof      bool
	enableHotSwitch  bool
	enableZPage      bool
}

func (o Options) validate() error {
	if len(o.addr) == 0 {
		return errors.New("must pass valid addr")
	}
	return nil
}

// WithAddr set server listen addr
func WithAddr(addr string) Option {
	return func(o *Options) {
		o.addr = addr
	}
}

// WithEnablePrometheus set whether to enable prometheus metrics http handler
func WithEnablePrometheus(enable bool) Option {
	return func(o *Options) {
		o.enablePrometheus = enable
	}
}

// WithEnablePprof set whether to enable go pprof http handler
func WithEnablePprof(enable bool) Option {
	return func(o *Options) {
		o.enablePprof = enable
	}
}

// WithEnableHotSwitch set whether to enable hot switch http handler
func WithEnableHotSwitch(enable bool) Option {
	return func(o *Options) {
		o.enableHotSwitch = enable
	}
}

// WithEnableZPage set up zPages processor
func WithEnableZPage(enable bool) Option {
	return func(o *Options) {
		o.enableZPage = enable
	}
}

func defaultOptions() *Options {
	return new(Options)
}
