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
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	v1proto "github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	commonproto "go.opentelemetry.io/proto/otlp/common/v1"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
	resourceproto "go.opentelemetry.io/proto/otlp/resource/v1"
	"go.uber.org/zap/zapcore"

	"trpc-system/go-opentelemetry/pkg/metrics"
	sdklog "trpc-system/go-opentelemetry/sdk/log"
)

const (
	DefaultMaxQueueSize         = 2048
	DefaultBatchTimeout         = 5000 * time.Millisecond
	DefaultMaxExportBatchSize   = 512
	DefaultBlockOnQueueFull     = false
	DefaultMaxBatchedPacketSize = 2097152 // need to be reported when accumulated log size reach 2MB
)

var _ zapcore.WriteSyncer = (*BatchWriteSyncer)(nil)

var (
	failedExportCounter      = metrics.BatchProcessCounter.WithLabelValues("failed", "logs")
	succeededExportCounter   = metrics.BatchProcessCounter.WithLabelValues("success", "logs")
	batchByCountCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchcount")
	batchByPacketSizeCounter = metrics.BatchProcessCounter.WithLabelValues("batched", "packetsize")
	batchByTimerCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchtimer")
	enqueueCounter           = metrics.BatchProcessCounter.WithLabelValues("enqueue", "logs")
	dropCounter              = metrics.BatchProcessCounter.WithLabelValues("dropped", "logs")
)

// BatchWriteSyncer implement zapcore.WriteSyncer
type BatchWriteSyncer struct {
	exporter    sdklog.Exporter
	opt         *BatchSyncerOptions
	queue       chan *logsproto.ScopeLogs
	dropped     uint32
	batch       []*logsproto.ScopeLogs
	timer       *time.Timer
	rs          *resource.Resource
	stopCh      chan struct{}
	rspb        *resourceproto.Resource
	batchedSize int
}

const (
	fieldSampled = "sampled"
	fieldLevel   = "level"
	fieldTraceID = "traceID"
	fieldSpanID  = "spanID"
	trueString   = "true"
)

// logsField ...
var logsField = map[string]bool{
	"msg":     true,
	"traceID": true,
	"spanID":  true,
	"sampled": true,
	"caller":  true,
	"level":   true,
	"ts":      true,
}

// isTagsField ...
func isTagsField(key string) bool {
	return !logsField[key]
}

func isValidRecord(l *logsproto.LogRecord) bool {
	if len(l.GetTraceId()) != 16 && len(l.GetTraceId()) != 0 {
		return false
	}
	if len(l.GetSpanId()) != 8 && len(l.GetSpanId()) != 0 {
		return false
	}

	return true
}

// Deprecated.
// convertToRecord just for unit test and benchmark.
func convertToRecord(raw map[string]interface{}) *logsproto.LogRecord {
	l := &logsproto.LogRecord{}
	msg, ok := raw["msg"]
	if !ok {
		return l
	}
	msgs, ok := msg.(string)
	if !ok {
		return l
	}
	l.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{
			StringValue: msgs,
		},
	}
	traceID, ok := raw["traceID"]
	if ok {
		var traceIDStr string
		traceIDStr, _ = traceID.(string)
		l.TraceId, _ = hex.DecodeString(traceIDStr)
	}
	spanID, ok := raw["spanID"]
	if ok {
		var spanIDStr string
		spanIDStr, _ = spanID.(string)
		l.SpanId, _ = hex.DecodeString(spanIDStr)
	}
	sampledRaw, ok := raw[fieldSampled]
	if ok {
		sampledStr, _ := sampledRaw.(string)
		sampled := (sampledStr == trueString)
		l.Attributes = append(l.Attributes, &commonproto.KeyValue{
			Key: fieldSampled,
			Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_BoolValue{
				BoolValue: sampled,
			}},
		})
	}
	lineRaw, ok := raw["caller"]
	if ok {
		var line string
		line, ok = lineRaw.(string)
		if ok {
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: "line",
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: line,
				}},
			})
		}
	}
	levelRaw, ok := raw[fieldLevel]
	if ok {
		var level string
		level, ok = levelRaw.(string)
		if ok {
			l.SeverityText = level
		}
	}

	tsRaw, ok := raw["ts"]
	if ok {
		ts, ok := tsRaw.(float64)
		if ok {
			l.TimeUnixNano = uint64(ts * float64(time.Second))
		}
	}

	for k, v := range raw {
		if vv, ok := v.(string); ok && isTagsField(k) {
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: k,
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: vv,
				}},
			})
		}
	}
	return l
}

func convertToRecordV2(iter *jsoniter.Iterator) (*logsproto.LogRecord, error) {
	l := &logsproto.LogRecord{}
	iter.ReadObjectCB(func(iterator *jsoniter.Iterator, f string) bool {
		switch f {
		case fieldSampled:
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: fieldSampled,
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_BoolValue{
					BoolValue: iter.ReadString() == trueString,
				}},
			})
		case fieldLevel:
			l.SeverityText = iter.ReadString()
		case "msg":
			l.Body = &commonproto.AnyValue{
				Value: &commonproto.AnyValue_StringValue{
					StringValue: iter.ReadString(),
				},
			}
		case fieldTraceID:
			l.TraceId, _ = hex.DecodeString(iter.ReadString())
		case fieldSpanID:
			l.SpanId, _ = hex.DecodeString(iter.ReadString())
		case "caller":
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: "line",
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: iter.ReadString(),
				}},
			})
		case "ts":
			l.TimeUnixNano = uint64(iter.ReadFloat64() * float64(time.Second))
		default:
			// support log field with any type
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: f,
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: iter.ReadAny().ToString(),
				}},
			})
		}
		return true
	})
	return l, iter.Error
}

// Write implement Write interface
func (bp *BatchWriteSyncer) Write(p []byte) (n int, err error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(p)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	keep, err := bp.shouldKeep(iter, p)
	if err != nil {
		return 0, err
	}
	if !keep {
		return 0, nil
	}
	l, err := convertToRecordV2(iter)
	if err != nil {
		return 0, err
	}
	if ok := isValidRecord(l); !ok {
		return 0, nil
	}
	metrics.LogsLevelTotal.WithLabelValues(l.SeverityText).Inc()
	sl := &logsproto.ScopeLogs{
		LogRecords: []*logsproto.LogRecord{l},
	}
	bp.Enqueue(sl, 1)
	return len(p), nil
}

func readSampledAndLevel(iter *jsoniter.Iterator, p []byte) (bool, string, error) {
	var (
		sampled    bool
		level      string
		readFields int
	)
	iter.ReadObjectCB(func(iterator *jsoniter.Iterator, f string) bool {
		switch f {
		case fieldSampled:
			sampled = iter.ReadString() == trueString
			readFields++
			return true
		case fieldLevel:
			level = iter.ReadString()
			readFields++
			return true
		default:
			if readFields > 2 {
				return false
			}
			iter.Skip()
			return true
		}
	})
	iter.ResetBytes(p)
	if err := iter.Error; err != nil {
		return false, "", err
	}
	return sampled, level, nil
}

func levelGreaterThanError(level string) bool {
	switch level {
	case "error", "fatal":
		return true
	default:
	}
	return false
}

func (bp *BatchWriteSyncer) shouldKeep(iter *jsoniter.Iterator, p []byte) (bool, error) {
	if bp.opt.EnableSampler { // only report log sampled
		sampled, level, err := readSampledAndLevel(iter, p)
		if err != nil {
			return false, err
		}
		if sampled {
			return true, nil
		}
		if bp.opt.EnableSamplerError {
			if levelGreaterThanError(level) {
				return true, nil
			}
		}
		return false, nil
	}

	return true, nil
}

// Sync implement Sync interface
func (bp *BatchWriteSyncer) Sync() error {
	bp.export()
	return nil
}

// NewBatchWriteSyncer return BatchWriteSyncer
func NewBatchWriteSyncer(exporter sdklog.Exporter, rs *resource.Resource, opts ...BatchSyncerOption) *BatchWriteSyncer {
	opt := &BatchSyncerOptions{
		MaxQueueSize:       DefaultMaxQueueSize,
		BatchTimeout:       DefaultBatchTimeout,
		MaxExportBatchSize: DefaultMaxExportBatchSize,
		BlockOnQueueFull:   DefaultBlockOnQueueFull,
		MaxPacketSize:      DefaultMaxBatchedPacketSize,
	}

	for _, o := range opts {
		o(opt)
	}

	bp := &BatchWriteSyncer{
		opt:      opt,
		rs:       rs,
		exporter: exporter,
		batch:    make([]*logsproto.ScopeLogs, 0, opt.MaxExportBatchSize),
		queue:    make(chan *logsproto.ScopeLogs, opt.MaxQueueSize),
		stopCh:   make(chan struct{}),
		timer:    time.NewTimer(opt.BatchTimeout),
	}
	if rs.Len() != 0 {
		rspb := &resourceproto.Resource{}
		for _, kv := range rs.Attributes() {
			rspb.Attributes = append(rspb.Attributes, &commonproto.KeyValue{
				Key: string(kv.Key),
				Value: &commonproto.AnyValue{
					Value: &commonproto.AnyValue_StringValue{StringValue: kv.Value.Emit()},
				},
			})
		}
		bp.rspb = rspb
	}

	go func() {
		bp.processQueue()
		bp.drainQueue()
	}()

	return bp
}

// Enqueue enqueue ResourceLogs to bp.queue
func (bp *BatchWriteSyncer) Enqueue(sl *logsproto.ScopeLogs, size int) {
	enqueueCounter.Add(float64(size))
	select {
	case <-bp.stopCh:
		return
	default:
	}

	if bp.opt.BlockOnQueueFull {
		bp.queue <- sl
		return
	}

	select {
	case bp.queue <- sl:
	default:
		dropCounter.Add(float64(size))
		otel.Handle(errors.New("opentelemetry export logs dropped"))
		atomic.AddUint32(&bp.dropped, 1)
	}
}

func (bp *BatchWriteSyncer) processQueue() {
	defer bp.timer.Stop()

	for {
		select {
		case <-bp.stopCh:
			return
		case <-bp.timer.C:
			batchByTimerCounter.Inc()
			bp.export()
		case ld := <-bp.queue:
			bp.batch = append(bp.batch, ld)
			bp.batchedSize += calcLogSize(ld)
			shouldExport := bp.shouldProcessInBatch()
			if shouldExport {
				if !bp.timer.Stop() {
					<-bp.timer.C
				}
				bp.export()
			}
		}
	}
}

func (bp *BatchWriteSyncer) export() {
	bp.timer.Reset(bp.opt.BatchTimeout)
	if len(bp.batch) > 0 {
		size := len(bp.batch)
		logs := []*logsproto.ResourceLogs{
			{
				Resource:  bp.rspb,
				ScopeLogs: bp.batch,
			},
		}
		err := bp.exporter.ExportLogs(context.Background(), logs)
		bp.batch = bp.batch[:0]
		bp.batchedSize = 0
		if err != nil {
			otel.Handle(fmt.Errorf("opentelemetry export logs failed: %v", err))
			failedExportCounter.Add(float64(size))
		} else {
			succeededExportCounter.Add(float64(size))
		}
	}
}

func (bp *BatchWriteSyncer) drainQueue() {
	for {
		select {
		case ld := <-bp.queue:
			if ld == nil {
				bp.export()
				return
			}
			bp.batch = append(bp.batch, ld)
			bp.batchedSize += calcLogSize(ld)
			shouldExport := bp.shouldProcessInBatch()
			if shouldExport {
				bp.export()
			}
		default:
			close(bp.queue)
		}
	}
}

// shouldProcessInBatch determines whether to export in batches
func (bp *BatchWriteSyncer) shouldProcessInBatch() bool {
	if len(bp.batch) == bp.opt.MaxExportBatchSize {
		batchByCountCounter.Inc()
		return true
	}
	if bp.batchedSize >= bp.opt.MaxPacketSize {
		batchByPacketSizeCounter.Inc()
		return true
	}
	return false
}

// calcLogSize calculates the packet size of a ScopeLogs
func calcLogSize(sl *logsproto.ScopeLogs) int {
	return v1proto.Size(sl)
}

// BatchSyncerOption apply changes to internalOptions.
type BatchSyncerOption func(o *BatchSyncerOptions)

// BatchSyncerOptions defines the configuration for the various elements of BatchSyncer
type BatchSyncerOptions struct {
	// MaxQueueSize is the maximum queue size to buffer spans for delayed processing. If the
	// queue gets full it drops the spans. Use BlockOnQueueFull to change this behavior.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available spans when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// MaxExportBatchSize is the maximum number of spans to process in a single batch.
	// If there are more than one batch worth of spans then it processes multiple batches
	// of spans one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int

	// BlockOnQueueFull blocks onEnd() and onStart() method if the queue is full
	// AND if BlockOnQueueFull is set to true.
	// Blocking option should be used carefully as it can severely affect the performance of an
	// application.
	BlockOnQueueFull bool

	// EnableSampler
	EnableSampler bool

	// EnableSamplerError
	EnableSamplerError bool

	// MaxPacketSize is the maximum number of packet size that will forcefully trigger a batch process.
	// The default value of MaxPacketSize is 2M (in bytes) .
	MaxPacketSize int
}

// WithMaxPacketSize WithMaxPacketSize
func WithMaxPacketSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxPacketSize = size
	}
}

// WithMaxQueueSize return BatchSyncerOption which to set MaxQueueSize
func WithMaxQueueSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize return BatchSyncerOption which to set  MaxExportBatchSize
func WithMaxExportBatchSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout return BatchSyncerOption which to set BatchTimeout
func WithBatchTimeout(delay time.Duration) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.BatchTimeout = delay
	}
}

// WithBlocking return BatchSyncerOption which to set BlockOnQueueFull
func WithBlocking() BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.BlockOnQueueFull = true
	}
}

// WithEnableSampler set sampler option
func WithEnableSampler(enableSampler bool) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.EnableSampler = enableSampler
	}
}

// WithEnableSamplerError set error sampler option
func WithEnableSamplerError(enableSamplerError bool) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.EnableSamplerError = enableSamplerError
	}
}
