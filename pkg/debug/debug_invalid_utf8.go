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

package debug

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/grpc/status"
)

// UTF8Debugger SDK debugger
type UTF8Debugger interface {
	Enabled() bool
	DebugSpansInvalidUTF8(exportErr error, batch []sdktrace.ReadOnlySpan)
	DebugLogsInvalidUTF8(exportErr error, batch []*logsproto.ResourceLogs)
}

// NewUTF8Debugger new debugger
func NewUTF8Debugger() UTF8Debugger {
	d := &debugger{}
	// export OTEL_SDK_DEBUG=utf8
	if sdkDebugEnv := os.Getenv("OTEL_SDK_DEBUG"); strings.Contains(sdkDebugEnv, "utf8") {
		log.Printf("opentelemetry: env OTEL_SDK_DEBUG:%s", sdkDebugEnv)
		d.enabled = true
	}
	return d
}

type debugger struct {
	enabled bool
}

// Enabled get enabled option
func (d *debugger) Enabled() bool {
	return d.enabled
}

// DebugSpansInvalidUTF8 debug invalid utf8 error when exporting spans
func (d *debugger) DebugSpansInvalidUTF8(exportErr error, batch []sdktrace.ReadOnlySpan) {
	s, ok := status.FromError(exportErr)
	if !ok {
		return
	}
	if !strings.Contains(s.String(), "invalid UTF-8") {
		return
	}
	for _, v := range batch {
		d.debugUTF8(telemetrySpan, "Name", v.Name())
		d.debugUTF8(telemetrySpan, "Status.Description", v.Status().Description)
		for _, attr := range v.Resource().Attributes() {
			d.debugUTF8(telemetrySpan, fmt.Sprintf("Resource.Attributes.Key.%s", attr.Key), string(attr.Key))
			d.debugUTF8(telemetrySpan, fmt.Sprintf("Resource.Attributes.%s", attr.Key), attr.Value.Emit())
		}
		for _, attr := range v.Attributes() {
			d.debugUTF8(telemetrySpan, fmt.Sprintf("Attributes.Key.%s", attr.Key), string(attr.Key))
			d.debugUTF8(telemetrySpan, fmt.Sprintf("Attributes.%s", attr.Key), attr.Value.Emit())
		}
		for i, event := range v.Events() {
			d.debugUTF8(telemetrySpan, fmt.Sprintf("Events.%d.Name", i), event.Name)
			for _, attr := range event.Attributes {
				d.debugUTF8(telemetrySpan,
					fmt.Sprintf("Events.%d.Attributes.Key.%s", i, attr.Key), string(attr.Key))
				d.debugUTF8(telemetrySpan, fmt.Sprintf("Events.%d.Attributes.%s",
					i, attr.Key), attr.Value.Emit())
			}
		}
	}
}

// DebugLogsInvalidUTF8 debug invalid utf8 error when exporting log
func (d *debugger) DebugLogsInvalidUTF8(exportErr error, batch []*logsproto.ResourceLogs) {
	s, ok := status.FromError(exportErr)
	if !ok {
		return
	}
	if !strings.Contains(s.String(), "invalid UTF-8") {
		return
	}
	for _, v := range batch {
		for _, attr := range v.GetResource().GetAttributes() {
			d.debugUTF8(telemetryLog,
				fmt.Sprintf("Attributes.Attributes.Key.%s", attr.Key),
				attr.Key,
			)
			d.debugUTF8(telemetryLog,
				fmt.Sprintf("Resource.Attributes.%s", attr.Key),
				attr.Value.GetStringValue(),
			)
		}
		for _, vv := range v.GetScopeLogs() {
			for _, vvv := range vv.GetLogRecords() {
				for _, attr := range vvv.GetAttributes() {
					d.debugUTF8(telemetryLog,
						fmt.Sprintf("Attributes.Key.%s", attr.Key),
						attr.Key,
					)
					d.debugUTF8(telemetryLog,
						fmt.Sprintf("Attributes.%s", attr.Key),
						attr.Value.GetStringValue(),
					)
				}
				d.debugUTF8(telemetryLog, "Message", vvv.GetBody().GetStringValue())
			}
		}
	}
}

func (d *debugger) debugUTF8(telemetry string, field string, value string) {
	if !utf8.ValidString(value) {
		log.Printf("opentelemetry: %v.%v is not a valid UTF-8 string, value:%s",
			telemetry, field, value)
	}
}

const (
	telemetrySpan = "span"
	telemetryLog  = "log"
)
