package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements Client interface using Redis
type RedisCache struct {
	client *redis.Client
	config *Config
	logger Logger
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *Config, logger Logger) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		PoolTimeout:  config.PoolTimeout,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	cache := &RedisCache{
		client: rdb,
		config: config,
		logger: logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return cache, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrKeyNotFound
		}
		return nil, &Error{Operation: "get", Key: key, Err: err}
	}
	return result, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = r.config.DefaultTTL
	}

	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return &Error{Operation: "set", Key: key, Err: err}
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return &Error{Operation: "delete", Key: key, Err: err}
	}
	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, &Error{Operation: "exists", Key: key, Err: err}
	}
	return result > 0, nil
}

func (r *RedisCache) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, &Error{Operation: "mget", Err: err}
	}

	data := make(map[string][]byte)
	for i, val := range result {
		if val != nil {
			if str, ok := val.(string); ok {
				data[keys[i]] = []byte(str)
			}
		}
	}

	return data, nil
}

func (r *RedisCache) SetMultiple(ctx context.Context, items map[string]Item) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, item := range items {
		ttl := item.TTL
		if ttl == 0 {
			ttl = r.config.DefaultTTL
		}
		pipe.Set(ctx, item.Key, item.Value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return &Error{Operation: "mset", Err: err}
	}
	return nil
}

func (r *RedisCache) DeleteMultiple(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		return &Error{Operation: "mdel", Err: err}
	}
	return nil
}

func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return &Error{Operation: "delete_pattern", Err: err}
	}

	if len(keys) > 0 {
		return r.DeleteMultiple(ctx, keys)
	}
	return nil
}

func (r *RedisCache) GetKeys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, &Error{Operation: "get_keys", Err: err}
	}
	return keys, nil
}

func (r *RedisCache) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incrCmd := pipe.IncrBy(ctx, key, delta)

	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	} else if ttl == 0 && r.config.DefaultTTL > 0 {
		pipe.Expire(ctx, key, r.config.DefaultTTL)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, &Error{Operation: "increment", Key: key, Err: err}
	}

	return incrCmd.Val(), nil
}

func (r *RedisCache) Decrement(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	return r.Increment(ctx, key, -delta, ttl)
}

func (r *RedisCache) HGet(ctx context.Context, key, field string) ([]byte, error) {
	result, err := r.client.HGet(ctx, key, field).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrKeyNotFound
		}
		return nil, &Error{Operation: "hget", Key: key, Err: err}
	}
	return result, nil
}

func (r *RedisCache) HSet(ctx context.Context, key, field string, value []byte, ttl time.Duration) error {
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, field, value)

	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	} else if ttl == 0 && r.config.DefaultTTL > 0 {
		pipe.Expire(ctx, key, r.config.DefaultTTL)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return &Error{Operation: "hset", Key: key, Err: err}
	}
	return nil
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, &Error{Operation: "hgetall", Key: key, Err: err}
	}

	data := make(map[string][]byte)
	for field, value := range result {
		data[field] = []byte(value)
	}

	return data, nil
}

func (r *RedisCache) HDelete(ctx context.Context, key string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}

	err := r.client.HDel(ctx, key, fields...).Err()
	if err != nil {
		return &Error{Operation: "hdel", Key: key, Err: err}
	}
	return nil
}

// SAdd adds one or more members to a set stored at key
// Returns the number of elements that were added to the set, not including all the elements already present
func (r *RedisCache) SAdd(ctx context.Context, key string, members ...[]byte) (int64, error) {
	// Convert []byte members to []interface{} for go-redis
	args := make([]interface{}, len(members))
	for i, member := range members {
		args[i] = member
	}

	// Call Redis SADD command
	result, err := r.client.SAdd(ctx, key, args...).Result()
	if err != nil {
		return 0, &Error{Operation: "sadd", Key: key, Err: err}
	}

	// Set expiration if DefaultTTL is configured
	if r.config.DefaultTTL > 0 {
		// Only set expiration for new keys - the Expire command accepts a NX option
		// but it's not available in the go-redis client, so we check if the set just got its first element
		if result > 0 {
			// Check set size - if it matches the number of elements we just added,
			// then this is a new key and we should set expiration
			size, err := r.client.SCard(ctx, key).Result()
			if err == nil && size == result {
				r.client.Expire(ctx, key, r.config.DefaultTTL)
			}
		}
	}

	return result, nil
}

// SRem removes one or more members from a set stored at key
// Returns the number of members that were removed from the set, not including non-existing members
func (r *RedisCache) SRem(ctx context.Context, key string, members ...[]byte) (int64, error) {
	// Convert []byte members to []interface{} for go-redis
	args := make([]interface{}, len(members))
	for i, member := range members {
		args[i] = member
	}

	// Call Redis SREM command
	result, err := r.client.SRem(ctx, key, args...).Result()
	if err != nil {
		return 0, &Error{Operation: "srem", Key: key, Err: err}
	}

	return result, nil
}

// SMembers returns all members of the set stored at key
func (r *RedisCache) SMembers(ctx context.Context, key string) ([][]byte, error) {
	// Call Redis SMEMBERS command
	strSlice, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, &Error{Operation: "smembers", Key: key, Err: err}
	}

	// Convert string slice to [][]byte
	result := make([][]byte, len(strSlice))
	for i, str := range strSlice {
		result[i] = []byte(str)
	}

	return result, nil
}

// SIsMember checks if member is a member of the set stored at key
func (r *RedisCache) SIsMember(ctx context.Context, key string, member []byte) (bool, error) {
	// Call Redis SISMEMBER command
	result, err := r.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		return false, &Error{Operation: "sismember", Key: key, Err: err}
	}

	return result, nil
}

// SCard returns the cardinality (number of elements) of the set stored at key
func (r *RedisCache) SCard(ctx context.Context, key string) (int64, error) {
	// Call Redis SCARD command
	result, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		return 0, &Error{Operation: "scard", Key: key, Err: err}
	}

	return result, nil
}

func (r *RedisCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := LockKey(key)
	result, err := r.client.SetNX(ctx, lockKey, "locked", ttl).Result()
	if err != nil {
		return false, &Error{Operation: "lock", Key: key, Err: err}
	}
	return result, nil
}

func (r *RedisCache) Unlock(ctx context.Context, key string) error {
	lockKey := LockKey(key)
	err := r.client.Del(ctx, lockKey).Err()
	if err != nil {
		return &Error{Operation: "unlock", Key: key, Err: err}
	}
	return nil
}

func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	result, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, &Error{Operation: "ttl", Key: key, Err: err}
	}
	return result, nil
}

func (r *RedisCache) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	err := r.client.Expire(ctx, key, ttl).Err()
	if err != nil {
		return &Error{Operation: "expire", Key: key, Err: err}
	}
	return nil
}

func (r *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, data, ttl)
}

func (r *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (r *RedisCache) FlushAll(ctx context.Context) error {
	err := r.client.FlushAll(ctx).Err()
	if err != nil {
		return &Error{Operation: "flushall", Err: err}
	}
	return nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	err := r.client.Ping(ctx).Err()
	if err != nil {
		return &Error{Operation: "ping", Err: err}
	}
	return nil
}

func (r *RedisCache) Stats(ctx context.Context) (Stats, error) {
	info, err := r.client.Info(ctx, "memory", "stats", "server").Result()
	if err != nil {
		return Stats{}, &Error{Operation: "stats", Err: err}
	}

	poolStats := r.client.PoolStats()

	// Parse Redis info for detailed stats
	stats := Stats{
		Connections: int64(poolStats.TotalConns),
		Metadata: map[string]string{
			"redis_version":     r.parseInfoField(info, "redis_version"),
			"used_memory":       r.parseInfoField(info, "used_memory"),
			"uptime_in_seconds": r.parseInfoField(info, "uptime_in_seconds"),
		},
	}

	// Get key count
	dbSize, err := r.client.DBSize(ctx).Result()
	if err == nil {
		stats.Keys = dbSize
	}

	return stats, nil
}

// Helper method to parse Redis INFO response
func (r *RedisCache) parseInfoField(info, field string) string {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, field+":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}
