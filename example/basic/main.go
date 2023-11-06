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

package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"trpc-ecosystem/go-opentelemetry/api"
	apilog "trpc-ecosystem/go-opentelemetry/api/log"
	ecotrace "trpc-ecosystem/go-opentelemetry/sdk/trace"
)

var meter = otel.Meter("my_meter")
var counter, _ = meter.Int64Counter("counter")

func main() {
	err := opentelemetry.Setup("your.own.server.ip:porto",
		opentelemetry.WithSampler(sdktrace.AlwaysSample()),
		opentelemetry.WithServiceName("helloworld"),
		opentelemetry.WithTenantID("default"),
		opentelemetry.WithMetricEnabled(true),
		opentelemetry.WithLogEnabled(true),
		opentelemetry.WithLevelEnable(apilog.DebugLevel),
		opentelemetry.WithBatchSpanProcessorOption( // 设置异步上报批量参数
			ecotrace.WithBatchTimeout(time.Second),
			ecotrace.WithExportTimeout(time.Second),
			ecotrace.WithMaxExportBatchSize(1)),
	)
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	// 利用Start，End示例
	ctx, span := opentelemetry.Start(ctx, "basic_step_one")
	span.SetAttributes(api.TpsDyeingKey.String("xxx"))
	err = doSomething(ctx)
	span.End()
	if err != nil {
		log.Fatalln(err)
	}

	span.AddEvent("", trace.WithAttributes(attribute.Int("article.num", 10)),
		trace.WithAttributes(attribute.String("article.name", "opentelemetry-example")))

	// 利用WithSpan示例，WithSpan内部是通过Start,End组合实现的，常用于简化内部Span上报
	err = opentelemetry.WithSpan(ctx, "basic_step_two", doSomething)
	if err != nil {
		log.Fatalln(err)
	}

	span = trace.SpanFromContext(ctx)
	log.Println(span.SpanContext().TraceID().String())

	if err = opentelemetry.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

func doSomething(ctx context.Context) error {
	// 获取span设置一些Log 和 Attributes
	counter.Add(ctx, 1)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Key("location").String("shenzhen"))
	span.AddEvent("",
		trace.WithAttributes(attribute.Key("msg").String("before doSomething log")),
	)

	apilog.Info(ctx, "do something")

	// 业务逻辑
	time.Sleep(time.Second)

	span.AddEvent("",
		trace.WithAttributes(attribute.Key("msg").String("after doSomething log")),
	)

	return nil
}
