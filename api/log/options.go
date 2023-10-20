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

package log

import (
	"go.opentelemetry.io/otel/attribute"
)

// Config defines the configuration for the various elements of log.
type Config struct {
	Level             Level
	Name              string
	Fields            []attribute.KeyValue
	StandaloneEnabled bool
}

// Option apply changes to internalOptions.
type Option func(*Config)

// WithLevel return Option which apply changes to level
func WithLevel(lvl Level) Option {
	return func(c *Config) {
		c.Level = lvl
	}
}

// WithStandaloneEnable return Option which apply changes to StandaloneEnabled
func WithStandaloneEnable() Option {
	return func(c *Config) {
		c.StandaloneEnabled = true
	}
}

// WithName return Option which set name
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithFields return Option which set fields
func WithFields(fields ...attribute.KeyValue) Option {
	return func(c *Config) {
		c.Fields = fields
	}
}
