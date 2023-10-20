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

package codes

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"trpc-system/go-opentelemetry/pkg/protocol/opentelemetry-ext/proto/operation"
	"trpc-system/go-opentelemetry/sdk/remote"
)

func TestCodeConverter_Map(t *testing.T) {
	type args struct {
		code    string
		service string
		method  string
	}
	tests := []struct {
		name string
		c    *CodeConverter
		args args
		want *Code
	}{
		{"test-default", New(), args{"", "", ""}, successCodeDesc},
		{"test-default-zero-code", New(), args{"0", "", ""}, successCodeDesc},
		{"test-default-non-zero-code", New(), args{"10003", "", ""}, exceptionCodeDesc},
		{"test-with-codes", New(WithCodes([]*Code{
			{
				Code: "10003",
				Type: CodeTypeSuccess.String(),
			},
		})), args{"10003", "", ""}, &Code{
			Code: "10003",
			Type: CodeTypeSuccess.String(),
		}},
		{"test-with-multiple-codes", New(WithCodes([]*Code{
			{
				Code: "10003",
				Type: CodeTypeSuccess.String(),
			},
			{
				Code:    "10003",
				Type:    CodeTypeTimeout.String(),
				Service: "myservice",
			},
		})), args{"10003", "myservice", ""}, &Code{
			Code:    "10003",
			Type:    CodeTypeTimeout.String(),
			Service: "myservice",
		}},
		{"test-multiple-codes-with-method", New(WithCodes([]*Code{
			{
				Code: "10003",
				Type: CodeTypeSuccess.String(),
			},
			{
				Code:    "10003",
				Type:    CodeTypeTimeout.String(),
				Service: "myservice",
			},
			{
				Code:   "10003",
				Type:   CodeTypeTimeout.String(),
				Method: "mymethod",
			},
		})), args{"10003", "myservice", "mymethod"}, &Code{
			Code:   "10003",
			Type:   CodeTypeTimeout.String(),
			Method: "mymethod",
		}},
		{"test-multiple-codes-with-service-method", New(WithCodes([]*Code{
			{
				Code: "10003",
				Type: CodeTypeSuccess.String(),
			},
			{
				Code:    "10003",
				Type:    CodeTypeTimeout.String(),
				Service: "myservice",
			},
			{
				Code:   "10003",
				Type:   CodeTypeTimeout.String(),
				Method: "mymethod",
			},
			{
				Code:    "10003",
				Type:    CodeTypeException.String(),
				Method:  "mymethod",
				Service: "myservice",
			},
		})), args{"10003", "myservice", "mymethod"}, &Code{
			Code:    "10003",
			Type:    CodeTypeException.String(),
			Method:  "mymethod",
			Service: "myservice",
		}},
		{"test-with-configurator", New(WithCodes([]*Code{
			{
				Code: "10003",
				Type: CodeTypeSuccess.String(),
			},
		}), WithConfigurator(newDummyConfigurator())), args{"10000", "", ""}, &Code{
			Code:        "10000",
			Type:        CodeTypeSuccess.String(),
			Description: "10000 to success",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "test-with-configurator" {
				time.Sleep(time.Second)
			}
			assert.Equalf(t, tt.want, tt.c.Mapping(tt.args.code, tt.args.service, tt.args.method),
				"Mapping(%v, %v, %v)", tt.args.code, tt.args.service, tt.args.method)
		})
	}
}

func newDummyConfigurator() remote.Configurator {
	os.Setenv("TPS_TRACE", "remote")
	addr := setupRemoteConfigServer(func() *operation.Operation {
		return &operation.Operation{
			Metric: &operation.Metric{
				Codes: []*operation.Code{
					{
						Type:        "success",
						Code:        10000,
						Description: "10000 to success",
					},
				},
			},
		}
	})
	time.Sleep(time.Millisecond * 10) // wait sync
	configurator := remote.NewRemoteConfigurator(addr, 0, "", "", "")
	return configurator
}

func setupRemoteConfigServer(handler func() *operation.Operation) string {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return ""
	}
	s := grpc.NewServer()
	operation.RegisterOperationServiceServer(s, &testOperationServiceServer{
		getHandler: handler,
	})
	go func() {
		_ = s.Serve(ln)
	}()
	return ln.Addr().String()
}

type testOperationServiceServer struct {
	operation.UnimplementedOperationServiceServer
	getHandler func() *operation.Operation
}

func (ts *testOperationServiceServer) SetOperation(ctx context.Context, req *operation.SetOperationRequest) (
	rsp *operation.SetOperationResponse, err error) {
	return nil, nil
}

func (ts *testOperationServiceServer) GetOperation(ctx context.Context, req *operation.GetOperationRequest) (
	rsp *operation.GetOperationResponse, err error) {
	return &operation.GetOperationResponse{Operation: ts.getHandler()}, nil
}
