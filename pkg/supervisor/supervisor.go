// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains a simple supervisor for goroutine workers.
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
)

// Timeout when waiting for workers to finish.
const DefaultSupervisorTimeout = time.Second * 15

// Start the workers in order, waiting for each one to be ready before starting the next one.
// When a worker exits, send a cancel signal to all of them and wait for them to finish.
type SupervisorWorker struct {
	Name    string
	Workers []Worker
	Timeout time.Duration
}

func (w SupervisorWorker) String() string {
	return w.Name
}

func (w SupervisorWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	timeout := w.Timeout
	if timeout == 0 {
		timeout = DefaultSupervisorTimeout
		slog.Debug("supervisor: using default timeout", "timeout", timeout)
	} else {
		slog.Debug("supervisor: using custom timeout", "timeout", timeout)
	}

	// Start workers
	var wg sync.WaitGroup
Loop:
	for _, worker := range w.Workers {
		ctx := commons.AddWorkerNameToContext(ctx, worker.String())

		wg.Add(1)
		innerReady := make(chan struct{})
		go func() {
			defer wg.Done()
			defer cancel()
			err := worker.Start(ctx, innerReady)
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.WarnContext(ctx, "supervisor: worker exitted with error", "error", err)
			} else {
				slog.DebugContext(ctx, "supervisor: worker exitted with success")
			}
		}()
		select {
		case <-innerReady:
			slog.DebugContext(ctx, "supervisor: worker is ready")
		case <-time.After(timeout):
			slog.WarnContext(ctx, "supervisor: worker timed out")
			cancel()
			break Loop
		case <-ctx.Done():
			break Loop
		}
	}

	// Wait for context to be done
	ready <- struct{}{}
	<-ctx.Done()

	// Wait for all workers
	wait := make(chan struct{})
	go func() {
		wg.Wait()
		wait <- struct{}{}
	}()
	select {
	case <-wait:
		slog.DebugContext(ctx, "supervisor: all workers exitted")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("supervisor: timed out waiting for workers")
	}
}
