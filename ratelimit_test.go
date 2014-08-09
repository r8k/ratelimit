package ratelimit

import "fmt"
import "net"
import "testing"

// flush helper, to help in Testing
// `FLUSH` data from `Redis`
func (s *RedisLimiter) flush() (bool, error) {
	conn := s.Pool.Get()
	defer conn.Close()

	data, err := conn.Do("FLUSHALL")
	if err != nil || data == nil {
		return false, err
	}

	return (data == "OK"), nil
}

func TestRedisLimiter(t *testing.T) {
	var Limit *Limit

	store, err := Init(&net.TCPAddr{Port: 6379})
	if err != nil {
		t.Fatal(err.Error())
	}

	// empty `Redis` for Testing
	store.flush()

	defer store.Close()
	defer store.flush()

	// Get `RateLimit` for `identifier` client_ip
	if Limit, err = store.Get("client_ip"); err != nil {
		t.Fatalf("Error getting session: %v", err)
	}

	if Limit.Quota != 5000 {
		t.Errorf("Expected Quota to be 5000; Got %v", Limit.Quota)
	}

	if Limit.Used != 1 {
		t.Errorf("Expected Used to be 1; Got %v", Limit.Used)
	}

	if Limit.Remaining != 4999 {
		t.Errorf("Expected Remaining to be 4999; Got %v", Limit.Remaining)
	}
}

func TestPingGoodPort(t *testing.T) {
	store, _ := Init(&net.TCPAddr{Port: 6379})
	defer store.Close()
	ok, err := store.ping()
	if err != nil {
		t.Error(err.Error())
	}
	if !ok {
		t.Error("Expected server to PONG")
	}
}

func TestPingBadPort(t *testing.T) {
	store, _ := Init(&net.TCPAddr{Port: 6380})
	defer store.Close()
	_, err := store.ping()
	if err == nil {
		t.Error("Expected error")
	}
}

func BenchmarkGetSequential(b *testing.B) {
	store, _ := Init(&net.TCPAddr{Port: 6379})
	store.flush()

	defer store.Close()
	defer store.flush()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get("client_ip")
	}
}

func BenchmarkGetParallel(b *testing.B) {
	store, _ := Init(&net.TCPAddr{Port: 6379})
	store.flush()

	defer store.Close()
	defer store.flush()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = store.Get("client_ip")
			}
		})
	}
}

func ExampleGet() {
	store, err := Init(&net.TCPAddr{Port: 6379})
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
	// Quota: 5000
	// Used: 1
	// Remaining: 4999
	// Retry After: 2014-08-09 17:14:55 +0530 IST
}
