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

package codes

import (
	"fmt"
	"strings"
)

// CodeTypeFunc ...
type CodeTypeFunc func(code, service, method string) *Code

// DefaultCodeTypeFunc default code mapping。
var DefaultCodeTypeFunc = defaultCodeTypeFunc

var (
	successCodeDesc = &Code{
		Type:        CodeTypeSuccess.String(),
		Description: "code=0",
	}
	exceptionCodeDesc = &Code{
		Type:        CodeTypeException.String(),
		Description: "code!=0",
	}
)

func defaultCodeTypeFunc(code, _, _ string) *Code {
	if code == "0" || code == "" {
		return successCodeDesc
	}
	return exceptionCodeDesc
}

// == Deprecated ==

// CodeTypeMappingDescription describe code_type_mapping information
// Deprecated
type CodeTypeMappingDescription struct {
	// CodeType ...
	CodeType string
	// Description ...
	Description string
}

// NewCodeTypeMappingDescription ...
// Deprecated
func NewCodeTypeMappingDescription(codeType CodeType, desc string) *CodeTypeMappingDescription {
	code := NewCode("", codeType, desc)
	return &CodeTypeMappingDescription{
		CodeType:    code.Type,
		Description: code.Description,
	}
}

// UnmarshalText unmarshal yaml string to *CodeTypeMappingDescription。
// Deprecated
func (d *CodeTypeMappingDescription) UnmarshalText(text []byte) error {
	seg := strings.Split(string(text), "|")
	desc := ""
	if len(seg) > 1 {
		desc = seg[1]
	}
	*d = *NewCodeTypeMappingDescription(CodeType(seg[0]), desc)
	return nil
}

// String ...
// Deprecated
func (d *CodeTypeMappingDescription) String() string {
	return fmt.Sprintf("%+v", *d)
}
