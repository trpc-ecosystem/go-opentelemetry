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

package main

import (
	"log"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"

	"trpc.group/trpc-go/trpc-opentelemetry/api"
	otlplog "trpc.group/trpc-go/trpc-opentelemetry/exporter/otlp"
	"trpc.group/trpc-go/trpc-opentelemetry/otelzap"
)

func main() {
	exp, err := otlplog.NewExporter(otlplog.WithInsecure(),
		otlplog.WithAddress("your.own.server.ip:port"))
	if err != nil {
		log.Fatalln(err)
	}
	res := resource.NewWithAttributes(
		"", api.TpsTenantIDKey.String("default"),
	)

	core := otelzap.NewBatchCore(otelzap.NewBatchWriteSyncer(exp, res))
	logger := zap.New(core)

	logger.Error("hello, world")
	select {}
}
