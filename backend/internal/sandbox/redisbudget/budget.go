// Package redisbudget implements a Redis-backed sandbox.Budget shared across
// worker replicas. Slots are ZSET members with expiry scores so crashed workers
// self-heal (hawk relay capacity pattern).
package redisbudget

import (
	"context"
	"sync"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/redis/go-redis/v9"
)

const (
	defaultKey           = "agentclash:sandbox:capacity"
	defaultSlotTTL       = 2 * time.Hour
	defaultPollInterval  = 100 * time.Millisecond
	defaultAcquireTimeout = 5 * time.Minute
)

// acquireScript prunes expired members then admits one if under max.
// KEYS[1]=zset key ARGV[1]=max ARGV[2]=member ARGV[3]=nowUnix ARGV[4]=expiryUnix
// Returns 1 on admit, 0 when full.
var acquireScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[3])
local n = redis.call('ZCARD', KEYS[1])
if tonumber(n) >= tonumber(ARGV[1]) then
  return 0
end
redis.call('ZADD', KEYS[1], ARGV[4], ARGV[2])
return 1
`)

var releaseScript = redis.NewScript(`
return redis.call('ZREM', KEYS[1], ARGV[1])
`)

// Config configures a Redis-backed budget.
type Config struct {
	Client         redis.Cmdable
	MaxConcurrent  int
	AcquireTimeout time.Duration
	SlotTTL        time.Duration
	Key            string
	PollInterval   time.Duration
	LeaseID        func() string
}

// Budget is a shared sandbox concurrency semaphore.
type Budget struct {
	client         redis.Cmdable
	max            int
	acquireTimeout time.Duration
	slotTTL        time.Duration
	key            string
	pollInterval   time.Duration
	leaseID        func() string
}

// New returns a Redis-backed Budget. max must be > 0.
func New(cfg Config) *Budget {
	if cfg.MaxConcurrent <= 0 {
		panic("redisbudget: MaxConcurrent must be > 0")
	}
	if cfg.Client == nil {
		panic("redisbudget: Client is required")
	}
	acquireTimeout := cfg.AcquireTimeout
	if acquireTimeout <= 0 {
		acquireTimeout = defaultAcquireTimeout
	}
	slotTTL := cfg.SlotTTL
	if slotTTL <= 0 {
		slotTTL = defaultSlotTTL
	}
	key := cfg.Key
	if key == "" {
		key = defaultKey
	}
	poll := cfg.PollInterval
	if poll <= 0 {
		poll = defaultPollInterval
	}
	leaseID := cfg.LeaseID
	if leaseID == nil {
		leaseID = sandbox.NewLeaseID
	}
	return &Budget{
		client:         cfg.Client,
		max:            cfg.MaxConcurrent,
		acquireTimeout: acquireTimeout,
		slotTTL:        slotTTL,
		key:            key,
		pollInterval:   poll,
		leaseID:        leaseID,
	}
}

func (b *Budget) Acquire(ctx context.Context) (func(), error) {
	ctx, cancel := context.WithTimeout(ctx, b.acquireTimeout)
	defer cancel()

	member := b.leaseID()
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	for {
		ok, err := b.tryAcquire(ctx, member)
		if err != nil {
			return nil, err
		}
		if ok {
			var once sync.Once
			return func() {
				once.Do(func() {
					releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer releaseCancel()
					_ = releaseScript.Run(releaseCtx, b.client, []string{b.key}, member).Err()
				})
			}, nil
		}
		select {
		case <-ctx.Done():
			return nil, sandbox.NewCapacityTimeoutError(5 * time.Second)
		case <-ticker.C:
		}
	}
}

func (b *Budget) tryAcquire(ctx context.Context, member string) (bool, error) {
	now := time.Now().Unix()
	expiry := time.Now().Add(b.slotTTL).Unix()
	res, err := acquireScript.Run(ctx, b.client, []string{b.key}, b.max, member, now, expiry).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}
