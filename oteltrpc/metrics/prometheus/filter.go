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

package prometheus

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/http"

	trpccodes "trpc-system/go-opentelemetry/oteltrpc/codes"
	"trpc-system/go-opentelemetry/sdk/metric"
)

// ServerFilter with prometheus metric
func ServerFilter(options ...ServerFilterOption) filter.ServerFilter {
	filterConfig := &serverFilterOption{}
	for _, opt := range options {
		opt(filterConfig)
	}
	return func(ctx context.Context, req interface{}, handle filter.ServerHandleFunc) (rsp interface{}, err error) {
		msg := trpc.Message(ctx)
		calleeMethod := msg.CalleeMethod()
		if head := http.Head(ctx); head != nil && head.Request != nil {
			calleeMethod = fmt.Sprintf("[%s]%s", head.Request.Method, metric.CleanRPCMethod(head.Request.URL.Path))
		}

		r := metric.NewServerReporter("trpc", msg.CallerServiceName(), msg.CallerMethod(),
			msg.CalleeServiceName(), calleeMethod, metric.WithServerTraceConfig(filterConfig.enableDeferredSample,
				filterConfig.deferredSampleError, filterConfig.deferredSampleSlowDuration))
		rsp, err = handle(ctx, req)
		code, _ := trpccodes.GetDefaultGetCodeFunc()(ctx, rsp, err)
		r.Handled(ctx, code)
		return rsp, err
	}
}

// ClientFilter with prometheus metric
func ClientFilter(options ...ClientFilterOption) filter.ClientFilter {
	filterConfig := &clientFilterOption{}
	for _, opt := range options {
		opt(filterConfig)
	}
	return func(ctx context.Context, req, rsp interface{}, handle filter.ClientHandleFunc) (err error) {
		msg := trpc.Message(ctx)
		md := msg.ClientMetaData()
		monitorRequestSize(req, md)

		r := metric.NewClientReporter("trpc", msg.CallerServiceName(), msg.CallerMethod(),
			msg.CalleeServiceName(), msg.CalleeMethod(), metric.WithClientTraceConfig(filterConfig.enableDeferredSample,
				filterConfig.deferredSampleError, filterConfig.deferredSampleSlowDuration))

		err = handle(ctx, req, rsp)

		code, _ := trpccodes.GetDefaultGetCodeFunc()(ctx, rsp, err)
		r.Handled(ctx, code)
		return err
	}
}

type (
	// ServerFilterOption defines the configuration for the various elements of BatchProcessor
	ServerFilterOption func(opt *serverFilterOption)
	// ClientFilterOption defines the configuration for the various elements of BatchProcessor
	ClientFilterOption func(opt *clientFilterOption)
)

// WithServerFilterTraceConfig return Option which apply changes to trace configuration
func WithServerFilterTraceConfig(enableDeferredSample, deferredSampleError bool,
	deferredSampleSlowDuration time.Duration) ServerFilterOption {
	return func(opt *serverFilterOption) {
		opt.enableDeferredSample = enableDeferredSample
		opt.deferredSampleError = deferredSampleError
		opt.deferredSampleSlowDuration = deferredSampleSlowDuration
	}
}

// WithClientFilterTraceConfig return Option which apply changes to trace configuration
func WithClientFilterTraceConfig(enableDeferredSample, deferredSampleError bool,
	deferredSampleSlowDuration time.Duration) ClientFilterOption {
	return func(opt *clientFilterOption) {
		opt.enableDeferredSample = enableDeferredSample
		opt.deferredSampleError = deferredSampleError
		opt.deferredSampleSlowDuration = deferredSampleSlowDuration
	}
}

// calcBodySize calc proto request size
func calcBodySize(body interface{}) int {
	switch req := body.(type) {
	case proto.Message:
		return proto.Size(req)
	default:
		return 0
	}
}

// calcMetaDataSize calc meta size
func calcMetaDataSize(md codec.MetaData) int {
	if len(md) == 0 {
		return 0
	}

	size := 0
	for _, v := range md {
		size += len(v)
	}
	return size
}

// monitorRequestSize monitor request size
func monitorRequestSize(req interface{}, md codec.MetaData) {
	ObserveRequestBodyBytes(calcBodySize(req))
	ObserveRequestMataDataBytes(calcMetaDataSize(md))
}

type serverFilterOption struct {
	enableDeferredSample       bool
	deferredSampleError        bool
	deferredSampleSlowDuration time.Duration
}

type clientFilterOption struct {
	enableDeferredSample       bool
	deferredSampleError        bool
	deferredSampleSlowDuration time.Duration
}
