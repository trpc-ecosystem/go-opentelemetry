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

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/debug"
	"trpc.group/trpc-go/trpc-opentelemetry/pkg/metrics"
)

// Defaults for BatchSpanProcessorOptions.
const (
	// DefaultMaxQueueSize default max queue size
	DefaultMaxQueueSize = 2048
	// DefaultBatchTimeout default batch timeout
	DefaultBatchTimeout = 5000 * time.Millisecond
	// DefaultExportTimeout default export timeout
	DefaultExportTimeout = 30000 * time.Millisecond
	// DefaultMaxExportBatchSize default max export batch size
	DefaultMaxExportBatchSize = 512
	// DefaultMaxBatchedPacketSize default max batched packet size
	DefaultMaxBatchedPacketSize = 2097152
)

var (
	failedExportCounter      = metrics.BatchProcessCounter.WithLabelValues("failed", "traces")
	succeededExportCounter   = metrics.BatchProcessCounter.WithLabelValues("success", "traces")
	batchByCountCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchcount")
	batchByPacketSizeCounter = metrics.BatchProcessCounter.WithLabelValues("batched", "packetsize")
	batchByTimerCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchtimer")
	enqueueCounter           = metrics.BatchProcessCounter.WithLabelValues("enqueue", "traces")
	dropCounter              = metrics.BatchProcessCounter.WithLabelValues("dropped", "traces")
)

// BatchSpanProcessorOption BatchSpanProcessor Option helper
type BatchSpanProcessorOption func(o *BatchSpanProcessorOptions)

// BatchSpanProcessorOptions BatchSpanProcessor options
type BatchSpanProcessorOptions struct {
	// MaxQueueSize is the maximum queue size to buffer spans for delayed processing. If the
	// queue gets full it drops the spans. Use BlockOnQueueFull to change this behavior.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available spans when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// ExportTimeout specifies the maximum duration for exporting spans. If the timeout
	// is reached, the export will be cancelled.
	// The default value of ExportTimeout is 30000 msec.
	ExportTimeout time.Duration

	// MaxExportBatchSize is the maximum number of spans to process in a single batch.
	// If there are more than one batch worth of spans then it processes multiple batches
	// of spans one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int

	// MaxPacketSize is the maximum number of packet size that will forcefully trigger a batch process.
	// The default value of MaxPacketSize is 2M (in bytes) .
	MaxPacketSize int

	// BlockOnQueueFull blocks onEnd() and onStart() method if the queue is full
	// AND if BlockOnQueueFull is set to true.
	// Blocking option should be used carefully as it can severely affect the performance of an
	// application.
	BlockOnQueueFull bool
}

// batchSpanProcessor is a SpanProcessor that batches asynchronously-received
// spans and sends them to a trace.Exporter when complete.
type batchSpanProcessor struct {
	e sdktrace.SpanExporter
	o BatchSpanProcessorOptions

	queue       chan sdktrace.ReadOnlySpan
	dropped     uint32
	batchedSize int

	batch      []sdktrace.ReadOnlySpan
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}

	debugger debug.UTF8Debugger
}

var _ sdktrace.SpanProcessor = (*batchSpanProcessor)(nil)

// NewBatchSpanProcessor creates a new SpanProcessor that will send completed
// span batches to the exporter with the supplied options.
//
// If the exporter is nil, the span processor will preform no action.
func NewBatchSpanProcessor(exporter sdktrace.SpanExporter, options ...BatchSpanProcessorOption) sdktrace.SpanProcessor {
	o := BatchSpanProcessorOptions{
		BatchTimeout:       DefaultBatchTimeout,
		ExportTimeout:      DefaultExportTimeout,
		MaxQueueSize:       DefaultMaxQueueSize,
		MaxExportBatchSize: DefaultMaxExportBatchSize,
		MaxPacketSize:      DefaultMaxBatchedPacketSize,
	}
	for _, opt := range options {
		opt(&o)
	}
	bsp := &batchSpanProcessor{
		e:        exporter,
		o:        o,
		batch:    make([]sdktrace.ReadOnlySpan, 0, o.MaxExportBatchSize),
		timer:    time.NewTimer(o.BatchTimeout),
		queue:    make(chan sdktrace.ReadOnlySpan, o.MaxQueueSize),
		stopCh:   make(chan struct{}),
		debugger: debug.NewUTF8Debugger(),
	}

	bsp.stopWait.Add(1)
	go func() {
		defer bsp.stopWait.Done()
		bsp.processQueue()
		bsp.drainQueue()
	}()

	return bsp
}

// OnStart method does nothing.
func (bsp *batchSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {}

// OnEnd method enqueues a ReadOnlySpan for later processing.
func (bsp *batchSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// Do not enqueue spans if we are just going to drop them.
	if bsp.e == nil {
		return
	}
	bsp.enqueue(s)
}

// Shutdown flushes the queue and waits until all spans are processed.
// It only executes once. Subsequent call does nothing.
func (bsp *batchSpanProcessor) Shutdown(ctx context.Context) error {
	var err error
	bsp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bsp.stopCh)
			bsp.stopWait.Wait()
			if bsp.e != nil {
				if err1 := bsp.e.Shutdown(ctx); err1 != nil {
					otel.Handle(err1)
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

type forceFlushSpan struct {
	sdktrace.ReadOnlySpan
	flushed chan struct{}
}

// SpanContext spanContext
func (f forceFlushSpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{TraceFlags: trace.FlagsSampled})
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *batchSpanProcessor) ForceFlush(ctx context.Context) error {
	var err error
	if bsp.e != nil {
		flushCh := make(chan struct{})
		if bsp.enqueueBlockOnQueueFull(ctx, forceFlushSpan{flushed: flushCh}, true) {
			select {
			case <-flushCh:
				// Processed any items in queue prior to ForceFlush being called
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		wait := make(chan error)
		go func() {
			wait <- bsp.exportSpans(ctx)
			close(wait)
		}()
		// Wait until the export is finished or the context is cancelled/timed out
		select {
		case err = <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	return err
}

// WithMaxQueueSize set MaxQueueSize helper
func WithMaxQueueSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize set MaxExportBatchSize helper
func WithMaxExportBatchSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout set BatchTimeout helper
func WithBatchTimeout(delay time.Duration) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithExportTimeout set ExportTimeout helper
func WithExportTimeout(timeout time.Duration) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.ExportTimeout = timeout
	}
}

// WithBlocking set Blocking helper
func WithBlocking() BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.BlockOnQueueFull = true
	}
}

// WithMaxPacketSize set MaxPacketSize helper
func WithMaxPacketSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxPacketSize = size
	}
}

// exportSpans is a subroutine of processing and draining the queue.
func (bsp *batchSpanProcessor) exportSpans(ctx context.Context) error {
	bsp.timer.Reset(bsp.o.BatchTimeout)

	bsp.batchMutex.Lock()
	defer bsp.batchMutex.Unlock()

	if bsp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bsp.o.ExportTimeout)
		defer cancel()
	}

	if l := len(bsp.batch); l > 0 {
		err := bsp.e.ExportSpans(ctx, bsp.batch)
		if err != nil && bsp.debugger.Enabled() {
			bsp.debugger.DebugSpansInvalidUTF8(err, bsp.batch)
		}
		size := len(bsp.batch)
		// A new batch is always created after exporting, even if the batch failed to be exported.
		//
		// It is up to the exporter to implement any type of retry logic if a batch is failing
		// to be exported, since it is specific to the protocol and backend being sent to.
		bsp.batch = bsp.batch[:0]
		bsp.batchedSize = 0

		if err != nil {
			failedExportCounter.Add(float64(size))
			return err
		}
		succeededExportCounter.Add(float64(size))
	}
	return nil
}

// processQueue removes spans from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bsp *batchSpanProcessor) processQueue() {
	defer bsp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-bsp.stopCh:
			return
		case <-bsp.timer.C:
			batchByTimerCounter.Inc()
			if err := bsp.exportSpans(ctx); err != nil {
				otel.Handle(err)
			}
		case sd := <-bsp.queue:
			if ffs, ok := sd.(forceFlushSpan); ok {
				close(ffs.flushed)
				continue
			}
			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			bsp.batchedSize += calcSpanSize(sd)
			shouldExport := bsp.shouldProcessInBatch()
			bsp.batchMutex.Unlock()
			if shouldExport {
				if !bsp.timer.Stop() {
					<-bsp.timer.C
				}
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		}
	}
}

// drainQueue awaits any caller that had added to bsp.stopWait
// to finish the enqueue, then exports the final batch.
func (bsp *batchSpanProcessor) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case sd := <-bsp.queue:
			if sd == nil {
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
				return
			}

			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			shouldExport := len(bsp.batch) == bsp.o.MaxExportBatchSize
			bsp.batchMutex.Unlock()

			if shouldExport {
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		default:
			close(bsp.queue)
		}
	}
}

func (bsp *batchSpanProcessor) enqueue(sd sdktrace.ReadOnlySpan) {
	bsp.enqueueBlockOnQueueFull(context.TODO(), sd, bsp.o.BlockOnQueueFull)
}

func (bsp *batchSpanProcessor) enqueueBlockOnQueueFull(ctx context.Context, sd sdktrace.ReadOnlySpan, block bool) bool {
	enqueueCounter.Inc()

	// This ensures the bsp.queue<- below does not panic as the
	// processor shuts down.
	defer func() {
		x := recover()
		switch err := x.(type) {
		case nil:
			return
		case runtime.Error:
			if err.Error() == "send on closed channel" {
				return
			}
		}
		panic(x)
	}()

	select {
	case <-bsp.stopCh:
		return false
	default:
	}

	if block {
		select {
		case bsp.queue <- sd:
			return true
		case <-ctx.Done():
			return false
		}
	}

	select {
	case bsp.queue <- sd:
		return true
	default:
		atomic.AddUint32(&bsp.dropped, 1)
		dropCounter.Inc()
	}
	return false
}

// shouldProcessInBatch determines whether to export in batches
func (bsp *batchSpanProcessor) shouldProcessInBatch() bool {
	if len(bsp.batch) == bsp.o.MaxExportBatchSize {
		batchByCountCounter.Inc()
		return true
	}

	if bsp.batchedSize >= bsp.o.MaxPacketSize {
		batchByPacketSizeCounter.Inc()
		return true
	}

	return false
}

// calcSpanSize calculates the packet size of a Span
func calcSpanSize(sd sdktrace.ReadOnlySpan) int {
	if sd == nil {
		return 0
	}

	size := 0
	// just calculate events size for now.
	for _, event := range sd.Events() {
		for _, kv := range event.Attributes {
			size += len(kv.Key)
			size += len(kv.Value.AsString())
		}
	}
	return size
}
