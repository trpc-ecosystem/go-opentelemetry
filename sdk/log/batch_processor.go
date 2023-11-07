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

package log

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/debug"
	"trpc.group/trpc-go/trpc-opentelemetry/pkg/metrics"
)

const (
	DefaultMaxQueueSize         = 2048
	DefaultBatchTimeout         = 5000 * time.Millisecond
	DefaultMaxExportBatchSize   = 512
	DefaultMaxBatchedPacketSize = 2097152
	DefaultBlockOnQueueFull     = false
)

// BatchProcessor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
type BatchProcessor struct {
	queue   chan *logsproto.ResourceLogs
	dropped uint32

	batch       []*logsproto.ResourceLogs
	batchedSize int

	timer *time.Timer

	exporter Exporter
	stopCh   chan struct{}
	stopWait sync.WaitGroup
	stopOnce sync.Once

	debugger debug.UTF8Debugger
}

// NewBatchProcessor return BatchProcessor
func NewBatchProcessor(exporter Exporter) *BatchProcessor {
	bp := &BatchProcessor{
		exporter: exporter,
		batch:    make([]*logsproto.ResourceLogs, 0, DefaultMaxExportBatchSize),
		queue:    make(chan *logsproto.ResourceLogs, DefaultMaxQueueSize),
		stopCh:   make(chan struct{}),
		timer:    time.NewTimer(DefaultBatchTimeout),
		debugger: debug.NewUTF8Debugger(),
	}
	bp.stopWait.Add(1)

	go func() {
		defer bp.stopWait.Done()
		bp.processQueue()
		bp.drainQueue()
	}()

	return bp
}

// Shutdown is invoked during service shutdown.
func (bp *BatchProcessor) Shutdown(ctx context.Context) (err error) {
	bp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bp.stopCh)
			bp.stopWait.Wait()
			if bp.exporter != nil {
				if shutdownErr := bp.exporter.Shutdown(ctx); shutdownErr != nil {
					otel.Handle(shutdownErr)
				}
			}
			close(wait)
		}()
		// Wait until the wait group is done or the context is cancelled
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// Enqueue enqueue ResourceLogs to batch queue
func (bp *BatchProcessor) Enqueue(rl *logsproto.ResourceLogs) {
	metrics.BatchProcessCounter.WithLabelValues("enqueue", "logs").Inc()
	select {
	case <-bp.stopCh:
		return
	default:
	}

	if DefaultBlockOnQueueFull {
		bp.queue <- rl
		return
	}

	select {
	case bp.queue <- rl:
	default:
		metrics.BatchProcessCounter.WithLabelValues("dropped", "logs").Add(1)
		atomic.AddUint32(&bp.dropped, 1)
	}
}

func (bp *BatchProcessor) shouldProcessInBatch() bool {
	if len(bp.batch) == DefaultMaxExportBatchSize {
		return true
	}
	if bp.batchedSize >= DefaultMaxBatchedPacketSize {
		return true
	}
	return false
}

func (bp *BatchProcessor) processQueue() {
	defer bp.timer.Stop()

	for {
		select {
		case <-bp.stopCh:
			return
		case <-bp.timer.C:
			bp.export()
		case ld := <-bp.queue:
			bp.batch = append(bp.batch, ld)
			bp.batchedSize += calcLogSize(ld)
			if bp.shouldProcessInBatch() {
				if !bp.timer.Stop() {
					<-bp.timer.C
				}
				bp.export()
			}
		}
	}
}

func (bp *BatchProcessor) export() {
	bp.timer.Reset(DefaultBatchTimeout)
	if len(bp.batch) > 0 {
		err := bp.exporter.ExportLogs(context.Background(), bp.batch)
		if err != nil {
			otel.Handle(err)
			metrics.BatchProcessCounter.WithLabelValues("failed", "logs").Add(1)
			if bp.debugger.Enabled() {
				bp.debugger.DebugLogsInvalidUTF8(err, bp.batch)
			}
		} else {
			metrics.BatchProcessCounter.WithLabelValues("success", "logs").Add(1)
		}
		bp.batch = bp.batch[:0]
		bp.batchedSize = 0
	}
}

func (bp *BatchProcessor) drainQueue() {
	for {
		select {
		case sd := <-bp.queue:
			if sd == nil {
				bp.export()
				return
			}

			bp.batch = append(bp.batch, sd)
			if len(bp.batch) == DefaultMaxExportBatchSize {
				bp.export()
			}
		default:
			close(bp.queue)
		}
	}
}

// BatchProcessorOption apply changes to internalOptions.
type BatchProcessorOption func(o *BatchProcessorOptions)

// BatchProcessorOptions defines the configuration for the various elements of BatchProcessor
type BatchProcessorOptions struct {
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
}

// WithMaxQueueSize return BatchProcessorOption which to set MaxQueueSize
func WithMaxQueueSize(size int) BatchProcessorOption {
	return func(o *BatchProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize return BatchProcessorOption which to set MaxExportBatchSize
func WithMaxExportBatchSize(size int) BatchProcessorOption {
	return func(o *BatchProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout return BatchProcessorOption which to set BatchTimeout
func WithBatchTimeout(delay time.Duration) BatchProcessorOption {
	return func(o *BatchProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithBlocking return BatchProcessorOption which to set BlockOnQueueFull
func WithBlocking() BatchProcessorOption {
	return func(o *BatchProcessorOptions) {
		o.BlockOnQueueFull = true
	}
}

func calcLogSize(l *logsproto.ResourceLogs) int {
	if l == nil {
		return 0
	}
	return proto.Size(l)
}
