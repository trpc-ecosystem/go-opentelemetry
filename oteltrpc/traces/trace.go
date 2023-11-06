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

package traces

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"

	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"

	"trpc.group/trpc-go/go-opentelemetry/oteltrpc/metrics/prometheus"
)

const (
	pluginType = "telemetry"
)

var _ plugin.Factory = (*factory)(nil)

type factory struct {
}

func (f factory) Type() string {
	return pluginType
}

func (f factory) Setup(name string, configDec plugin.Decoder) error {
	return nil
}

// addEvent returns messageStr so that subsequent processing can be reused to reduce serialization consumption.
// The upper layer needs to judge whether it is empty. If it is empty,
// it means that the package body is not proto.Message and has not been serialized to string
func addEvent(ctx context.Context, message interface{},
	messageType attribute.KeyValue, deadline time.Duration, timeStamp time.Time) (messageStr string) {
	span := trace.SpanFromContext(ctx)
	defer func() {
		if err := recover(); err != nil {
			log.ErrorContextf(ctx, "opentelemetry addEvent err: %v", err)
			// add sdk panic metrics
			prometheus.IncrSDKPanicTotal()
			// panic is usually because defaultTraceEventMarshalerWithContext puts related information
			// into the event for business analysis
			messageStr = fixStringTooLong(fmt.Sprintf("addEvent panic: %v", err))
			span.AddEvent(messageType.Value.AsString(),
				trace.WithAttributes(
					// RPCMessageUncompressedSizeKey is not accurate,
					// but it is much smaller than the proto.Size consumed by implicit serialization before pb1.4.0
					semconv.MessageUncompressedSizeKey.Int(len(messageStr)),
					attribute.Key("message.detail").String(messageStr),
					attribute.Key("ctx.deadline").String(deadline.String()),
				),
				trace.WithTimestamp(timeStamp),
			)
		}
	}()

	messageStr = fixStringTooLong(defaultTraceEventMarshalerWithContext(ctx, message))
	span.AddEvent(messageType.Value.AsString(),
		trace.WithAttributes(
			// RPCMessageUncompressedSizeKey is not accurate,
			// but it is much smaller than the proto.Size consumed by implicit serialization before pb1.4.0
			semconv.MessageUncompressedSizeKey.Int(len(messageStr)),
			attribute.Key("message.detail").String(messageStr),
			attribute.Key("ctx.deadline").String(deadline.String()),
		),
		trace.WithTimestamp(timeStamp),
	)
	return messageStr
}

const fixedStringSuffix = "...stringLengthTooLong"
const defaultMaxStringLength = 32766

var maxStringLength = defaultMaxStringLength

// SetMaxStringLength sets the maximum length of a string attribute value.
func SetMaxStringLength(limit int) {
	if limit > defaultMaxStringLength {
		return
	}
	maxStringLength = limit
}

// isStringTooLong
func isStringTooLong(s string) bool {
	return len(s) > maxStringLength
}

// fixStringTooLong
// Document contains at least one immense term in field=\"logs.fields.value\"
// (whose UTF8 encoding is longer than the max length 32766)
func fixStringTooLong(s string) (result string) {
	if isStringTooLong(s) {
		return strings.ToValidUTF8(s[:maxStringLength-len(fixedStringSuffix)]+fixedStringSuffix, "")
	}
	return s
}
