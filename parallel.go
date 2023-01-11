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
	"sync"
)

type contextKey int

const (
	parallelKey contextKey = iota
)

// TestSerialize forces the number of workers in Map to be 1, thereby running
// jobs serially and in the strict FIFO order. This helps make tests
// deterministic.
func TestSerialize(ctx context.Context) context.Context {
	return context.WithValue(ctx, parallelKey, true)
}

func isSerialized(ctx context.Context) bool {
	v, ok := ctx.Value(parallelKey).(bool)
	return ok && v
}

// Job is a unit of work to be done, which returns a result.
type Job[T any] func() T

type mapIter[T any] struct {
	ctx     context.Context  // potentially cancelable context
	workers int              // maximum number of parallel jobs allowed
	jobs    int              // number of jobs currently running
	it      Iterator[Job[T]] // job iterator
	resCh   chan T           // workers send their results to this channel
	done    bool
	mux     sync.Mutex // to make Next() go routine safe
}

var _ Iterator[int] = &mapIter[int]{}

// ParallelMap runs multiple jobs in parallel on a given number of workers,
// 0=unlimited, collects their results and returns as an iterator. The order of
// results is not guaranteed, unless the number of workers is 1.
//
// Canceling the supplied context immediately stops queuing new jobs, but the
// jobs that already started will finish and their results will be returned.
// Therefore, it is important to flush the iterator after canceling the context
// to release all the resources.
//
// No job is started by this method itself. Jobs begin to run on the first
// Next() call on the result iterator, which is go routine safe.
//
// Example usage:
//
//	m := ParallelMap(context.Background(), 2, jobsIter)
//	for v, ok := m.Next(); ok; v, ok = m.Next() {
//	  // Process v and err.
//	}
func ParallelMap[T any](ctx context.Context, workers int, it Iterator[Job[T]]) Iterator[T] {
	if isSerialized(ctx) {
		workers = 1
	}
	return &mapIter[T]{
		ctx:     ctx,
		workers: workers,
		resCh:   make(chan T),
		it:      it,
	}
}

// startJobs starts as many jobs as possible given the number of workers.
func (m *mapIter[T]) startJobs() {
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
		j, ok := m.it.Next()
		if !ok {
			m.done = true
			return
		}
		go func() { m.resCh <- j() }()
	}
}

// Next implements ResultsIter. It runs jobs in parallel up to the number of
// workers, blocks till at least one finishes (if any), and returns its result.
// When no more jobs are left, return Done error. Go routine safe.
func (m *mapIter[T]) Next() (T, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.startJobs()
	if m.jobs == 0 {
		m.done = true
		var zero T
		return zero, false
	}
	r := <-m.resCh
	m.jobs--
	m.startJobs()
	return r, true
}

// Jobs creates a JobsIter from a slice of Job.
func Jobs[T any](js []Job[T]) Iterator[Job[T]] {
	return FromSlice[Job[T]](js)
}

// ParallelMapSlice is a convenience method around Map. It runs a slice of jobs in
// parallel, waits for them to finish, and returns the results in a slice.
func ParallelMapSlice[T any](ctx context.Context, workers int, jobs []Job[T]) []T {
	// Jobs() never returns non-nil error, it's safe to ignore it.
	return ToSlice(ParallelMap(ctx, workers, Jobs(jobs)))
}
