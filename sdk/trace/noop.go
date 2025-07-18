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

// Package trace trace
package trace

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// NoopTracerProvider implementation a tracer provider that will do nothing
// use code lines blow to set a noop tracer
// otel.SetTracerProvider(NoopTracerProvider)
var NoopTracerProvider = trace.NewNoopTracerProvider()

// SetNoopTracerProvider set the global trace provider as noop
func SetNoopTracerProvider() {
	otel.SetTracerProvider(NoopTracerProvider)
}
