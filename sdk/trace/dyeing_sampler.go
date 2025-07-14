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

// Package trace trace
package trace

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/protocol/opentelemetry-ext/proto/sampler"
)

const (
	dyeingHeader       = "x-tps-tenantid"
	dyeingSamplerDebug = "DyeingSamplerDebug"
)

var (
	// ForceSamplerKey for debug
	ForceSamplerKey     = attribute.Key("trace.force.sample")
	traceStateDyeing    = attribute.Key("trace_dyeing")
	dyeingTraceState, _ = trace.TraceState{}.Insert(string(traceStateDyeing), "true")
)

var _ sdktrace.Sampler = &Sampler{}

// DefaultSampler return sdktrace.Sampler
func DefaultSampler() sdktrace.Sampler {
	return NewSampler("", SamplerConfig{
		Fraction:           0.001,
		SamplerServiceAddr: "",
		SyncInterval:       0,
	})
}

// SamplerOptions .
type SamplerOptions struct {
	// DefaultSamplingDecision Default sampling decision
	DefaultSamplingDecision sdktrace.SamplingDecision
}

// SamplerConfig sampler fractions config
type SamplerConfig struct {
	// Fraction default sampling fraction
	Fraction float64
	// SpecialFractions specified fractions
	SpecialFractions map[string]SpecialFraction
	// SamplerServiceAddr sampler service address
	SamplerServiceAddr string
	// SyncInterval sampler sync interval
	SyncInterval time.Duration
	// traceIDUpperBound sampler traceIDUpperBound
	traceIDUpperBound uint64
}

// defaultSamplerOptions .
var defaultSamplerOptions = SamplerOptions{DefaultSamplingDecision: sdktrace.Drop}

// SamplerOption .
type SamplerOption func(*SamplerOptions)

// MethodInfo .
type MethodInfo struct {
	CalleeService string
	CalleeMethod  string
}

// DefaultGetCalleeMethodInfo .
var DefaultGetCalleeMethodInfo GetCalleeMethodInfo = nil

// GetCalleeMethodInfo .
type GetCalleeMethodInfo func(context.Context) MethodInfo

// Sampler .
type Sampler struct {
	samplerConfig SamplerConfig
	description   string
	tenantID      string
	client        sampler.SamplerServiceClient
	sampledKvs    atomic.Value // map[string]map[string]bool
	debug         bool
	opt           SamplerOptions
}

// NewSampler .
func NewSampler(
	tpsTenantID string,
	samplerConfig SamplerConfig,
	opts ...SamplerOption,
) sdktrace.Sampler {
	ws := &Sampler{
		samplerConfig: getSamplerConfig(samplerConfig),
		tenantID:      tpsTenantID,
		description: fmt.Sprintf("TpsSampler{fraction=%g,tenantID=%s}",
			samplerConfig.Fraction, tpsTenantID),
		sampledKvs: atomic.Value{},
		opt:        defaultSamplerOptions,
	}
	for _, v := range opts {
		v(&ws.opt)
	}
	if ws.samplerConfig.SamplerServiceAddr != "" {
		ws.sampledKvs.Store(make(map[string]map[string]bool))
		go ws.updateDyeingMetadataDaemon()
	}

	if ws.isDebugEnabled() {
		log.Printf("[opentelemetry][I] %s : enabled", dyeingSamplerDebug)
		ws.debug = true
	}
	return ws
}

func (ws *Sampler) isDebugEnabled() bool {
	if debug := os.Getenv(dyeingSamplerDebug); debug == "true" {
		return true
	}
	return false
}

func (ws *Sampler) updateDyeingMetadataDaemon() {
	for {
		ws.updateDyeingMetadata()
		time.Sleep(ws.samplerConfig.SyncInterval)
	}
}

func (ws *Sampler) updateDyeingMetadata() {
	if ws.client == nil {
		cc, err := grpc.Dial(ws.samplerConfig.SamplerServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return
		}
		ws.client = sampler.NewSamplerServiceClient(cc)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		dyeingHeader: ws.tenantID,
	}))
	rsp, err := ws.client.GetSampler(ctx, &sampler.GetSamplerRequest{}, grpc.WaitForReady(true))
	if err != nil {
		if ws.debug {
			log.Printf("[opentelemetry][E] Get sampler err:%v", err)
		}
		return
	}
	sampledKvs := make(map[string]map[string]bool)
	for _, v := range rsp.GetAttributes() {
		sampledKv := sampledKvs[v.Key]
		if sampledKv == nil {
			sampledKv = make(map[string]bool)
			sampledKvs[v.Key] = sampledKv
		}
		for _, vv := range v.GetValues() {
			sampledKv[vv] = true
		}
	}
	if ws.debug {
		log.Printf("[opentelemetry][I] sampledKvs:%+v", sampledKvs)
	}
	ws.sampledKvs.Store(sampledKvs)
}

// ShouldSample sampler ShouldSample implementation
func (ws *Sampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if psc := trace.SpanContextFromContext(p.ParentContext); psc.IsSampled() {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	if ws.samplerConfig.SamplerServiceAddr != "" {
		sampledKvs, ok := ws.sampledKvs.Load().(map[string]map[string]bool)
		for _, attr := range p.Attributes {
			key := string(attr.Key)
			if attr.Key == ForceSamplerKey && attr.Value.Emit() != "" {
				return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample,
					Tracestate: dyeingTraceState}
			}

			if ok {
				if values, ok2 := sampledKvs[key]; ok2 && values[attr.Value.Emit()] {
					return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample,
						Tracestate: dyeingTraceState}
				}
			}
		}
	}

	return ws.shouldSample(p)
}

// Description return the description of Sampler
func (ws *Sampler) Description() string {
	return ws.description
}

func (ws *Sampler) shouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	x := binary.BigEndian.Uint64(p.TraceID[0:8]) >> 1

	traceIDUpperBound := getSamplerTraceIDUpperBound(p.ParentContext, ws.samplerConfig)
	if x < traceIDUpperBound {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	}
	return sdktrace.SamplingResult{Decision: ws.opt.DefaultSamplingDecision}
}

func getSamplerTraceIDUpperBound(ctx context.Context, config SamplerConfig) uint64 {
	if DefaultGetCalleeMethodInfo == nil || config.SpecialFractions == nil {
		return config.traceIDUpperBound
	}

	methodInfo := DefaultGetCalleeMethodInfo(ctx)
	serviceFraction, ok := config.SpecialFractions[methodInfo.CalleeService]
	if !ok {
		return config.traceIDUpperBound
	}

	if serviceFraction.Methods == nil {
		return serviceFraction.defaultTraceIDUpperBound
	}

	methodFraction, ok := serviceFraction.Methods[methodInfo.CalleeMethod]
	if !ok {
		return serviceFraction.defaultTraceIDUpperBound
	}
	return methodFraction.traceIDUpperBound
}
