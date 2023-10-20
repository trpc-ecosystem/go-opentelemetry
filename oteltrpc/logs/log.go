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

package logs

import (
	"context"
	"fmt"
	"strings"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// FlowLog log model for rpc
type FlowLog struct {
	Kind     FlowKind `json:"kind,omitempty"`
	Source   Service  `json:"source,omitempty"`
	Target   Service  `json:"target,omitempty"`
	Request  Request  `json:"request,omitempty"`
	Response Response `json:"response,omitempty"`
	Cost     string   `json:"cost,omitempty"`
	Status   Status   `json:"status,omitempty"`
}

// String ...
func (f FlowLog) String() string {
	return f.OneLineString()
}

// MultilineString ...
func (f FlowLog) MultilineString() string {
	var sb strings.Builder
	switch trace.SpanKind(f.Kind) {
	case trace.SpanKindServer:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s\n", f.Kind.String(), f.Target.String()))
		sb.WriteString(fmt.Sprintf("RecvFrom: %s\n", f.Source.String()))
	case trace.SpanKindClient:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s\n", f.Kind.String(), f.Source.String()))
		sb.WriteString(fmt.Sprintf("SentTo: %s\n", f.Target.String()))
	}
	sb.WriteString(fmt.Sprintf("Status: %s\n", f.Status))
	sb.WriteString(fmt.Sprintf("Cost: %s\n", f.Cost))
	sb.WriteString(fmt.Sprintf("Request.Body: %s\n", f.Request.Body))
	sb.WriteString(fmt.Sprintf("Response.Body: %s\n", f.Response.Body))
	return sb.String()
}

// OneLineString ...
func (f FlowLog) OneLineString() string {
	var sb strings.Builder
	switch trace.SpanKind(f.Kind) {
	case trace.SpanKindServer:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s ", f.Kind.String(), f.Target.String()))
		sb.WriteString(fmt.Sprintf(" RecvFrom: %s ", f.Source.String()))
	case trace.SpanKindClient:
		sb.WriteString(fmt.Sprintf(" [FLOW(%s)] %s ", f.Kind.String(), f.Source.String()))
		sb.WriteString(fmt.Sprintf(" SentTo: %s ", f.Target.String()))
	}
	sb.WriteString(fmt.Sprintf(" Status: %s ", f.Status))
	sb.WriteString(fmt.Sprintf(" Cost: %s ", f.Cost))
	sb.WriteString(fmt.Sprintf(" Request.Body: %s ", f.Request.Body))
	sb.WriteString(fmt.Sprintf(" Response.Body: %s", f.Response.Body))
	return sb.String()
}

type FlowKind trace.SpanKind

// MarshalJSON return byte slice of flowkind
func (k FlowKind) MarshalJSON() ([]byte, error) {
	switch k {
	case FlowKindServer:
		return []byte("\"server\""), nil
	case FlowKindClient:
		return []byte("\"client\""), nil
	default:
		return []byte("\"internal\""), nil
	}
}

const (
	FlowKindServer FlowKind = FlowKind(trace.SpanKindServer)
	FlowKindClient FlowKind = FlowKind(trace.SpanKindClient)
)

// String return string of flowkind
func (k FlowKind) String() string {
	switch trace.SpanKind(k) {
	case trace.SpanKindServer:
		return "SERVER"
	case trace.SpanKindClient:
		return "CLIENT"
	default:
		return "INTERNAL"
	}
}

// Status rpc status
type Status struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

// Service rpc service
type Service struct {
	Name      string `json:"service,omitempty"`
	Method    string `json:"method,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Address   string `json:"address,omitempty"`
}

// String return status as string
func (s Status) String() string {
	return fmt.Sprintf("%d %s(%s)", s.Code, s.Type, s.Message)
}

// String return service as string
func (s Service) String() string {
	return fmt.Sprintf("%s/%s@%s(%s)", s.Name, s.Method, s.Namespace, s.Address)
}

// Request rpc request
type Request struct {
	Head string `json:"head,omitempty"`
	Body string `json:"body,omitempty"`
}

// Response rpc response
type Response struct {
	Head string `json:"head,omitempty"`
	Body string `json:"body,omitempty"`
}

func spanLogf(ctx context.Context, level log.Level, format string, v []interface{}) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	var msg string
	if format == "" {
		msg = fmt.Sprint(v...)
	} else {
		msg = fmt.Sprintf(format, v...)
	}
	span.AddEvent("", trace.WithAttributes(attribute.String("msg", msg), attribute.String("level", level.String())))
}

// Debug log with debug level
func Debug(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelDebug, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Debug(args...)
	default:
		log.DefaultLogger.Debug(args...)
	}
}

// Debugf Debug
func Debugf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelDebug, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Debugf(format, args...)
	default:
		log.DefaultLogger.Debugf(format, args...)
	}
}

// Info log with Info level
func Info(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelInfo, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Info(args...)
	default:
		log.DefaultLogger.Info(args...)
	}
}

// Infof Info
func Infof(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelInfo, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Infof(format, args...)
	default:
		log.DefaultLogger.Infof(format, args...)
	}
}

// Warn log with warn level
func Warn(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelWarn, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Warn(args...)
	default:
		log.DefaultLogger.Warn(args...)
	}
}

// Warnf warn
func Warnf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelWarn, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Warnf(format, args...)
	default:
		log.DefaultLogger.Warnf(format, args...)
	}
}

// Error log with error level
func Error(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelError, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Error(args...)
	default:
		log.DefaultLogger.Error(args...)
	}
}

// Errorf error
func Errorf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelError, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Errorf(format, args...)
	default:
		log.DefaultLogger.Errorf(format, args...)
	}
}

// Fatal log with debug level
func Fatal(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelFatal, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Fatal(args...)
	default:
		log.DefaultLogger.Fatal(args...)
	}
}

// Fatalf fatal
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelFatal, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case log.Logger:
		l.Fatalf(format, args...)
	default:
		log.DefaultLogger.Fatalf(format, args...)
	}
}
