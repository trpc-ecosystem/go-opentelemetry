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

package zaplogs

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/filter"
)

var (
	systemField = zap.String("system", "trpc")
	serverField = zap.String("span.kind", "server")
	clientField = zap.String("span.kind", "client")
)

var globalLogger = zap.NewNop()

// SetLogger set logger
func SetLogger(l *zap.Logger) {
	globalLogger = l
}

// ServerFilter get server filters
func ServerFilter(opts ...Option) filter.ServerFilter {
	o := evaluateServerOpt(opts)
	return func(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (rsp interface{}, err error) {
		msg := trpc.Message(ctx)
		startTime := time.Now()
		var fields []zapcore.Field
		fields = append(fields, zap.String("trpc.start_time", startTime.Format(time.RFC3339)))
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, zap.String("trpc.request.deadline", d.Format(time.RFC3339)))
		}
		fields = append(fields, zap.String("trpc.service", msg.CalleeServiceName()))
		fields = append(fields, zap.String("trpc.method", msg.CalleeMethod()))
		fields = append(fields, []zapcore.Field{systemField, serverField}...)

		globalLogger = globalLogger.With(fields...)
		ctx = ContextWith(ctx, globalLogger)

		rsp, err = handle(ctx, req)

		if !o.shouldLog(msg.CalleeServiceName(), err) {
			return rsp, err
		}

		code := o.codeFunc(err)
		level := o.levelFunc(code)
		duration := o.durationFunc(time.Since(startTime))
		o.messageFunc(ctx, "", level, code, err, duration)
		return rsp, err
	}
}

// ClientFilter get client filters
func ClientFilter(opts ...Option) filter.ClientFilter {
	o := evaluateClientOpt(opts)
	return func(ctx context.Context, req interface{}, rsp interface{}, handle filter.ClientHandleFunc) (err error) {
		msg := trpc.Message(ctx)
		fields := []zapcore.Field{
			systemField,
			clientField,
			zap.String("trpc.service", msg.CalleeService()),
			zap.String("trpc.method", msg.CalleeMethod()),
		}
		globalLogger = globalLogger.With(fields...)
		ctx = ContextWith(ctx, globalLogger)
		startTime := time.Now()
		err = handle(ctx, req, rsp)
		code := o.codeFunc(err)
		level := o.levelFunc(code)
		duration := o.durationFunc(time.Since(startTime))
		o.messageFunc(ctx, "", level, code, err, duration)
		return err
	}
}
