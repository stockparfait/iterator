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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIterator(t *testing.T) {
	t.Parallel()

	Convey("FromSlice and ToSlice", t, func() {
		slice := []int{1, 2, 5}
		So(ToSlice(FromSlice(slice)), ShouldResemble, slice)
	})

	Convey("Map, MapCloser", t, func() {
		var closed bool
		f := func(i int) int { return i + 1 }
		it := WithClose(FromSlice([]int{1, 5}), func() { closed = true })
		mp := MapCloser(it, f)
		So(ToSlice(mp), ShouldResemble, []int{2, 6})
		mp.Close()
		So(closed, ShouldBeTrue)
	})

	Convey("Reduce", t, func() {
		f := func(i int, out float64) float64 { return float64(i) + out }
		So(Reduce(FromSlice([]int{1, 2, 3}), 1.0, f), ShouldResemble, 7.0)
	})

	Convey("Batch", t, func() {
		Convey("even batches", func() {
			it := FromSlice([]int{1, 2, 3, 4, 5, 6})
			So(ToSlice(Batch(it, 3)), ShouldResemble, [][]int{
				{1, 2, 3}, {4, 5, 6}})
		})

		Convey("uneven batches", func() {
			it := FromSlice([]int{1, 2, 3, 4, 5, 6, 7})
			So(ToSlice(Batch(it, 3)), ShouldResemble, [][]int{
				{1, 2, 3}, {4, 5, 6}, {7}})
		})

		Convey("empty input", func() {
			it := FromSlice([]int{})
			So(len(ToSlice(Batch(it, 3))), ShouldEqual, 0)
		})
	})

	Convey("BatchCloser", t, func() {
		var closed bool
		it := WithClose(FromSlice([]int{1, 2, 3, 4, 5}), func() { closed = true })
		bit := BatchCloser(it, 3)
		So(ToSlice(bit), ShouldResemble, [][]int{{1, 2, 3}, {4, 5}})
		bit.Close()
		So(closed, ShouldBeTrue)
	})

	Convey("ChainCloser", t, func() {
		var closedTop, closed1, closed2 bool
		it1 := WithClose(FromSlice([]int{1, 2, 3}), func() { closed1 = true })
		it2 := WithClose(FromSlice([]int{4, 5}), func() { closed2 = true })
		it := WithClose(FromSlice([]IteratorCloser[int]{it1, it2}),
			func() { closedTop = true })
		out := ChainCloser(it)

		Convey("drain then close", func() {
			So(ToSlice(out), ShouldEqual, []int{1, 2, 3, 4, 5})
			out.Close()
			So(closedTop, ShouldBeTrue)
			So(closed1, ShouldBeTrue)
			So(closed2, ShouldBeTrue)
		})

		Convey("close midway", func() {
			v, ok := out.Next()
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, 1)
			out.Close()
			So(closedTop, ShouldBeTrue)
			So(closed1, ShouldBeTrue)  // current
			So(closed2, ShouldBeFalse) // never requested
		})
	})

	Convey("Unbatch", t, func() {
		it := FromSlice([][]int{{1, 2}, {}, {3, 4, 5}})
		So(ToSlice(Unbatch(it)), ShouldResemble, []int{1, 2, 3, 4, 5})
	})

	Convey("UnbatchCloser", t, func() {
		var closed bool
		it := WithClose(FromSlice([][]int{{1, 2}, {}, {3, 4, 5}}),
			func() { closed = true })
		ub := UnbatchCloser(it)
		So(ToSlice(ub), ShouldResemble, []int{1, 2, 3, 4, 5})
		ub.Close()
		So(closed, ShouldBeTrue)
	})

	Convey("Flush", t, func() {
		it := FromSlice([]int{1, 2, 3})
		Flush(it)
		So(len(ToSlice(it)), ShouldEqual, 0)
	})

	Convey("WithClose", t, func() {
		it := FromSlice([]int{1, 2, 3})
		var closeCalls int
		closer := WithClose(it, func() { Flush(it); closeCalls++ })
		v, ok := closer.Next()
		So(ok, ShouldBeTrue)
		So(v, ShouldEqual, 1)
		closer.Close()
		_, ok = closer.Next()
		So(ok, ShouldBeFalse)
		closer.Close()
		So(closeCalls, ShouldEqual, 1) // no duplicate calls to close
	})

	Convey("Repeat", t, func() {
		So(ToSlice(Repeat(42, 5)), ShouldResemble, []int{42, 42, 42, 42, 42})
	})
}
