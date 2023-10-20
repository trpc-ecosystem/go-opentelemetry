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

package otlp

import (
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"trpc-system/go-opentelemetry/api"
	"trpc-system/go-opentelemetry/exporter/retry"
)

const (
	// DefaultCollectorPort is the port the Exporter will attempt connect to
	// if no collector port is provided.
	DefaultCollectorPort uint16 = 55680
	// DefaultCollectorHost is the host address the Exporter will attempt
	// connect to if no collector address is provided.
	DefaultCollectorHost string = "localhost"
	// DefaultNumWorkers is the number of goroutines the Exporter will use when
	// processing telemetry.
	DefaultNumWorkers uint = 1

	// DefaultGRPCServiceConfig is the gRPC service config used if none is
	// provided by the user.
	//
	// For more info on gRPC service configs:
	// https://github.com/grpc/proposal/blob/master/A6-client-retries.md
	//
	// Note: MaxAttempts > 5 are treated as 5. See
	// https://github.com/grpc/proposal/blob/master/A6-client-retries.md#validation-of-retrypolicy
	// for more details.
	DefaultGRPCServiceConfig = `{
	"methodConfig":[{
		"name":[
			{ "service":"opentelemetry.proto.collector.metrics.v1.MetricsService" },
			{ "service":"opentelemetry.proto.collector.trace.v1.TraceService" }
		],
		"retryPolicy":{
			"MaxAttempts":5,
			"InitialBackoff":"3s",
			"MaxBackoff":"10s",
			"BackoffMultiplier":3,
			"RetryableStatusCodes":[
				"UNAVAILABLE",
				"CANCELLED",
				"DEADLINE_EXCEEDED",
				"RESOURCE_EXHAUSTED",
				"ABORTED",
				"OUT_OF_RANGE",
				"UNAVAILABLE",
				"DATA_LOSS"
			]
		}
	}]
}`
)

// ExporterOption are setting options passed to an Exporter on creation.
type ExporterOption func(*config)

type config struct {
	collectorAddr      string
	compressor         string
	reconnectionPeriod time.Duration
	grpcServiceConfig  string
	grpcDialOptions    []grpc.DialOption
	headers            map[string]string
	clientCredentials  credentials.TransportCredentials
	numWorkers         uint
	requestFunc        retry.RequestFunc
}

// WorkerCount sets the number of Goroutines to use when processing telemetry.
func WorkerCount(n uint) ExporterOption {
	if n == 0 {
		n = DefaultNumWorkers
	}
	return func(cfg *config) {
		cfg.numWorkers = n
	}
}

// WithInsecure disables client transport security for the exporter's gRPC connection
// just like grpc.WithInsecure() https://pkg.go.dev/google.golang.org/grpc#WithInsecure
// does. Note, by default, client security is required unless WithInsecure is used.
func WithInsecure() ExporterOption {
	return func(cfg *config) {
		// grpc.WithInsecure is deprecated
		cfg.clientCredentials = insecure.NewCredentials()
	}
}

// WithAddress allows one to set the address that the exporter will
// connect to the collector on. If unset, it will instead try to use
// connect to DefaultCollectorHost:DefaultCollectorPort.
func WithAddress(addr string) ExporterOption {
	return func(cfg *config) {
		cfg.collectorAddr = addr
	}
}

// WithReconnectionPeriod allows one to set the delay between next connection attempt
// after failing to connect with the collector.
func WithReconnectionPeriod(rp time.Duration) ExporterOption {
	return func(cfg *config) {
		cfg.reconnectionPeriod = rp
	}
}

// WithCompressor will set the compressor for the gRPC client to use when sending requests.
// It is the responsibility of the caller to ensure that the compressor set has been registered
// with google.golang.org/grpc/encoding. This can be done by encoding.RegisterCompressor. Some
// compressors auto-register on import, such as gzip, which can be registered by calling
// `import _ "google.golang.org/grpc/encoding/gzip"`
func WithCompressor(compressor string) ExporterOption {
	return func(cfg *config) {
		cfg.compressor = compressor
	}
}

// WithHeaders will send the provided headers with gRPC requests
func WithHeaders(headers map[string]string) ExporterOption {
	return func(cfg *config) {
		cfg.headers = headers
	}
}

// WithTenantID will set 'X-Tps-TenantID' as grpc header
func WithTenantID(tenantID string) ExporterOption {
	return func(cfg *config) {
		if cfg.headers == nil {
			cfg.headers = make(map[string]string)
		}
		cfg.headers[api.TenantHeaderKey] = tenantID
	}
}

// WithTLSCredentials allows the connection to use TLS credentials
// when talking to the server. It takes in grpc.TransportCredentials instead
// of say a Certificate file or a tls.Certificate, because the retrieving
// these credentials can be done in many ways e.g. plain file, in code tls.Config
// or by certificate rotation, so it is up to the caller to decide what to use.
func WithTLSCredentials(creds credentials.TransportCredentials) ExporterOption {
	return func(cfg *config) {
		cfg.clientCredentials = creds
	}
}

// WithGRPCServiceConfig defines the default gRPC service config used.
func WithGRPCServiceConfig(serviceConfig string) ExporterOption {
	return func(cfg *config) {
		cfg.grpcServiceConfig = serviceConfig
	}
}

// WithGRPCDialOption opens support to any grpc.DialOption to be used. If it conflicts
// with some other configuration the GRPC specified via the collector the ones here will
// take preference since they are set last.
func WithGRPCDialOption(opts ...grpc.DialOption) ExporterOption {
	return func(cfg *config) {
		cfg.grpcDialOptions = opts
	}
}

// WithRetryConfig set retry config, suggesting retry.DefaultConfig
func WithRetryConfig(retryCfg retry.Config) ExporterOption {
	return func(c *config) {
		c.requestFunc = retryCfg.RequestFunc(retryable)
	}
}

// retryable returns if err identifies a request that can be retried and a
// duration to wait for if an explicit throttle time is included in err.
func retryable(err error) (bool, time.Duration) {
	s := status.Convert(err)
	switch s.Code() {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		return true, throttleDelay(s)
	}

	// Not a retry-able error.
	return false, 0
}

// throttleDelay returns a duration to wait for if an explicit throttle time
// is included in the response status.
func throttleDelay(s *status.Status) time.Duration {
	for _, detail := range s.Details() {
		if t, ok := detail.(*errdetails.RetryInfo); ok {
			return t.RetryDelay.AsDuration()
		}
	}
	return 0
}
