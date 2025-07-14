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

package trace

import "time"

// SpecialFraction special fraction config
type SpecialFraction struct {
	DefaultFraction float64
	Methods         map[string]MethodFraction
	// defaultTraceIDUpperBound The upper limit of traceID corresponding to the default sampling rate,
	// calculated during initialization and not exposed to the outside world.
	defaultTraceIDUpperBound uint64
}

// MethodFraction method special fraction
type MethodFraction struct {
	// Fraction Specified sampling rate
	Fraction float64
	// traceIDUpperBound The upper limit of traceID corresponding to the sampling rate is calculated
	// during initialization and is not exposed to the outside world.
	traceIDUpperBound uint64
}

func getSamplerConfig(config SamplerConfig) SamplerConfig {
	return SamplerConfig{
		Fraction:           config.Fraction,
		SpecialFractions:   getSpecialFraction(config.SpecialFractions),
		SamplerServiceAddr: config.SamplerServiceAddr,
		SyncInterval:       getSamplerSyncInterval(config.SyncInterval),
		traceIDUpperBound:  getTraceIDUpperBound(config.Fraction),
	}
}

func getSamplerSyncInterval(syncInterval time.Duration) time.Duration {
	if syncInterval != 0 {
		return syncInterval
	}
	return time.Second * 10
}

func getSpecialFraction(fractions map[string]SpecialFraction) map[string]SpecialFraction {
	result := make(map[string]SpecialFraction, len(fractions))
	for k, v := range fractions {
		result[k] = SpecialFraction{
			DefaultFraction:          v.DefaultFraction,
			Methods:                  getMethodsSpecialFraction(v.Methods),
			defaultTraceIDUpperBound: getTraceIDUpperBound(v.DefaultFraction),
		}
	}
	return result
}

func getMethodsSpecialFraction(methodsFraction map[string]MethodFraction) map[string]MethodFraction {
	result := make(map[string]MethodFraction, len(methodsFraction))
	for k, v := range methodsFraction {
		result[k] = MethodFraction{
			Fraction:          v.Fraction,
			traceIDUpperBound: getTraceIDUpperBound(v.Fraction),
		}
	}
	return result
}

func getTraceIDUpperBound(fraction float64) uint64 {
	if fraction >= 1 {
		fraction = 1
	}
	if fraction <= 0 {
		fraction = 0
	}
	return uint64(fraction * (1 << 63))
}
