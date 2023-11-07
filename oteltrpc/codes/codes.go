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

// Package codes trpc
package codes

import (
	"context"
	"strconv"

	"trpc.group/trpc-go/trpc-go/errs"

	"trpc.group/trpc-go/trpc-opentelemetry/config/codes"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

// ConvertMetricsCodes converts codes to metric.CodeTypeMappingDescription
func ConvertMetricsCodes(
	codeTypeMapping map[string]*metric.CodeTypeMappingDescription,
	codes []*codes.Code) []*codes.Code {
	defaultCodeTypeMapping := map[string]*metric.CodeTypeMappingDescription{
		"0":   metric.NewCodeTypeMappingDescription(metric.CodeTypeSuccess, "code=0"),
		"":    metric.NewCodeTypeMappingDescription(metric.CodeTypeSuccess, "code="),
		"101": metric.NewCodeTypeMappingDescription(metric.CodeTypeTimeout, "client timeout"),
		"21":  metric.NewCodeTypeMappingDescription(metric.CodeTypeTimeout, "server timeout"),
	}
	merged := make(map[string]*metric.CodeTypeMappingDescription, len(codeTypeMapping)+len(defaultCodeTypeMapping))
	for k, v := range codeTypeMapping {
		merged[k] = v
	}
	for code, v := range defaultCodeTypeMapping {
		if _, ok := merged[code]; !ok {
			merged[code] = v
		}
	}
	for k, v := range merged {
		codes = append(codes, metric.NewCode(k, metric.CodeType(v.CodeType), v.Description))
	}
	return codes
}

// GetCodeFunc DefaultGetCodeFunc
// Deprecated
type GetCodeFunc func(ctx context.Context, rsp interface{}, err error) (int, error)

// DefaultGetCodeFunc returns the default GetCodeFunc
// Deprecated: Use defaultGetCodeFunc instead, use SetDefaultGetCodeFunc to set it, use GetDefaultGetCodeFunc to get it
var DefaultGetCodeFunc GetCodeFunc = func(ctx context.Context, rsp interface{}, err error) (int, error) {
	if err != nil {
		return int(errs.Code(err)), err
	}
	switch v := rsp.(type) {
	case interface {
		GetRetcode() int32
	}:
		return int(v.GetRetcode()), err
	case interface {
		GetRetCode() int32
	}:
		return int(v.GetRetCode()), err
	case interface {
		GetCode() int32
	}:
		return int(v.GetCode()), err
	default:
		return 0, err
	}
}

// GetStringCodeFunc defaultGetCodeFunc
type GetStringCodeFunc func(ctx context.Context, rsp interface{}, err error) (string, error)

// defaultGetCodeFunc defaultGetCodeFunc
// user can set a custom GetStringCodeFunc
var defaultGetCodeFunc GetStringCodeFunc = func(ctx context.Context, rsp interface{}, err error) (string, error) {
	code, err := DefaultGetCodeFunc(ctx, rsp, err)
	return strconv.Itoa(code), err
}

// SetDefaultGetCodeFunc user can set a custom GetStringCodeFunc
func SetDefaultGetCodeFunc(f GetStringCodeFunc) {
	defaultGetCodeFunc = f
}

// GetDefaultGetCodeFunc getDefaultGetCodeFunc
func GetDefaultGetCodeFunc() GetStringCodeFunc {
	return defaultGetCodeFunc
}
