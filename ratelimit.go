package ratelimit

// module dependencies
import "net"
import "time"
import "github.com/garyburd/redigo/redis"

// module constants
const MaxQuota = 5000
const LimitInterval = 3600 * time.Second
const MilliSecond = int64(time.Millisecond)
const MaxIdle = 40
const IdleTimeout = 240 * time.Second

// `RedisLimiter` Limits connections based
// on a max quota that they (`end_user_of_api`)
// are entitled to, and the quota that they  have
// already consumed. Useful for Web Applications,
// to avoid abuse of APIs etc, by limiting everyone
// to a max quota of ex: 5000 requests per hour per
// `identifier`. The `identifier` has to be choosen
// by the `user` cosuming `RedisLimiter`
type RedisLimiter struct {
	Pool            *redis.Pool
	PrefixQuota     string
	PrefixRemaining string
	PrefixReset     string
	Duration        time.Duration
	Quota           int
}

// `Limit` struct defines the model of `Limiter`.
// essential fields include:
//
// 		- `Quota`
// 		- `Used`
// 		- `Remaining`
// 		- `RetryAfter` epoch timestamp
type Limit struct {
	Quota      int
	Used       int
	Remaining  int
	RetryAfter time.Time
}

// `Bucket` defines the model for
// `Quota`, `Used` & `Remaining`
type Bucket struct {
	id    string
	value int
}

// `RetryAfter` defines for `self`
type RetryAfter struct {
	id    string
	value int64
}

// Init returns a new RedisLimiter.
// Options:
//   - `address` net.Addr
//
// @return *RedisLimiter, error
func Init(address net.Addr) (*RedisLimiter, error) {
	rl := &RedisLimiter{
		// http://godoc.org/github.com/garyburd/redigo/redis#Pool
		Pool: &redis.Pool{
			MaxIdle:     MaxIdle,
			IdleTimeout: IdleTimeout,
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial(address.Network(), address.String())
				if err != nil {
					return nil, err
				}

				return c, err
			},
		},
		PrefixQuota:     "RateLimit:Quota:",
		PrefixRemaining: "RateLimit:Remaining:",
		PrefixReset:     "RateLimit:Reset:",
		Duration:        LimitInterval,
		Quota:           MaxQuota,
	}

	_, err := rl.ping()
	return rl, err
}

// ping does an internal ping against
// `Redis` to check if it is alive.
func (s *RedisLimiter) ping() (bool, error) {
	conn := s.Pool.Get()
	defer conn.Close()

	data, err := conn.Do("PING")
	if err != nil || data == nil {
		return false, err
	}

	return (data == "PONG"), nil
}

// Get `Limit` for an `identifier` from Redis
// and implicitly apply `RateLimit` calculations
// against this `identifier` back into Redis.
func (s *RedisLimiter) Get(id string) (*Limit, error) {
	// Local variables
	var (
		err   error
		reply []interface{}
	)

	// Fetch Redis Pool
	conn := s.Pool.Get()
	defer conn.Close()
	if err = conn.Err(); err != nil {
		return nil, err
	}

	// create `Bucket` for holding `id` & `value`
	quota := &Bucket{id: s.PrefixQuota + id}
	remaining := &Bucket{id: s.PrefixRemaining + id}
	reset := &RetryAfter{id: s.PrefixReset + id}

	// calculate `expiry` based on: time.Now() + LimitInterval
	expiry := (time.Now().UnixNano()/MilliSecond + int64(s.Duration)/MilliSecond) / 1000

	// WATCH for changes at key:remaining
	// also, defer to UNWATCH
	conn.Send("WATCH", remaining)
	defer conn.Send("UNWATCH", remaining)

	// fetch quota, remaining & reset time
	// to check if it already exists.
	//
	// if already exists, use these values
	// and decrement. else, create them
	if reply, err = redis.Values(conn.Do("MGET", quota.id, remaining.id, reset.id)); err != nil {
		return nil, err
	}

	// get the above MGET values copied into the respective variables
	if _, err = redis.Scan(reply, &quota.value, &remaining.value, &reset.value); err != nil {
		return nil, err
	}

	// as noted above, keys do not exist
	// which means that we've to create
	// them now, to use it further
	if quota.value == 0 {
		// @TODO: can we do this better?
		// EX: expire in time.Seconds
		// NX: create, only if it does not exist
		conn.Send("MULTI")
		conn.Send("SET", quota.id, s.Quota, "EX", s.Duration.Seconds(), "NX")
		conn.Send("SET", remaining.id, s.Quota-1, "EX", s.Duration.Seconds(), "NX")
		conn.Send("SET", reset.id, expiry, "EX", s.Duration.Seconds(), "NX")

		if reply, err = redis.Values(conn.Do("EXEC")); err != nil {
			return nil, err
		} else if reply[0] != "OK" {
			// @TODO: race condition
		}

		// copy the values, to use them in a uniform way
		quota.value = s.Quota
		remaining.value = s.Quota - 1
		reset.value = expiry
	} else if remaining.value > 0 {
		// keys exist, and the remaining value
		// is greater than 0. decrement it.
		conn.Do("DECR", remaining.id)
		remaining.value--
	}
	// we do not have to handle the case of
	// remaining value <= 0, since the MGET
	// command initialized it to 0 for us.

	// finally, return the bucket
	// holding the `Limit` values
	return &Limit{
		Quota:      quota.value,
		Used:       quota.value - remaining.value,
		Remaining:  remaining.value,
		RetryAfter: time.Unix(reset.value, 0),
	}, nil
}

// Close closes the underlying *redis.Pool
func (s *RedisLimiter) Close() error {
	return s.Pool.Close()
}
