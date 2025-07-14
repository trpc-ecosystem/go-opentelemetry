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

package metric

import (
	"trpc.group/trpc-go/trpc-opentelemetry/config/codes"
)

type Code = codes.Code
type CodeType = codes.CodeType

const (
	CodeTypeSuccess   = codes.CodeTypeSuccess
	CodeTypeException = codes.CodeTypeException
	CodeTypeTimeout   = codes.CodeTypeTimeout
)

var (
	NewCode = codes.NewCode
)

type CodeTypeFunc = codes.CodeTypeFunc
type CodeTypeMappingDescription = codes.CodeTypeMappingDescription

var (
	DefaultCodeTypeFunc           = codes.DefaultCodeTypeFunc
	defaultCodeTypeFunc           = codes.DefaultCodeTypeFunc
	NewCodeTypeMappingDescription = codes.NewCodeTypeMappingDescription
)
