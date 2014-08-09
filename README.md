# ratelimit
[![GoDoc](https://godoc.org/github.com/r8k/ratelimit?status.svg)](https://godoc.org/github.com/r8k/ratelimit)

 Rate Limiter for Go, backed by Redis.

 View the [docs](http://godoc.org/github.com/r8k/ticker).

## Dependencies
 * [Redis](http://redis.io/download) 2.8.13+
 * [Redigo](https://github.com/garyburd/redigo/)

## Features
 * Utilises connection pool from Redigo
 * Handles race conditions
 * [Efficient](https://github.com/r8k/ratelimit#benchmark)
 * Distributed Store: coming soon

## Installation

```
$ go get github.com/r8k/ratelimit
```

## Example

```go
package main

import "fmt"
import "net"
import "github.com/r8k/ratelimit"

func main() {
    store, err := ratelimit.Init(&net.TCPAddr{Port: 6379})
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Get `RateLimit` for `identifier` client_ip
    Limit, err := store.Get("client_ip")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Quota: %d\n", Limit.Quota)
    fmt.Printf("Used: %d\n", Limit.Used)
    fmt.Printf("Remaining: %d\n", Limit.Remaining)
    fmt.Printf("Retry After: %s\n", Limit.RetryAfter)
}

```

Run the above example
````
❯ go run main.go

Quota: 5000
Used: 1
Remaining: 4999
Retry After: 2014-08-09 17:14:55 +0530 IST
```

## Benchmark
````
❯ go test -bench=.

PASS
BenchmarkGetSequential     10000        192374 ns/op
BenchmarkGetParallel         100      20119241 ns/op
ok      github.com/r8k/ratelimit    4.016s
````

## License

MIT
