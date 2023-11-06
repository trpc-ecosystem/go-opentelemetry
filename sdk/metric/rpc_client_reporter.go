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

package metric

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"

	"trpc-ecosystem/go-opentelemetry/config/codes"
)

// ClientReporter ...
type ClientReporter struct {
	systemName    string
	callerService string
	callerMethod  string
	calleeService string
	calleeMethod  string
	startTime     time.Time
	endTime       time.Time
	extraLabels   []string

	// only used for stream rpc now
	metrics *ClientMetrics
	rpcType RPCType

	enableDeferredSample       bool
	deferredSampleError        bool
	deferredSampleSlowDuration time.Duration
}

// ClientOption Client 调用参数工具函数。
type ClientOption func(*ClientReporter)

// WithClientExtraLabels 设置 Labels。
func WithClientExtraLabels(labels []string) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.extraLabels = labels
	}
}

// WithClientTraceConfig 设置 Trace 相关配置
func WithClientTraceConfig(enableDeferredSample, deferredSampleError bool,
	deferredSampleSlowDuration time.Duration) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.enableDeferredSample = enableDeferredSample
		clientReporter.deferredSampleError = deferredSampleError
		clientReporter.deferredSampleSlowDuration = deferredSampleSlowDuration
	}
}

// WithClientStartTime 设置startTime
func WithClientStartTime(startTime time.Time) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.startTime = startTime
	}
}

// WithClientEndTime 设置endTime
func WithClientEndTime(endTime time.Time) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.endTime = endTime
	}
}

// WithClientMetrics 设置metrics
func WithClientMetrics(metrics *ClientMetrics) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.metrics = metrics
	}
}

// WithClientRPCType 设置rpcType
func WithClientRPCType(rpcType RPCType) ClientOption {
	return func(clientReporter *ClientReporter) {
		clientReporter.rpcType = rpcType
	}
}

// NewClientReporter create a client reporter
// systemName, callerService, callerMethod, calleeService, calleeMethod 为保留字段。
// 新增labels 为扩展字段，注意使用扩展字段需要重定义 sdk/metric/rpc_client_metrics.go:40 所在的初始化函数。
func NewClientReporter(systemName, callerService, callerMethod, calleeService, calleeMethod string,
	options ...ClientOption,
) *ClientReporter {
	r := &ClientReporter{
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
	clientStartedCounter.WithLabelValues(labelValues...).Inc()
	return r
}

// Handled is called after the request is processed.
// systemName, callerService, callerMethod, calleeService, calleeMethod, code,
// codeType.Type, codeType.Description are reserved fields.
// Add labels as extended fields. Note that using extended fields requires redefining the initialization function where sdk/metric/rpc_client_metrics.go:40 is located.
func (r *ClientReporter) Handled(ctx context.Context, code string) {
	codeType := codes.CodeMapping(code, r.calleeService, r.calleeMethod)
	counterLabelValues := []string{
		r.systemName, r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
		code, codeType.Type, codeType.Description,
	}
	counterLabelValues = append(counterLabelValues, r.extraLabels...)
	c := clientHandledCounter.WithLabelValues(counterLabelValues...)
	histogramLabelValues := []string{
		r.systemName, r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
		code, codeType.Type, codeType.Description,
	}
	histogramLabelValues = append(histogramLabelValues, r.extraLabels...)
	h := clientHandledHistogram.WithLabelValues(histogramLabelValues...)

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
func (r *ClientReporter) counterNeedUseExemplar(sp trace.SpanContext, codeType string) bool {
	if r.enableDeferredSample && r.deferredSampleError {
		// 如果启用 deferred sample, 仅需判断是否为 error
		return codeType != CodeTypeSuccess.String()
	}
	// 如果未启用 deferred sample, 则按是否命中采样选择上报
	return sp.IsSampled()
}

// histogramNeedUseExemplar Check whether histogram needs to be reported exemplar
func (r *ClientReporter) histogramNeedUseExemplar(sp trace.SpanContext, costSecs float64) bool {
	if r.enableDeferredSample {
		// 如果启用 deferred sample, 且耗时超时限制时需要上报 exemplar, 以此来减少不是高耗时的 exemplar 点
		return costSecs >= r.deferredSampleSlowDuration.Seconds()
	}
	// If deferred sample is not enabled, reporting will be selected based on whether the sample is hit or not.
	return sp.IsSampled()
}

// Timer is a helper interface to time functions.
type Timer interface {
	ObserveDuration() time.Duration
}

type noOpTimer struct{}

// ObserveDuration returns time.Duration for noOpTimer.
func (noOpTimer) ObserveDuration() time.Duration {
	return 0
}

var emptyTimer = noOpTimer{}

// ReceiveMessageTimer returns Timer for receive message.
func (r *ClientReporter) ReceiveMessageTimer() Timer {
	if r.metrics.clientStreamRecvHistogramEnabled {
		hist := r.metrics.clientStreamRecvHistogram.WithLabelValues(r.streamLabels()...)
		return prometheus.NewTimer(hist)
	}

	return emptyTimer
}

// ReceivedMessage reports metrics after stream rpc received a message.
func (r *ClientReporter) ReceivedMessage() {
	r.metrics.clientStreamMsgReceived.WithLabelValues(r.streamLabels()...).Inc()
}

// SendMessageTimer returns Timer for sending a message.
func (r *ClientReporter) SendMessageTimer() Timer {
	if r.metrics.clientStreamSendHistogramEnabled {
		hist := r.metrics.clientStreamSendHistogram.WithLabelValues(r.streamLabels()...)
		return prometheus.NewTimer(hist)
	}

	return emptyTimer
}

// SentMessage reports metrics after stream rpc message was sent.
func (r *ClientReporter) SentMessage() {
	r.metrics.clientStreamMsgSent.WithLabelValues(r.streamLabels()...).Inc()
}

func (r *ClientReporter) streamLabels() []string {
	return []string{
		r.systemName, string(r.rpcType), r.callerService, r.callerMethod, r.calleeService, r.calleeMethod,
	}
}
