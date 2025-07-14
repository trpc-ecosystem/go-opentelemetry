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

// Package traces
package traces

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"

	"trpc.group/trpc-go/trpc-opentelemetry/api"
	"trpc.group/trpc-go/trpc-opentelemetry/config"
	ecocodes "trpc.group/trpc-go/trpc-opentelemetry/config/codes"
	trpccodes "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/codes"
	"trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/logs"
	trpcsemconv "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/semconv"
	oteladmin "trpc.group/trpc-go/trpc-opentelemetry/pkg/admin"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

var (
	defaultTracer     trace.Tracer
	defaultTracerOnce sync.Once
)

// Init trace filter
func Init() {
	admin.HandleFunc("/cmds/disabletrace", oteladmin.DisableTrace)
	admin.HandleFunc("/cmds/enabletrace", oteladmin.EnableTrace)
	admin.HandleFunc("/cmds/tracestatus", oteladmin.TraceStatus)
}

func getDefaultTracer() trace.Tracer {
	defaultTracerOnce.Do(func() {
		defaultTracer = otel.Tracer("")
	})
	return defaultTracer
}

// FilterOptions FilterOptions
type FilterOptions struct {
	// TraceLogMode trace log mode
	TraceLogMode config.LogMode
	// TraceLogOption trace_log option
	TraceLogOption config.TraceLogOption
	// DisableTraceBody disable req/rsp event log
	DisableTraceBody bool
	// DisableParentSampling ignore parent sampling
	DisableParentSampling bool
}

// FilterOption filter option
type FilterOption func(*FilterOptions)

var defaultFilterOptions = FilterOptions{
	TraceLogMode:          config.LogModeOneLine,
	DisableTraceBody:      false,
	DisableParentSampling: false,
}

// ServerFilter opentelemetry server filter in trpc
func ServerFilter(opts ...FilterOption) filter.ServerFilter {
	opt := defaultFilterOptions
	for _, v := range opts {
		v(&opt)
	}
	return func(ctx context.Context, req interface{}, f filter.ServerHandleFunc) (rsp interface{}, err error) {
		if oteladmin.TraceDisabled() {
			return f(ctx, req)
		}

		start := time.Now()
		msg := trpc.Message(ctx)
		md := msg.ServerMetaData()
		if md == nil {
			md = codec.MetaData{}
		}

		ctx, span := startServerSpan(ctx, req, msg, md, opt)
		defer span.End()

		log.WithContextFields(ctx, "traceID", span.SpanContext().TraceID().String(),
			"spanID", span.SpanContext().SpanID().String(),
			"sampled", strconv.FormatBool(span.SpanContext().IsSampled()))

		receivedDeadline := getDeadline(ctx)
		receivedTime := time.Now()

		rsp, err = f(ctx, req)

		sentDeadline := getDeadline(ctx)
		sentTime := time.Now()

		var code int
		codeStr, err1 := trpccodes.GetDefaultGetCodeFunc()(ctx, rsp, err)
		if c, e := strconv.Atoi(codeStr); e == nil {
			code = c
		}
		flow := buildFlowLog(msg, trace.SpanKindServer)
		handleError(code, err1, span, flow)
		if needToTraceBody(span, opt, err1) {
			flow.Request.Body = addEvent(ctx, req, otelsemconv.MessageTypeReceived, receivedDeadline, receivedTime)
			flow.Response.Body = addEvent(ctx, rsp, otelsemconv.MessageTypeSent, sentDeadline, sentTime)
		}

		span.SetAttributes(DefaultAttributesAfterServerHandle(ctx, rsp)...)
		flow.Cost = time.Since(start).String()
		doFlowLog(ctx, flow, opt)
		return rsp, err
	}
}

func buildFlowLog(msg codec.Msg, kind trace.SpanKind) *logs.FlowLog {
	var sourceAddr, targetAddr string
	if msg.RemoteAddr() != nil {
		targetAddr = msg.RemoteAddr().String()
	}
	if msg.LocalAddr() != nil {
		sourceAddr = msg.LocalAddr().String()
	}

	if kind == trace.SpanKindServer {
		sourceAddr, targetAddr = targetAddr, sourceAddr
	}

	flow := &logs.FlowLog{
		Kind: logs.FlowKind(kind),
		Source: logs.Service{
			Name:      msg.CallerServiceName(),
			Method:    msg.CallerMethod(),
			Namespace: msg.EnvName(),
			Address:   sourceAddr,
		},
		Target: logs.Service{
			Name:      msg.CalleeServiceName(),
			Method:    msg.CalleeMethod(),
			Address:   targetAddr,
			Namespace: msg.EnvName(),
		},
	}

	return flow
}

func startServerSpan(ctx context.Context,
	req interface{}, msg codec.Msg, md codec.MetaData, opt FilterOptions) (context.Context, trace.Span) {
	suppliers := GetTextMapCarriers(md, msg)
	ctx = otel.GetTextMapPropagator().Extract(ctx, suppliers)
	spanContext := trace.SpanContextFromContext(ctx)

	spanKind := trace.SpanKindServer
	if kind, ok := msg.CommonMeta()[SpanKindServer].(trace.SpanKind); ok {
		spanKind = kind
	}

	spanStartOptions := []trace.SpanStartOption{
		trace.WithSpanKind(spanKind),
		trace.WithAttributes(peerInfo(msg.RemoteAddr())...),
		trace.WithAttributes(hostInfo(msg.LocalAddr())...),
		trace.WithAttributes(trpcsemconv.CallerServiceKey.String(msg.CallerServiceName())),
		trace.WithAttributes(trpcsemconv.CallerMethodKey.String(msg.CallerMethod())),
		trace.WithAttributes(trpcsemconv.CalleeServiceKey.String(msg.CalleeServiceName())),
		trace.WithAttributes(trpcsemconv.CalleeMethodKey.String(msg.CalleeMethod())),
		trace.WithAttributes(trpcsemconv.NamespaceKey.String(msg.Namespace()),
			trpcsemconv.EnvNameKey.String(msg.EnvName())),
		trace.WithAttributes(fromTRPCDyeingKey(msg.DyeingKey())...),
		trace.WithAttributes(forceSampleFromMetadata(msg.ServerMetaData())...),
		trace.WithAttributes(DefaultTraceAttributesFunc(ctx, req)...),
		trace.WithAttributes(attribute.String(api.BaggageHeader, baggage.FromContext(ctx).String())),
	}

	if opt.DisableParentSampling {
		spanContext = spanContext.WithTraceFlags(spanContext.TraceFlags() &^ trace.FlagsSampled)
	}

	return getDefaultTracer().Start(
		trace.ContextWithRemoteSpanContext(ctx, spanContext),
		msg.ServerRPCName(),
		spanStartOptions...)
}

func needToTraceBody(span trace.Span, opt FilterOptions, err error) bool {
	if opt.DisableTraceBody {
		return false
	}

	if span.SpanContext().IsSampled() {
		return true
	}
	return opt.TraceLogMode != config.LogModeDisable || err != nil
}

func handleError(errCode int, err error, span trace.Span, flow *logs.FlowLog) {
	code, msg, errType := getErrCode(errCode, err)
	calleeService, calleeMethod := flow.Target.Name, flow.Target.Method
	codeType := ecocodes.CodeMapping(strconv.Itoa(code), calleeService, calleeMethod)
	if codeType.Type != ecocodes.CodeTypeSuccess.String() {
		span.SetStatus(codes.Error, msg)
	} else {
		span.SetStatus(codes.Ok, msg)
	}

	span.SetAttributes(
		trpcsemconv.StatusCode.Int64(int64(code)),
		trpcsemconv.StatusMsg.String(msg),
		trpcsemconv.StatusType.Int(errType),
	)
	flow.Status = logs.Status{
		Code:    int32(code),
		Message: msg,
		Type:    toErrorType(errType),
	}
}

func getErrCode(errCode int, err error) (int, string, int) {
	var (
		code, errType int
		msg           string
	)

	if err == nil {
		return errCode, msg, errType
	}

	if e, ok := err.(*errs.Error); ok {
		code, msg, errType = int(e.Code), e.Msg, e.Type
	} else {
		code, msg = int(errs.RetUnknown), err.Error()
	}
	return code, msg, errType
}

// ClientFilter client filter in trpc
func ClientFilter(opts ...FilterOption) filter.ClientFilter {
	opt := defaultFilterOptions
	for _, v := range opts {
		v(&opt)
	}
	return func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
		if oteladmin.TraceDisabled() {
			return f(ctx, req, rsp)
		}

		start := time.Now()
		msg := trpc.Message(ctx)
		md := msg.ClientMetaData()
		if md == nil {
			md = codec.MetaData{}
		}
		suppliers := GetTextMapCarriers(md, msg)
		ctx, span := startClientSpan(ctx, req, msg)
		defer span.End()

		otel.GetTextMapPropagator().Inject(ctx, suppliers)
		msg.WithClientMetaData(md)

		sentDeadline := getDeadline(ctx)
		sentTime := time.Now()
		err := f(ctx, req, rsp)
		receivedDeadline := getDeadline(ctx)
		receivedTime := time.Now()
		var code int
		codeStr, err1 := trpccodes.GetDefaultGetCodeFunc()(ctx, rsp, err)
		if c, e := strconv.Atoi(codeStr); e == nil {
			code = c
		}
		flow := buildFlowLog(msg, trace.SpanKindClient)
		handleError(code, err1, span, flow)
		if needToTraceBody(span, opt, err1) {
			flow.Request.Body = addEvent(ctx, req, otelsemconv.MessageTypeSent, sentDeadline, sentTime)
			flow.Response.Body = addEvent(ctx, rsp, otelsemconv.MessageTypeReceived, receivedDeadline, receivedTime)
		}
		handleComponent(msg, span) // add component tags
		span.SetAttributes(DefaultAttributesAfterClientHandle(ctx, rsp)...)
		span.SetAttributes(peerInfo(msg.RemoteAddr())...)
		span.SetAttributes(hostInfo(msg.LocalAddr())...)
		flow.Cost = time.Since(start).String()

		doFlowLog(ctx, flow, opt)
		return err
	}
}

func startClientSpan(ctx context.Context, req interface{}, msg codec.Msg) (context.Context, trace.Span) {
	var spanKind = trace.SpanKindClient
	if kind, ok := msg.CommonMeta()[SpanKindClient].(trace.SpanKind); ok {
		spanKind = kind
	}
	return getDefaultTracer().Start(ctx,
		// msg.ClientRPCName(),
		msg.CalleeServiceName()+"/"+strings.TrimLeft(msg.CalleeMethod(), "/"),
		trace.WithSpanKind(spanKind),
		trace.WithAttributes(trpcsemconv.CallerServiceKey.String(msg.CallerServiceName())),
		trace.WithAttributes(trpcsemconv.CallerMethodKey.String(metric.CleanRPCMethod(msg.CallerMethod()))),
		trace.WithAttributes(trpcsemconv.CalleeServiceKey.String(msg.CalleeServiceName())),
		trace.WithAttributes(trpcsemconv.CalleeMethodKey.String(metric.CleanRPCMethod(msg.CalleeMethod()))),
		trace.WithAttributes(trpcsemconv.NamespaceKey.String(msg.Namespace()),
			trpcsemconv.EnvNameKey.String(msg.EnvName())),
		trace.WithAttributes(fromTRPCDyeingKey(msg.DyeingKey())...),
		trace.WithAttributes(forceSampleFromMetadata(msg.ClientMetaData())...),
		trace.WithAttributes(DefaultTraceAttributesFunc(ctx, req)...),
		trace.WithAttributes(attribute.String(api.BaggageHeader, baggage.FromContext(ctx).String())))
}

func toErrorType(t int) string {
	switch t {
	case errs.ErrorTypeBusiness:
		return "business"
	case errs.ErrorTypeCalleeFramework:
		return "callee_framework"
	case errs.ErrorTypeFramework:
		return "framework"
	default:
		return ""
	}
}

func getDeadline(ctx context.Context) time.Duration {
	var t time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		t = time.Until(deadline)
	}
	return t
}
