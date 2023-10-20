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

package zaplogs

import (
	"context"

	"go.uber.org/zap"
)

type ctxMarker struct{}

type ctxLogger struct {
	logger *zap.Logger
}

var (
	ctxZapKey = &ctxMarker{}
	nopLogger = zap.NewNop()
)

// FromContext get logger from ctx
func FromContext(ctx context.Context) *zap.Logger {
	l, ok := ctx.Value(ctxZapKey).(*ctxLogger)

	if !ok || l == nil {
		return nopLogger
	}

	return l.logger
}

// ContextWith set logger to ctx
func ContextWith(ctx context.Context, logger *zap.Logger) context.Context {
	l := &ctxLogger{
		logger: logger,
	}
	return context.WithValue(ctx, ctxZapKey, l)
}
