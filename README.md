# Generic iterator library

This package standardizes iterator implementation using generic types. It is
designed for ease of use in a `for` loop, and includes several commonly used
iterator methods, such as converting a slice to an iterator and back, a
sequential map, etc. Of special note is `ParallelMap`, see below.

This package requires Go 1.18 or higher, as it uses generics.

## Installation

```
go get github.com/stockparfait/iterator
```

## Example usage

```go
package main

import (
	"fmt"

	"github.com/stockparfait/iterator"
)

func main() {
	it := iterator.FromSlice([]int{1, 2, 3, 4})

	for v, ok := it.Next(); ok; v, ok = it.Next() {
		fmt.Println(v)
	}
}
```

## ParallelMap - parallel execution with a specified number of workers

This method implements processing units of work (`Job`) in parallel using a
specified number of workers (0 - unlimited). In the most general case
(`ParallelMap`), the user supplies an iterator over `Job`'s which are processed
in parallel and the results are returned as an iterator of the results.

For simpler situations when all jobs can be created ahead of time and the
results collected before further processing, use `ParallelMapSlice`.

Example:

```go
package main

import (
	"context"
	"fmt"

	"github.com/stockparfait/iterator"
)

func main() {
	ctx := context.Background()
	in := []int{5, 10, 15}
	f := func(i int) int { return i + 1 }
	for _, r := range iterator.ParallelMapSlice(ctx, 2, in, f) {
		fmt.Printf("result = %d\n", r)
	}
}
```
