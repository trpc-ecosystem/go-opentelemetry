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

package logs

import (
	"context"
	"crypto/tls"
	"errors"

	v1proto "github.com/golang/protobuf/proto"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/plugin"

	"trpc.group/trpc-go/go-opentelemetry"

	"trpc.group/trpc-go/go-opentelemetry/api"
	logtps "trpc.group/trpc-go/go-opentelemetry/api/log"
	"trpc.group/trpc-go/go-opentelemetry/config"
	"trpc.group/trpc-go/go-opentelemetry/exporter/asyncexporter"
	otlplog "trpc.group/trpc-go/go-opentelemetry/exporter/otlp"
	"trpc.group/trpc-go/go-opentelemetry/oteltrpc/consts"
	otelprometheus "trpc.group/trpc-go/go-opentelemetry/oteltrpc/metrics/prometheus"
	"trpc.group/trpc-go/go-opentelemetry/otelzap"
	sdklog "trpc.group/trpc-go/go-opentelemetry/sdk/log"
)

const (
	writerType  = consts.PluginName
	otelSection = consts.PluginName
)

func init() {
	log.RegisterWriter(writerType, &writer{})
}

var _ plugin.Factory = (*writer)(nil)

type writer struct {
}

// Type trpc Type interface
func (w writer) Type() string {
	return writerType
}

// Setup trpc plugin interface
func (w writer) Setup(name string, configDec plugin.Decoder) error {
	decoder, ok := configDec.(*log.Decoder)
	if !ok {
		return errors.New("log decoder type invalid")
	}
	cfg, err := ParseConfig(decoder)
	if err != nil {
		return err
	}

	if !cfg.Logs.Enabled {
		decoder.Core = zapcore.NewNopCore()
		return nil
	}
	var exp sdklog.Exporter
	if asyncexporter.Concurrency > 1 {
		exp, err = newAsyncExporter(cfg, asyncexporter.Concurrency)
	} else {
		exp, err = newOtlpExporter(cfg)
	}
	if err != nil {
		return errors.New("opentelemetry log exporter create fail: " + err.Error())
	}

	kvs := []attribute.KeyValue{
		api.TpsTenantIDKey.String(cfg.TenantID),
		attribute.Key(api.ServerKey).String(trpc.GlobalConfig().Server.App + "." + trpc.GlobalConfig().Server.Server),
		attribute.Key(api.EnvKey).String(trpc.GlobalConfig().Global.EnvName),
		attribute.Key(api.InstanceKey).String(trpc.GlobalConfig().Global.LocalIP),
	}
	var opts []sdklog.LoggerOption
	if cfg.Logs.Level != "" {
		opts = append(opts, sdklog.WithLevelEnable(cfg.Logs.Level))
	}
	opts = append(opts, sdklog.WithEnableSampler(cfg.Logs.EnableSampler))
	opts = append(opts, sdklog.WithEnableSamplerError(cfg.Logs.EnableSamplerError))
	decoder.Core, decoder.ZapLevel = otelzap.NewBatchCoreAndLevel(
		otelzap.NewBatchWriteSyncer(
			exp,
			resource.NewWithAttributes(semconv.SchemaURL, kvs...),
			getBatchSyncerOptions(cfg.Logs)...,
		),
		opts...,
	)

	if enableLogRateLimit(cfg) {
		decoder.Core = zapcore.NewSamplerWithOptions(decoder.Core,
			cfg.Logs.RateLimit.Tick, cfg.Logs.RateLimit.First, cfg.Logs.RateLimit.Thereafter)
	}

	log.Info("opentelemetry zap log setup success")
	return nil
}

func getBatchSyncerOptions(cfg config.LogsConfig) []otelzap.BatchSyncerOption {
	queueSize := otelzap.DefaultMaxQueueSize
	batchSize := otelzap.DefaultMaxExportBatchSize
	batchTimeout := otelzap.DefaultBatchTimeout
	maxBatchPacketSize := otelzap.DefaultMaxBatchedPacketSize
	exportOpt := cfg.ExportOption
	// configuration in yaml has lower priority than settings in code
	if exportOpt.QueueSize > 0 {
		queueSize = exportOpt.QueueSize
	}
	if asyncexporter.QueueSize > queueSize {
		queueSize = asyncexporter.QueueSize
	}
	if exportOpt.BatchSize > 0 {
		batchSize = exportOpt.BatchSize
	}
	if exportOpt.BatchTimeout > 0 {
		batchTimeout = exportOpt.BatchTimeout
	}
	if exportOpt.MaxBatchPacketSize > 0 {
		maxBatchPacketSize = exportOpt.MaxBatchPacketSize
	}
	return []otelzap.BatchSyncerOption{
		otelzap.WithEnableSampler(cfg.EnableSampler),
		otelzap.WithMaxQueueSize(queueSize),
		otelzap.WithMaxExportBatchSize(batchSize),
		otelzap.WithBatchTimeout(batchTimeout),
		otelzap.WithMaxPacketSize(maxBatchPacketSize),
		otelzap.WithEnableSamplerError(cfg.EnableSamplerError),
	}
}

func enableLogRateLimit(cfg *config.Config) bool {
	if cfg == nil || !cfg.Logs.RateLimit.EnableRateLimit {
		return false
	}
	return cfg.Logs.RateLimit.First != 0 && cfg.Logs.RateLimit.Thereafter != 0 && cfg.Logs.RateLimit.Tick != 0
}

// ParseConfig parse config from decoder
var ParseConfig = func(decoder *log.Decoder) (*config.Config, error) {
	cfg := &config.Config{
		Addr:     opentelemetry.DefaultExporterAddr,
		TenantID: opentelemetry.DefaultTenantID,
		Logs: config.LogsConfig{
			Enabled: false,
			Level:   opentelemetry.DefaultLogLevel,
		},
	}

	if err := loadConfig(cfg); err != nil {
		return nil, err
	}

	if decoder.OutputConfig.Level != "" {
		var s logtps.Level
		err := s.UnmarshalText([]byte(decoder.OutputConfig.Level))
		if err != nil {
			return nil, errors.New("opentelemetry level invalid: " + decoder.OutputConfig.Level)
		}

		cfg.Logs.Level = s
	}
	if cfg.Logs.Addr != "" {
		cfg.Addr = cfg.Logs.Addr
	}
	return cfg, nil
}

func loadConfig(cfg *config.Config) error {
	telemetrys, ok := trpc.GlobalConfig().Plugins["telemetry"]
	if ok {
		if otelDec, ok := telemetrys[otelSection]; ok {
			if err := otelDec.Decode(cfg); err != nil {
				return errors.New("opentelemetry config decode fail: " + err.Error())
			}
		}
	}
	return nil
}

func newOtlpExporter(cfg *config.Config) (*otlplog.Exporter, error) {
	return otlplog.NewExporter(otlpTLSOption(&cfg.Logs),
		otlplog.WithAddress(cfg.Addr),
		otlplog.WithCompressor("gzip"),
		otlplog.WithHeaders(map[string]string{api.TenantHeaderKey: cfg.TenantID}),
		otlplog.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			grpc_prometheus.UnaryClientInterceptor,
			packetLogSizeMetric(),
		)))
}

func newAsyncExporter(cfg *config.Config, concurrency int) (*asyncexporter.Exporter, error) {
	return asyncexporter.NewExporter(asyncTLSOption(&cfg.Logs),
		asyncexporter.WithAddress(cfg.Addr),
		asyncexporter.WithCompressor("gzip"),
		asyncexporter.WithConcurrency(concurrency),
		asyncexporter.WithHeaders(map[string]string{api.TenantHeaderKey: cfg.TenantID}),
		asyncexporter.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			grpc_prometheus.UnaryClientInterceptor,
			packetLogSizeMetric(),
		)))
}

func otlpTLSOption(cfg *config.LogsConfig) otlplog.ExporterOption {
	if !cfg.TLS.Enabled {
		return otlplog.WithInsecure()
	}
	tlsConfig := &tls.Config{}
	if cfg.TLS.InsecureSkipVeriry {
		tlsConfig.InsecureSkipVerify = true
	}
	cred := credentials.NewTLS(tlsConfig)
	return otlplog.WithTLSCredentials(cred)
}

func asyncTLSOption(cfg *config.LogsConfig) asyncexporter.ExporterOption {
	if !cfg.TLS.Enabled {
		return asyncexporter.WithInsecure()
	}
	tlsConfig := &tls.Config{}
	if cfg.TLS.InsecureSkipVeriry {
		tlsConfig.InsecureSkipVerify = true
	}
	cred := credentials.NewTLS(tlsConfig)
	return asyncexporter.WithTLSCredentials(cred)
}

// packetLogSizeMetric metric for log pakcet isze
func packetLogSizeMetric() func(ctx context.Context, method string,
	req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		switch req := req.(type) {
		case proto.Message:
			otelprometheus.ObserveExportLogsBytes(proto.Size(req))
		case v1proto.Message:
			otelprometheus.ObserveExportLogsBytes(v1proto.Size(req))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
