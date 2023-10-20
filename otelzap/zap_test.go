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

package otelzap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	commonproto "go.opentelemetry.io/proto/otlp/common/v1"
	"go.uber.org/zap/zapcore"
)

func Test_encoder_convertField(t *testing.T) {
	e := &encoder{
		kvs: []*commonproto.KeyValue{},
	}
	e.convertField(zapcore.Field{
		Key:     "test",
		Type:    zapcore.BoolType,
		Integer: 0,
	})
	kv := e.kvs
	assert.Equal(t, 1, len(kv))
	assert.Equal(t, "test", kv[0].Key)
	assert.Equal(t, &commonproto.AnyValue{
		Value: &commonproto.AnyValue_BoolValue{
			BoolValue: false,
		},
	}, kv[0].Value)
}
