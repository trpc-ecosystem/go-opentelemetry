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
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// ClientStartedCounter the total number of RPCs started on the client.
	ClientStartedCounter = "ClientStartedCounter"
	// ClientHandledCounter the total number of RPCs completed by the client, regardless of success or failure.
	ClientHandledCounter = "ClientHandledCounter"
	// ClientHandledHistogram the Histogram of response latency (seconds) of the RPC
	// until it is finished by the application.
	ClientHandledHistogram = "ClientHandledHistogram"
)

var clientHandledHistogramBuckets = []float64{.005, .01, .1, .5, 1, 5}

var (
	clientStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_started_total",
			Help:      "Total number of RPCs started on the client.",
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
	clientHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_handled_total",
			Help:      "Total number of RPCs completed by the client, regardless of success or failure.",
		},
		[]string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		},
	)
	clientHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "client_handled_seconds",
			Help:      "Histogram of response latency (seconds) of the RPC until it is finished by the application.",
			Buckets:   clientHandledHistogramBuckets,
		},
		[]string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		},
	)
)

// clientLabelsOption list of extra labels for init client metrics.
// according to the Client to dynamically initialize extraLabels.
var clientLabelsOption = func(name string) []string {
	switch name {
	case ClientStartedCounter:
		return []string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"}
	case ClientHandledCounter, ClientHandledHistogram:
		return []string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		}
	default:
		return []string{}
	}
}

// SetClientLabelsOption other users can use this function to redefine clientLabelsOption.
func SetClientLabelsOption(f func(name string) []string) {
	clientLabelsOption = f
}

func registerRPCClientCounter() {
	initClientCollectors()
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			clientStartedCounter,
			"clientStartedCounter",
			rpcMetricsCardinalityLimit,
		})
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			clientHandledCounter,
			"clientHandledCounter",
			rpcMetricsCardinalityLimit,
		})
}

func initClientCollectors() {
	clientStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_started_total",
			Help:      "Total number of RPCs started on the client.",
		},
		clientLabelsOption(ClientStartedCounter),
	)
	clientHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_handled_total",
			Help:      "Total number of RPCs completed by the client, regardless of success or failure.",
		},
		clientLabelsOption(ClientHandledCounter),
	)
}

// setClientHandledHistogramBuckets user customizes clientHandledHistogramBuckets through configuration
func setClientHandledHistogramBuckets(buckets []float64) {
	if newBuckets, ok := validateBuckets(buckets); ok {
		clientHandledHistogramBuckets = newBuckets
	}
}

// DefaultClientMetrics is the default instance of ClientMetrics. It is
// intended to be used in conjunction the default Prometheus metrics
// registry.
// Note: only used stream metrics now
var DefaultClientMetrics = NewClientMetrics()

func init() {
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			DefaultClientMetrics.clientStreamMsgReceived,
			"clientStreamMsgReceived",
			rpcMetricsCardinalityLimit,
		})
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			DefaultClientMetrics.clientStreamMsgSent,
			"clientStreamMsgSent",
			rpcMetricsCardinalityLimit,
		})
}

// ClientMetrics represents a collection of metrics to be registered on a Prometheus metrics registry for a rpc client.
type ClientMetrics struct {
	clientStartedCounter    *prometheus.CounterVec
	clientStreamMsgReceived *prometheus.CounterVec
	clientStreamMsgSent     *prometheus.CounterVec

	clientStreamRecvHistogramEnabled bool
	clientStreamRecvHistogramOpts    prometheus.HistogramOpts
	clientStreamRecvHistogram        *prometheus.HistogramVec

	clientStreamSendHistogramEnabled bool
	clientStreamSendHistogramOpts    prometheus.HistogramOpts
	clientStreamSendHistogram        *prometheus.HistogramVec
}

var defaultStreamLabels = []string{
	"system_name", "rpc_type", "caller_service", "caller_method", "callee_service", "callee_method",
}

// NewClientMetrics returns a ClientMetrics object. Use a new instance of
// ClientMetrics when not using the default Prometheus metrics registry, for
// example when wanting to control which metrics are added to a registry as
// opposed to automatically adding metrics via init functions.
func NewClientMetrics(counterOpts ...CounterOption) *ClientMetrics {
	opts := counterOptions(counterOpts)
	return &ClientMetrics{
		clientStartedCounter: clientStartedCounter,
		clientStreamMsgReceived: prometheus.NewCounterVec(
			opts.apply(prometheus.CounterOpts{
				Subsystem: "rpc",
				Name:      "client_msg_received_total",
				Help:      "Total number of rpc stream messages received by the client.",
			}),
			defaultStreamLabels),
		clientStreamMsgSent: prometheus.NewCounterVec(
			opts.apply(prometheus.CounterOpts{
				Subsystem: "rpc",
				Name:      "client_msg_sent_total",
				Help:      "Total number of rpc stream messages sent by the client.",
			}),
			defaultStreamLabels),
		clientStreamRecvHistogramOpts: prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "client_stream_recv_handling_seconds",
			Help:      "Histogram of response latency (seconds) of the rpc single message receive.",
			Buckets:   prometheus.DefBuckets,
		},
		clientStreamSendHistogramOpts: prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "client_stream_send_handling_seconds",
			Help:      "Histogram of response latency (seconds) of the rpc single message send.",
			Buckets:   prometheus.DefBuckets,
		},
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *ClientMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.clientStreamMsgReceived.Describe(ch)
	m.clientStreamMsgSent.Describe(ch)
	if m.clientStreamRecvHistogramEnabled {
		m.clientStreamRecvHistogram.Describe(ch)
	}
	if m.clientStreamSendHistogramEnabled {
		m.clientStreamSendHistogram.Describe(ch)
	}
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *ClientMetrics) Collect(ch chan<- prometheus.Metric) {
	m.clientStreamMsgReceived.Collect(ch)
	m.clientStreamMsgSent.Collect(ch)
	if m.clientStreamRecvHistogramEnabled {
		m.clientStreamRecvHistogram.Collect(ch)
	}
	if m.clientStreamSendHistogramEnabled {
		m.clientStreamSendHistogram.Collect(ch)
	}
}

// EnableClientStreamReceiveTimeHistogram turns on recording of single message receive time of streaming RPCs.
// Histogram metrics can be very expensive for Prometheus to retain and query.
func (m *ClientMetrics) EnableClientStreamReceiveTimeHistogram(opts ...HistogramOption) {
	for _, o := range opts {
		o(&m.clientStreamRecvHistogramOpts)
	}

	if !m.clientStreamRecvHistogramEnabled {
		m.clientStreamRecvHistogram = prometheus.NewHistogramVec(
			m.clientStreamRecvHistogramOpts,
			defaultStreamLabels,
		)
	}

	m.clientStreamRecvHistogramEnabled = true
}

// EnableClientStreamSendTimeHistogram turns on recording of single message send time of streaming RPCs.
// Histogram metrics can be very expensive for Prometheus to retain and query.
func (m *ClientMetrics) EnableClientStreamSendTimeHistogram(opts ...HistogramOption) {
	for _, o := range opts {
		o(&m.clientStreamSendHistogramOpts)
	}

	if !m.clientStreamSendHistogramEnabled {
		m.clientStreamSendHistogram = prometheus.NewHistogramVec(
			m.clientStreamSendHistogramOpts,
			defaultStreamLabels,
		)
	}

	m.clientStreamSendHistogramEnabled = true
}

// Reset resets the metrics.
func (m *ClientMetrics) Reset() {
	m.clientStreamMsgReceived.Reset()
	m.clientStreamMsgSent.Reset()
	m.clientStartedCounter.Reset()
	if m.clientStreamRecvHistogramEnabled {
		m.clientStreamRecvHistogram.Reset()
	}
	if m.clientStreamSendHistogramEnabled {
		m.clientStreamSendHistogram.Reset()
	}
}

// ClientStreamMsgReceived returns clientStreamMsgReceived counter vec
func (m *ClientMetrics) ClientStreamMsgReceived() *prometheus.CounterVec {
	return m.clientStreamMsgReceived
}

// ClientStreamMsgSent returns clientStreamMsgSent counter vec
func (m *ClientMetrics) ClientStreamMsgSent() *prometheus.CounterVec {
	return m.clientStreamMsgSent
}

// ClientStreamRecvHistogram returns clientStreamRecvHistogram histogram vec
func (m *ClientMetrics) ClientStreamRecvHistogram() *prometheus.HistogramVec {
	return m.clientStreamRecvHistogram
}

// ClientStreamSendHistogram returns clientStreamSendHistogram histogram vec
func (m *ClientMetrics) ClientStreamSendHistogram() *prometheus.HistogramVec {
	return m.clientStreamSendHistogram
}

// ClientStartedCounter returns clientStarted counter vec
func (m *ClientMetrics) ClientStartedCounter() *prometheus.CounterVec {
	return m.clientStartedCounter
}

// EnableClientStreamReceiveTimeHistogram turns on recording of
// single message receive time of streaming RPCs.
// This function acts on the DefaultClientMetrics variable and the
// default Prometheus metrics registry.
func EnableClientStreamReceiveTimeHistogram(opts ...HistogramOption) {
	DefaultClientMetrics.EnableClientStreamReceiveTimeHistogram(opts...)
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			metricCollector: DefaultClientMetrics.clientStreamRecvHistogram,
			desc:            "clientStreamRecvHistogram",
			limit:           rpcMetricsCardinalityLimit,
		})
}

// EnableClientStreamSendTimeHistogram turns on recording of
// single message send time of streaming RPCs.
// This function acts on the DefaultClientMetrics variable and the
// default Prometheus metrics registry.
func EnableClientStreamSendTimeHistogram(opts ...HistogramOption) {
	DefaultClientMetrics.EnableClientStreamSendTimeHistogram(opts...)
	prometheus.MustRegister(
		&LimitCardinalityCollector{
			metricCollector: DefaultClientMetrics.clientStreamSendHistogram,
			desc:            "clientStreamSendHistogram",
			limit:           rpcMetricsCardinalityLimit,
		})
}

func enableClientStreamHistograms(opts ...HistogramOption) {
	EnableClientStreamReceiveTimeHistogram(opts...)
	EnableClientStreamSendTimeHistogram(opts...)
}
