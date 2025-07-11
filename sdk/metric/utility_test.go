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

// Package metric metric
package metric

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestClientMetricBeforeHandler 确保不会因为label个数不匹配而panic
func TestMetricHandler(t *testing.T) {
	NewClientReporter("", "", "", "", "").Handled(context.Background(), "")
	NewServerReporter("", "", "", "", "").Handled(context.Background(), "")
	t.Run("非UTF8", func(t *testing.T) {
		NewClientReporter("", "\xff非UTF8", "\xff", "\xff非UTF8", "\xff").Handled(context.Background(), "")
		NewServerReporter("", "\xff非UTF8", "\xff", "\xff非UTF8", "\xff").Handled(context.Background(), "")
	})
}

// TestClientMetricBeforeHandler
func TestMetricHandlerWithExemplar(t *testing.T) {
	bsp := sdktrace.NewBatchSpanProcessor(otlptracegrpc.NewUnstarted())
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	ctx, sp := tp.Tracer("").Start(context.Background(), "testSpan")
	defer sp.End()
	NewClientReporter("", "", "", "", "").Handled(ctx, "1")
	NewServerReporter("", "", "", "", "").Handled(ctx, "1")
}

// Test_DefaultCleanRPCMethod()
func Test_DefaultCleanRPCMethod(t *testing.T) {
	type args struct {
		method string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{method: "GetWeDetailPage"},
			want: "GetWeDetailPage",
		},
		{
			name: "",
			args: args{method: "/a/b "},
			want: "default_pattern_method",
		},
		{
			name: "",
			args: args{method: ""},
			want: "-",
		},
		{
			name: "",
			args: args{method: "GetUserByID"},
			want: "GetUserByID",
		},
		{
			name: "",
			args: args{method: "/query/1/book/2"},
			want: "default_pattern_method",
		},
		{
			name: "",
			args: args{method: "?"},
			want: "?",
		},
		{
			name: "",
			args: args{method: "中文"},
			want: "中文",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultCleanRPCMethod(tt.args.method); got != tt.want {
				t.Errorf("defaultCleanRPCMethod()() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMethodToPattern(t *testing.T) {
	RegisterMethodMapping(`/query/\d+/book/\d+`, "/query/:id/book/:id")
	RegisterMethodMapping(`/user/\d+`, "queryUserByID")
	type args struct {
		method string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{method: "/query/1/book/2"},
			want: "/query/:id/book/:id",
		},
		{
			name: "",
			args: args{method: "/user/2?aa=1"},
			want: "queryUserByID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultCleanRPCMethod(tt.args.method); got != tt.want {
				t.Errorf("defaultCleanRPCMethod()() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark_DefaultCleanRPCMethod()
func Benchmark_DefaultCleanRPCMethod(b *testing.B) {
	for i := 0; i < b.N; i++ {
		defaultCleanRPCMethod("/GetWeDetailPage?gjxyasd")
		defaultCleanRPCMethod("Get")
		defaultCleanRPCMethod("Get select")
	}
}
