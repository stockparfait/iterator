# Generic iterator library

[![Build Status](https://github.com/stockparfait/iterator/workflows/Tests/badge.svg)](https://github.com/stockparfait/iterator/actions?query=workflow%3ATests)
[![GoDoc](https://godoc.org/github.com/stockparfait/iterator?status.svg)](http://godoc.org/github.com/stockparfait/iterator)

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

`ParallelMap` is similar to `Map`, except that it runs multiple function calls
`f(In)` ("jobs") in parallel on a given number of workers (unlimited when 0).
The order of the results is undefined, unless the number of workers is 1.

Canceling the supplied context immediately stops queuing new jobs, but the jobs
that already started will finish and their results will be returned.  Therefore,
it is important to flush the iterator after canceling the context to release all
the resources.

No job is started by this method itself. Jobs begin to run on the first `Next()`
call on the result iterator.

Note, that `Next()` is _not_ go routine safe, and it doesn't require go routine
safety of the input iterator.

For simpler situations when the input sequence can be created ahead of time and
the results collected in a slice before further processing, use `ParallelMapSlice`:

```go
package main

import (
	"context"
	"fmt"

	"github.com/stockparfait/iterator"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	it := iterator.FromSlice([]int{5, 10, 15})
	stop := func() {
		cancel()           // stop queuing new jobs
		iterator.Flush(it) // flush the remaining parallel jobs, release resources
	}
	defer stop()

	f := func(i int) int { return i + 1 }
	pm := iterator.ParallelMap(ctx, 2, it, f)
	for r, ok := pm.Next(); ok; r, ok = pm.Next() {
		fmt.Printf("result = %d\n", r)
		// Early exit is safe, resources will be released.
	}
}
```
