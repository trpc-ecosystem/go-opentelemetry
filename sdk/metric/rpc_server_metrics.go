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
	"math"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// ServerStartedCounter the number of RPCs started on the server.
	ServerStartedCounter = "ServerStartedCounter"
	// ServerHandledCounter the number of RPCs completed on the server, regardless of success or failure.
	ServerHandledCounter = "ServerHandledCounter"
	// ServerHandledHistogram the response latency (seconds) of RPC that had been application-level handled by the server.
	ServerHandledHistogram = "ServerHandledHistogram"
)

var (
	// ServerPanicTotal server panic count total
	ServerPanicTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_panic_total",
			Help:      "Total number of RPCs panic on the server.",
		},
		[]string{"system_name"})

	serverHandledHistogramBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 5}
)

var (
	serverStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_started_total",
			Help:      "Total number of RPCs started on the server.",
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
	serverHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_handled_total",
			Help:      "Total number of RPCs completed on the server, regardless of success or failure.",
		},
		[]string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		},
	)
	serverHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "server_handled_seconds",
			Help:      "Histogram of response latency (seconds) of RPC that had been application-level handled by the server.",
			Buckets:   serverHandledHistogramBuckets,
		},
		[]string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		},
	)
)

// serverLabelsOption initializes the extraLabels list monitored by the server.
// Dynamically initialize extraLabels based on this Client.
var serverLabelsOption = func(name string) []string {
	switch name {
	case ServerStartedCounter:
		return []string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"}
	case ServerHandledCounter, ServerHandledHistogram:
		return []string{
			"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc",
		}
	default:
		return []string{}
	}
}

// SetServerLabelsOption users can use this function to redefine serverLabelsOption.
func SetServerLabelsOption(f func(name string) []string) {
	serverLabelsOption = f
}

func registerRPCServerCounter() {
	initServerCollectors()
	prometheus.MustRegister(
		&LimitCardinalityCollector{serverStartedCounter, "serverStartedCounter", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(
		&LimitCardinalityCollector{serverHandledCounter, "serverHandledCounter", rpcMetricsCardinalityLimit})
}

func initServerCollectors() {
	serverStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_started_total",
			Help:      "Total number of RPCs started on the server.",
		},
		serverLabelsOption(ServerStartedCounter),
	)
	serverHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_handled_total",
			Help:      "Total number of RPCs completed on the server, regardless of success or failure.",
		},
		serverLabelsOption(ServerHandledCounter),
	)
}

// registerRPCHandledHistograms register server and client rpc metrics with histogram
func registerRPCHandledHistograms() {
	serverHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "server_handled_seconds",
			Help:      "Histogram of response latency (seconds) of RPC that had been application-level handled by the server.",
			Buckets:   serverHandledHistogramBuckets,
		},
		serverLabelsOption(ServerHandledHistogram),
	)
	clientHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "client_handled_seconds",
			Help:      "Histogram of response latency (seconds) of the RPC until it is finished by the application.",
			Buckets:   clientHandledHistogramBuckets,
		},
		clientLabelsOption(ClientHandledHistogram),
	)

	prometheus.MustRegister(
		&LimitCardinalityCollector{serverHandledHistogram, "serverHandledHistogram", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(
		&LimitCardinalityCollector{clientHandledHistogram, "clientHandledHistogram", rpcMetricsCardinalityLimit})
}

// setServerHandledHistogramBuckets user customizes serverHandledHistogramBuckets through configuration
func setServerHandledHistogramBuckets(buckets []float64) {
	if newBuckets, ok := validateBuckets(buckets); ok {
		serverHandledHistogramBuckets = newBuckets
	}
}

func validateBuckets(buckets []float64) ([]float64, bool) {
	if len(buckets) == 0 || len(buckets) > 10 {
		return nil, false
	}

	// first check if the buckets are sorted
	if !sort.SliceIsSorted(buckets, func(i, j int) bool {
		return buckets[i] < buckets[j]
	}) {
		return nil, false
	}

	for _, upperBound := range buckets {
		if upperBound <= 0 {
			return nil, false
		}
	}
	last := len(buckets) - 1
	if math.IsInf(buckets[last], +1) {
		// The +Inf bucket is implicit. Remove it here.
		buckets = buckets[:last]
	}

	return buckets, true
}

// DefaultServerMetrics is the default instance of ServerMetrics. It is
// intended to be used in conjunction the default Prometheus metrics
// registry.
var DefaultServerMetrics = NewServerMetrics()

func init() {
	prometheus.MustRegister(&LimitCardinalityCollector{
		metricCollector: DefaultServerMetrics.serverStreamMsgReceived,
		desc:            "serverStreamMsgReceived",
		limit:           rpcMetricsCardinalityLimit,
	})
	prometheus.MustRegister(&LimitCardinalityCollector{
		metricCollector: DefaultServerMetrics.serverStreamMsgSent,
		desc:            "serverStreamMsgSent",
		limit:           rpcMetricsCardinalityLimit,
	})
	prometheus.MustRegister(ServerPanicTotal)
}

// ServerMetrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a rpc server.
// NOTE: only used for stream rpc now
type ServerMetrics struct {
	serverStartedCounter    *prometheus.CounterVec
	serverStreamMsgReceived *prometheus.CounterVec
	serverStreamMsgSent     *prometheus.CounterVec
}

// NewServerMetrics returns a ServerMetrics object. Use a new instance of
// ServerMetrics when not using the default Prometheus metrics registry, for
// example when wanting to control which metrics are added to a registry as
// opposed to automatically adding metrics via init functions.
func NewServerMetrics(counterOpts ...CounterOption) *ServerMetrics {
	opts := counterOptions(counterOpts)
	return &ServerMetrics{
		serverStartedCounter: serverStartedCounter,
		serverStreamMsgReceived: prometheus.NewCounterVec(
			opts.apply(prometheus.CounterOpts{
				Subsystem: "rpc",
				Name:      "server_stream_msg_received_total",
				Help:      "Total number of messages received on the server streaming interface.",
			}),
			defaultStreamLabels,
		),
		serverStreamMsgSent: prometheus.NewCounterVec(
			opts.apply(prometheus.CounterOpts{
				Subsystem: "rpc",
				Name:      "server_stream_msg_sent_total",
				Help:      "Total number of messages sent on the server streaming interface.",
			}),
			defaultStreamLabels,
		),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *ServerMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.serverStreamMsgReceived.Describe(ch)
	m.serverStreamMsgSent.Describe(ch)
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *ServerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.serverStreamMsgReceived.Collect(ch)
	m.serverStreamMsgSent.Collect(ch)
}

// Reset resets the metrics.
func (m *ServerMetrics) Reset() {
	m.serverStreamMsgReceived.Reset()
	m.serverStreamMsgSent.Reset()
	m.serverStartedCounter.Reset()
}

// ServerStartedCounter returns serverStartedCounter counter vec
func (m *ServerMetrics) ServerStartedCounter() *prometheus.CounterVec {
	return m.serverStartedCounter
}

// ServerStreamMsgReceived returns serverStreamMsgReceived counter vec
func (m *ServerMetrics) ServerStreamMsgReceived() *prometheus.CounterVec {
	return m.serverStreamMsgReceived
}

// ServerStreamMsgSent returns serverStreamMsgSent counter vec
func (m *ServerMetrics) ServerStreamMsgSent() *prometheus.CounterVec {
	return m.serverStreamMsgSent
}
