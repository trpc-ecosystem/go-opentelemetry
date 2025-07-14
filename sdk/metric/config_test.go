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

// Package metric
package metric

import "testing"

func Test_toKey(t *testing.T) {
	type args struct {
		ins *Instance
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"null-key-suffix",
			args{
				ins: &Instance{
					Addr:     "127.0.0.1:9091",
					TenantID: "default",
				},
			},
			"/opentelemetry/metrics/services/default/127.0.0.1:9091",
		},
		{
			"with-key-suffix",
			args{
				ins: &Instance{
					Addr:      "127.0.0.1:9091",
					TenantID:  "default",
					KeySuffix: "_default",
				},
			},
			"/opentelemetry/metrics/services/default/127.0.0.1:9091_default",
		},
		{
			"with-key",
			args{
				ins: &Instance{
					Addr:     "127.0.0.1:9091",
					TenantID: "default",
					Key:      "/opentelemetry/custom/key",
				},
			},
			"/opentelemetry/custom/key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.ins.GetKey(); got != tt.want {
				t.Errorf("toKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toValue(t *testing.T) {
	type args struct {
		ins *Instance
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"null-key-suffix",
			args{
				ins: &Instance{
					Addr:     "127.0.0.1:9091",
					TenantID: "default",
				},
			},
			`{"addr":"127.0.0.1:9091","tenant_id":"default","metadata":null}`,
		},
		{
			"with-key-suffix",
			args{
				ins: &Instance{
					Addr:      "127.0.0.1:9091",
					TenantID:  "default",
					KeySuffix: "_default",
				},
			},
			`{"addr":"127.0.0.1:9091","tenant_id":"default","metadata":null}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.ins.GetValue(); got != tt.want {
				t.Errorf("toValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
