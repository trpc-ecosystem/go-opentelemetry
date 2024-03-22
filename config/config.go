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

package config

import (
	"strings"
	"time"

	opentelemetry "trpc.group/trpc-go/trpc-opentelemetry"
	"trpc.group/trpc-go/trpc-opentelemetry/api/log"
	"trpc.group/trpc-go/trpc-opentelemetry/config/codes"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

// Config opentelemetry trpc plugin config
type Config struct {
	Addr       string            `yaml:"addr"`
	TenantID   string            `yaml:"tenant_id"`
	Sampler    SamplerConfig     `yaml:"sampler"`
	Metrics    MetricsConfig     `yaml:"metrics"`
	Logs       LogsConfig        `yaml:"logs"`
	Traces     TracesConfig      `yaml:"traces"`
	Codes      []*codes.Code     `yaml:"codes"`
	Attributes []*Attribute      `yaml:"attributes"`
	Headers    map[string]string `yaml:"headers"`
}

// TracesConfig traces config
type TracesConfig struct {
	// DisableTraceBody if true, the trace of req and rsp will be closed, which can improve the reporting performance
	DisableTraceBody bool `yaml:"disable_trace_body"`
	// EnableDeferredSample if true, the trace will be sampled after the request is completed
	EnableDeferredSample bool `yaml:"enable_deferred_sample"`
	// DeferredSampleError deferred sample with error
	DeferredSampleError bool `yaml:"deferred_sample_error"`
	// DeferredSampleSlowDuration deferred sample with slow duration
	DeferredSampleSlowDuration time.Duration `yaml:"deferred_sample_slow_duration"`
	// DisableParentSampling ignore parent sampling
	DisableParentSampling bool `yaml:"disable_parent_sampling"`
	// EnableZPage local zpage
	EnableZPage bool `yaml:"enable_zpage"`

	// ExportConfig config of trace exporter
	ExportConfig TraceExporterOption `yaml:"export_config"`
}

// TraceExporterOption defines the behavior of the trace span exporter.
// For detailed parameter description, ref to sdk/trace/batch_span_processor.go (BatchSpanProcessorOptions)
type TraceExporterOption struct {
	MaxQueueSize       int           `yaml:"max_queue_size"`
	BatchTimeout       time.Duration `yaml:"batch_timeout"`
	ExportTimeout      time.Duration `yaml:"export_timeout"`
	MaxExportBatchSize int           `yaml:"max_export_batch_size"`
	MaxPacketSize      int           `yaml:"max_packet_size"`
	BlockOnQueueFull   bool          `yaml:"block_on_queue_full"`
}

// Attribute defines struct of k-v data
type Attribute struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// SamplerConfig sampler config
type SamplerConfig struct {
	Fraction          float64           `yaml:"fraction"`
	SpecialFractions  []SpecialFraction `yaml:"special_fractions"`
	SamplerServerAddr string            `yaml:"sampler_server_addr"`
	SyncInterval      time.Duration     `yaml:"sync_interval"`
}

// SpecialFraction special fraction config
type SpecialFraction struct {
	CalleeService   string           `yaml:"callee_service"`
	DefaultFraction float64          `yaml:"default_fraction"`
	CalleeMethods   []MethodFraction `yaml:"callee_methods"`
}

// MethodFraction method special fraction
type MethodFraction struct {
	Method   string  `yaml:"method"`
	Fraction float64 `yaml:"fraction"`
}

// MetricsConfig defines the configuration for the various elements of Metrics
type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
	// EnabledRegister if true, register service to registry
	EnabledRegister   bool     `yaml:"enable_register"`
	RegistryEndpoints []string `yaml:"registry_endpoints"`
	// TLSCert certificate chain, private key, and root CA certificate
	TLSCert     metric.TLSCert `yaml:"tls_cert"`
	ServerOwner string         `yaml:"server_owner"`
	// Deprecated CodeTypeMapping codeType mapping: key: code value: type(success/exception/timeout) default:success
	CodeTypeMapping        map[string]*metric.CodeTypeMappingDescription `yaml:"code_type_mapping"`
	Codes                  []*codes.Code                                 `yaml:"codes"`
	ClientHistogramBuckets []float64                                     `yaml:"client_histogram_buckets"`
	ServerHistogramBuckets []float64                                     `yaml:"server_histogram_buckets"`
	// DisableRPCMethodMapping do not process with RPCName (cannot be true when using restful API)
	DisableRPCMethodMapping bool `yaml:"disable_rpc_method_mapping"`
	// PrometheusPush prometheus push config
	PrometheusPush metric.PrometheusPushConfig `yaml:"prometheus_push"`
}

// LogsConfig defines the configuration for the various elements of Logs
type LogsConfig struct {
	Addr           string         `yaml:"addr"`
	TLS            TLSConfig      `yaml:"tls"`
	TraceLogMode   LogMode        `yaml:"trace_log_mode"`
	TraceLogOption TraceLogOption `yaml:"trace_log_option"`
	Level          log.Level      `yaml:"level"`
	Enabled        bool           `yaml:"enabled"`
	// DisableRecovery default false and recovery panic and report metrics
	DisableRecovery bool `yaml:"disable_recovery"`
	// enable sampler and only report log when the request is sampled
	EnableSampler bool `yaml:"enable_sampler"`
	// work with EnableSampler, report error when the request is sampled
	EnableSamplerError bool `yaml:"enable_sampler_error"`
	// log rate limit config
	RateLimit RateLimit `yaml:"rate_limit"`
	// log exporter config
	ExportOption ExportOption `yaml:"export_option"`
}

// TraceLogOption defines trace_log option, which also called flow log, print request and response.
type TraceLogOption struct {
	// Exclude some requests, do not care about the log of some requests: such as file upload and download interface, etc.
	Exclude []TraceLogRule `yaml:"exclude"`
}

// TraceLogRule defines the rule of trace log, similar to codec.Code
type TraceLogRule struct {
	// Codeerror code, empty means all error code
	Code string `yaml:"code"`
	// Service empty means full match
	Service string `yaml:"service"`
	// Method empty means full match
	Method string `yaml:"method"`
}

// ExportOption defines the behavior of the log exporter
type ExportOption struct {
	// QueueSize the size of the channel used to save logs locally
	QueueSize int `yaml:"queue_size"`
	// BatchSize batch size of log to send to remote server, when the number of logs in buffer exceeds this config,
	// the logs will be sent to remote server
	BatchSize int `yaml:"batch_size"`
	// BatchTimeout batch timeout of log to send to remote server, when the time since the last log exceeds this config,
	// the logs will be sent to remote server
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	// MaxBatchPacketSize max batch size of log to send to remote server, when the size of logs in buffer exceeds this
	// config, the logs will be sent to remote server
	MaxBatchPacketSize int `yaml:"max_batch_packet_size"`
}

// TLSConfig defines tls config
type TLSConfig struct {
	Enabled            bool `yaml:"enabled"`
	InsecureSkipVeriry bool `yaml:"insecure_skip_veriry"`
}

// RateLimit defines the rate limit config
// For example, tick = 1s, first = 100, thereafter = 3 means that if the same log is printed
// more than 100 times within 1 second, then the same log will be printed every 3 times.
// Here, the "same log" is defined as a duplicate log with exactly the same content and level.
type RateLimit struct {
	// EnableRateLimit if true, enable rate limit
	EnableRateLimit bool `yaml:"enable_rate_limit"`
	// Tick is the effective period of log flow control (that is, from the time when a log is printed,
	// the counter for the same log will be reset to zero after tick time,
	// regardless of whether the flow control is triggered or not)
	Tick time.Duration `yaml:"tick"`
	// First is the threshold of rate limit, that is, when the same log reaches the first threshold,
	// the rate limit is triggered
	First int `yaml:"first"`
	// Thereafter is the number of logs that need to be printed after the rate limit is triggered
	Thereafter int `yaml:"thereafter"`
}

// LogMode log mode
type LogMode int32

const (
	LogModeDefault   LogMode = 0 // default
	LogModeOneLine   LogMode = 1
	LogModeDisable   LogMode = 2
	LogModeMultiLine LogMode = 3
)

// DefaultConfig return the default configuration
func DefaultConfig() Config {
	cfg := Config{
		Addr:     "localhost:12520",
		TenantID: opentelemetry.DefaultTenantID,
		Sampler: SamplerConfig{
			Fraction:          0.001,
			SamplerServerAddr: "localhost:14941",
		},
		Metrics: MetricsConfig{
			Enabled:           true,
			EnabledRegister:   true,
			RegistryEndpoints: []string{"localhost:2379"},
		},
		Logs: LogsConfig{
			Enabled:      false,
			TraceLogMode: LogModeDisable,
		},
		Traces: TracesConfig{DisableTraceBody: false},
	}
	return cfg
}

var logModeMap = map[string]LogMode{
	"disable":   LogModeDisable,   // do not print
	"verbose":   LogModeOneLine,   // single line include body
	"":          LogModeOneLine,   // default value
	"multiline": LogModeMultiLine, // multi lines
	"oneline":   LogModeOneLine,   // single line
}

// UnmarshalText unmarshals text to a log mode.
func (m *LogMode) UnmarshalText(text []byte) error {
	switch v := logModeMap[strings.ToLower(string(text))]; v {
	case LogModeDisable, LogModeOneLine, LogModeMultiLine:
		*m = v
		return nil
	default:
		v = LogModeOneLine
		*m = v
		return nil
	}
}

// MarshalText marshal log mode to text.
func (m LogMode) MarshalText() (text []byte, err error) {
	switch m {
	case LogModeDefault, LogModeOneLine:
		return []byte("oneline"), nil
	case LogModeDisable:
		return []byte("disable"), nil
	case LogModeMultiLine:
		return []byte("multiline"), nil
	default:
		return []byte("oneline"), nil
	}
}
