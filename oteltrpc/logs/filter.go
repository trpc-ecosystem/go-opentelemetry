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

package logs

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"

	apilog "trpc.group/trpc-go/trpc-opentelemetry/api/log"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/semconv"
)

var (
	systemField = attribute.String("system", "trpc")
	serverField = attribute.String("span.kind", "server")
	clientField = attribute.String("span.kind", "client")
)

func decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}
func durationToMilliseconds(duration time.Duration) float64 {
	return decimal(float64(duration.Nanoseconds()/1000) / 1000)
}

// ServerFilter get server filters
func ServerFilter() filter.ServerFilter {
	return func(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (rsp interface{}, err error) {
		startTime := time.Now()
		var fields []attribute.KeyValue
		fields = append(fields, attribute.String("trpc.start_time", startTime.Format(time.RFC3339)))
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, attribute.String("trpc.request.deadline", d.Format(time.RFC3339)))
		}

		rsp, err = handle(ctx, req)

		code := errs.Code(err)

		fields = append(fields, semconv.KeyValues(ctx)...)
		fields = append(fields, []attribute.KeyValue{systemField, serverField}...)
		fields = append(fields, attribute.Float64("trpc.time_ms", durationToMilliseconds(time.Since(startTime))))
		fields = append(fields, attribute.Int("trpc.code", int(code)))
		apilog.GlobalLogger().Log(ctx, "", apilog.WithLevel(apilog.InfoLevel),
			apilog.WithFields(fields...),
			apilog.WithStandaloneEnable())
		return rsp, err
	}
}

// ClientFilter get client filters
func ClientFilter() filter.ClientFilter {
	return func(ctx context.Context, req interface{}, rsp interface{}, handle filter.ClientHandleFunc) (err error) {
		fields := []attribute.KeyValue{
			systemField,
			clientField,
		}
		startTime := time.Now()
		err = handle(ctx, req, rsp)
		code := errs.Code(err)

		fields = append(fields, semconv.KeyValues(ctx)...)
		fields = append(fields, attribute.Float64("trpc.time_ms", durationToMilliseconds(time.Since(startTime))))
		fields = append(fields, attribute.Int("trpc.code", int(code)))
		apilog.Info(ctx, "", fields...)
		return err
	}
}
