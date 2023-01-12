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
	ctx := context.Background()
	in := []int{5, 10, 15}
	f := func(i int) int { return i + 1 }
	for _, r := range iterator.ParallelMapSlice(ctx, 2, in, f) {
		fmt.Printf("result = %d\n", r)
	}
}
```
