package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with faucet-specific operations
type RedisClient struct {
	client         *redis.Client
	cooldownHours  int
	maxPerHour     int
	maxPerDay      int
	maxChallengesPerHour int
}

// NewRedisClient creates a new Redis client
func NewRedisClient(redisURL string, cooldownHours, maxPerHour, maxPerDay, maxChallengesPerHour int) (*RedisClient, error) {
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
		client:               client,
		cooldownHours:        cooldownHours,
		maxPerHour:           maxPerHour,
		maxPerDay:            maxPerDay,
		maxChallengesPerHour: maxChallengesPerHour,
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

// Cooldown-related operations

// SetAddressCooldown sets the cooldown period for an address
func (r *RedisClient) SetAddressCooldown(ctx context.Context, address string) error {
	key := fmt.Sprintf("cooldown:address:%s", address)
	ttl := time.Duration(r.cooldownHours) * time.Hour
	return r.client.Set(ctx, key, time.Now().Unix(), ttl).Err()
}

// IsAddressInCooldown checks if an address is in cooldown
func (r *RedisClient) IsAddressInCooldown(ctx context.Context, address string) (bool, *time.Time, error) {
	key := fmt.Sprintf("cooldown:address:%s", address)

	// Check if key exists
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}

	if exists == 0 {
		return false, nil, nil
	}

	// Get TTL
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}

	// Calculate next request time
	nextRequestTime := time.Now().Add(ttl)

	return true, &nextRequestTime, nil
}

// Rate limiting operations

// CheckAddressRateLimit checks if an address has exceeded rate limits
func (r *RedisClient) CheckAddressRateLimit(ctx context.Context, address string) (bool, error) {
	// Check hourly limit
	hourlyKey := fmt.Sprintf("ratelimit:address:hour:%s", address)
	hourlyCount, err := r.client.Get(ctx, hourlyKey).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if hourlyCount >= r.maxPerHour {
		return false, nil
	}

	// Check daily limit
	dailyKey := fmt.Sprintf("ratelimit:address:day:%s", address)
	dailyCount, err := r.client.Get(ctx, dailyKey).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if dailyCount >= r.maxPerDay {
		return false, nil
	}

	return true, nil
}

// IncrementAddressRateLimit increments the rate limit counters for an address
func (r *RedisClient) IncrementAddressRateLimit(ctx context.Context, address string) error {
	// Increment hourly counter
	hourlyKey := fmt.Sprintf("ratelimit:address:hour:%s", address)
	pipe := r.client.Pipeline()
	pipe.Incr(ctx, hourlyKey)
	pipe.Expire(ctx, hourlyKey, time.Hour)

	// Increment daily counter
	dailyKey := fmt.Sprintf("ratelimit:address:day:%s", address)
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// CheckIPRateLimit checks if an IP has exceeded rate limits (deprecated, kept for backwards compatibility)
func (r *RedisClient) CheckIPRateLimit(ctx context.Context, ip string) (bool, error) {
	// Check hourly limit
	hourlyKey := fmt.Sprintf("ratelimit:ip:hour:%s", ip)
	hourlyCount, err := r.client.Get(ctx, hourlyKey).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if hourlyCount >= r.maxPerHour {
		return false, nil
	}

	// Check daily limit
	dailyKey := fmt.Sprintf("ratelimit:ip:day:%s", ip)
	dailyCount, err := r.client.Get(ctx, dailyKey).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if dailyCount >= r.maxPerDay {
		return false, nil
	}

	return true, nil
}

// IncrementIPRateLimit increments the rate limit counters for an IP (deprecated, kept for backwards compatibility)
func (r *RedisClient) IncrementIPRateLimit(ctx context.Context, ip string) error {
	// Increment hourly counter
	hourlyKey := fmt.Sprintf("ratelimit:ip:hour:%s", ip)
	pipe := r.client.Pipeline()
	pipe.Incr(ctx, hourlyKey)
	pipe.Expire(ctx, hourlyKey, time.Hour)

	// Increment daily counter
	dailyKey := fmt.Sprintf("ratelimit:ip:day:%s", ip)
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
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
