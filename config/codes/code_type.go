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

const (
	CodeTypeSuccess   CodeType = "success"
	CodeTypeException CodeType = "exception"
	CodeTypeTimeout   CodeType = "timeout"
)

// CodeType ...
type CodeType string

// String ...
func (c CodeType) String() string {
	s := string(c)
	if s == "" {
		return "success"
	}
	return s
}
