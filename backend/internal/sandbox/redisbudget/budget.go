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
	defaultKey            = "agentclash:sandbox:capacity"
	defaultSlotTTL        = 2 * time.Hour
	defaultPollInterval   = 100 * time.Millisecond
	defaultAcquireTimeout = 5 * time.Minute
	minRenewInterval      = 30 * time.Second
)

const (
	acquireScriptSource = `
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[3])
local n = redis.call('ZCARD', KEYS[1])
if tonumber(n) >= tonumber(ARGV[1]) then
  return 0
end
redis.call('ZADD', KEYS[1], ARGV[4], ARGV[2])
return 1
`
	releaseScriptSource = `
return redis.call('ZREM', KEYS[1], ARGV[1])
`
	renewScriptSource = `
if redis.call('ZSCORE', KEYS[1], ARGV[1]) then
  redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
  return 1
end
return 0
`
)

// Config configures a Redis-backed budget.
type Config struct {
	Client         redis.Cmdable
	MaxConcurrent  int
	AcquireTimeout time.Duration
	SlotTTL        time.Duration
	RenewInterval  time.Duration
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
	renewInterval  time.Duration
	key            string
	pollInterval   time.Duration
	leaseID        func() string
	acquireScript  *redis.Script
	releaseScript  *redis.Script
	renewScript    *redis.Script
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
	renewInterval := cfg.RenewInterval
	if renewInterval <= 0 {
		renewInterval = slotTTL / 3
		if renewInterval < minRenewInterval {
			renewInterval = minRenewInterval
		}
		if renewInterval >= slotTTL {
			renewInterval = slotTTL / 2
		}
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
		renewInterval:  renewInterval,
		key:            key,
		pollInterval:   poll,
		leaseID:        leaseID,
		acquireScript:  redis.NewScript(acquireScriptSource),
		releaseScript:  redis.NewScript(releaseScriptSource),
		renewScript:    redis.NewScript(renewScriptSource),
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
			stopHeartbeat := make(chan struct{})
			var heartbeatWG sync.WaitGroup
			heartbeatWG.Add(1)
			go b.runHeartbeat(member, stopHeartbeat, &heartbeatWG)

			var once sync.Once
			return func() {
				once.Do(func() {
					close(stopHeartbeat)
					heartbeatWG.Wait()
					releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer releaseCancel()
					_ = b.releaseScript.Run(releaseCtx, b.client, []string{b.key}, member).Err()
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
	res, err := b.acquireScript.Run(ctx, b.client, []string{b.key}, b.max, member, now, expiry).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

func (b *Budget) renew(ctx context.Context, member string) error {
	expiry := time.Now().Add(b.slotTTL).Unix()
	_, err := b.renewScript.Run(ctx, b.client, []string{b.key}, member, expiry).Int()
	return err
}

func (b *Budget) runHeartbeat(member string, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(b.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := b.renew(ctx, member)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
