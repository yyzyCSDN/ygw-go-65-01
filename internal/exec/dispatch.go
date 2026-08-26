package exec

import (
	"context"
	"sync"
)

// Run starts the configured number of dispatcher workers until ctx is done.
func (e *Executor) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.worker(ctx)
		}()
	}
	<-ctx.Done()
	wg.Wait()
}

func (e *Executor) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.queue.Notify():
		}
		e.drain(ctx)
	}
}

func (e *Executor) drain(ctx context.Context) {
	for {
		batch := e.queue.DequeueBatch(e.batchSize)
		if len(batch.Calls) == 0 {
			if batch.More {
				continue
			}
			return
		}
		for _, call := range batch.Calls {
			if err := ctx.Err(); err != nil {
				return
			}
			e.Dispatch(ctx, call)
		}
		if !batch.More {
			if e.queue.Len() > 0 {
				continue
			}
			return
		}
	}
}
