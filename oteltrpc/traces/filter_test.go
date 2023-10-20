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

package traces

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	"trpc-system/go-opentelemetry/api"
	"trpc-system/go-opentelemetry/config"
	"trpc-system/go-opentelemetry/oteltrpc/codes"
)

// BenchmarkServerFilter
// BenchmarkServerFilter-12    	  711921	      1708 ns/op	    1440 B/op	      22 allocs/op
func BenchmarkServerFilter(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	handle := func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return &pb.HelloReply{}, nil
	}
	f := ServerFilter(func(options *FilterOptions) {
		options.TraceLogMode = config.LogModeDisable
	})
	log.SetLogger(log.NewZapLog(nil))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f(ctx, req, handle)
	}
}

// BenchmarkClientFilter
// BenchmarkClientFilter-12    	 1564944	       756.6 ns/op	     661 B/op	       8 allocs/op
func BenchmarkClientFilter(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{}
	handle := func(ctx context.Context, req interface{}, rsp interface{}) (err error) { return nil }
	f := ClientFilter(func(options *FilterOptions) {
		options.TraceLogMode = config.LogModeDisable
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f(ctx, req, rsp, handle)
	}
}

func Test_startServerSpan(t *testing.T) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{},
		propagation.Baggage{}))

	ctx := trpc.BackgroundContext()
	msg := trpc.Message(ctx)
	md := codec.MetaData{
		api.TraceparentHeader: []byte("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
	}
	msg.WithServerMetaData(md)
	type args struct {
		ctx context.Context
		req interface{}
		msg codec.Msg
		md  codec.MetaData
		opt FilterOptions
	}
	tests := []struct {
		name    string
		args    args
		sampled bool
	}{
		{"test-parent-trace", args{ctx: ctx, req: &pb.HelloRequest{},
			msg: msg, md: md, opt: FilterOptions{
				TraceLogMode:          config.LogModeOneLine,
				DisableParentSampling: false}}, true},
		{"test-disable-parent-trace", args{ctx: ctx, req: &pb.HelloRequest{},
			msg: msg, md: md, opt: FilterOptions{
				TraceLogMode:          config.LogModeOneLine,
				DisableParentSampling: true}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := startServerSpan(tt.args.ctx, tt.args.req, tt.args.msg, tt.args.md, tt.args.opt)
			if span.SpanContext().IsSampled() != tt.sampled {
				t.Errorf("startServerSpan() sampled = %v, want %v", span.SpanContext().IsSampled(), tt.sampled)
			}
		})
	}
}

func TestServerFilter(t *testing.T) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{},
		propagation.Baggage{}))

	msg := "no"
	f := ServerFilter(func(options *FilterOptions) {})
	type args struct {
		ctx    context.Context
		req    interface{}
		handle func(ctx context.Context, req interface{}) (rsp interface{}, err error)
	}
	tests := []struct {
		name    string
		args    args
		setup   func()
		want    interface{}
		wantErr bool
	}{
		{"test-get-code-with-nil-error",
			args{
				ctx: trpc.BackgroundContext(),
				req: &pb.HelloRequest{},
				handle: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return &pb.HelloReply{}, nil
				},
			}, func() {
			},
			&pb.HelloReply{},
			false,
		},
		{"test-get-code-with-codefunc-error",
			args{
				ctx: trpc.BackgroundContext(),
				req: &pb.HelloRequest{},
				handle: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return &pb.HelloReply{Msg: msg}, nil
				},
			}, func() {
				codes.DefaultGetCodeFunc = func(ctx context.Context, rsp interface{}, err error) (int, error) {
					if err != nil {
						return int(errs.Code(err)), err
					}

					if r, ok := rsp.(*pb.HelloReply); ok {
						if r.GetMsg() == msg {
							return 10000, err
						}
					}
					return 0, err
				}
			},
			&pb.HelloReply{Msg: msg},
			false,
		},
		{"test-get-code-with-trpc-error",
			args{
				ctx: trpc.BackgroundContext(),
				req: &pb.HelloRequest{},
				handle: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return nil, errs.New(10001, "10001 error")
				},
			}, func() {
			},
			nil,
			true,
		},
		{"test-get-code-with-errors-error",
			args{
				ctx: trpc.BackgroundContext(),
				req: &pb.HelloRequest{},
				handle: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return nil, errors.New("errors error")
				},
			}, func() {
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got, err := f(tt.args.ctx, tt.args.req, tt.args.handle)
			assert.Equal(t, tt.wantErr, err != nil, "ServerFilter() err = %v, wantErr %t", err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
