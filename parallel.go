// Copyright 2023 Stock Parfait

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iterator

import (
	"context"
)

type contextKey int

const (
	parallelKey contextKey = iota
)

// TestSerialize forces the number of workers in ParallelMap to be 1, thereby
// running jobs serially and in the strict FIFO order. This helps make tests
// deterministic.
func TestSerialize(ctx context.Context) context.Context {
	return context.WithValue(ctx, parallelKey, true)
}

func isSerialized(ctx context.Context) bool {
	v, ok := ctx.Value(parallelKey).(bool)
	return ok && v
}

type parallelMapIter[In, Out any] struct {
	ctx     context.Context // potentially cancelable context
	workers int             // maximum number of parallel jobs allowed
	jobs    int             // number of jobs currently running
	it      Iterator[In]
	f       func(In) Out
	resCh   chan Out // workers send their results to this channel
	done    bool
}

// ParallelMap runs multiple function calls f(In) in parallel on a given number
// of workers (0=unlimited), collects their results and returns as an iterator.
// The order of the results is undefined, unless the number of workers is 1.
//
// Canceling the supplied context immediately stops queuing new jobs, but the
// jobs that already started will finish and their results will be returned.
// Therefore, it is important to flush the iterator after canceling the context
// to release all the resources.
//
// Similarly, any early exit from the iterator loop must ensure that the context
// is canceled and the iterator is flushed.
//
// No job is started by this method itself. Jobs begin to run on the first
// Next() call on the result iterator, which is go routine safe.
//
// Example usage:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	m := ParallelMap(ctx, 2, it, f)
//	stop := func() {
//	  cancel()  // stop queuing new jobs
//	  Flush(m)  // flush the remaining parallel jobs, release resources
//	}
//	defer stop()
//	for v, ok := m.Next(); ok; v, ok = m.Next() {
//	  // Process v.
//	  // Exiting early is safe, m will be stopped and resources released.
//	}
func ParallelMap[In, Out any](ctx context.Context, workers int, it Iterator[In], f func(In) Out) Iterator[Out] {
	if isSerialized(ctx) {
		workers = 1
	}
	return &parallelMapIter[In, Out]{
		ctx:     ctx,
		workers: workers,
		resCh:   make(chan Out),
		it:      it,
		f:       f,
	}
}

// startJobs starts as many jobs as possible given the number of workers.
func (m *parallelMapIter[In, Out]) startJobs() {
	if m.done {
		return
	}
	for ; m.workers <= 0 || m.jobs < m.workers; m.jobs++ {
		select {
		case <-m.ctx.Done():
			m.done = true
			return
		default:
		}
		v, ok := m.it.Next()
		if !ok {
			m.done = true
			return
		}
		go func() { m.resCh <- m.f(v) }()
	}
}

// Next implements Iterator. It runs jobs in parallel up to the number of
// workers, blocks till at least one finishes (if any), and returns its result.
func (m *parallelMapIter[In, Out]) Next() (Out, bool) {
	m.startJobs()
	if m.jobs == 0 {
		m.done = true
		var zero Out
		return zero, false
	}
	r := <-m.resCh
	m.jobs--
	m.startJobs()
	return r, true
}

// ParallelMapSlice maps an input slice into the output slice using ParallelMap.
func ParallelMapSlice[In, Out any](ctx context.Context, workers int, in []In, f func(In) Out) []Out {
	return ToSlice(ParallelMap(ctx, workers, FromSlice(in), f))
}

// BatchReduce reduces the input iterator in parallel batches, returning an
// iterator of the results that can be further reduced by sequential Reduce, or
// another layer of BatchReduce. Panics if batchSize < 1.
//
// Same as ParallelMap, canceling context stops queuing new jobs, but the
// iterator needs to Flush to release resources. See ParallelMap for an example.
func BatchReduce[In, Out any](ctx context.Context, workers int, it Iterator[In], batchSize int, zero Out, f func(In, Out) Out) Iterator[Out] {
	batchIt := Batch(it, batchSize)
	batchF := func(in []In) Out { return Reduce(FromSlice(in), zero, f) }
	pm := ParallelMap(ctx, workers, batchIt, batchF)
	return pm
}
