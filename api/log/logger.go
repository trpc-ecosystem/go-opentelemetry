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

package log

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

var logger Logger = NewNopLogger()

var _ Logger = (*NopLogger)(nil)

// NewNopLogger returns a no-op Logger
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// NopLogger implement Logger
type NopLogger struct{}

// With set fields
func (n *NopLogger) With(ctx context.Context, values []attribute.KeyValue) context.Context {
	return ctx
}

// Log log a message
func (n *NopLogger) Log(ctx context.Context, msg string, opts ...Option) {
}

// Logger provides a interface of logging
type Logger interface {
	Log(context.Context, string, ...Option)
	With(context.Context, []attribute.KeyValue) context.Context
}

// GlobalLogger return global logger
func GlobalLogger() Logger {
	return logger
}

// SetGlobalLogger set global logger
func SetGlobalLogger(l Logger) {
	logger = l
}

type ctxMarker struct{}

var (
	ctxKey = &ctxMarker{}
)

// FromContext takes a context and returns []attribute.KeyValue from it, if present.
func FromContext(ctx context.Context) []attribute.KeyValue {
	l, ok := ctx.Value(ctxKey).([]attribute.KeyValue)

	if !ok || l == nil {
		return []attribute.KeyValue{}
	}

	return l
}

// ContextWith takes an existing context and derives a new context with the []attribute.KeyValue stored on it
func ContextWith(ctx context.Context, kvs []attribute.KeyValue) context.Context {
	labels := FromContext(ctx)
	labels = append(labels, kvs...)
	return context.WithValue(ctx, ctxKey, labels)
}
