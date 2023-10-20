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

package admin

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

const (
	// TraceFilterOn toggle
	TraceFilterOn int32 = 0
	// TraceFilterOff toggle
	TraceFilterOff int32 = 1
)

var traceStatus int32 // trace filter：0-on，1-off

// TraceDisabled ...
func TraceDisabled() bool {
	return atomic.LoadInt32(&traceStatus) == TraceFilterOff
}

// DisableTrace ...
func DisableTrace(w http.ResponseWriter, _ *http.Request) {
	atomic.StoreInt32(&traceStatus, TraceFilterOff)
	log.Print("opentelemetry: close trace filter")
	response(w, "disable trace filter success")
}

// EnableTrace ...
func EnableTrace(w http.ResponseWriter, _ *http.Request) {
	atomic.StoreInt32(&traceStatus, TraceFilterOn)
	log.Print("opentelemetry: open trace filter")
	response(w, "enable trace filter success")
}

// TraceStatus ...
func TraceStatus(w http.ResponseWriter, _ *http.Request) {
	status := atomic.LoadInt32(&traceStatus)
	if status == TraceFilterOff {
		response(w, "opentelemetry: trace filter is closed")
	} else {
		response(w, "opentelemetry: trace filter is opening")
	}
}

func response(w http.ResponseWriter, message string) {
	_, err := w.Write([]byte(fmt.Sprintf("{\"code\":0, \"message\": %q}", message)))
	if err != nil {
		log.Printf("write http response err: %v", err)
	}
}
