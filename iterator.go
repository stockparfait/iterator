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

type Iterator[T any] interface {
	Next() (T, bool)
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

func Map[In, Out any](it Iterator[In], f func(In) Out) Iterator[Out] {
	return &mapIter[In, Out]{it: it, f: f}
}
