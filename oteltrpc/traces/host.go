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
	"net"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var (
	hostname     string
	hostnameOnce sync.Once
)

const localhost = "127.0.0.1"

func peerInfo(addr net.Addr) []attribute.KeyValue {
	if addr == nil {
		return nil
	}
	host, port, err := net.SplitHostPort(addr.String())

	if err != nil {
		return []attribute.KeyValue{}
	}

	if host == "" {
		host = localhost
	}

	return []attribute.KeyValue{
		semconv.NetPeerIPKey.String(host),
		semconv.NetPeerPortKey.String(port),
	}
}

func hostInfo(addr net.Addr) []attribute.KeyValue {
	if addr == nil {
		return []attribute.KeyValue{
			semconv.NetHostNameKey.String(getHostname()),
		}
	}
	host, port, err := net.SplitHostPort(addr.String())

	if err != nil {
		return []attribute.KeyValue{
			semconv.NetHostNameKey.String(getHostname()),
		}
	}

	if host == "" {
		host = localhost
	}

	return []attribute.KeyValue{
		semconv.NetHostIPKey.String(host),
		semconv.NetHostPortKey.String(port),
		semconv.NetHostNameKey.String(getHostname()),
	}
}

// getHostname
func getHostname() string {
	hostnameOnce.Do(func() {
		hostname, _ = os.Hostname()
	})
	return hostname
}
