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

package oteltrpc

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"

	v1proto "github.com/golang/protobuf/proto"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/plugin"

	opentelemetry "trpc.group/trpc-go/trpc-opentelemetry"

	"trpc.group/trpc-go/trpc-opentelemetry/config"
	"trpc.group/trpc-go/trpc-opentelemetry/config/codes"
	trpccodes "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/codes"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/consts"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/logs"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/metrics/prometheus"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/traces"
	"trpc.group/trpc-go/trpc-opentelemetry/pkg/zpage"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/remote"
	ecosystemtrace "trpc.group/trpc-go/trpc-opentelemetry/sdk/trace"
)

func init() {
	Register()
	prometheus.MonitorTRPCSDKMeta()
}

// RegisterTextMapSupplier .
func RegisterTextMapSupplier(carrier func(md codec.MetaData, msg codec.Msg) propagation.TextMapCarrier) {
	traces.RegisterCarrier(carrier)
}

var _ plugin.Factory = (*factory)(nil)

type factory struct{}

func (f factory) Type() string {
	return consts.PluginType
}

func packetSizeMetric() func(ctx context.Context, method string,
	req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		switch req := req.(type) {
		case proto.Message:
			prometheus.ObserveExportSpansBytes(proto.Size(req))
		case v1proto.Message:
			prometheus.ObserveExportSpansBytes(v1proto.Size(req))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// DefaultRecoveryHandler default recovery handler can be set by user
var DefaultRecoveryHandler = func(ctx context.Context, panicErr interface{}) error {
	return fmt.Errorf("panic:%v", panicErr)
}

func recovery() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				buf := make([]byte, 2048)
				buf = buf[:runtime.Stack(buf, false)]
				// panic may happen during print, need recover again
				func() {
					defer func() {
						_ = recover()
					}()
					log.Printf("opentelemetry: otel export panic:%v, req:%T, stack:%s",
						rec, req, buf)
					log.Printf("opentelemetry: otel export panic req detail:%#v", req)
				}()
				prometheus.IncrSDKPanicTotal()
				err = DefaultRecoveryHandler(ctx, rec)
			}
		}()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// DefaultSampler sampler could be set by user
var DefaultSampler sdktrace.Sampler

func (f factory) Setup(name string, configDec plugin.Decoder) error {
	cfg, err := ParseConfig(configDec)
	if err != nil {
		return err
	}
	ecosystemtrace.DefaultGetCalleeMethodInfo = getCalleeMethodInfoFunc()
	if DefaultSampler == nil {
		DefaultSampler = ecosystemtrace.NewSampler(
			cfg.TenantID,
			ecosystemtrace.SamplerConfig{
				Fraction:           cfg.Sampler.Fraction,
				SpecialFractions:   getSpecialFractions(cfg.Sampler.SpecialFractions),
				SamplerServiceAddr: cfg.Sampler.SamplerServerAddr,
				SyncInterval:       cfg.Sampler.SyncInterval,
			},
			func(opt *ecosystemtrace.SamplerOptions) {
				if cfg.Traces.EnableDeferredSample {
					opt.DefaultSamplingDecision = sdktrace.RecordOnly
				}
			})
	}
	DeferredSampler := ecosystemtrace.NewDeferredSampler(ecosystemtrace.DeferredSampleConfig{
		Enabled:            cfg.Traces.EnableDeferredSample,
		SampleError:        cfg.Traces.DeferredSampleError,
		SampleSlowDuration: cfg.Traces.DeferredSampleSlowDuration,
	})
	var isHTTPEnabled bool
	if strings.HasPrefix(cfg.Addr, "http://") || strings.HasPrefix(cfg.Addr, "https://") {
		isHTTPEnabled = true
	}
	serviceName := trpc.GlobalConfig().Server.App + "." + trpc.GlobalConfig().Server.Server
	if cfg.Traces.EnableZPage {
		admin.HandleFunc("/debug/tracez", zpage.GetZPageHandlerFunc())
	}
	err = opentelemetry.Setup(cfg.Addr,
		opentelemetry.WithHeader(cfg.Header),
		opentelemetry.WithTenantID(cfg.TenantID),
		opentelemetry.WithSampler(DefaultSampler),
		opentelemetry.WithDeferredSampler(DeferredSampler),
		opentelemetry.WithServiceName(serviceName),
		opentelemetry.WithLabels(getResourceAttributes(cfg)...),
		opentelemetry.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			recovery(),
			grpcprometheus.UnaryClientInterceptor,
			packetSizeMetric())),
		opentelemetry.WithServerOwner(cfg.Metrics.ServerOwner),
		opentelemetry.WithHTTPEnabled(isHTTPEnabled),
		opentelemetry.WithBatchSpanProcessorOption(buildBatchSpanProcessorOptions(cfg.Traces.ExportConfig)...),
		opentelemetry.WithIDGenerator(opentelemetry.GlobalIDGenerator()),
		opentelemetry.WithZPageSpanProcessor(cfg.Traces.EnableZPage),
	)
	if err != nil {
		return err
	}
	configurator := remote.NewRemoteConfigurator(cfg.Sampler.SamplerServerAddr, 0,
		cfg.TenantID, trpc.GlobalConfig().Server.App, trpc.GlobalConfig().Server.Server,
	)
	if cfg.Metrics.Enabled {
		prometheus.Setup(cfg.TenantID, cfg.Metrics.RegistryEndpoints,
			metric.WithEnabledZPage(cfg.Traces.EnableZPage),
			metric.WithConfigurator(configurator),
			metric.WithClientHistogramBuckets(cfg.Metrics.ClientHistogramBuckets),
			metric.WithServerHistogramBuckets(cfg.Metrics.ServerHistogramBuckets),
			metric.WithTLSCert(cfg.Metrics.TLSCert),
			metric.WithEnabled(true),
			metric.WithEnabledRegister(cfg.Metrics.EnabledRegister),
			metric.WithMetricsPrometheusPush(cfg.Metrics.PrometheusPush),
		)
	}
	setupCodes(cfg, configurator)
	setupFilters(cfg)
	return nil
}

func getSpecialFractions(fractions []config.SpecialFraction) map[string]ecosystemtrace.SpecialFraction {
	result := make(map[string]ecosystemtrace.SpecialFraction)
	for _, f := range fractions {
		result[f.CalleeService] = ecosystemtrace.SpecialFraction{
			DefaultFraction: f.DefaultFraction,
			Methods:         getSpecialFractionMethods(f.CalleeMethods),
		}
	}

	return result
}

func getSpecialFractionMethods(methods []config.MethodFraction) map[string]ecosystemtrace.MethodFraction {
	result := make(map[string]ecosystemtrace.MethodFraction)
	for _, m := range methods {
		result[m.Method] = ecosystemtrace.MethodFraction{Fraction: m.Fraction}
	}
	return result
}

func getCalleeMethodInfoFunc() ecosystemtrace.GetCalleeMethodInfo {
	return func(ctx context.Context) ecosystemtrace.MethodInfo {
		msg := trpc.Message(ctx)
		return ecosystemtrace.MethodInfo{
			CalleeService: msg.CalleeService(),
			CalleeMethod:  msg.CalleeMethod(),
		}
	}
}

func buildBatchSpanProcessorOptions(c config.TraceExporterOption) (options []ecosystemtrace.BatchSpanProcessorOption) {
	if c.BlockOnQueueFull {
		options = append(options, ecosystemtrace.WithBlocking())
	}

	if c.MaxQueueSize > 0 {
		options = append(options, ecosystemtrace.WithMaxQueueSize(c.MaxQueueSize))
	}
	if c.BatchTimeout > 0 {
		options = append(options, ecosystemtrace.WithBatchTimeout(c.BatchTimeout))
	}
	if c.ExportTimeout > 0 {
		options = append(options, ecosystemtrace.WithExportTimeout(c.ExportTimeout))
	}
	if c.MaxExportBatchSize > 0 {
		options = append(options, ecosystemtrace.WithMaxExportBatchSize(c.MaxExportBatchSize))
	}
	if c.MaxPacketSize > 0 {
		options = append(options, ecosystemtrace.WithMaxPacketSize(c.MaxPacketSize))
	}
	return
}

func setupCodes(cfg *config.Config, configurator remote.Configurator) {
	var c []*codes.Code
	c = append(c, cfg.Codes...)
	c = append(c, cfg.Metrics.Codes...)
	metricsCodes := trpccodes.ConvertMetricsCodes(cfg.Metrics.CodeTypeMapping, c)
	if len(metricsCodes) == 0 {
		return
	}

	codes.SetMapper(codes.New(codes.WithCodes(metricsCodes), codes.WithConfigurator(configurator)))
}

func getResourceAttributes(cfg *config.Config) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if len(cfg.Attributes) > 0 {
		for _, attr := range cfg.Attributes {
			attrs = append(attrs, attribute.String(attr.Key, attr.Value))
		}
	}
	return attrs
}

func setupFilters(cfg *config.Config) {
	filterOpts := func(o *traces.FilterOptions) {
		o.TraceLogMode = cfg.Logs.TraceLogMode
		o.TraceLogOption = cfg.Logs.TraceLogOption
		o.DisableTraceBody = cfg.Traces.DisableTraceBody
		o.DisableParentSampling = cfg.Traces.DisableParentSampling
	}
	logFilterOpts := func(o *logs.FilterOptions) {
		o.DisableRecovery = cfg.Logs.DisableRecovery
	}

	// override register filter with config options
	serverFilterChain := filter.ServerChain{traces.ServerFilter(filterOpts)}
	clientFilterChain := filter.ClientChain{traces.ClientFilter(filterOpts)}
	if cfg.Metrics.Enabled {
		if cfg.Metrics.DisableRPCMethodMapping {
			metric.SetCleanRPCMethodFunc(func(s string) string {
				return strings.ToValidUTF8(s, "")
			})
		}
		serverFilterChain = append(serverFilterChain, prometheus.ServerFilter(prometheus.WithServerFilterTraceConfig(
			cfg.Traces.EnableDeferredSample, cfg.Traces.DeferredSampleError, cfg.Traces.DeferredSampleSlowDuration)))
		clientFilterChain = append(clientFilterChain, prometheus.ClientFilter(prometheus.WithClientFilterTraceConfig(
			cfg.Traces.EnableDeferredSample, cfg.Traces.DeferredSampleError, cfg.Traces.DeferredSampleSlowDuration)))
	}
	serverFilterChain = append(serverFilterChain, logs.LogRecoveryFilter(logFilterOpts))
	serverFilter := serverFilterChain.Filter
	clientFilter := clientFilterChain.Filter
	filter.Register(consts.PluginName,
		serverFilter,
		clientFilter,
	)
	sf := &ServerFilter
	*sf = serverFilter
	cf := &ClientFilter
	*cf = clientFilter
}

// ParseConfig can be set by the user to override the config
var ParseConfig = func(configDec plugin.Decoder) (*config.Config, error) {
	cfg := &config.Config{}
	*cfg = config.DefaultConfig()
	if err := configDec.Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Register plugin and filter
func Register() {
	filter.Register(consts.PluginName, ServerFilter, ClientFilter)
	plugin.Register(consts.PluginName, &factory{})
}

var (
	ServerFilter = filter.ServerChain{traces.ServerFilter(), prometheus.ServerFilter(), logs.LogRecoveryFilter()}.Filter
	ClientFilter = filter.ClientChain{traces.ClientFilter(), prometheus.ClientFilter()}.Filter
)
