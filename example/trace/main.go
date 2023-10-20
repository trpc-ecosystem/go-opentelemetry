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
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"trpc-system/go-opentelemetry"
	apilog "trpc-system/go-opentelemetry/api/log"
	"trpc-system/go-opentelemetry/pkg/zpage"
	ecotrace "trpc-system/go-opentelemetry/sdk/trace"
)

func main() {
	// set up tracer
	if err := opentelemetry.Setup("your.own.server.ip:Port",
		opentelemetry.WithSampler(sdktrace.AlwaysSample()),
		opentelemetry.WithServiceName("opentelemetryDemoForSdk"),
		opentelemetry.WithTenantID("default"),
		opentelemetry.WithLogEnabled(true),
		opentelemetry.WithLevelEnable(apilog.DebugLevel),
		opentelemetry.WithBatchSpanProcessorOption( // batch options
			ecotrace.WithBatchTimeout(time.Second),
			ecotrace.WithExportTimeout(time.Second),
			ecotrace.WithMaxExportBatchSize(1)),
		opentelemetry.WithZPageSpanProcessor(true),
	); err != nil {
		log.Fatalln(err)
	}

	// parent span
	ctx, span := opentelemetry.Start(context.Background(), "mainFunc")
	span.SetAttributes(
		attribute.String("func", "main"),
		attribute.Int64("start_time", time.Now().Unix()),
	)
	span.AddEvent("",
		trace.WithAttributes(attribute.String("article.name", "basic demo")),
	)

	// end span
	defer func() {
		span.SetAttributes(attribute.Int64("end_time", time.Now().Unix()))
		// print span
		spanJSON, _ := span.SpanContext().MarshalJSON()
		log.Printf("span:%s", string(spanJSON))
		span.End()
		time.Sleep(2 * time.Second)
	}()

	// biz logic, get information from parent span in ctx
	_ = doSomething(ctx)
	http.HandleFunc("/debug/tracez", zpage.GetZPageHandlerFunc())
	_ = http.ListenAndServe(":8000", nil)
}

// doSomething ...
func doSomething(ctx context.Context) error {
	// span for biz logic
	_, span := opentelemetry.Start(ctx, "doSomething")

	// before do doSomething
	span.SetAttributes(attribute.Key("location").String("shenzhen"))
	span.AddEvent("",
		trace.WithAttributes(attribute.Key("msg").String("before doSomething log")),
	)
	span.SetAttributes(attribute.Int64("start_time", time.Now().Unix()))

	// end span
	defer func() {
		span.SetAttributes(attribute.Int64("end_time", time.Now().Unix()))
		span.AddEvent("",
			trace.WithAttributes(attribute.Key("msg").String("after doSomething log")),
		)
		// print span
		spanJSON, _ := span.SpanContext().MarshalJSON()
		log.Printf("span:%s", string(spanJSON))
		defer span.End()
	}()

	time.Sleep(2 * time.Second)

	return nil
}
