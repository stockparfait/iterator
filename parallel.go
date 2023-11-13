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

type enumerated[Out any] struct {
	v Out
	i int // sequence number
}

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
//	m := ParallelMap(context.Background(), 2, it, f)
//	defer m.Close()
//	for v, ok := m.Next(); ok; v, ok = m.Next() {
//	  // Process v.
//	  // Exiting early is safe, m will be closed and resources released.
//	}
func ParallelMap[In, Out any](ctx context.Context, workers int, it Iterator[In], f func(In) Out) IteratorCloser[Out] {
	if isSerialized(ctx) {
		workers = 1
	}
	if workers <= 0 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	return &parallelMapIter[In, Out]{
		ctx:     ctx,
		cancel:  cancel,
		workers: workers,
		resCh:   make(chan enumerated[Out]),
		it:      it,
		f:       f,
	}
}

// OrderedParallelMap is like ParallMap, only it preserves the output order. The
// bufferSize must be >= workers, otherwise it effectively reduces parallelism
// to bufferSize workers. Larger bufferSize may improve parallelization when
// some jobs take a lot longer than others.
func OrderedParallelMap[In, Out any](ctx context.Context, workers, bufferSize int, it Iterator[In], f func(In) Out) IteratorCloser[Out] {
	if isSerialized(ctx) {
		workers = 1
	}
	if workers <= 0 {
		workers = 1
	}
	if bufferSize <= 0 {
		bufferSize = 1
	}
	ctx, cancel := context.WithCancel(ctx)
	return &parallelMapIter[In, Out]{
		ctx:     ctx,
		cancel:  cancel,
		workers: workers,
		it:      it,
		f:       f,
		resCh:   make(chan enumerated[Out]),
		buffer:  make([]Out, bufferSize),
		ready:   make([]bool, bufferSize),
	}
}

type parallelMapIter[In, Out any] struct {
	ctx       context.Context // potentially cancelable context
	cancel    func()          // cancels the context
	workers   int             // maximum number of parallel jobs allowed
	jobs      int             // number of jobs currently running
	it        Iterator[In]
	f         func(In) Out
	resCh     chan enumerated[Out] // workers send their results to this channel
	done      bool
	buffer    []Out  // reorder buffer; ignore if nil
	ready     []bool // which cells are ready in buffer
	numReady  int    // number of ready cells
	nextSched int    // next cell to schedule
	next      int    // next cell to return in Next()
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
		if len(m.buffer) > 0 && m.jobs == len(m.buffer) {
			return
		}
		v, ok := m.it.Next()
		if !ok {
			m.done = true
			return
		}
		go func(i int) {
			m.resCh <- enumerated[Out]{v: m.f(v), i: i}
		}(m.nextSched)
		if len(m.buffer) > 0 {
			m.nextSched = (m.nextSched + 1) % len(m.buffer)
		}
	}
}

// Next implements Iterator. It runs jobs in parallel up to the number of
// workers, blocks till at least one finishes (if any), and returns its result.
func (m *parallelMapIter[In, Out]) Next() (Out, bool) {
	m.startJobs()
	if len(m.buffer) == 0 {
		if m.jobs == 0 {
			m.done = true
			var zero Out
			return zero, false
		}

		r := <-m.resCh
		m.jobs--
		m.startJobs()
		return r.v, true
	}
	if m.numReady == 0 && m.jobs == 0 {
		m.done = true
		var zero Out
		return zero, false
	}
	for !m.ready[m.next] {
		r := <-m.resCh
		m.jobs--
		m.buffer[r.i] = r.v
		m.ready[r.i] = true
		m.numReady++
	}
	res := m.buffer[m.next]
	m.ready[m.next] = false
	m.numReady--
	m.next = (m.next + 1) % len(m.buffer)
	m.startJobs()
	return res, true
}

func (m *parallelMapIter[In, Out]) Close() {
	m.cancel()
	Flush[Out](m)
}

// ParallelMapSlice maps an input slice into the output slice using ParallelMap.
func ParallelMapSlice[In, Out any](ctx context.Context, workers int, in []In, f func(In) Out) []Out {
	return ToSlice[Out](ParallelMap(ctx, workers, FromSlice(in), f))
}

// BatchReduce reduces the input iterator in parallel batches, returning an
// iterator of the results that can be further reduced by sequential Reduce, or
// another layer of BatchReduce. Panics if batchSize < 1.
//
// Same as ParallelMap, canceling context stops queuing new jobs, but the
// iterator needs to Flush to release resources. See ParallelMap for an example.
func BatchReduce[In, Out any](ctx context.Context, workers int, it Iterator[In], batchSize int, zero Out, f func(In, Out) Out) IteratorCloser[Out] {
	batchIt := Batch(it, batchSize)
	batchF := func(in []In) Out { return Reduce(FromSlice(in), zero, f) }
	pm := ParallelMap(ctx, workers, batchIt, batchF)
	return pm
}
