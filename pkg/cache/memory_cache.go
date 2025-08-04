package cache

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryCache implements Client interface using in-memory storage
type MemoryCache struct {
	data      map[string]*memoryItem
	mu        sync.RWMutex
	config    *Config
	logger    Logger
	stopCh    chan struct{}
	stats     *memoryStats
	startTime time.Time
}

// memoryItem represents a cache item in memory
type memoryItem struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
}

// memoryStats tracks cache statistics
type memoryStats struct {
	hits   int64
	misses int64
	mu     sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache(config *Config, logger Logger) *MemoryCache {
	cache := &MemoryCache{
		data:      make(map[string]*memoryItem),
		config:    config,
		logger:    logger,
		stopCh:    make(chan struct{}),
		stats:     &memoryStats{},
		startTime: time.Now(),
	}
	go cache.cleanupExpired()
	return cache
}

// cleanupExpired removes expired items periodically
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCh:
			return
		}
	}
}

// cleanup removes expired items
func (m *MemoryCache) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, item := range m.data {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		delete(m.data, key)
	}

	if len(expired) > 0 {
		m.logger.Debugf("Cleaned up expired cache items: expired_count=%d", len(expired))
	}
}

// isExpired checks if an item is expired
func (m *MemoryCache) isExpired(item *memoryItem) bool {
	if item.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(item.expiresAt)
}

// recordHit increments hit counter
func (m *MemoryCache) recordHit() {
	m.stats.mu.Lock()
	m.stats.hits++
	m.stats.mu.Unlock()
}

// recordMiss increments miss counter
func (m *MemoryCache) recordMiss() {
	m.stats.mu.Lock()
	m.stats.misses++
	m.stats.mu.Unlock()
}

func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists || m.isExpired(item) {
		m.recordMiss()
		return nil, ErrKeyNotFound
	}

	m.recordHit()

	// Return copy to prevent external modification
	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, nil
}

func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Copy value to prevent external modification
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	item := &memoryItem{
		value:     valueCopy,
		expiresAt: expiresAt,
		createdAt: time.Now(),
	}

	m.mu.Lock()
	m.data[key] = item
	m.mu.Unlock()

	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}

func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists || m.isExpired(item) {
		return false, nil
	}
	return true, nil
}

func (m *MemoryCache) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	m.mu.RLock()
	for _, key := range keys {
		if item, exists := m.data[key]; exists && !m.isExpired(item) {
			valueCopy := make([]byte, len(item.value))
			copy(valueCopy, item.value)
			result[key] = valueCopy
			m.recordHit()
		} else {
			m.recordMiss()
		}
	}
	m.mu.RUnlock()

	return result, nil
}

func (m *MemoryCache) SetMultiple(ctx context.Context, items map[string]Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range items {
		ttl := item.TTL
		if ttl == 0 {
			ttl = m.config.DefaultTTL
		}

		var expiresAt time.Time
		if ttl > 0 {
			expiresAt = time.Now().Add(ttl)
		}

		valueCopy := make([]byte, len(item.Value))
		copy(valueCopy, item.Value)

		m.data[item.Key] = &memoryItem{
			value:     valueCopy,
			expiresAt: expiresAt,
			createdAt: time.Now(),
		}
	}

	return nil
}

func (m *MemoryCache) DeleteMultiple(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := m.GetKeys(ctx, pattern)
	if err != nil {
		return err
	}
	return m.DeleteMultiple(ctx, keys)
}

func (m *MemoryCache) GetKeys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key := range m.data {
		if matched, _ := filepath.Match(pattern, key); matched {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

func (m *MemoryCache) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var current int64 = 0

	if item, exists := m.data[key]; exists && !m.isExpired(item) {
		// Try to parse existing value as int64
		if len(item.value) > 0 {
			if val, err := parseInt64(item.value); err == nil {
				current = val
			}
		}
	}

	newValue := current + delta
	valueBytes := formatInt64(newValue)

	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.data[key] = &memoryItem{
		value:     valueBytes,
		expiresAt: expiresAt,
		createdAt: time.Now(),
	}

	return newValue, nil
}

func (m *MemoryCache) Decrement(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	return m.Increment(ctx, key, -delta, ttl)
}

func (m *MemoryCache) HGet(ctx context.Context, key, field string) ([]byte, error) {
	hashKey := fmt.Sprintf("%s::%s", key, field)
	return m.Get(ctx, hashKey)
}

func (m *MemoryCache) HSet(ctx context.Context, key, field string, value []byte, ttl time.Duration) error {
	hashKey := fmt.Sprintf("%s::%s", key, field)
	return m.Set(ctx, hashKey, value, ttl)
}

func (m *MemoryCache) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	pattern := key + "::*"
	keys, err := m.GetKeys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	prefix := key + "::"

	for _, fullKey := range keys {
		if strings.HasPrefix(fullKey, prefix) {
			field := strings.TrimPrefix(fullKey, prefix)
			value, err := m.Get(ctx, fullKey)
			if err == nil {
				result[field] = value
			}
		}
	}

	return result, nil
}

func (m *MemoryCache) HDelete(ctx context.Context, key string, fields ...string) error {
	keys := make([]string, len(fields))
	for i, field := range fields {
		keys[i] = fmt.Sprintf("%s::%s", key, field)
	}
	return m.DeleteMultiple(ctx, keys)
}

// SAdd adds one or more members to a set stored at key
func (m *MemoryCache) SAdd(ctx context.Context, key string, members ...[]byte) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if key exists and is a set
	item, exists := m.data[key]

	// If key doesn't exist or is expired, create a new set
	if !exists || m.isExpired(item) {
		setMap := make(map[string]struct{})

		// Add all members to the new set
		added := int64(0)
		for _, member := range members {
			memberStr := string(member)
			if _, exists := setMap[memberStr]; !exists {
				setMap[memberStr] = struct{}{}
				added++
			}
		}

		// Serialize the set map to JSON and store it
		setData, err := serializeSetMap(setMap)
		if err != nil {
			return 0, err
		}

		// Create a new item with default TTL
		ttl := m.config.DefaultTTL
		var expiresAt time.Time
		if ttl > 0 {
			expiresAt = time.Now().Add(ttl)
		}

		m.data[key] = &memoryItem{
			value:     setData,
			expiresAt: expiresAt,
			createdAt: time.Now(),
		}

		return added, nil
	}

	// Key exists, deserialize the set
	setMap, err := deserializeSetMap(item.value)
	if err != nil {
		return 0, err
	}

	// Add members to the existing set
	added := int64(0)
	for _, member := range members {
		memberStr := string(member)
		if _, exists := setMap[memberStr]; !exists {
			setMap[memberStr] = struct{}{}
			added++
		}
	}

	// If nothing was added, return early
	if added == 0 {
		return 0, nil
	}

	// Serialize the updated set and store it
	setData, err := serializeSetMap(setMap)
	if err != nil {
		return 0, err
	}

	// Update the item with the new set data
	item.value = setData

	return added, nil
}

// SRem removes one or more members from a set stored at key
func (m *MemoryCache) SRem(ctx context.Context, key string, members ...[]byte) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if key exists and is a set
	item, exists := m.data[key]

	// If key doesn't exist or is expired, return 0 (nothing removed)
	if !exists || m.isExpired(item) {
		return 0, nil
	}

	// Deserialize the set
	setMap, err := deserializeSetMap(item.value)
	if err != nil {
		return 0, err
	}

	// Remove members from the set
	removed := int64(0)
	for _, member := range members {
		memberStr := string(member)
		if _, exists := setMap[memberStr]; exists {
			delete(setMap, memberStr)
			removed++
		}
	}

	// If nothing was removed, return early
	if removed == 0 {
		return 0, nil
	}

	// If the set is now empty, delete the key
	if len(setMap) == 0 {
		delete(m.data, key)
		return removed, nil
	}

	// Serialize the updated set and store it
	setData, err := serializeSetMap(setMap)
	if err != nil {
		return 0, err
	}

	// Update the item with the new set data
	item.value = setData

	return removed, nil
}

// SMembers returns all members of the set stored at key
func (m *MemoryCache) SMembers(ctx context.Context, key string) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if key exists and is a set
	item, exists := m.data[key]

	// If key doesn't exist or is expired, return empty slice
	if !exists || m.isExpired(item) {
		return [][]byte{}, nil
	}

	// Deserialize the set
	setMap, err := deserializeSetMap(item.value)
	if err != nil {
		return nil, err
	}

	// Convert set members to slice of byte slices
	members := make([][]byte, 0, len(setMap))
	for memberStr := range setMap {
		members = append(members, []byte(memberStr))
	}

	return members, nil
}

// SIsMember returns if member is a member of the set stored at key
func (m *MemoryCache) SIsMember(ctx context.Context, key string, member []byte) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if key exists and is a set
	item, exists := m.data[key]

	// If key doesn't exist or is expired, return false
	if !exists || m.isExpired(item) {
		return false, nil
	}

	// Deserialize the set
	setMap, err := deserializeSetMap(item.value)
	if err != nil {
		return false, err
	}

	// Check if member exists in the set
	_, exists = setMap[string(member)]
	return exists, nil
}

// SCard returns the cardinality (number of elements) of the set stored at key
func (m *MemoryCache) SCard(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if key exists and is a set
	item, exists := m.data[key]

	// If key doesn't exist or is expired, return 0
	if !exists || m.isExpired(item) {
		return 0, nil
	}

	// Deserialize the set
	setMap, err := deserializeSetMap(item.value)
	if err != nil {
		return 0, err
	}

	// Return the number of elements in the set
	return int64(len(setMap)), nil
}

func (m *MemoryCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := LockKey(key)

	m.mu.Lock()
	defer m.mu.Unlock()

	if item, exists := m.data[lockKey]; exists && !m.isExpired(item) {
		return false, nil // Lock already held
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.data[lockKey] = &memoryItem{
		value:     []byte("locked"),
		expiresAt: expiresAt,
		createdAt: time.Now(),
	}

	return true, nil
}

func (m *MemoryCache) Unlock(ctx context.Context, key string) error {
	lockKey := LockKey(key)
	return m.Delete(ctx, lockKey)
}

func (m *MemoryCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		return 0, ErrKeyNotFound
	}

	if item.expiresAt.IsZero() {
		return -1, nil // No expiration
	}

	ttl := time.Until(item.expiresAt)
	if ttl < 0 {
		return 0, nil // Already expired
	}

	return ttl, nil
}

func (m *MemoryCache) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.data[key]
	if !exists {
		return ErrKeyNotFound
	}

	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	} else {
		item.expiresAt = time.Time{} // No expiration
	}

	return nil
}

func (m *MemoryCache) FlushAll(ctx context.Context) error {
	m.mu.Lock()
	m.data = make(map[string]*memoryItem)
	m.mu.Unlock()

	m.stats.mu.Lock()
	m.stats.hits = 0
	m.stats.misses = 0
	m.stats.mu.Unlock()

	return nil
}

func (m *MemoryCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.Set(ctx, key, data, ttl)
}

func (m *MemoryCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := m.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (m *MemoryCache) Close() error {
	close(m.stopCh)
	return nil
}

func (m *MemoryCache) Ping(ctx context.Context) error {
	return nil // Always available
}

func (m *MemoryCache) Stats(ctx context.Context) (Stats, error) {
	m.mu.RLock()
	keyCount := int64(len(m.data))
	m.mu.RUnlock()

	m.stats.mu.RLock()
	hits := m.stats.hits
	misses := m.stats.misses
	m.stats.mu.RUnlock()

	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return Stats{
		Hits:    hits,
		Misses:  misses,
		HitRate: hitRate,
		Keys:    keyCount,
		Memory:  0, // Memory calculation would be complex
		Uptime:  time.Since(m.startTime),
		Metadata: map[string]string{
			"type":        "memory",
			"default_ttl": m.config.DefaultTTL.String(),
		},
	}, nil
}
