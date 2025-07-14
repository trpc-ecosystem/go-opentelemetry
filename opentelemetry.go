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

package opentelemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	apitrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"trpc.group/trpc-go/trpc-opentelemetry/api"
	apilog "trpc.group/trpc-go/trpc-opentelemetry/api/log"
	ecosystemotlp "trpc.group/trpc-go/trpc-opentelemetry/exporter/otlp"
	"trpc.group/trpc-go/trpc-opentelemetry/exporter/retry"
	"trpc.group/trpc-go/trpc-opentelemetry/pkg/zpage"
	sdklog "trpc.group/trpc-go/trpc-opentelemetry/sdk/log"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/trace"

	_ "google.golang.org/grpc/encoding/gzip" // open gzip
)

var globalTracer = apitrace.NewNoopTracerProvider().Tracer("")

var globalIDGenerator sdktrace.IDGenerator

var (
	DefaultTenantID     = "default"
	DefaultExporterAddr = "localhost:12520"
	DefaultLogLevel     = apilog.InfoLevel
	MaxSendMessageSize  = 4194304
)

// GlobalTracer global tracer
func GlobalTracer() apitrace.Tracer {
	return globalTracer
}

// SetGlobalIDGenerator set global id generator
func SetGlobalIDGenerator(gen sdktrace.IDGenerator) {
	globalIDGenerator = gen
}

// GlobalIDGenerator global id generator
func GlobalIDGenerator() sdktrace.IDGenerator {
	return globalIDGenerator
}

// Start opentelemetry enables helper function
func Start(ctx context.Context, spanName string, opts ...apitrace.SpanStartOption) (context.Context, apitrace.Span) {
	return globalTracer.Start(ctx, spanName, opts...)
}

// WithSpan sets up a span with the given name and calls the supplied function.
func WithSpan(ctx context.Context, spanName string, fn func(ctx context.Context) error,
	opts ...apitrace.SpanStartOption,
) error {
	ctx, sp := globalTracer.Start(ctx, spanName, opts...)
	defer sp.End()
	return fn(ctx)
}

// AsyncSetup init async setup, used in long running service, report data by batch
func AsyncSetup(addr string, tenantID string, sampler sdktrace.Sampler) error {
	return setup(addr, WithTenantID(tenantID), WithSampler(sampler))
}

// Setup enables oteltrpc plugin
func Setup(addr string, opts ...SetupOption) error {
	return setup(addr, opts...)
}

func newTraceHTTPExporter(addr string, o *setupOptions) (sdktrace.SpanExporter, error) {
	otlpTraceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(addr),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithHeaders(o.otlptraceHeader),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: retry.DefaultConfig.InitialInterval,
			MaxInterval:     retry.DefaultConfig.MaxInterval,
			MaxElapsedTime:  retry.DefaultConfig.MaxElapsedTime,
		}),
	}
	switch {
	case strings.HasPrefix(addr, "http://"):
		otlpTraceOpts = append(otlpTraceOpts, otlptracehttp.WithInsecure())
		otlpTraceOpts = append(otlpTraceOpts, otlptracehttp.WithEndpoint(strings.TrimPrefix(addr, "http://")))
	case strings.HasPrefix(addr, "https://"):
		otlpTraceOpts = append(otlpTraceOpts, otlptracehttp.WithEndpoint(strings.TrimPrefix(addr, "https://")))
	default:
		otlpTraceOpts = append(otlpTraceOpts, otlptracehttp.WithEndpoint(addr))
	}
	exporter, err := otlptracehttp.New(context.Background(), otlpTraceOpts...)
	if err != nil {
		return nil, err
	}
	return exporter, nil
}

func newTraceGRPCExporter(addr string, o *setupOptions) (sdktrace.SpanExporter, error) {
	otlpTraceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(addr),
		otlptracegrpc.WithCompressor("gzip"),
		otlptracegrpc.WithHeaders(o.otlptraceHeader),
		otlptracegrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMessageSize))),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: retry.DefaultConfig.InitialInterval,
			MaxInterval:     retry.DefaultConfig.MaxInterval,
			MaxElapsedTime:  retry.DefaultConfig.MaxElapsedTime,
		}),
	}
	if len(o.grpcDialOptions) > 0 {
		otlpTraceOpts = append(otlpTraceOpts, otlptracegrpc.WithDialOption(o.grpcDialOptions...))
	}
	exporter, err := otlptracegrpc.New(context.Background(), otlpTraceOpts...)
	if err != nil {
		return nil, err
	}
	return exporter, nil
}

func newExporter(addr string, o *setupOptions) (sdktrace.SpanExporter, error) {
	if o.httpEnabled {
		return newTraceHTTPExporter(addr, o)
	}
	return newTraceGRPCExporter(addr, o)
}

func setup(addr string, options ...SetupOption) error {
	o := defaultSetupOptions()
	for _, opt := range options {
		opt(o)
	}

	exp, err := newExporter(addr, o)
	if err != nil {
		return err
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithSampler(o.sampler))
	opts = append(opts, sdktrace.WithSpanProcessor(
		trace.NewDeferredSampleProcessor(
			trace.NewBatchSpanProcessor(exp, o.batchSpanOption...), o.deferredSampler)))

	if o.zPageEnabled {
		opts = append(opts, sdktrace.WithSpanProcessor(zpage.GetZPageProcessor()))
	}

	kvs := []attribute.KeyValue{
		api.TpsTenantIDKey.String(o.tenantID),
		api.TpsOwnerKey.String(o.ServerOwner),
		api.TpsCmdbIDKey.String(o.CmdbID),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKNameKey.String(api.OpenTelemetryName),
	}
	kvs = append(kvs, o.additionalLabels...)
	if o.serviceName != "" {
		kvs = append(kvs, semconv.ServiceNameKey.String(o.serviceName))
	}
	if o.serviceNamespace != "" {
		kvs = append(kvs, semconv.ServiceNamespaceKey.String(o.serviceNamespace))
	}

	if o.logEnabled {
		if err = setupLog(addr, o, kvs); err != nil {
			return err
		}
	}

	res := resource.NewWithAttributes(semconv.SchemaURL, kvs...)

	if o.metricEnabled {
		if err = setupMetric(addr, res, o); err != nil {
			return err
		}
	}
	opts = append(opts, sdktrace.WithResource(res))
	if o.idGenerator != nil {
		opts = append(opts, sdktrace.WithIDGenerator(o.idGenerator))
	}

	traceProvider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{},
		propagation.Baggage{}))
	globalTracer = otel.Tracer("")
	return nil
}

var meterProvider *sdkmetric.MeterProvider

func newMetricHTTPExporter(addr string, o *setupOptions) (*sdkmetric.Exporter, error) {
	otlpMetricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithEndpoint(addr),
		otlpmetrichttp.WithHeaders(o.otlptraceHeader),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{
			Enabled:         true,
			InitialInterval: retry.DefaultConfig.InitialInterval,
			MaxInterval:     retry.DefaultConfig.MaxInterval,
			MaxElapsedTime:  retry.DefaultConfig.MaxElapsedTime,
		}),
	}
	exp, err := otlpmetrichttp.New(context.Background(), otlpMetricOpts...)
	return &exp, err
}

func newMetricGrpcExporter(addr string, o *setupOptions) (*sdkmetric.Exporter, error) {
	otlpMetricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(addr),
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithHeaders(o.otlptraceHeader),
		otlpmetricgrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMessageSize))),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: retry.DefaultConfig.InitialInterval,
			MaxInterval:     retry.DefaultConfig.MaxInterval,
			MaxElapsedTime:  retry.DefaultConfig.MaxElapsedTime,
		}),
	}
	if len(o.grpcDialOptions) > 0 {
		otlpMetricOpts = append(otlpMetricOpts, otlpmetricgrpc.WithDialOption(o.grpcDialOptions...))
	}
	exp, err := otlpmetricgrpc.New(context.Background(), otlpMetricOpts...)
	return &exp, err
}

func setupMetric(addr string, res *resource.Resource, o *setupOptions) (err error) {
	var exporter *sdkmetric.Exporter
	if o.httpEnabled {
		exporter, err = newMetricHTTPExporter(addr, o)
	} else {
		exporter, err = newMetricGrpcExporter(addr, o)
	}
	if err != nil {
		return err
	}
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(*exporter)), sdkmetric.WithResource(res))
	otel.SetMeterProvider(meterProvider)
	return nil
}

func setupLog(addr string, o *setupOptions, kvs []attribute.KeyValue) (err error) {
	exporter, err := ecosystemotlp.NewExporter(
		ecosystemotlp.WithInsecure(),
		ecosystemotlp.WithAddress(addr),
		ecosystemotlp.WithTenantID(o.tenantID),
		ecosystemotlp.WithCompressor("gzip"),
		ecosystemotlp.WithHeaders(o.otlptraceHeader),
		ecosystemotlp.WithRetryConfig(retry.DefaultConfig),
	)
	if err != nil {
		return err
	}
	logger := sdklog.NewLogger(
		sdklog.WithResource(resource.NewWithAttributes(semconv.SchemaURL, kvs...)),
		sdklog.WithBatcher(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithLevelEnable(o.enabledLogLevel),
	)
	apilog.SetGlobalLogger(logger)
	return nil
}

type setupOptions struct {
	tenantID         string
	sampler          sdktrace.Sampler
	serviceName      string
	serviceNamespace string
	grpcDialOptions  []grpc.DialOption
	resourceLabels   *resource.Resource
	logEnabled       bool
	enabledLogLevel  apilog.Level
	metricEnabled    bool
	httpEnabled      bool
	zPageEnabled     bool
	ServerOwner      string
	CmdbID           string
	additionalLabels []attribute.KeyValue
	deferredSampler  trace.DeferredSampler
	batchSpanOption  []trace.BatchSpanProcessorOption
	idGenerator      sdktrace.IDGenerator
	otlptraceHeader  map[string]string
}

func defaultSetupOptions() *setupOptions {
	return &setupOptions{
		tenantID:        DefaultTenantID,
		sampler:         sdktrace.AlwaysSample(),
		logEnabled:      false,
		enabledLogLevel: apilog.InfoLevel,
		otlptraceHeader: make(map[string]string),
		deferredSampler: trace.NewDeferredSampler(trace.DeferredSampleConfig{}),
	}
}

// SetupOption opentelemetry setup option
type SetupOption func(*setupOptions)

// WithLogEnabled log is enabled or not
func WithLogEnabled(enabled bool) SetupOption {
	return func(options *setupOptions) {
		options.logEnabled = enabled
	}
}

// WithLevelEnable with log level
func WithLevelEnable(level apilog.Level) SetupOption {
	return func(options *setupOptions) {
		options.enabledLogLevel = level
	}
}

// WithServerOwner with server owner
func WithServerOwner(owner string) SetupOption {
	return func(options *setupOptions) {
		options.ServerOwner = owner
	}
}

// WithHeader with server header
func WithHeader(header map[string]string) SetupOption {
	return func(options *setupOptions) {
		for k, v := range header {
			options.otlptraceHeader[k] = v
		}
	}
}

// WithTenantID with tenant id
func WithTenantID(tenantID string) SetupOption {
	return func(options *setupOptions) {
		options.tenantID = tenantID
		options.otlptraceHeader[api.TenantHeaderKey] = tenantID
	}
}

// WithGRPCDialOption with grpc dial option
func WithGRPCDialOption(opts ...grpc.DialOption) SetupOption {
	return func(cfg *setupOptions) {
		cfg.grpcDialOptions = opts
	}
}

// WithResource with resource
func WithResource(rs *resource.Resource) SetupOption {
	return func(options *setupOptions) {
		options.resourceLabels = rs
	}
}

// WithSampler with sampler
func WithSampler(sampler sdktrace.Sampler) SetupOption {
	return func(options *setupOptions) {
		options.sampler = sampler
	}
}

// WithServiceName with service name
func WithServiceName(serviceName string) SetupOption {
	return func(options *setupOptions) {
		options.serviceName = serviceName
	}
}

// WithLabels with extra attribute labels
func WithLabels(opts ...attribute.KeyValue) SetupOption {
	return func(cfg *setupOptions) {
		cfg.additionalLabels = opts
	}
}

// WithServiceNamespace with namespace
func WithServiceNamespace(namespace string) SetupOption {
	return func(cfg *setupOptions) {
		cfg.serviceNamespace = namespace
	}
}

// WithDeferredSampler with deferred sampler filter function
func WithDeferredSampler(deferredSampler trace.DeferredSampler) SetupOption {
	return func(cfg *setupOptions) {
		cfg.deferredSampler = deferredSampler
	}
}

// WithBatchSpanProcessorOption sets the options to configure a BatchSpanProcessor.
func WithBatchSpanProcessorOption(opts ...trace.BatchSpanProcessorOption) SetupOption {
	return func(cfg *setupOptions) {
		cfg.batchSpanOption = opts
	}
}

// WithMetricEnabled enables metric
func WithMetricEnabled(enabled bool) SetupOption {
	return func(cfg *setupOptions) {
		cfg.metricEnabled = enabled
	}
}

// WithHTTPEnabled enabled http protocol, default is grpc
func WithHTTPEnabled(enabled bool) SetupOption {
	return func(cfg *setupOptions) {
		cfg.httpEnabled = enabled
	}
}

// WithIDGenerator with custom id generator
func WithIDGenerator(idGenerator sdktrace.IDGenerator) SetupOption {
	return func(cfg *setupOptions) {
		cfg.idGenerator = idGenerator
	}
}

// WithZPageSpanProcessor enables zpage span processor
func WithZPageSpanProcessor(enable bool) SetupOption {
	return func(cfg *setupOptions) {
		cfg.zPageEnabled = enable
	}
}

// Shutdown report all data before process exit
func Shutdown(ctx context.Context) error {
	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
	}
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
	}
	if logger, ok := apilog.GlobalLogger().(interface{ Shutdown(context.Context) error }); ok {
		if err := logger.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
