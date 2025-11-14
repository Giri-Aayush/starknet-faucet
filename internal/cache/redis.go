package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with faucet-specific operations
type RedisClient struct {
	client                *redis.Client
	maxDailyRequestsIP    int // Max requests per IP per day (5)
	maxChallengesPerHour  int // Max PoW challenges per IP per hour (8)
}

// NewRedisClient creates a new Redis client
func NewRedisClient(redisURL string, maxDailyRequestsIP, maxChallengesPerHour int) (*RedisClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client:                client,
		maxDailyRequestsIP:    maxDailyRequestsIP,
		maxChallengesPerHour:  maxChallengesPerHour,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Challenge-related operations

// StoreChallenge stores a challenge in Redis with TTL
func (r *RedisClient) StoreChallenge(ctx context.Context, challengeID, challenge string, ttl time.Duration) error {
	key := fmt.Sprintf("challenge:%s", challengeID)
	return r.client.Set(ctx, key, challenge, ttl).Err()
}

// GetChallenge retrieves a challenge from Redis
func (r *RedisClient) GetChallenge(ctx context.Context, challengeID string) (string, error) {
	key := fmt.Sprintf("challenge:%s", challengeID)
	return r.client.Get(ctx, key).Result()
}

// DeleteChallenge removes a challenge from Redis (prevents reuse)
func (r *RedisClient) DeleteChallenge(ctx context.Context, challengeID string) error {
	key := fmt.Sprintf("challenge:%s", challengeID)
	return r.client.Del(ctx, key).Err()
}

// New Simplified Rate Limiting Operations

// CheckIPDailyLimit checks if IP has exceeded daily request limit (5/day)
// Returns (canRequest, currentCount, error)
func (r *RedisClient) CheckIPDailyLimit(ctx context.Context, ip string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:ip:day:%s", ip)
	count, err := r.client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, 0, err
	}
	if count >= r.maxDailyRequestsIP {
		return false, count, nil
	}
	return true, count, nil
}

// IncrementIPDailyLimit increments IP daily counter by specified amount (1 for single token, 2 for BOTH)
func (r *RedisClient) IncrementIPDailyLimit(ctx context.Context, ip string, incrementBy int) error {
	key := fmt.Sprintf("ratelimit:ip:day:%s", ip)
	pipe := r.client.Pipeline()
	pipe.IncrBy(ctx, key, int64(incrementBy))
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// CheckTokenHourlyThrottle checks if a specific token was requested in the last hour
// Returns (canRequest, nextAvailableTime, error)
func (r *RedisClient) CheckTokenHourlyThrottle(ctx context.Context, ip, token string) (bool, *time.Time, error) {
	key := fmt.Sprintf("throttle:ip:token:%s:%s", ip, token)

	// Check if key exists
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}

	if exists == 0 {
		return true, nil, nil // No throttle active
	}

	// Get TTL to calculate when next request is available
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}

	nextAvailable := time.Now().Add(ttl)
	return false, &nextAvailable, nil
}

// SetTokenHourlyThrottle sets hourly throttle for a token (1 hour cooldown)
func (r *RedisClient) SetTokenHourlyThrottle(ctx context.Context, ip, token string) error {
	key := fmt.Sprintf("throttle:ip:token:%s:%s", ip, token)
	return r.client.Set(ctx, key, time.Now().Unix(), time.Hour).Err()
}

// GetIPDailyQuota returns current usage and remaining quota for an IP
func (r *RedisClient) GetIPDailyQuota(ctx context.Context, ip string) (used, remaining int, err error) {
	key := fmt.Sprintf("ratelimit:ip:day:%s", ip)
	count, err := r.client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	if err == redis.Nil {
		count = 0
	}
	remaining = r.maxDailyRequestsIP - count
	if remaining < 0 {
		remaining = 0
	}
	return count, remaining, nil
}


// Global distribution tracking (anti-drain protection)

// TrackGlobalDistribution tracks tokens distributed globally and checks limits
// If maxHour or maxDay is 0, that limit is disabled
func (r *RedisClient) TrackGlobalDistribution(ctx context.Context, tokenType string, amount float64, maxHour, maxDay float64) (bool, error) {
	// If both limits are 0, skip tracking entirely
	if maxHour == 0 && maxDay == 0 {
		return true, nil
	}

	hourlyKey := fmt.Sprintf("global:distributed:hour:%s", tokenType)
	dailyKey := fmt.Sprintf("global:distributed:day:%s", tokenType)

	// Check hourly limit (only if enabled)
	if maxHour > 0 {
		hourlyTotal, err := r.client.Get(ctx, hourlyKey).Float64()
		if err != nil && err != redis.Nil {
			return false, err
		}
		if hourlyTotal+amount > maxHour {
			return false, nil // Would exceed hourly limit
		}
	}

	// Check daily limit (only if enabled)
	if maxDay > 0 {
		dailyTotal, err := r.client.Get(ctx, dailyKey).Float64()
		if err != nil && err != redis.Nil {
			return false, err
		}
		if dailyTotal+amount > maxDay {
			return false, nil // Would exceed daily limit
		}
	}

	// Increment counters (only if limits are enabled)
	pipe := r.client.Pipeline()
	if maxHour > 0 {
		pipe.IncrByFloat(ctx, hourlyKey, amount)
		pipe.Expire(ctx, hourlyKey, time.Hour)
	}
	if maxDay > 0 {
		pipe.IncrByFloat(ctx, dailyKey, amount)
		pipe.Expire(ctx, dailyKey, 24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	return err == nil, err
}

// GetGlobalDistribution returns current global distribution totals
func (r *RedisClient) GetGlobalDistribution(ctx context.Context, tokenType string) (hourly, daily float64, err error) {
	hourlyKey := fmt.Sprintf("global:distributed:hour:%s", tokenType)
	dailyKey := fmt.Sprintf("global:distributed:day:%s", tokenType)

	hourly, err = r.client.Get(ctx, hourlyKey).Float64()
	if err == redis.Nil {
		hourly = 0
		err = nil
	} else if err != nil {
		return 0, 0, err
	}

	daily, err = r.client.Get(ctx, dailyKey).Float64()
	if err == redis.Nil {
		daily = 0
		err = nil
	}

	return hourly, daily, err
}

// Challenge rate limiting

// CheckChallengeRateLimit checks if an IP has exceeded challenge request limits
func (r *RedisClient) CheckChallengeRateLimit(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ratelimit:challenge:hour:%s", ip)
	count, err := r.client.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if count >= r.maxChallengesPerHour {
		return false, nil
	}
	return true, nil
}

// IncrementChallengeRateLimit increments the challenge rate limit counter for an IP
func (r *RedisClient) IncrementChallengeRateLimit(ctx context.Context, ip string) error {
	key := fmt.Sprintf("ratelimit:challenge:hour:%s", ip)
	pipe := r.client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}


// Health check

// Ping checks if Redis is responsive
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
