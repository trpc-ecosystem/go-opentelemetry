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

package log

import "strings"

// Level is a logging priority. Higher levels are more important.
type Level string

const (
	// TraceLevel A fine-grained debugging event. Typically disabled in default configurations.
	TraceLevel Level = "TRACE"

	// DebugLevel A debugging event.
	DebugLevel Level = "DEBUG"

	// InfoLevel An informational event. Indicates that an event happened.
	InfoLevel Level = "INFO"

	// WarnLevel A warning event. Not an error but is likely more important than an informational event.
	WarnLevel Level = "WARN"

	// ErrorLevel An error event. Something went wrong.
	ErrorLevel Level = "ERROR"

	// FatalLevel A fatal error such as application or system crash.
	FatalLevel Level = "FATAL"
)

// UnmarshalText unmarshals text to a level.
func (m *Level) UnmarshalText(text []byte) error {
	*m = Level(strings.ToUpper(string(text)))
	return nil
}
