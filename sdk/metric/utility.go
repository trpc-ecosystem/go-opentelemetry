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

package metric

import (
	"regexp"
	"strings"
)

// CleanRPCMethodFunc CleanRPCMethod func
type CleanRPCMethodFunc func(string) string

// CleanRPCMethod for high-cardinality problem
var CleanRPCMethod = defaultCleanRPCMethod

// SetCleanRPCMethodFunc set cleanRPCMethod for high-cardinality problem
func SetCleanRPCMethodFunc(f CleanRPCMethodFunc) {
	CleanRPCMethod = f
}

// defaultCleanRPCMethod for high-cardinality problem
func defaultCleanRPCMethod(method string) string {
	if method == "" {
		return "-"
	}
	if strings.HasPrefix(method, "/0x") {
		// oidb method
		return strings.ToValidUTF8(method, "")
	}
	if method[0] == '/' { // http path
		// 1. trim http query params (after char '?')
		if idx := strings.IndexByte(method, '?'); idx > 0 {
			method = method[:idx]
		}
		if v, ok := methodToPattern(method); ok {
			return strings.ToValidUTF8(v, "")
		}
		// http服务只信任通过RegisterMethodMapping的pattern, 避免高基数问题
		return "default_pattern_method"
	}
	// 3. limit length<64
	const maxLength = 64
	if len(method) > maxLength {
		method = method[:maxLength]
	}
	return strings.ToValidUTF8(method, "")
}

func cleanServiceName(s string) string {
	if s == "" {
		return "-"
	}
	return strings.ToValidUTF8(s, "")
}

var methodMappings []*MethodMapping

// MethodMapping ...
type MethodMapping struct {
	Regex   *regexp.Regexp
	Pattern string
}

// RegisterMethodMapping 在初始化函数中注册 method regex->pattern 映射, 将含有 path 参数的 高基数method 转换为 低基数的 method pattern,
// regexStr 不合法将 Panic.
func RegisterMethodMapping(regexStr string, pattern string) {
	if !strings.HasPrefix(regexStr, "^") { // 添加完全匹配
		regexStr = "^" + regexStr + "$"
	}
	regex := regexp.MustCompile(regexStr)
	methodMappings = append(methodMappings, &MethodMapping{
		Regex:   regex,
		Pattern: pattern,
	})
}

// methodToPattern
func methodToPattern(method string) (string, bool) {
	if methodMappings == nil {
		return method, false
	}
	for _, v := range methodMappings {
		if v.Regex.MatchString(method) {
			return v.Pattern, true
		}
	}
	return method, false
}
