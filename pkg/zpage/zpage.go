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

package zpage

import (
	"net/http"
	"sync"

	"go.opentelemetry.io/contrib/zpages"
)

var processor *zpages.SpanProcessor
var once sync.Once

// GetZPageProcessor get globally unique zPage SpanProcessor
func GetZPageProcessor() *zpages.SpanProcessor {
	once.Do(func() {
		processor = zpages.NewSpanProcessor()
	})
	return processor
}

// GetZPageHandlerFunc  get new handler for zPage SpanProcessor
func GetZPageHandlerFunc() func(http.ResponseWriter, *http.Request) {
	return zpages.NewTracezHandler(GetZPageProcessor()).ServeHTTP
}
