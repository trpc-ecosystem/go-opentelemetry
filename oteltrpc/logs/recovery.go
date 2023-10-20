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

package logs

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"

	"trpc-system/go-opentelemetry/pkg/stack"
	"trpc-system/go-opentelemetry/sdk/metric"
)

// RecoveryHandler recovery
type RecoveryHandler func(ctx context.Context, panicErr interface{}) error

// DefaultRecoveryHandler default recovery
var DefaultRecoveryHandler = handlePanic

// FilterOptions filter
type FilterOptions struct {
	DisableRecovery bool
}

// FilterOption filter options
type FilterOption func(*FilterOptions)

// LogRecoveryFilter  log recovery filter
func LogRecoveryFilter(opts ...FilterOption) filter.ServerFilter {
	var opt FilterOptions
	for _, o := range opts {
		o(&opt)
	}
	return func(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (rsp interface{}, err error) {
		defer func() {
			if rerr := recover(); rerr != nil {
				err = DefaultRecoveryHandler(ctx, rerr)
				metric.ServerPanicTotal.WithLabelValues("trpc").Inc()
				if opt.DisableRecovery {
					panic(rerr)
				}
			}
		}()

		return handle(ctx, req)
	}
}

func handlePanic(ctx context.Context, err interface{}) error {
	stackInfo := stack.Stack(3)
	stacktrace := fmt.Sprintf("[RECOVERY] %s panic recovered:\n%s\n%s",
		stack.TimeFormat(time.Now()), err, stackInfo)

	log.ErrorContext(ctx, stacktrace)
	return errs.NewFrameError(errs.RetServerSystemErr, fmt.Sprint(err))
}
