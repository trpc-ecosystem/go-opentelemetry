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

package semconv

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"trpc.group/trpc-go/trpc-go"

	tpsapi "trpc.group/trpc-go/go-opentelemetry/api"
)

const (
	NamespaceKey = attribute.Key("trpc.namespace")
	EnvNameKey   = attribute.Key("trpc.envname")

	StatusCode = attribute.Key("trpc.status_code")
	StatusMsg  = attribute.Key("trpc.status_msg")
	StatusType = attribute.Key("trpc.status_type")

	ProtocolKey = attribute.Key("trpc.protocol")

	CallerServiceKey = attribute.Key("trpc.caller_service")
	CallerMethodKey  = attribute.Key("trpc.caller_method")
	CalleeServiceKey = attribute.Key("trpc.callee_service")
	CalleeMethodKey  = attribute.Key("trpc.callee_method")
)

var once sync.Once
var serviceProtocols map[string]string

func getProtocol(ctx context.Context) string {
	once.Do(func() {
		serviceProtocols = make(map[string]string)
		for _, service := range trpc.GlobalConfig().Server.Service {
			serviceProtocols[service.Name] = service.Protocol
		}
	})
	if serviceProtocols == nil {
		return ""
	}
	msg := trpc.Message(ctx)
	protocol, ok := serviceProtocols[msg.CalleeServiceName()]
	if !ok {
		return ""
	}
	return protocol
}

// KeyValues get KeyValue array from ctx
func KeyValues(ctx context.Context) []attribute.KeyValue {
	msg := trpc.Message(ctx)
	var kvs []attribute.KeyValue
	kvs = append(kvs, NamespaceKey.String(msg.Namespace()))
	kvs = append(kvs, EnvNameKey.String(msg.EnvName()))
	kvs = append(kvs, CallerServiceKey.String(msg.CallerServiceName()))
	kvs = append(kvs, CallerMethodKey.String(msg.CallerMethod()))
	kvs = append(kvs, CalleeServiceKey.String(msg.CalleeServiceName()))
	kvs = append(kvs, CalleeMethodKey.String(msg.CalleeMethod()))
	kvs = append(kvs, tpsapi.TpsDyeingKey.String(msg.DyeingKey()))
	kvs = append(kvs, ProtocolKey.String(getProtocol(ctx)))
	kvs = append(kvs, semconv.EnduserIDKey.String(DefaultUserIDInjectFunc(ctx)))
	return kvs
}

var DefaultUserIDInjectFunc = func(ctx context.Context) string {
	return trpc.Message(ctx).DyeingKey()
}
