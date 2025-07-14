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

package metric

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"

	"trpc.group/trpc-go/trpc-opentelemetry/config/codes"
)

// ServerReporter server metrics reporter
type ServerReporter struct {
	systemName    string
	callerService string
	callerMethod  string
	calleeService string
	calleeMethod  string
	startTime     time.Time
	endTime       time.Time
	extraLabels   []string

	// only used for stream rpc now
	metrics *ServerMetrics
	rpcType RPCType

	enableDeferredSample       bool
	deferredSampleError        bool
	deferredSampleSlowDuration time.Duration
}

// ServerOption Server option
type ServerOption func(reporter *ServerReporter)

// WithServerExtraLabels  set server extraLabels.
func WithServerExtraLabels(labels []string) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.extraLabels = labels
	}
}

// WithServerTraceConfig set server trace config
func WithServerTraceConfig(enableDeferredSample, deferredSampleError bool,
	deferredSampleSlowDuration time.Duration) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.enableDeferredSample = enableDeferredSample
		serverReporter.deferredSampleError = deferredSampleError
		serverReporter.deferredSampleSlowDuration = deferredSampleSlowDuration
	}
}

// WithServerStartTime set startTime
func WithServerStartTime(startTime time.Time) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.startTime = startTime
	}
}

// WithServerEndTime set endTime
func WithServerEndTime(endTime time.Time) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.endTime = endTime
	}
}

// WithServerMetrics set metrics
func WithServerMetrics(metrics *ServerMetrics) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.metrics = metrics
	}
}

// WithServerRPCType set rpcType
func WithServerRPCType(rpcType RPCType) ServerOption {
	return func(serverReporter *ServerReporter) {
		serverReporter.rpcType = rpcType
	}
}

// NewServerReporter create a server reporter.
// systemName, callerService, callerMethod, calleeService, calleeMethod are reserved fields.
// Add labels as extended fields. Note that using extended fields requires redefining the initialization function
// in sdk/metric/rpc_server_metrics.go.
func NewServerReporter(systemName, callerService, callerMethod, calleeService, calleeMethod string,
	options ...ServerOption,
) *ServerReporter {
	r := &ServerReporter{
		systemName:    systemName,
		callerService: cleanServiceName(callerService),
		callerMethod:  CleanRPCMethod(callerMethod),
		calleeService: cleanServiceName(calleeService),
		calleeMethod:  CleanRPCMethod(calleeMethod),
		startTime:     time.Now(),
	}
	for _, opt := range options {
		opt(r)
	}
	labelValues := []string{r.systemName, r.callerService, r.callerMethod, r.calleeService, r.calleeMethod}
	labelValues = append(labelValues, r.extraLabels...)
	serverStartedCounter.WithLabelValues(labelValues...).Inc()
	return r
}

// Handled called after the request has been processed.
// systemName, callerService, callerMethod, calleeService, calleeMethod, code,
// codeType.Type, codeType.Description are reserved fields.
// Add labels as extended fields. Note that using extended fields requires redefining the initialization function
// in sdk/metric/rpc_server_metrics.go.
func (r *ServerReporter) Handled(ctx context.Context, code string) {
	codeType := codes.CodeMapping(code, r.calleeService, r.calleeMethod)
	counterLabelValues := []string{
		r.systemName, r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
		code, codeType.Type, codeType.Description,
	}
	counterLabelValues = append(counterLabelValues, r.extraLabels...)
	c := serverHandledCounter.WithLabelValues(counterLabelValues...)
	histogramLabelValues := []string{
		r.systemName, r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
		code, codeType.Type, codeType.Description,
	}
	histogramLabelValues = append(histogramLabelValues, r.extraLabels...)
	h := serverHandledHistogram.WithLabelValues(histogramLabelValues...)

	if r.endTime.IsZero() {
		r.endTime = time.Now()
	}
	sp := trace.SpanFromContext(ctx).SpanContext()
	exemplar := prometheus.Labels{
		"traceID": sp.TraceID().String(),
	}
	costSecs := r.endTime.Sub(r.startTime).Seconds()
	if r.counterNeedUseExemplar(sp, codeType.Type) {
		if v, ok := c.(prometheus.ExemplarAdder); ok {
			v.AddWithExemplar(1, exemplar)
		}
	} else {
		c.Inc()
	}
	if r.histogramNeedUseExemplar(sp, costSecs) {
		if v, ok := h.(prometheus.ExemplarObserver); ok {
			v.ObserveWithExemplar(costSecs, exemplar)
		}
	} else {
		h.Observe(costSecs)
	}
}

// counterNeedUseExemplar Check whether counter needs to be reported exemplar
func (r *ServerReporter) counterNeedUseExemplar(sp trace.SpanContext, codeType string) bool {
	if r.enableDeferredSample && r.deferredSampleError {
		// 如果启用 deferred sample, 仅需判断是否为 error
		return codeType != CodeTypeSuccess.String()
	}
	// If deferred sample is not enabled, reporting will be selected based on whether the sample is hit or not.
	return sp.IsSampled()
}

// histogramNeedUseExemplar Check whether the histogram needs to be reported as exemplar
func (r *ServerReporter) histogramNeedUseExemplar(sp trace.SpanContext, costSecs float64) bool {
	if r.enableDeferredSample {
		// 如果启用 deferred sample, 且耗时超时限制时需要上报 exemplar, 以此来减少不是高耗时的 exemplar 点.
		return costSecs >= r.deferredSampleSlowDuration.Seconds()
	}
	// If deferred sample is not enabled, reporting will be selected based on whether the sample is hit or not.
	return sp.IsSampled()
}

// ReceivedMessage reports metrics for server stream received message.
func (r *ServerReporter) ReceivedMessage() {
	r.metrics.serverStreamMsgReceived.WithLabelValues(r.streamLabels()...).Inc()
}

// SentMessage reports metrics for server stream sent message.
func (r *ServerReporter) SentMessage() {
	r.metrics.serverStreamMsgSent.WithLabelValues(r.streamLabels()...).Inc()
}

func (r *ServerReporter) streamLabels() []string {
	return []string{
		r.systemName, string(r.rpcType), r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
	}
}
