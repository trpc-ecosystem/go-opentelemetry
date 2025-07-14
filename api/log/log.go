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

// Package log log interface
package log

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

// Trace log with trace level
func Trace(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(TraceLevel), WithFields(fields...))
}

// Debug log with debug level
func Debug(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(DebugLevel), WithFields(fields...))
}

// Info log with info level
func Info(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(InfoLevel), WithFields(fields...))
}

// Warn log with warn level
func Warn(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(WarnLevel), WithFields(fields...))
}

// Error log with error level
func Error(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(ErrorLevel), WithFields(fields...))
}

// Fatal log with fatal level
func Fatal(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(FatalLevel), WithFields(fields...))
}
