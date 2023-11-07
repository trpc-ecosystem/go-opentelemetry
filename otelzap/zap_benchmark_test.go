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

package otelzap

import (
	"context"
	"testing"
	"time"

	sdklog "trpc.group/trpc-go/trpc-opentelemetry/sdk/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	commonproto "go.opentelemetry.io/proto/otlp/common/v1"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
)

// BenchmarkBatchCore
// before:                BenchmarkBatchCore-12                     475426              2770 ns/op
// after jsoniter parser: BenchmarkBatchCore-12                     615273              1916 ns/op
func BenchmarkBatchCore(b *testing.B) {
	bw := NewBatchWriteSyncer(&nopExporter{}, nil)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		bw, zapcore.DebugLevel)
	logger := zap.New(core)
	for n := 0; n < b.N; n++ {
		logger.Info("maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined")
	}
}

func BenchmarkProtoCore(b *testing.B) {
	core := zapcore.NewCore(NewEncoder(zap.NewProductionEncoderConfig()),
		&protoWriteSyncer{}, zapcore.DebugLevel)
	logger := zap.New(core)
	for n := 0; n < b.N; n++ {
		logger.Info("maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined")
	}
}

func BenchmarkJsonWriteSyncer_Write(b *testing.B) {
	jw := &jsonWriteSyncer{}
	data := []byte(`{\"level\":\"debug\",\"ts\":1603700525.1244118,
\"caller\":\"maxprocs/maxprocs.go:47\",\"msg\":\"maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined\"}`)
	for n := 0; n < b.N; n++ {
		_, _ = jw.Write(data)
	}
}

func BenchmarkProtoWriteSyncer_Write(b *testing.B) {
	l := &logsproto.LogRecord{
		SeverityText: "debug",
		TimeUnixNano: uint64(time.Now().UnixNano()),
		Body: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
			StringValue: `maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined`},
		},
	}
	data, err := proto.Marshal(l)
	if err != nil {
		b.Fatal(err)
	}
	pw := &protoWriteSyncer{}
	for n := 0; n < b.N; n++ {
		_, _ = pw.Write(data)
	}
}

var _ zapcore.WriteSyncer = (*protoWriteSyncer)(nil)

type protoWriteSyncer struct {
}

func (pw *protoWriteSyncer) Write(p []byte) (n int, err error) {
	l := &logsproto.LogRecord{}
	err = proto.Unmarshal(p, l)
	if err != nil {
		return 0, nil
	}
	return len(p), nil
}

func (pw *protoWriteSyncer) Sync() error {
	return nil
}

var _ sdklog.Exporter = (*nopExporter)(nil)

type nopExporter struct {
}

func (n nopExporter) Shutdown(_ context.Context) error {
	return nil
}

func (n nopExporter) ExportLogs(ctx context.Context, logs []*logsproto.ResourceLogs) error {
	return nil
}
