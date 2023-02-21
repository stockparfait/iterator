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
	"github.com/stockparfait/errors"
)

// Iterator is a generic interface for generating sequences of values of type T.
//
// When the second Next()'s result is true it returns the next value. When it's
// false, the iterator is considered "empty", and subsequent calls to Next() are
// expected to return false.
//
// Example use of an iterator "it":
//
//	for v, ok := it.Next(); ok; v, ok = it.Next() {
//	  // use v
//	}
type Iterator[T any] interface {
	Next() (T, bool)
}

// IteratorCloser is an iterator with an additional Close() method which empties
// the iterator (a subsequent Next() call returns ok=false) and releases all
// associated resources, such as active go-routines.
//
// Example use of a closing iterator "it":
//
//	defer it.Close()
//	for v; ok := it.Next(); ok; v, ok = it.Next() {
//	  // use v; can safely exit early
//	}
type IteratorCloser[T any] interface {
	Iterator[T]
	Close()
}

type sliceIter[T any] struct {
	slice []T
}

func (it *sliceIter[T]) Next() (T, bool) {
	if len(it.slice) == 0 {
		var zero T
		return zero, false
	}
	x := it.slice[0]
	it.slice = it.slice[1:]
	return x, true
}

func FromSlice[T any](s []T) Iterator[T] {
	return &sliceIter[T]{slice: s}
}

func ToSlice[T any](it Iterator[T]) []T {
	var s []T
	for x, ok := it.Next(); ok; x, ok = it.Next() {
		s = append(s, x)
	}
	return s
}

type mapIter[In, Out any] struct {
	it Iterator[In]
	f  func(In) Out
}

func (it *mapIter[In, Out]) Next() (Out, bool) {
	v, ok := it.it.Next()
	if !ok {
		var zero Out
		return zero, false
	}
	return it.f(v), true
}

// Map transforms Iterator[In] into Iterator[Out] by applying f elementwise.
func Map[In, Out any](it Iterator[In], f func(In) Out) Iterator[Out] {
	return &mapIter[In, Out]{it: it, f: f}
}

// Reduce Iterator[In] into a single value Out by applying res[n+1] = f(x[n],
// res[n]), starting with res[0] = zero.
func Reduce[In, Out any](it Iterator[In], zero Out, f func(In, Out) Out) Out {
	res := zero
	for v, ok := it.Next(); ok; v, ok = it.Next() {
		res = f(v, res)
	}
	return res
}

type batchIter[T any] struct {
	it   Iterator[T]
	n    int
	done bool
}

func (it *batchIter[T]) Next() ([]T, bool) {
	if it.done {
		return nil, false
	}
	v, ok := it.it.Next()
	if !ok {
		it.done = true
		return nil, false
	}
	var res []T
	for i := 0; ok && i < it.n; i++ {
		res = append(res, v)
		if i+1 < it.n {
			v, ok = it.it.Next()
		}
	}
	it.done = !ok
	return res, true
}

// Batch the input iterator values into n-sized slices and return them as a new
// iterator. Panics if n < 1.
func Batch[T any](it Iterator[T], n int) Iterator[[]T] {
	if n < 1 {
		panic(errors.Reason("n=%d must be >= 1", n))
	}
	return &batchIter[T]{it: it, n: n}
}

// Flush the remaining elements from the iterator. This can be useful for a
// custom IteratorCloser when the iterator needs to flush remaining elements to
// release resources.
func Flush[T any](it Iterator[T]) {
	for _, ok := it.Next(); ok; _, ok = it.Next() {
	}
}

type itCloser[T any] struct {
	it     Iterator[T]
	close  func()
	closed bool
}

func (it *itCloser[T]) Next() (T, bool) {
	if it.closed {
		var zero T
		return zero, false
	}
	return it.it.Next()
}

func (it *itCloser[T]) Close() {
	if it.closed {
		return
	}
	it.close()
	it.closed = true
}

// WithClose attaches a close function to at iterator.
func WithClose[T any](it Iterator[T], close func()) IteratorCloser[T] {
	return &itCloser[T]{it: it, close: close}
}
