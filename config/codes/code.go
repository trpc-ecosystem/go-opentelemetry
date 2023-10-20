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
	"fmt"
	"strings"
)

const (
	descriptionMaxLength = 32 // description max length
)

func legalizeDescription(description string) string {
	if len(description) > descriptionMaxLength {
		description = description[:descriptionMaxLength]
	}
	// description needs to be utf-8
	return strings.ToValidUTF8(description, "")
}

// Code structure.
type Code struct {
	// Code in string
	Code string `yaml:"code"`
	// Type of code
	Type string `yaml:"type"`
	// Description code description
	Description string `yaml:"description"`
	// Service empty means full match
	Service string `yaml:"service"`
	// Method empty means full match
	Method string `yaml:"method"`
}

func newCode(code, codeType, desc string) *Code {
	return &Code{
		Code:        code,
		Type:        codeType,
		Description: desc,
	}
}

// NewCode create a new code
func NewCode(code string, codeType CodeType, description string) *Code {
	desc := legalizeDescription(description)
	switch v := codeType; v {
	case CodeTypeSuccess, CodeTypeException, CodeTypeTimeout:
		return newCode(code, v.String(), desc)
	default:
		return newCode(code, CodeTypeSuccess.String(), desc)
	}
}

// String ...
func (c *Code) String() string {
	return fmt.Sprintf("%+v", *c)
}
