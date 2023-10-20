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

package zaplogs

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"trpc.group/trpc-go/trpc-go/errs"
)

var (
	defaultOptions = &options{
		levelFunc:    DefaultCodeToLevel,
		shouldLog:    DefaultDeciderMethod,
		codeFunc:     DefaultErrorToCode,
		durationFunc: DefaultDurationToField,
		messageFunc:  DefaultMessageProducer,
	}
)

var DefaultDurationToField = DurationToTimeMillisField

// DurationToTimeMillisField convert time.Duration to zap Field
func DurationToTimeMillisField(duration time.Duration) zapcore.Field {
	return zap.Float32("trpc.time_ms", durationToMilliseconds(duration))
}

func durationToMilliseconds(duration time.Duration) float32 {
	return float32(duration.Nanoseconds()/1000) / 1000
}

// DefaultDeciderMethod ...
func DefaultDeciderMethod(fullMethodName string, err error) bool {
	return true
}

// DefaultErrorToCode default error code function
func DefaultErrorToCode(err error) int {
	return int(errs.Code(err))
}

// DefaultMessageProducer default message producer
func DefaultMessageProducer(ctx context.Context, msg string, level zapcore.Level,
	code int, err error, duration zapcore.Field) {
	FromContext(ctx).Check(level, msg).Write(
		zap.Error(err),
		zap.Int("trpc.code", code),
		duration,
	)
}

// WithDecider customizes the function for deciding if the gRPC interceptor logs should log.
func WithDecider(f Decider) Option {
	return func(o *options) {
		o.shouldLog = f
	}
}

// WithLevels customizes the function for mapping gRPC return codes and interceptor log level statements.
func WithLevels(f CodeToLevel) Option {
	return func(o *options) {
		o.levelFunc = f
	}
}

// WithCodes customizes the function for mapping errors to error codes.
func WithCodes(f ErrorToCode) Option {
	return func(o *options) {
		o.codeFunc = f
	}
}

// WithDurationField customizes the function for mapping request durations to Zap fields.
func WithDurationField(f DurationToField) Option {
	return func(o *options) {
		o.durationFunc = f
	}
}

// WithMessageProducer customizes the function for message formation.
func WithMessageProducer(f MessageProducer) Option {
	return func(o *options) {
		o.messageFunc = f
	}
}

func evaluateClientOpt(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	optCopy.levelFunc = DefaultCodeToLevel
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

func evaluateServerOpt(opts []Option) *options {
	optCopy := &options{}
	*optCopy = *defaultOptions
	optCopy.levelFunc = DefaultCodeToLevel
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

type Option func(*options)

type options struct {
	levelFunc    CodeToLevel
	shouldLog    Decider
	codeFunc     ErrorToCode
	durationFunc DurationToField
	messageFunc  MessageProducer
}

type CodeToLevel func(code int) zapcore.Level

type Decider func(fullMethodName string, err error) bool

type ErrorToCode func(err error) int

type DurationToField func(duration time.Duration) zapcore.Field

type MessageProducer func(ctx context.Context, msg string, level zapcore.Level,
	code int, err error, duration zapcore.Field)

// DefaultCodeToLevel code转换为zap level
func DefaultCodeToLevel(code int) zapcore.Level {
	switch code {
	case 0:
		return zap.InfoLevel
	default:
		return zap.ErrorLevel
	}
}
