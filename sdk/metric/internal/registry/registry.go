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

package registry

import (
	"context"
	"time"
)

// Instance instance for registry
type Instance interface {
	GetKey() string
	GetValue() string
}

// Registry register or unregister instance to registry
type Registry interface {
	// Register register a instance to the registry.
	Register(ctx context.Context, ins Instance, ttl time.Duration) (context.CancelFunc, error)
}
