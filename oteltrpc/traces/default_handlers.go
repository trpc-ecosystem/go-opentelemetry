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

// Package traces
package traces

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"trpc.group/trpc-go/trpc-go/codec"

	tpsapi "trpc.group/trpc-go/trpc-opentelemetry/api"
	sdktrace "trpc.group/trpc-go/trpc-opentelemetry/sdk/trace"
)

// DefaultTraceAttributesFunc can be set by user
var DefaultTraceAttributesFunc TraceAttributesFunc = func(ctx context.Context, req interface{}) []attribute.KeyValue {
	return nil
}

// TraceAttributesFunc hook for get trace attribute from ctx and req
type TraceAttributesFunc func(ctx context.Context, req interface{}) []attribute.KeyValue

// TraceEventMsgMarshaler marshaler for trace event msg
type TraceEventMsgMarshaler func(message interface{}) string

// TraceEventMsgMarshalerWithContext marshaler for trace event msg with ctx
type TraceEventMsgMarshalerWithContext func(ctx context.Context, message interface{}) string

// DefaultTraceEventMsgMarshaler can be set by user
// Deprecated: use SetTraceEventMsgMarshaler instead
var DefaultTraceEventMsgMarshaler TraceEventMsgMarshaler = ProtoMessageToCustomJSONString

// defaultTraceEventMsgMarshalerWithContext can be set by user
var defaultTraceEventMarshalerWithContext = ProtoMessageToCustomJSONStringWithContext

// SetTraceEventMsgMarshaler set marshaler for trace event msg
func SetTraceEventMsgMarshaler(f TraceEventMsgMarshalerWithContext) {
	defaultTraceEventMarshalerWithContext = f
}

var (
	metadataKeyTraceForceSample = "trace-force-sample"
)

// AttributesAfterHandle hook
type AttributesAfterHandle func(ctx context.Context, rsp interface{}) []attribute.KeyValue

// DefaultAttributesAfterServerHandle set by user
var DefaultAttributesAfterServerHandle AttributesAfterHandle = func(ctx context.Context,
	rsp interface{}) []attribute.KeyValue {
	return nil
}

// DefaultAttributesAfterClientHandle set by user
var DefaultAttributesAfterClientHandle AttributesAfterHandle = func(ctx context.Context,
	rsp interface{}) []attribute.KeyValue {
	return nil
}

// forceSampleFromMetadata for force sample
func forceSampleFromMetadata(metadata codec.MetaData) []attribute.KeyValue {
	v := metadata[metadataKeyTraceForceSample]
	if len(v) == 0 {
		return nil
	}
	return []attribute.KeyValue{sdktrace.ForceSamplerKey.String(string(v))}
}

// fromTRPCDyeingKey get attribute from tRPC dyeing key
func fromTRPCDyeingKey(dyeingKey string) []attribute.KeyValue {
	if dyeingKey == "" {
		return nil
	}
	return []attribute.KeyValue{tpsapi.TpsDyeingKey.String(dyeingKey)}
}
