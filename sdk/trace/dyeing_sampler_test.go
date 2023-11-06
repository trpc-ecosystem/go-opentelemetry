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

package trace

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"

	"trpc.group/trpc-go/go-opentelemetry/pkg/protocol/opentelemetry-ext/proto/sampler"
)

func TestSampler_shouldSample(t *testing.T) {
	type fields struct {
		samplerConfig SamplerConfig
		description   string
		tenantID      string
		client        sampler.SamplerServiceClient
		sampledKvs    atomic.Value
		debug         bool
		opt           SamplerOptions
	}
	type args struct {
		p trace.SamplingParameters
	}

	DefaultGetCalleeMethodInfo = func(ctx context.Context) MethodInfo {
		return MethodInfo{
			CalleeService: "service1",
			CalleeMethod:  "method1",
		}
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   trace.SamplingResult
	}{
		{
			name: "should sample",
			fields: fields{
				samplerConfig: SamplerConfig{
					Fraction: 0.5,
					SpecialFractions: map[string]SpecialFraction{
						"service1": {
							DefaultFraction:          1,
							Methods:                  nil,
							defaultTraceIDUpperBound: 9223372036854775808,
						},
					},
					traceIDUpperBound: 4611686018427387904,
				},
			},
			args: args{
				p: trace.SamplingParameters{
					ParentContext: context.Background(),
					TraceID:       [16]byte{34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49},
				},
			},
			want: trace.SamplingResult{Decision: trace.RecordAndSample},
		},
		{
			name: "should not sample",
			fields: fields{
				samplerConfig: SamplerConfig{
					Fraction: 0.5,
					SpecialFractions: map[string]SpecialFraction{
						"service1": {
							DefaultFraction:          0,
							Methods:                  nil,
							defaultTraceIDUpperBound: 0,
						},
					},
					traceIDUpperBound: 4611686018427387904,
				},
			},
			args: args{
				p: trace.SamplingParameters{
					ParentContext: context.Background(),
					TraceID:       [16]byte{34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49},
				},
			},
			want: trace.SamplingResult{Decision: trace.Drop},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Sampler{
				samplerConfig: tt.fields.samplerConfig,
				description:   tt.fields.description,
				tenantID:      tt.fields.tenantID,
				client:        tt.fields.client,
				sampledKvs:    tt.fields.sampledKvs,
				debug:         tt.fields.debug,
				opt:           tt.fields.opt,
			}
			if got := ws.shouldSample(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shouldSample() = %v, want %v", got, tt.want)
			}
		})
	}
}
