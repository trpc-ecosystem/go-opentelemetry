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

package metric

import (
	"trpc.group/trpc-go/go-opentelemetry/sdk/metric/internal/registry"
)

type Registry = registry.Registry
type RegistryInstance = registry.Instance

var (
	// DefaultRegisterTTL default register ttl
	DefaultRegisterTTL = registry.DefaultRegisterTTL
	// DefaultDialTimeout default dail timeout
	DefaultDialTimeout = registry.DefaultDialTimeout
)

var (
	NewEtcdRegistry = registry.NewEtcdRegistry
)
