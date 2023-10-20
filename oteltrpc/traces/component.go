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
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"

	"trpc.group/trpc-go/trpc-go/codec"
)

func handleComponent(msg codec.Msg, span trace.Span) {
	var attrs []attribute.KeyValue
	if _, ok := msg.CommonMeta()[semconv.DBSystemKey]; ok {
		attrs = databaseAttribute(msg)
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

func databaseAttribute(msg codec.Msg) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	for k, v := range msg.CommonMeta() {
		key, kok := k.(attribute.Key)
		val, vok := v.(string)
		if !kok || !vok {
			continue
		}
		switch key {
		case semconv.DBSystemKey, semconv.DBNameKey,
			semconv.DBOperationKey, semconv.DBStatementKey,
			semconv.DBMongoDBCollectionKey, semconv.DBSQLTableKey:
			attrs = append(attrs, attribute.String(string(key), val))
		default:
		}
	}
	return attrs
}
