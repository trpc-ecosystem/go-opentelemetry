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

package oteltrpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/plugin"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	"trpc-system/go-opentelemetry/config"
	"trpc-system/go-opentelemetry/oteltrpc/consts"
	ecosystemtrace "trpc-system/go-opentelemetry/sdk/trace"
)

// language: yaml
const testYaml = `
    opentelemetry:
      attributes:
        - key: tag1
          value: value1
      metrics:
        code_type_mapping:
          0: success|成功
          101: timeout|client超时
          100012: success|session过期
          100013: exception|session错误
        codes:
          - code: 101
            type: timeout
            description: client超时
          - code: 100014
            type: success
            description: desc4
          - code: 100015
            type: exception
            description: desc5
          - code: 100016
            type: success
            description: desc6
            service: service1
          - code: 100017
            type: success
            description: desc7
            method: method1
          - code: 100018
            type: success
            description: desc8
            service: service1
            method: method1
`

func Test_factory_Setup(t *testing.T) {
	type args struct {
		yaml string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "配置codes兼容code_type_mapping",
			args: args{
				yaml: testYaml,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factory{}
			var config map[string]yaml.Node
			err := yaml.Unmarshal([]byte(tt.args.yaml), &config)
			assert.NoError(t, err)
			node := config[consts.PluginName]
			configDec := &plugin.YamlNodeDecoder{Node: &node}
			if err := f.Setup(consts.PluginName, configDec); (err != nil) != tt.wantErr {
				t.Errorf("Setup() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(time.Millisecond * 100) // wait sync setup
		})
	}
}

func Test_recovery(t *testing.T) {
	_ = recovery()(nil, "", &pb.HelloRequest{Msg: "1"}, nil, nil, nil)
}

const testCustomSamplerYaml = `
    opentelemetry:
      sampler:
        fraction: 1.0
        special_fractions:                   
          - callee_service: service1           
            default_fraction: 0.0003          
            callee_methods:               
              - method: method1 
                fraction: 0.004
              - method: method2
                fraction: 0.005
          - callee_service: service2
            default_fraction: 0.0001
            callee_methods:
              - method: method1
                fraction: 0.06
              - method: method2
                fraction: 0.07
      metrics:
        enabled: false
`

func Test_custom_sampler(t *testing.T) {
	DefaultSampler = sdktrace.AlwaysSample()
	var config map[string]yaml.Node
	err := yaml.Unmarshal([]byte(testCustomSamplerYaml), &config)
	assert.NoError(t, err)
	node := config[consts.PluginName]
	f := factory{}
	err = f.Setup(consts.PluginName, &plugin.YamlNodeDecoder{Node: &node})
	assert.NoError(t, err)

	assert.Equal(t, sdktrace.AlwaysSample().Description(), DefaultSampler.Description())
}

func Test_getSpecialFractions(t *testing.T) {
	type args struct {
		fractions []config.SpecialFraction
	}
	tests := []struct {
		name string
		args args
		want map[string]ecosystemtrace.SpecialFraction
	}{
		{
			name: "test config",
			args: args{fractions: []config.SpecialFraction{
				{
					CalleeService:   "service1",
					DefaultFraction: 1,
					CalleeMethods: []config.MethodFraction{
						{
							Method:   "method1",
							Fraction: 0.1,
						},
					},
				},
			}},
			want: map[string]ecosystemtrace.SpecialFraction{
				"service1": {
					DefaultFraction: 1,
					Methods: map[string]ecosystemtrace.MethodFraction{
						"method1": {
							Fraction: 0.1,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getSpecialFractions(tt.args.fractions), "getSpecialFractions(%v)",
				tt.args.fractions)
		})
	}
}
