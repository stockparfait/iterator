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
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParallelMap(t *testing.T) {
	t.Parallel()

	Convey("ParallelMap works", t, func() {
		ctx := context.Background()
		var running, maxRunning int
		var sequence []int
		var mux sync.Mutex

		start := func(i int) {
			mux.Lock()
			sequence = append(sequence, i)
			running++
			if running > maxRunning {
				maxRunning = running
			}
			mux.Unlock()
		}

		end := func() {
			mux.Lock()
			running--
			mux.Unlock()
		}

		f := func(i int) int {
			start(i)
			time.Sleep(1 * time.Millisecond)
			end()
			return i
		}

		slice := func(n int) (res []int) {
			for i := 0; i < n; i++ {
				res = append(res, i)
			}
			return
		}

		Convey("with limited workers", func() {
			res := ParallelMapSlice(ctx, 5, slice(15), f)
			So(len(res), ShouldEqual, 15)
			So(len(sequence), ShouldEqual, 15)
			So(maxRunning, ShouldEqual, 5)
		})

		Convey("serialized", func() {
			res := ParallelMapSlice(TestSerialize(ctx), 5, slice(15), f)
			So(res, ShouldResemble, slice(15))
			So(maxRunning, ShouldEqual, 1)
			So(len(sequence), ShouldEqual, 15)
		})

		Convey("with no jobs", func() {
			res := ParallelMapSlice[int](ctx, 0, nil, f)
			So(len(res), ShouldEqual, 0)
		})

		Convey("canceling context stops enqueuing jobs", func() {
			cc, cancel := context.WithCancel(ctx)
			m := ParallelMap(cc, 3, FromSlice(slice(15)), f)
			_, ok := m.Next()
			So(ok, ShouldBeTrue)
			cancel() // 3 more still in flight
			_, ok = m.Next()
			So(ok, ShouldBeTrue)
			_, ok = m.Next()
			So(ok, ShouldBeTrue)
			_, ok = m.Next()
			So(ok, ShouldBeTrue)
			_, ok = m.Next()
			So(ok, ShouldBeFalse)
			So(len(sequence), ShouldEqual, 4)
		})

		Convey("Close stops and flushes the iterator", func() {
			m := ParallelMap(ctx, 3, FromSlice(slice(15)), f)
			_, ok := m.Next()
			So(ok, ShouldBeTrue)
			m.Close() // 3 more were in flight, but should be flushed
			_, ok = m.Next()
			So(ok, ShouldBeFalse)
			So(len(sequence), ShouldEqual, 4)
		})
	})

	Convey("BatchReduce works", t, func() {
		ctx := TestSerialize(context.Background())
		f := func(n, res int) int { return n + res }

		Convey("even batches", func() {
			it := FromSlice([]int{1, 2, 3, 4, 5, 6})
			So(ToSlice[int](BatchReduce[int, int](ctx, 1, it, 3, 0, f)), ShouldResemble, []int{6, 15})
		})

		Convey("uneven batches", func() {
			it := FromSlice([]int{1, 2, 3, 4, 5, 6, 7})
			So(ToSlice[int](BatchReduce(ctx, 1, it, 3, 0, f)), ShouldResemble, []int{
				6, 15, 7})
		})

		Convey("empty input", func() {
			it := FromSlice([]int{})
			So(len(ToSlice[int](BatchReduce(ctx, 1, it, 3, 0, f))), ShouldEqual, 0)
		})
	})
}
