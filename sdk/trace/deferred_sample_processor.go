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

// Package trace trace
package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/metrics"
)

var _ sdktrace.SpanProcessor = (*DeferredSampleProcessor)(nil)

// DeferredSampler deferred sampling, processing the filtering conditions after span.End.
// If true is returned, it is retained, and if false is returned, it is dropped.
type DeferredSampler func(sdktrace.ReadOnlySpan) bool

// DeferredSampleConfig deferred sampling configuration
type DeferredSampleConfig struct {
	Enabled            bool          // Whether to enable it, if not enabled, there will be no filtering
	SampleError        bool          // Sampling error
	SampleSlowDuration time.Duration // Sampling slow operation
}

// NewDeferredSampler crate a new deferred sampler
func NewDeferredSampler(cfg DeferredSampleConfig) DeferredSampler {
	sampledCounter := metrics.DeferredProcessCounter.WithLabelValues("sampled", "traces")
	errorCounter := metrics.DeferredProcessCounter.WithLabelValues("deferred_error", "traces")
	slowCounter := metrics.DeferredProcessCounter.WithLabelValues("deferred_slow", "traces")
	unsampledCounter := metrics.DeferredProcessCounter.WithLabelValues("unsampled", "traces")
	return func(s sdktrace.ReadOnlySpan) bool {
		// already sampled
		if s.SpanContext().IsSampled() {
			sampledCounter.Inc()
			return true
		}
		if cfg.Enabled && cfg.SampleError && s.Status().Code != codes.Ok {
			// error
			errorCounter.Inc()
			return true
		}
		if cfg.Enabled && cfg.SampleSlowDuration != 0 && s.EndTime().Sub(s.StartTime()) >= cfg.SampleSlowDuration {
			// slow duration
			slowCounter.Inc()
			return true
		}
		unsampledCounter.Inc()
		return false
	}
}

// DeferredSampleProcessor deferred sampling processor, processing filter conditions after span.End
type DeferredSampleProcessor struct {
	next            sdktrace.SpanProcessor
	deferredSampler DeferredSampler
}

// NewDeferredSampleProcessor create a new deferred sample processor
func NewDeferredSampleProcessor(next sdktrace.SpanProcessor,
	sampleFunc func(sdktrace.ReadOnlySpan) bool) *DeferredSampleProcessor {
	return &DeferredSampleProcessor{
		next:            next,
		deferredSampler: sampleFunc,
	}
}

// OnStart is called when a span is started. It is called synchronously
// and should not block.
func (p *DeferredSampleProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	p.next.OnStart(parent, s)
}

// OnEnd is called when span is finished. It is called synchronously and
// hence not block.
func (p *DeferredSampleProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	if p.deferredSampler == nil {
		// keep
		p.next.OnEnd(s)
		return
	}
	if p.deferredSampler(s) {
		// keep
		p.next.OnEnd(s)
		return
	}
	// drop
}

// Shutdown is called when the SDK shuts down. Any cleanup or release of
// resources held by the processor should be done in this call.
//
// Calls to OnStart, OnEnd, or ForceFlush after this has been called
// should be ignored.
//
// All timeouts and cancellations contained in ctx must be honored, this
// should not block indefinitely.
func (p *DeferredSampleProcessor) Shutdown(ctx context.Context) error {
	return p.next.Shutdown(ctx)
}

// ForceFlush exports all ended spans to the configured Exporter that have not yet
// been exported.  It should only be called when absolutely necessary, such as when
// using a FaaS provider that may suspend the process after an invocation, but before
// the Processor can export the completed spans.
func (p *DeferredSampleProcessor) ForceFlush(ctx context.Context) error {
	return p.next.ForceFlush(ctx)
}
