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
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	lemonsKey = attribute.Key("otlp.exporter")
)

// initProvider Initializes an OTLP exporter, and configures the corresponding
// trace and metric providers.
func initProvider() func() {
	ctx := context.Background()
	// creates a new HTTP driver.
	// creates a new OTLP exporter
	exp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithEndpoint("your.own.endpoint.ip:port"))
	handleErr(err, "failed to create exporter")
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
	otel.SetMeterProvider(meterProvider)
	return func() {
		// Push any last metric events to the exporter.
		handleErr(meterProvider.Shutdown(context.Background()), "failed to stop controller")
	}
}

func main() {
	shutdown := initProvider()
	defer shutdown()
	meter := noop.NewMeterProvider().Meter("test-meter")
	// Init observer value and labels
	gauge, _ := meter.Float64UpDownCounter("chocolate", metric.WithDescription("A GaugeObserver set to 1.0"))
	// init instrument with histogram and counter
	histogram, _ := meter.Float64Histogram("raspberry")
	counter, _ := meter.Float64Counter("vanilla")

	// init labels with attribute
	commonLabels := []attribute.KeyValue{lemonsKey.Int(10), attribute.String("A", "1"),
		attribute.String("B", "2"), attribute.String("C", "3")}
	notSoCommonLabels := []attribute.KeyValue{lemonsKey.Int(13)}

	ctx := context.Background()
	_ = observeReport(ctx, observeParam{
		meter:                  meter,
		Labels:                 commonLabels,
		observerValueToReport:  1.0,
		observerLabelsToReport: commonLabels,
		histogram:              histogram,
		counter:                counter,
		gauge:                  gauge,
		hisValue:               2.0,
		counValue:              12.0,
		gaugeValue:             22.0,
	})

	time.Sleep(5 * time.Second)

	_ = observeReport(ctx, observeParam{
		meter:                  meter,
		Labels:                 notSoCommonLabels,
		observerValueToReport:  1.0,
		observerLabelsToReport: notSoCommonLabels,
		histogram:              histogram,
		counter:                counter,
		gauge:                  gauge,
		hisValue:               2.0,
		counValue:              22.0,
		gaugeValue:             222.0,
	})

	time.Sleep(5 * time.Second)

	_ = observeReport(ctx, observeParam{
		meter:                  meter,
		Labels:                 commonLabels,
		observerValueToReport:  13.0,
		observerLabelsToReport: commonLabels,
		histogram:              histogram,
		counter:                counter,
		gauge:                  gauge,
		hisValue:               12.0,
		counValue:              13.0,
		gaugeValue:             14.0,
	})

	fmt.Println("Example finished updating")
	select {}
}

// observeParam define param to record metrics
type observeParam struct {
	meter                  metric.Meter
	Labels                 []attribute.KeyValue
	observerValueToReport  float64
	observerLabelsToReport []attribute.KeyValue
	histogram              metric.Float64Histogram
	counter                metric.Float64Counter
	gauge                  metric.Float64UpDownCounter
	hisValue               float64
	counValue              float64
	gaugeValue             float64
}

// observeReport record metrics
func observeReport(ctx context.Context, observeParam observeParam) error {
	observeParam.observerValueToReport = 13.0
	observeParam.observerLabelsToReport = observeParam.Labels
	observeParam.histogram.Record(ctx, 12.0, metric.WithAttributes(observeParam.Labels...))
	observeParam.counter.Add(ctx, 13.0, metric.WithAttributes(observeParam.Labels...))
	observeParam.gauge.Add(ctx, 13.0, metric.WithAttributes(observeParam.Labels...))
	return nil
}

// handleError ...
func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
