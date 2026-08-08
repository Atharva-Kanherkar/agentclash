package throttle

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

const (
	redisKeyPrefix     = "agentclash:provider:throttle:"
	cooldownKeyPrefix  = "agentclash:provider:cooldown:"
	concurrentKeyPref  = "agentclash:provider:inflight:"
)

// RedisLimiter shares budgets across worker replicas.
type RedisLimiter struct {
	cfg     Config
	rdb     redis.Cmdable
	limiter *redis_rate.Limiter
}

// NewRedisLimiter constructs a Redis-backed limiter.
func NewRedisLimiter(rdb redis.Cmdable, cfg Config) *RedisLimiter {
	return &RedisLimiter{
		cfg:     cfg,
		rdb:     rdb,
		limiter: redis_rate.NewLimiter(rdb),
	}
}

func (l *RedisLimiter) CoolDown(key Key, d time.Duration) {
	if d <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ck := cooldownKeyPrefix + key.String()
	// Set + PExpire is more reliable across Redis stubs than Set(key,val,ttl)
	// alone (some test doubles ignore the expiration argument on Set).
	_ = l.rdb.Set(ctx, ck, "1", 0).Err()
	_ = l.rdb.PExpire(ctx, ck, d).Err()
}

func (l *RedisLimiter) Acquire(ctx context.Context, key Key, estimatedTokens int64) (Lease, error) {
	limits := l.cfg.limitsFor(key.Provider)
	if !limits.Enabled() {
		return noopLease{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, l.cfg.acquireTimeout())
	defer cancel()

	id := key.String()

	for {
		if err := ctx.Err(); err != nil {
			return nil, ErrAcquireTimeout
		}
		// Cooldown
		ck := cooldownKeyPrefix + id
		exists, err := l.rdb.Exists(ctx, ck).Result()
		if err == nil && exists > 0 {
			ttl, ttlErr := l.rdb.PTTL(ctx, ck).Result()
			if ttlErr != nil || ttl <= 0 {
				ttl = 50 * time.Millisecond
			}
			timer := time.NewTimer(ttl)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ErrAcquireTimeout
			case <-timer.C:
			}
			continue
		}

		if limits.MaxConcurrent > 0 {
			n, err := l.rdb.Incr(ctx, concurrentKeyPref+id).Result()
			if err != nil {
				return nil, err
			}
			_ = l.rdb.Expire(ctx, concurrentKeyPref+id, time.Hour).Err()
			if n > int64(limits.MaxConcurrent) {
				_ = l.rdb.Decr(ctx, concurrentKeyPref+id).Err()
				timer := time.NewTimer(50 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ErrAcquireTimeout
				case <-timer.C:
				}
				continue
			}
		}

		// Reserve TPM before consuming an RPM token — redis_rate.Allow is not
		// refundable, so a later TPM reject would permanently burn RPM budget.
		reserved := estimatedTokens
		if reserved < 0 {
			reserved = 0
		}
		if limits.TPM > 0 && reserved > 0 {
			ok, err := l.reserveTPM(ctx, id, limits.TPM, reserved)
			if err != nil {
				l.releaseConcurrent(ctx, id, limits)
				return nil, err
			}
			if !ok {
				l.releaseConcurrent(ctx, id, limits)
				timer := time.NewTimer(50 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ErrAcquireTimeout
				case <-timer.C:
				}
				continue
			}
		}

		if limits.RPM > 0 {
			res, err := l.limiter.Allow(ctx, redisKeyPrefix+"rpm:"+id, redis_rate.PerMinute(limits.RPM))
			if err != nil {
				if limits.TPM > 0 && reserved > 0 {
					l.returnTPM(ctx, id, reserved)
				}
				l.releaseConcurrent(ctx, id, limits)
				return nil, err
			}
			if res.Allowed == 0 {
				if limits.TPM > 0 && reserved > 0 {
					l.returnTPM(ctx, id, reserved)
				}
				l.releaseConcurrent(ctx, id, limits)
				wait := res.RetryAfter
				if wait <= 0 {
					wait = 50 * time.Millisecond
				}
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ErrAcquireTimeout
				case <-timer.C:
				}
				continue
			}
		}

		return &redisLease{
			limiter:  l,
			key:      id,
			limits:   limits,
			reserved: reserved,
		}, nil
	}
}

func (l *RedisLimiter) releaseConcurrent(ctx context.Context, id string, limits Limits) {
	if limits.MaxConcurrent > 0 {
		_ = l.rdb.Decr(ctx, concurrentKeyPref+id).Err()
	}
}

// reserveTPM uses a minute sliding counter of tokens.
func (l *RedisLimiter) reserveTPM(ctx context.Context, id string, tpm int64, tokens int64) (bool, error) {
	key := redisKeyPrefix + "tpm:" + id
	// Simple fixed-window: INCRBY + EXPIRE 60s; reject if > tpm.
	n, err := l.rdb.IncrBy(ctx, key, tokens).Result()
	if err != nil {
		return false, err
	}
	if n == tokens {
		_ = l.rdb.Expire(ctx, key, time.Minute).Err()
	}
	if n > tpm {
		_ = l.rdb.DecrBy(ctx, key, tokens).Err()
		return false, nil
	}
	return true, nil
}

func (l *RedisLimiter) returnTPM(ctx context.Context, id string, tokens int64) {
	if tokens <= 0 {
		return
	}
	key := redisKeyPrefix + "tpm:" + id
	_ = l.rdb.DecrBy(ctx, key, tokens).Err()
}

type redisLease struct {
	limiter  *RedisLimiter
	key      string
	limits   Limits
	reserved int64
	once     sync.Once
}

func (l *redisLease) Reconcile(actualTokens int64) {
	if l.limits.TPM <= 0 {
		return
	}
	if actualTokens < 0 {
		actualTokens = 0
	}
	surplus := l.reserved - actualTokens
	if surplus > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		l.limiter.returnTPM(ctx, l.key, surplus)
	}
	l.reserved = 0
}

func (l *redisLease) Release() {
	l.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if l.limits.TPM > 0 && l.reserved > 0 {
			l.limiter.returnTPM(ctx, l.key, l.reserved)
			l.reserved = 0
		}
		l.limiter.releaseConcurrent(ctx, l.key, l.limits)
	})
}
