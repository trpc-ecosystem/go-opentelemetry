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
	"reflect"
	"testing"
	"time"
)

func Test_getMethodsSpecialFraction(t *testing.T) {
	type args struct {
		methodsFraction map[string]MethodFraction
	}
	tests := []struct {
		name string
		args args
		want map[string]MethodFraction
	}{
		{
			name: "normal",
			args: args{
				methodsFraction: map[string]MethodFraction{
					"method1": {
						Fraction: 0.5,
					},
					"method fraction > 1": {
						Fraction: 2,
					},
					"method fraction < 0": {
						Fraction: -1,
					},
				},
			},
			want: map[string]MethodFraction{
				"method1": {
					Fraction:          0.5,
					traceIDUpperBound: 4611686018427387904,
				},
				"method fraction > 1": {
					Fraction:          2,
					traceIDUpperBound: 9223372036854775808,
				},
				"method fraction < 0": {
					Fraction:          -1,
					traceIDUpperBound: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMethodsSpecialFraction(tt.args.methodsFraction); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMethodsSpecialFraction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSamplerConfig(t *testing.T) {
	type args struct {
		config SamplerConfig
	}
	tests := []struct {
		name string
		args args
		want SamplerConfig
	}{
		{
			name: "Test_getSamplerConfig_OK",
			args: args{
				config: SamplerConfig{
					Fraction: 0.01,
					SpecialFractions: map[string]SpecialFraction{
						"service": {
							DefaultFraction: 0.5,
							Methods: map[string]MethodFraction{
								"method": {
									Fraction: 1,
								},
							},
						},
					},
					SamplerServiceAddr: "your.own.gateway.ip:port ",
					SyncInterval:       time.Millisecond,
				},
			},
			want: SamplerConfig{
				Fraction: 0.01,
				SpecialFractions: map[string]SpecialFraction{
					"service": {
						DefaultFraction: 0.5,
						Methods: map[string]MethodFraction{
							"method": {
								Fraction:          1,
								traceIDUpperBound: 9223372036854775808,
							},
						},
						defaultTraceIDUpperBound: 4611686018427387904,
					},
				},
				SamplerServiceAddr: "your.own.gateway.ip:port",
				SyncInterval:       time.Millisecond,
				traceIDUpperBound:  92233720368547760,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSamplerConfig(tt.args.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSamplerConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSamplerSyncInterval(t *testing.T) {
	type args struct {
		syncInterval time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "normal",
			args: args{syncInterval: time.Millisecond},
			want: time.Millisecond,
		},
		{
			name: "default",
			args: args{syncInterval: 0},
			want: time.Second * 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSamplerSyncInterval(tt.args.syncInterval); got != tt.want {
				t.Errorf("getSamplerSyncInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSpecialFraction(t *testing.T) {
	type args struct {
		fractions map[string]SpecialFraction
	}
	tests := []struct {
		name string
		args args
		want map[string]SpecialFraction
	}{
		{
			name: "normal",
			args: args{
				fractions: map[string]SpecialFraction{
					"service": {
						DefaultFraction: 0.5,
						Methods: map[string]MethodFraction{
							"method": {
								Fraction: 1,
							},
						},
					},
				},
			},
			want: map[string]SpecialFraction{
				"service": {
					DefaultFraction: 0.5,
					Methods: map[string]MethodFraction{
						"method": {
							Fraction:          1,
							traceIDUpperBound: 9223372036854775808,
						},
					},
					defaultTraceIDUpperBound: 4611686018427387904,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSpecialFraction(tt.args.fractions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSpecialFraction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTraceIDUpperBound(t *testing.T) {
	type args struct {
		fraction float64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "normal",
			args: args{
				fraction: 0.5,
			},
			want: 4611686018427387904,
		},
		{
			name: "> 1=",
			args: args{fraction: 1},
			want: 9223372036854775808,
		},
		{
			name: "<=0",
			args: args{fraction: -1},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTraceIDUpperBound(tt.args.fraction); got != tt.want {
				t.Errorf("getTraceIDUpperBound() = %v, want %v", got, tt.want)
			}
		})
	}
}
