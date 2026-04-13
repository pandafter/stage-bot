package queue

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Processor executes the actual work for a claimed job.
// The payload is opaque to the queue — the processor decodes it.
type Processor interface {
	Process(ctx context.Context, senderID string, payload []byte) error
}

type Pool struct {
	queue        Queue
	processor    Processor
	concurrency  int
	maxAttempts  int
	pollInterval time.Duration
	jobTimeout   time.Duration
	logger       *zap.Logger
	wg           sync.WaitGroup
}

type Config struct {
	Concurrency  int
	MaxAttempts  int
	PollInterval time.Duration
	JobTimeout   time.Duration
}

func NewPool(q Queue, p Processor, cfg Config, logger *zap.Logger) *Pool {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.JobTimeout <= 0 {
		cfg.JobTimeout = 90 * time.Second
	}
	return &Pool{
		queue:        q,
		processor:    p,
		concurrency:  cfg.Concurrency,
		maxAttempts:  cfg.MaxAttempts,
		pollInterval: cfg.PollInterval,
		jobTimeout:   cfg.JobTimeout,
		logger:       logger,
	}
}

// Start launches the worker goroutines and a recovery sweep. Blocks nothing.
func (p *Pool) Start(ctx context.Context) {
	recoverCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if n, err := p.queue.RecoverStuck(recoverCtx, 5*time.Minute); err != nil {
		p.logger.Warn("recover stuck jobs failed", zap.Error(err))
	} else if n > 0 {
		p.logger.Info("recovered stuck jobs", zap.Int("count", n))
	}

	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	p.logger.Info("queue pool started",
		zap.Int("workers", p.concurrency),
		zap.Int("max_attempts", p.maxAttempts),
	)
}

// Wait blocks until all workers exit (after ctx cancel).
func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log := p.logger.With(zap.Int("worker", id))

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopping")
			return
		case <-ticker.C:
			p.runOne(ctx, log)
		}
	}
}

func (p *Pool) runOne(ctx context.Context, log *zap.Logger) {
	claimCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	job, err := p.queue.Claim(claimCtx)
	if err != nil {
		log.Error("claim failed", zap.Error(err))
		return
	}
	if job == nil {
		return
	}

	log = log.With(zap.Int64("job_id", job.ID), zap.String("sender", job.SenderID), zap.Int("attempt", job.Attempts))
	log.Info("processing job")

	procCtx, procCancel := context.WithTimeout(ctx, p.jobTimeout)
	defer procCancel()

	procErr := p.processor.Process(procCtx, job.SenderID, job.Payload)

	finCtx, finCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finCancel()

	if procErr != nil {
		log.Error("job failed", zap.Error(procErr))
		if err := p.queue.Fail(finCtx, job.ID, procErr.Error(), p.maxAttempts); err != nil {
			log.Error("mark fail failed", zap.Error(err))
		}
		return
	}

	if err := p.queue.Complete(finCtx, job.ID); err != nil {
		log.Error("mark complete failed", zap.Error(err))
		return
	}
	log.Info("job done")
}
