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
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// BenchmarkServerFilter
// BenchmarkServerFilter-12    	 1150906	       988.6 ns/op	     496 B/op	       6 allocs/op
func BenchmarkServerFilter(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	handle := func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return &pb.HelloReply{}, nil
	}
	f := ServerFilter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f(ctx, req, handle)
	}
}

// BenchmarkClientFilter
// BenchmarkClientFilter-12    	 1231050	       988.1 ns/op	     496 B/op	       6 allocs/op
func BenchmarkClientFilter(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{}
	handle := func(ctx context.Context, req interface{}, rsp interface{}) (err error) { return nil }
	f := ClientFilter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f(ctx, req, rsp, handle)
	}
}

// BenchmarkServerFilterWithSpanSampled add exemplar benchmark
// BenchmarkServerFilterWithSpanSampled-12    	  452018	      2716 ns/op	    1952 B/op	      25 allocs/op
func BenchmarkServerFilterWithSpanSampled(b *testing.B) {
	bsp := sdktrace.NewBatchSpanProcessor(otlptracegrpc.NewUnstarted())
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	ctx, sp := tp.Tracer("").Start(context.Background(), "testSpan")
	defer sp.End()
	req := &pb.HelloRequest{}
	handle := func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		return nil, errors.New("error")
	}
	f := ServerFilter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f(ctx, req, handle)
	}
}

// BenchmarkClientFilterWithSpanSampled add exemplar benchmark
// BenchmarkClientFilterWithSpanSampled-12    	  442316	      2557 ns/op	    1952 B/op	      25 allocs/op
func BenchmarkClientFilterWithSpanSampled(b *testing.B) {
	bsp := sdktrace.NewBatchSpanProcessor(otlptracegrpc.NewUnstarted())
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	ctx, sp := tp.Tracer("").Start(context.Background(), "testSpan")
	defer sp.End()
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{}
	handle := func(ctx context.Context, req interface{}, rsp interface{}) (err error) {
		return errors.New("error")
	}
	f := ClientFilter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f(ctx, req, rsp, handle)
	}
}
