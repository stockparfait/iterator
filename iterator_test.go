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

	Convey("Map", t, func() {
		f := func(i int) int { return i + 1 }
		So(ToSlice(Map(FromSlice([]int{1, 5}), f)), ShouldResemble, []int{2, 6})
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

	Convey("Flush", t, func() {
		it := FromSlice([]int{1, 2, 3})
		Flush(it)
		So(len(ToSlice(it)), ShouldEqual, 0)
	})
}
