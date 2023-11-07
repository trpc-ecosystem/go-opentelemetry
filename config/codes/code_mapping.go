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
	"sort"
	"strconv"
	"sync/atomic"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/protocol/opentelemetry-ext/proto/operation"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/remote"
)

var (
	mapper = defaultMapper()
)

// CodeMapper mapping interface
type CodeMapper interface {
	Mapping(code, service, method string) *Code
}

// CodeConverter implements CodeMapper interface
type CodeConverter struct {
	configurator        remote.Configurator
	codes               []*Code
	atomicCodeMapping   *atomic.Value
	defaultCodeTypeFunc CodeTypeFunc
}

// New trpc codes
func New(opts ...SetupOption) *CodeConverter {
	cfg := &setupOption{}
	for _, opt := range opts {
		opt(cfg)
	}

	f := cfg.defaultCodeTypeFunc
	if f == nil {
		f = DefaultCodeTypeFunc
	}
	c := &CodeConverter{codes: cfg.Codes,
		configurator:        cfg.Configurator,
		defaultCodeTypeFunc: f}
	c.init()
	return c
}

func (cm *CodeConverter) init() {
	cm.atomicCodeMapping = &atomic.Value{}
	cm.atomicCodeMapping.Store(covertCodesToMapping(cm.codes))
	if cm.configurator != nil {
		cm.configurator.RegisterConfigApplyFunc(genConfigApplyFunc(cm.codes, cm.atomicCodeMapping))
	}
}

// Mapping redefine the return code
func (cm *CodeConverter) Mapping(code, service, method string) *Code {
	if v, ok := cm.atomicCodeMapping.Load().(map[string][]*Code)[code]; ok {
		if c := cm.matchCode(v, service, method); c != nil {
			return c
		}
	}

	return cm.defaultCodeTypeFunc(code, service, method)
}

func (cm *CodeConverter) matchCode(codeArray []*Code, service string, method string) *Code {
	for _, vv := range codeArray {
		if vv.Service == service || vv.Method == method {
			return vv
		}

		// If the service/method is not configured, check whether there is a global configuration
		if vv.Service == "" && vv.Method == "" {
			return vv
		}
	}
	return nil
}

type setupOption struct {
	Codes []*Code `yaml:"codes"`
	// Configurator supports dynamic configuration, experimental features
	Configurator        remote.Configurator `yaml:"-"`
	defaultCodeTypeFunc CodeTypeFunc
}

// SetupOption Configuration method of codes, specify codes, configurator, etc.
type SetupOption func(setupOption *setupOption)

// WithCodes ...
func WithCodes(codes []*Code) SetupOption {
	return func(o *setupOption) {
		o.Codes = codes
	}
}

// WithConfigurator set configurator
func WithConfigurator(configurator remote.Configurator) SetupOption {
	return func(o *setupOption) {
		o.Configurator = configurator
	}
}

// WithDefaultFunc set the default return code redefinition method
func WithDefaultFunc(f CodeTypeFunc) SetupOption {
	return func(o *setupOption) {
		o.defaultCodeTypeFunc = f
	}
}

func defaultMapper() *atomic.Value {
	v := &atomic.Value{}
	v.Store(New())
	return v
}

// SetMapper set user-defined mapper
func SetMapper(m CodeMapper) {
	mapper.Store(m)
}

// CodeMapping redefine code
func CodeMapping(code, service, method string) *Code {
	if m, ok := mapper.Load().(CodeMapper); ok {
		return m.Mapping(code, service, method)
	}

	return nil
}

func covertCodesToMapping(codes []*Code) map[string][]*Code {
	codeMapping := make(map[string][]*Code, len(codes))

	// Matching specific service and method at the same time has the highest priority,
	// and service takes precedence over method
	sort.Slice(codes, func(i, j int) bool {
		return (codes[i].Service != "" && codes[j].Service == "") ||
			(codes[i].Method != "" && codes[j].Method == "")
	})
	for _, v := range codes {
		codeMapping[v.Code] = append(codeMapping[v.Code], v)
	}
	return codeMapping
}

func genConfigApplyFunc(codes []*Code, atomicCodeMapping *atomic.Value) remote.ConfigApplyFunc {
	return func(config *operation.Operation) error {
		// codeMapping
		codeMapping := covertCodesToMapping(codes)
		for _, v := range config.GetMetric().GetCodes() {
			codeStr := strconv.FormatInt(int64(v.GetCode()), 10)
			desc := NewCode(codeStr, CodeType(v.GetType()), v.GetDescription())
			desc.Service = v.GetService()
			desc.Method = v.GetMethod()
			codeMapping[codeStr] = append(codeMapping[codeStr], desc)
		}
		atomicCodeMapping.Store(codeMapping)
		return nil
	}
}
