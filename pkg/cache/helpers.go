package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"sort"
	"strconv"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func LockKey(resource string) string {
	return fmt.Sprintf("lock:%s", resource)
}

// GenerateCacheKey generates a deterministic cache key for search results based on query and filters.
//
// This function creates a consistent, unique cache key by hashing the search query and filters.
// The key generation is deterministic, meaning the same query and filters will always produce
// the same cache key, enabling efficient cache lookups for identical search requests.
//
// Parameters:
//   - query: The search query string (e.g., "pizza", "sushi near me", "fast food")
//   - filters: A map of filter criteria where key is the filter name and value is the filter value
//
// Returns:
//   - A string cache key in the format "search:<16-char-hash>"
//
// Key Generation Process:
//  1. Creates a SHA256 hash starting with the query string
//  2. Sorts filter keys alphabetically to ensure deterministic ordering
//  3. Appends each filter key-value pair to the hash
//  4. Returns the first 16 characters of the hex-encoded hash with "search:" prefix
//
// Example Usage:
//
//	// Basic search query
//	key := GenerateCacheKey("pizza", nil)
//	// Returns: "search:a1b2c3d4e5f6g7h8"
//
//	// Search with filters
//	filters := map[string]string{
//		"cuisine":     "italian",
//		"price_range": "2",
//		"distance":    "5km",
//		"rating":      "4+",
//	}
//	key := GenerateCacheKey("pizza", filters)
//	// Returns: "search:x9y8z7w6v5u4t3s2"
//
//	// Same query and filters will produce identical keys
//	key1 := GenerateCacheKey("pizza", map[string]string{"cuisine": "italian", "price": "$$"})
//	key2 := GenerateCacheKey("pizza", map[string]string{"price": "$$", "cuisine": "italian"})
//	// key1 == key2 (true) - order of filters doesn't matter
func GenerateCacheKey(query string, filters map[string]string) string {
	// Create a deterministic key from query and filters
	h := sha256.New()
	h.Write([]byte(query))

	// Sort filters for deterministic key generation
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(filters[k]))
	}

	hash := hex.EncodeToString(h.Sum(nil))[:16]
	return fmt.Sprintf("search:%s", hash)
}

func GetString(cache Client, ctx context.Context, key string) (string, error) {
	data, err := cache.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SetString(cache Client, ctx context.Context, key, value string, ttl time.Duration) error {
	return cache.Set(ctx, key, []byte(value), ttl)
}

func GetJSON(cache Client, ctx context.Context, key string, v interface{}) error {
	data, err := cache.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func SetJSON(cache Client, ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return &Error{Operation: "serialize", Key: key, Err: err}
	}
	return cache.Set(ctx, key, data, ttl)
}

// serializeSetMap serializes a set (represented as a map) to a byte slice
// Format: Each member is stored as a length-prefixed string
// [len1][member1][len2][member2]...
func serializeSetMap(setMap map[string]struct{}) ([]byte, error) {
	var buf bytes.Buffer

	for member := range setMap {
		memberBytes := []byte(member)
		memberLen := len(memberBytes)

		// Write length as a 4-byte integer
		lenBytes := make([]byte, 4)
		lenBytes[0] = byte(memberLen >> 24)
		lenBytes[1] = byte(memberLen >> 16)
		lenBytes[2] = byte(memberLen >> 8)
		lenBytes[3] = byte(memberLen)

		buf.Write(lenBytes)
		buf.Write(memberBytes)
	}

	return buf.Bytes(), nil
}

// deserializeSetMap deserializes a byte slice to a set (represented as a map)
func deserializeSetMap(data []byte) (map[string]struct{}, error) {
	setMap := make(map[string]struct{})

	for i := 0; i < len(data); {
		// Need at least 4 bytes for the length
		if i+4 > len(data) {
			return nil, fmt.Errorf("invalid set data format")
		}

		// Read the member length
		memberLen := int(data[i])<<24 | int(data[i+1])<<16 | int(data[i+2])<<8 | int(data[i+3])
		i += 4

		// Check if we have enough bytes for the member
		if i+memberLen > len(data) {
			return nil, fmt.Errorf("invalid set data format")
		}

		// Read the member
		member := string(data[i : i+memberLen])
		i += memberLen

		// Add the member to the set
		setMap[member] = struct{}{}
	}

	return setMap, nil
}

// Helper functions for int64 conversion
func parseInt64(data []byte) (int64, error) {
	str := string(data)
	return strconv.ParseInt(str, 10, 64)
}

func formatInt64(value int64) []byte {
	return []byte(strconv.FormatInt(value, 10))
}
