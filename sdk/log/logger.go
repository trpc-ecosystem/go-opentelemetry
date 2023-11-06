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
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	commonproto "go.opentelemetry.io/proto/otlp/common/v1"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
	resourceproto "go.opentelemetry.io/proto/otlp/resource/v1"

	"trpc-ecosystem/go-opentelemetry/api/log"
)

var _ log.Logger = (*Logger)(nil)

// NewLogger ...
func NewLogger(opts ...LoggerOption) *Logger {
	options := &LoggerOptions{}
	for _, o := range opts {
		o(options)
	}

	return &Logger{
		opts: options,
	}
}

// LoggerOptions logger options detail
type LoggerOptions struct {
	// Resource contains attributes representing an entity that produces telemetry.
	Resource *resource.Resource

	// Processor export logs
	Processor *BatchProcessor

	// LevelEnabled enabled level
	LevelEnabled log.Level

	// LevelNumber enabled level number according to LevelEnabled
	LevelNumber logsproto.SeverityNumber

	// EnableSampler enable sampler.
	EnableSampler bool

	// EnableSamplerError when EnableSampler is true，report error log when not sampled
	EnableSamplerError bool
}

// LoggerOption logger option func
type LoggerOption func(*LoggerOptions)

// WithLevelEnable enable level
func WithLevelEnable(level log.Level) LoggerOption {
	return func(options *LoggerOptions) {
		options.LevelEnabled = level
		options.LevelNumber = toSeverityNumber(level)
	}
}

// WithEnableSampler enable sample along with trace.
func WithEnableSampler(enableSampler bool) LoggerOption {
	return func(options *LoggerOptions) {
		options.EnableSampler = enableSampler
	}
}

// WithEnableSamplerError when EnableSampler is true，report error log when not sampled
func WithEnableSamplerError(enableSampler bool) LoggerOption {
	return func(options *LoggerOptions) {
		options.EnableSamplerError = enableSampler
	}
}

// WithResource setting resource info
func WithResource(rs *resource.Resource) LoggerOption {
	return func(options *LoggerOptions) {
		options.Resource = rs
	}
}

// WithBatcher setting Processor
func WithBatcher(batcher *BatchProcessor) LoggerOption {
	return func(options *LoggerOptions) {
		options.Processor = batcher
	}
}

// Logger logger impl
type Logger struct {
	opts *LoggerOptions
}

// Shutdown is invoked during service shutdown.
func (l *Logger) Shutdown(ctx context.Context) error {
	return l.opts.Processor.Shutdown(ctx)
}

// With  set fields
func (l *Logger) With(ctx context.Context, values []attribute.KeyValue) context.Context {
	return log.ContextWith(ctx, values)
}

// Log record a log
func (l *Logger) Log(ctx context.Context, msg string, opts ...log.Option) {
	cfg := &log.Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	sampled := false
	levelNumber := toSeverityNumber(cfg.Level)
	if l.opts.EnableSampler && levelNumber >= l.opts.LevelNumber {
		if trace.SpanFromContext(ctx).SpanContext().IsSampled() ||
			(l.opts.EnableSamplerError && levelNumber >= logsproto.SeverityNumber_SEVERITY_NUMBER_ERROR) {
			sampled = true
		}
	}
	if !l.opts.EnableSampler && levelNumber >= l.opts.LevelNumber {
		sampled = true
	}

	l.log(ctx, msg, cfg, sampled)
}

func toSeverityNumber(level log.Level) logsproto.SeverityNumber {
	var number logsproto.SeverityNumber
	switch level {
	case log.TraceLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_TRACE
	case log.DebugLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case log.InfoLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_INFO
	case log.WarnLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_WARN
	case log.ErrorLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_ERROR
	case log.FatalLevel:
		number = logsproto.SeverityNumber_SEVERITY_NUMBER_FATAL
	}

	return number
}

func (l *Logger) log(ctx context.Context, msg string, cfg *log.Config, sampled bool) {
	if !sampled {
		return
	}
	span := trace.SpanFromContext(ctx)
	record := &logsproto.LogRecord{
		TimeUnixNano:   uint64(time.Now().UnixNano()),
		SeverityText:   string(cfg.Level),
		SeverityNumber: toSeverityNumber(cfg.Level),
	}
	record.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{StringValue: msg},
	}
	record.Flags = uint32(span.SpanContext().TraceFlags())
	kvs := log.FromContext(ctx)
	cfg.Fields = append(cfg.Fields, kvs...)
	if span.SpanContext().IsSampled() {
		traceID := span.SpanContext().TraceID()
		spanID := span.SpanContext().SpanID()
		if span != nil {
			if span.SpanContext().HasSpanID() {
				record.SpanId = spanID[:]
			}
			if span.SpanContext().HasTraceID() {
				record.TraceId = traceID[:]
			}
		}
	}
	for _, field := range cfg.Fields {
		record.Attributes = append(record.Attributes, toAttribute(field))
	}
	logs := &logsproto.ResourceLogs{
		Resource: Resource(l.opts.Resource),
		ScopeLogs: []*logsproto.ScopeLogs{
			{
				LogRecords: []*logsproto.LogRecord{record},
			},
		},
	}
	l.opts.Processor.Enqueue(logs)
}

func toAttribute(v attribute.KeyValue) *commonproto.KeyValue {
	switch v.Value.Type() {
	case attribute.BOOL:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_BoolValue{BoolValue: v.Value.AsBool()},
			},
		}
	case attribute.INT64:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_IntValue{IntValue: v.Value.AsInt64()},
			},
		}
	case attribute.FLOAT64:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_DoubleValue{DoubleValue: v.Value.AsFloat64()},
			},
		}
	case attribute.STRING:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_StringValue{StringValue: v.Value.AsString()},
			},
		}
	default:
		return nil
	}
}

// Resource transforms a Resource into an OTLP Resource.
func Resource(r *resource.Resource) *resourceproto.Resource {
	if r == nil {
		return nil
	}
	return &resourceproto.Resource{Attributes: ResourceAttributes(r)}
}

// ResourceAttributes transforms a Resource into a slice of OTLP attribute key-values.
func ResourceAttributes(resource *resource.Resource) []*commonproto.KeyValue {
	if resource.Len() == 0 {
		return nil
	}

	out := make([]*commonproto.KeyValue, 0, resource.Len())
	for iter := resource.Iter(); iter.Next(); {
		out = append(out, toAttribute(iter.Attribute()))
	}

	return out
}
