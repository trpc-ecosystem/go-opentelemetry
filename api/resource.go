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

package api

import "go.opentelemetry.io/otel/attribute"

const (
	TpsTenantIDKey   = attribute.Key("tps.tenant.id")
	TpsTenantNameKey = attribute.Key("tps.tenant.name")
	TpsDyeingKey     = attribute.Key("tps.dyeing")
	TpsOwnerKey      = attribute.Key("server.owner")
	TpsCmdbIDKey     = attribute.Key("cmdb.module.id")

	OpenTelemetryName = "opentelemetry"
	TenantHeaderKey   = "X-Tps-TenantID"
)

const (
	// TraceparentHeader w3c trace header
	TraceparentHeader = "traceparent"
	// TracestateHeader w3c trace  state
	TracestateHeader = "tracestate"
	// BaggageHeader full link awareness information customized by the business side
	BaggageHeader = "baggage"
)

const (
	AppKey           = "app"
	NamespaceKey     = "namespace"
	EnvNameKey       = "env_name"
	ServerKey        = "server"
	ContainerNameKey = "container_name"
	EnvKey           = "env"
	InstanceKey      = "instance"
)
