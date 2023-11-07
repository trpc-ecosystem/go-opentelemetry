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
	"math/rand"
	"testing"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	"trpc.group/trpc-go/trpc-opentelemetry/config"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// BenchmarkServerFilter_DisableTraceBody disable trace body
func BenchmarkServerFilter_DisableTraceBody(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	handle := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
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

// BenchmarkServerFilter1024 report body
func BenchmarkServerFilter1024(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{
		Msg: randStr(1024),
	}
	handle := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
	f := ServerFilter()
	log.SetLogger(log.NewZapLog(nil))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f(ctx, req, handle)
	}
}

// BenchmarkServerFilter10240 report body
func BenchmarkServerFilter10240(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{
		Msg: randStr(10240),
	}
	handle := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
	f := ServerFilter(func(options *FilterOptions) {
		options.TraceLogMode = config.LogModeDefault
	})
	log.SetLogger(log.NewZapLog(nil))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f(ctx, req, handle)
	}
}

// BenchmarkClientFilter_DisableTraceBody
func BenchmarkClientFilter_DisableTraceBody(b *testing.B) {
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

// BenchmarkClientFilter1024
// BenchmarkClientFilter-12   	 1564944	       756.6 ns/op	     661 B/op	       8 allocs/op
func BenchmarkClientFilter1024(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{
		Msg: randStr(1024),
	}
	handle := func(ctx context.Context, req interface{}, rsp interface{}) (err error) { return nil }
	f := ClientFilter(func(options *FilterOptions) {
		options.TraceLogMode = config.LogModeDefault
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f(ctx, req, rsp, handle)
	}
}

// BenchmarkClientFilter10240
// BenchmarkClientFilter-12   	 1564944	       756.6 ns/op	     661 B/op	       8 allocs/op
func BenchmarkClientFilter10240(b *testing.B) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{
		Msg: randStr(10240),
	}
	handle := func(ctx context.Context, req interface{}, rsp interface{}) (err error) { return nil }
	f := ClientFilter(func(options *FilterOptions) {
		options.TraceLogMode = config.LogModeDefault
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f(ctx, req, rsp, handle)
	}
}
