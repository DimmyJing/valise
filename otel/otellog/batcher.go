package otellog

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	logFlushTimeout = 5 * time.Second
	logBatchSize    = 2048
)

type batcher struct {
	exporter LogExporter
	batch    []ReadOnlyLog
	lock     sync.Mutex
	timer    *time.Timer
	queue    chan ReadOnlyLog
	stopCh   chan struct{}
	stopWait sync.WaitGroup
	err      error
}

func newBatcher(logs LogExporter) *batcher {
	handler := batcher{
		exporter: logs,
		batch:    nil,
		lock:     sync.Mutex{},
		timer:    time.NewTimer(logFlushTimeout),
		queue:    make(chan ReadOnlyLog, logBatchSize),
		stopCh:   make(chan struct{}),
		stopWait: sync.WaitGroup{},
		err:      nil,
	}
	handler.stopWait.Add(1)

	go func() {
		defer handler.stopWait.Done()
		handler.processQueue()
		handler.drainQueue()
	}()

	return &handler
}

func (b *batcher) ExportLogs(ctx context.Context, logs []ReadOnlyLog) error {
	for _, log := range logs {
		b.queue <- log
	}

	if b.err != nil {
		err := b.err
		b.err = nil

		return err
	}

	return nil
}

func (b *batcher) Shutdown(ctx context.Context) error {
	err := b.exporter.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed shutting down log exporter: %w", err)
	}

	return nil
}

func (b *batcher) processQueue() {
	defer b.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-b.stopCh:
			return
		case <-b.timer.C:
			b.exportLogs(ctx)
		case log := <-b.queue:
			b.handleLog(ctx, log)
		}
	}
}

func (b *batcher) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case log := <-b.queue:
			b.handleLog(ctx, log)
		default:
			b.exportLogs(ctx)

			return
		}
	}
}

func (b *batcher) handleLog(ctx context.Context, log ReadOnlyLog) {
	b.lock.Lock()
	b.batch = append(b.batch, log)
	shouldExport := len(b.batch) >= logBatchSize
	b.lock.Unlock()

	if shouldExport {
		if !b.timer.Stop() {
			<-b.timer.C
		}

		b.exportLogs(ctx)
	}
}

func (b *batcher) exportLogs(ctx context.Context) {
	b.timer.Reset(logFlushTimeout)
	b.lock.Lock()
	defer b.lock.Unlock()

	if len(b.batch) > 0 {
		err := b.exporter.ExportLogs(ctx, b.batch)
		if err != nil {
			b.err = fmt.Errorf("error exporting %d logs: %w", len(b.batch), err)
		}

		b.batch = nil
	}
}
