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
