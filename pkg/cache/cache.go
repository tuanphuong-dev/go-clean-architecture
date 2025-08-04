package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Provider string

const (
	Redis  Provider = "redis"
	Memory Provider = "memory"
)

var (
	ErrKeyNotFound    = errors.New("key not found")
	ErrKeyExists      = errors.New("key already exists")
	ErrInvalidTTL     = errors.New("invalid TTL")
	ErrLockFailed     = errors.New("failed to acquire lock")
	ErrConnectionFail = errors.New("cache connection failed")
	ErrSerialization  = errors.New("serialization failed")
)

type Error struct {
	Operation string
	Key       string
	Err       error
}

func (e *Error) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("cache %s operation failed for key '%s': %v", e.Operation, e.Key, e.Err)
	}
	return fmt.Sprintf("cache %s operation failed: %v", e.Operation, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Client interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error)
	SetMultiple(ctx context.Context, items map[string]Item) error
	DeleteMultiple(ctx context.Context, keys []string) error

	DeletePattern(ctx context.Context, pattern string) error
	GetKeys(ctx context.Context, pattern string) ([]string, error)

	Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)
	Decrement(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)

	HGet(ctx context.Context, key, field string) ([]byte, error)
	HSet(ctx context.Context, key, field string, value []byte, ttl time.Duration) error
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
	HDelete(ctx context.Context, key string, fields ...string) error

	SAdd(ctx context.Context, key string, members ...[]byte) (int64, error)
	SRem(ctx context.Context, key string, members ...[]byte) (int64, error)
	SMembers(ctx context.Context, key string) ([][]byte, error)
	SIsMember(ctx context.Context, key string, member []byte) (bool, error)

	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error

	GetTTL(ctx context.Context, key string) (time.Duration, error)
	SetTTL(ctx context.Context, key string, ttl time.Duration) error

	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error

	FlushAll(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
	Stats(ctx context.Context) (Stats, error)
}

type Item struct {
	Key   string
	Value []byte
	TTL   time.Duration
}

type Stats struct {
	Hits        int64             `json:"hits"`
	Misses      int64             `json:"misses"`
	HitRate     float64           `json:"hit_rate"`
	Keys        int64             `json:"keys"`
	Memory      int64             `json:"memory_bytes"`
	Connections int64             `json:"connections"`
	Uptime      time.Duration     `json:"uptime"`
	Metadata    map[string]string `json:"metadata"`
}

type Config struct {
	// Connection settings
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`

	// Pool settings
	PoolSize     int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	PoolTimeout  time.Duration `json:"pool_timeout" yaml:"pool_timeout"`

	// Operation settings
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// Default TTL
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`

	// Serialization
	Serializer string `json:"serializer" yaml:"serializer"` // json, msgpack, gob

	// Memory cache settings (for memory implementation)
	MaxSize int           `json:"max_size" yaml:"max_size"`
	TTL     time.Duration `json:"ttl" yaml:"ttl"`
}

type Factory struct {
	logger Logger
}

// NewCacheFactory creates a new cache factory
func NewCacheFactory(logger Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateCache creates a cache instance based on the configuration
func (f *Factory) CreateCache(cacheType Provider, config *Config) (Client, error) {
	switch cacheType {
	case Redis:
		return f.createRedisCache(config)
	case Memory:
		return f.createMemoryCache(config)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
}

// createRedisCache creates a Redis cache instance
func (f *Factory) createRedisCache(config *Config) (Client, error) {
	// Set default values if not provided
	f.setRedisDefaults(config)

	cache, err := NewRedisCache(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis cache: %w", err)
	}

	f.logger.Info("Redis cache created successfully",
		"host", config.Host,
		"port", config.Port,
		"db", config.DB,
		"pool_size", config.PoolSize,
		"default_ttl", config.DefaultTTL.String(),
	)

	return cache, nil
}

// createMemoryCache creates an in-memory cache instance
func (f *Factory) createMemoryCache(config *Config) (Client, error) {
	// Set default values if not provided
	f.setMemoryDefaults(config)

	cache := NewMemoryCache(config, f.logger)

	f.logger.Info("Memory cache created successfully",
		"max_size", config.MaxSize,
		"default_ttl", config.DefaultTTL.String(),
	)

	return cache, nil
}

// setRedisDefaults sets default values for Redis configuration
func (f *Factory) setRedisDefaults(config *Config) {
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 6379
	}
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 2
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = 4 * time.Second
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 1 * time.Hour
	}
	if config.Serializer == "" {
		config.Serializer = "json"
	}
}

// setMemoryDefaults sets default values for memory cache configuration
func (f *Factory) setMemoryDefaults(config *Config) {
	if config.MaxSize == 0 {
		config.MaxSize = 1000 // Default max 1000 items
	}
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 5 * time.Minute // Shorter TTL for memory cache
	}
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
}

// CacheBuilder provides a fluent interface for building cache configurations
type CacheBuilder struct {
	config *Config
	logger Logger
}

// NewCacheBuilder creates a new cache builder
func NewCacheBuilder(logger Logger) *CacheBuilder {
	return &CacheBuilder{
		config: &Config{},
		logger: logger,
	}
}

// WithRedis configures Redis connection settings
func (b *CacheBuilder) WithRedis(host string, port int, password string, db int) *CacheBuilder {
	b.config.Host = host
	b.config.Port = port
	b.config.Password = password
	b.config.DB = db
	return b
}

// WithPool configures connection pool settings
func (b *CacheBuilder) WithPool(size, minIdle int, poolTimeout time.Duration) *CacheBuilder {
	b.config.PoolSize = size
	b.config.MinIdleConns = minIdle
	b.config.PoolTimeout = poolTimeout
	return b
}

// WithTimeouts configures operation timeouts
func (b *CacheBuilder) WithTimeouts(dial, read, write time.Duration) *CacheBuilder {
	b.config.DialTimeout = dial
	b.config.ReadTimeout = read
	b.config.WriteTimeout = write
	return b
}

// WithTTL configures default TTL
func (b *CacheBuilder) WithTTL(ttl time.Duration) *CacheBuilder {
	b.config.DefaultTTL = ttl
	return b
}

// WithMemory configures memory cache settings
func (b *CacheBuilder) WithMemory(maxSize int, ttl time.Duration) *CacheBuilder {
	b.config.MaxSize = maxSize
	b.config.TTL = ttl
	return b
}

// WithSerialization configures serialization method
func (b *CacheBuilder) WithSerialization(method string) *CacheBuilder {
	b.config.Serializer = method
	return b
}

// Build creates the cache instance
func (b *CacheBuilder) Build(cacheType Provider) (Client, error) {
	factory := NewCacheFactory(b.logger)
	return factory.CreateCache(cacheType, b.config)
}

// BuildRedis creates a Redis cache instance
func (b *CacheBuilder) BuildRedis() (Client, error) {
	return b.Build(Redis)
}

// BuildMemory creates a memory cache instance
func (b *CacheBuilder) BuildMemory() (Client, error) {
	return b.Build(Memory)
}

// GetCacheFromConfig creates a cache from environment configuration
func GetCacheFromConfig(config *Config, logger Logger) (Client, error) {
	factory := NewCacheFactory(logger)

	// Determine cache type based on configuration
	var cacheType Provider
	if config.Host != "" {
		cacheType = Redis
	} else {
		cacheType = Memory
	}

	return factory.CreateCache(cacheType, config)
}
