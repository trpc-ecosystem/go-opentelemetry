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

package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
)

var (
	trpcMethod  = attribute.Key("trpc.method")
	trpcService = attribute.Key("trpc.service")
	trpcCode    = attribute.Key("trpc.code")
)

// ServerFilter get server filters
func ServerFilter() filter.ServerFilter {
	return func(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (rsp interface{}, err error) {
		msg := trpc.Message(ctx)
		startTime := time.Now()
		serverStartedCounter.Add(ctx, 1, metric.WithAttributes(trpcService.String(msg.CalleeServiceName()),
			trpcMethod.String(msg.CalleeMethod())))

		rsp, err = handle(ctx, req)

		code := errs.Code(err)
		serverHandledCounter.Add(ctx, 1, metric.WithAttributes(trpcService.String(msg.CalleeServiceName()),
			trpcMethod.String(msg.CalleeMethod()), trpcCode.Int(int(code))))
		serverHandledHistogram.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(trpcService.String(msg.CalleeServiceName()), trpcMethod.String(msg.CalleeMethod())))
		return rsp, err
	}
}

// ClientFilter get client filters
func ClientFilter() filter.ClientFilter {
	return func(ctx context.Context, req, rsp interface{}, handle filter.ClientHandleFunc) (err error) {
		msg := trpc.Message(ctx)
		startTime := time.Now()
		clientStartedCounter.Add(ctx, 1, metric.WithAttributes(trpcService.String(msg.CalleeServiceName()),
			trpcMethod.String(msg.CalleeMethod())))

		err = handle(ctx, req, rsp)

		code := errs.Code(err)
		clientHandledCounter.Add(ctx, 1, metric.WithAttributes(trpcService.String(msg.CalleeServiceName()),
			trpcMethod.String(msg.CalleeMethod()), trpcCode.Int(int(code))))
		clientHandledHistogram.Record(ctx, time.Since(startTime).Seconds(),
			metric.WithAttributes(trpcService.String(msg.CalleeServiceName()), trpcMethod.String(msg.CalleeMethod())))
		return err
	}
}
