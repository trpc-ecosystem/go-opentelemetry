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
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/log"

	"trpc.group/trpc-go/go-opentelemetry/config"
	"trpc.group/trpc-go/go-opentelemetry/oteltrpc/logs"
)

func Test_doFlowLog(t *testing.T) {
	type args struct {
		flow    *logs.FlowLog
		options FilterOptions
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "disable",
			args: args{
				flow:    &logs.FlowLog{},
				options: FilterOptions{TraceLogMode: config.LogModeDisable},
			},
			want: 0,
		},
		{
			name: "enable",
			args: args{
				flow:    &logs.FlowLog{},
				options: FilterOptions{},
			},
			want: 1,
		},
		{
			name: "exclude by service",
			args: args{
				flow: &logs.FlowLog{Source: logs.Service{Name: "serviceName"}},
				options: FilterOptions{
					TraceLogOption: config.TraceLogOption{
						Exclude: []config.TraceLogRule{
							{
								Service: "serviceName",
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "exclude by method",
			args: args{
				flow: &logs.FlowLog{Source: logs.Service{Method: "method"}},
				options: FilterOptions{
					TraceLogOption: config.TraceLogOption{
						Exclude: []config.TraceLogRule{
							{
								Method: "method",
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "exclude by code",
			args: args{
				flow: &logs.FlowLog{Status: logs.Status{Code: 0}},
				options: FilterOptions{
					TraceLogOption: config.TraceLogOption{
						Exclude: []config.TraceLogRule{
							{
								Code: "0",
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "exclude multi",
			args: args{
				flow: &logs.FlowLog{Status: logs.Status{Code: 123}},
				options: FilterOptions{
					TraceLogOption: config.TraceLogOption{
						Exclude: []config.TraceLogRule{
							{
								Service: "serviceName",
							},
							{
								Code: "123",
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "not exclude by service",
			args: args{
				flow: &logs.FlowLog{Source: logs.Service{Name: "serviceName1"}},
				options: FilterOptions{
					TraceLogOption: config.TraceLogOption{
						Exclude: []config.TraceLogRule{
							{
								Service: "serviceName",
							},
						},
					},
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLogger()
			log.SetLogger(logger)
			doFlowLog(context.Background(), tt.args.flow, tt.args.options)
			assert.Equal(t, tt.want, logger.count)
		})
	}
}

type testLogger struct {
	log.Logger
	count int
}

func newTestLogger() *testLogger {
	return &testLogger{
		Logger: log.DefaultLogger,
	}
}

// Debugf implement tRPC log.Logger
func (t *testLogger) Debugf(format string, args ...interface{}) {
	t.count++
	t.Logger.Debugf(format, args...)
}
