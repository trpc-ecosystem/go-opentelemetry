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

// Package traces
package traces

import (
	"go.opentelemetry.io/otel/propagation"

	"trpc.group/trpc-go/trpc-go/codec"
	thttp "trpc.group/trpc-go/trpc-go/http"
)

const (
	SpanKindClient = "spanKindClient"
	SpanKindServer = "spanKindServer"
)

var registeredCarriers []func(md codec.MetaData, msg codec.Msg) propagation.TextMapCarrier

// RegisterCarrier register otelcarrier
func RegisterCarrier(carrier func(md codec.MetaData, msg codec.Msg) propagation.TextMapCarrier) {
	registeredCarriers = append(registeredCarriers, carrier)
}

// GetTextMapCarriers ...
func GetTextMapCarriers(md codec.MetaData, msg codec.Msg) propagation.TextMapCarrier {
	var textMapCarriers compositeTextMapCarrier
	textMapCarriers = append(textMapCarriers, newSupplier(md, msg))
	for _, r := range registeredCarriers {
		supplier := r(md, msg)
		if supplier != nil {
			textMapCarriers = append(textMapCarriers, supplier)
		}
	}

	return &textMapCarriers
}

// defaultCarrier
func newSupplier(md codec.MetaData, msg codec.Msg) *supplier {
	serverHeader := thttp.Head(msg.Context())
	clientHeader, _ := msg.ClientReqHead().(*thttp.ClientReqHeader)
	return &supplier{
		md:           md,
		serverHeader: serverHeader,
		clientHeader: clientHeader,
		keys:         make([]string, 0),
	}
}

type supplier struct {
	md           codec.MetaData
	serverHeader *thttp.Header
	clientHeader *thttp.ClientReqHeader
	keys         []string
}

// Get implement TextMapCarrier Get interface
func (s *supplier) Get(key string) string {
	value := s.md[key]
	if len(value) == 0 {
		if s.serverHeader != nil && s.serverHeader.Request != nil && s.serverHeader.Request.Header != nil {
			return s.serverHeader.Request.Header.Get(key)
		}
		return ""
	}
	return string(value)
}

// Set implement TextMapCarrier Set interface
func (s *supplier) Set(key string, value string) {
	s.md[key] = []byte(value)
	if s.clientHeader != nil {
		s.clientHeader.AddHeader(key, value)
	}
	s.keys = append(s.keys, key)
}

// Keys implement TextMapCarrier Keys interface
func (s *supplier) Keys() []string {
	return s.keys
}

type compositeTextMapCarrier []propagation.TextMapCarrier

// Get implement TextMapCarrier Get interface
func (c *compositeTextMapCarrier) Get(key string) string {
	v := ""
	for _, carrier := range *c {
		v = carrier.Get(key)
		if v != "" {
			break
		}
	}
	return v
}

// Set implement TextMapCarrier Set interface
func (c *compositeTextMapCarrier) Set(key string, value string) {
	for _, carrier := range *c {
		carrier.Set(key, value)
	}
}

// Keys implement TextMapCarrier Keys interface
func (c *compositeTextMapCarrier) Keys() []string {
	keys := make(map[string]bool)
	var list []string

	for _, carrier := range *c {
		for _, k := range carrier.Keys() {
			if _, value := keys[k]; !value {
				keys[k] = true
				list = append(list, k)
			}
		}
	}
	return list
}
