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

package semconv

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	ServiceNameKey      = semconv.ServiceNameKey
	ServiceNamespaceKey = semconv.ServiceNamespaceKey
	ServiceVersionKey   = semconv.ServiceVersionKey
)

const (
	TenantIDKey    = attribute.Key("tenant.id")
	TenantNameKey  = attribute.Key("tenant.name")
	ModuleIDKey    = attribute.Key("module.id")
	ModuleNameKey  = attribute.Key("module.name")
	ServerIDKey    = attribute.Key("server.id")
	ServerNameKey  = attribute.Key("server.name")
	ServerOwnerKey = attribute.Key("server.owner")
	EnvKey         = attribute.Key("server.env")
	DyeingKey      = attribute.Key("tps.dyeing")
	StatusCode     = attribute.Key("status_code")
)

const (
	CMDBRegionIDKey   = attribute.Key("cmdb.region.id")
	CMDBRegionNameKey = attribute.Key("cmdb.region.name")
	CMDBZoneIDKey     = attribute.Key("cmdb.zone.id")
	CMDBZoneNameKey   = attribute.Key("cmdb.zone.name")
	CMDBModuleIDKey   = attribute.Key("cmdb.module.id")
	CMDBModuleNameKey = attribute.Key("cmdb.module.name")
)

const (
	CalleeServiceKey        = attribute.Key("callee.service")
	CalleeMethodKey         = attribute.Key("callee.method")
	CalleeContainerKey      = attribute.Key("callee.container")
	CalleeSystemNameKey     = attribute.Key("callee.system.name")
	CalleeSystemCategoryKey = attribute.Key("callee.system.category")
	CalleeSystemProtocolKey = attribute.Key("callee.system.protocol")

	CallerServiceKey        = attribute.Key("caller.service")
	CallerMethodKey         = attribute.Key("caller.method")
	CallerContainerKey      = attribute.Key("caller.container")
	CallerSystemNameKey     = attribute.Key("caller.system.name")
	CallerSystemCategoryKey = attribute.Key("caller.system.category")
	CallerSystemProtocolKey = attribute.Key("caller.system.protocol")
	CallerSetKey            = attribute.Key("caller.system.set")
)

const (
	NetHostIPKey   = semconv.NetHostIPKey
	NetHostPortKey = semconv.NetHostPortKey
	NetPeerIPKey   = semconv.NetPeerIPKey
	NetPeerPortKey = semconv.NetPeerPortKey
)
