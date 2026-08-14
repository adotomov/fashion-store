package application

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// locationCache memoizes Speedy Location API lookups (sites, complexes,
// streets, offices) in process. Location data is effectively static, so a
// generous TTL keeps typeahead responsive and spares the Speedy account from a
// request per keystroke (their docs explicitly recommend caching it). Entries
// are keyed by kind|siteId|normalizedQuery and hold the already-mapped result
// slice. The map is bounded: once it grows past maxLocationCacheEntries it is
// dropped wholesale rather than tracking per-entry LRU, which is fine for a
// short-lived typeahead cache.
type locationCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]locationCacheEntry
}

type locationCacheEntry struct {
	value   any
	expires time.Time
}

const maxLocationCacheEntries = 4096

func newLocationCache(ttl time.Duration) *locationCache {
	return &locationCache{ttl: ttl, items: map[string]locationCacheEntry{}}
}

func locationCacheKey(kind string, siteID int64, query string) string {
	return kind + "|" + strconv.FormatInt(siteID, 10) + "|" + strings.ToLower(strings.TrimSpace(query))
}

func (c *locationCache) get(key string, now time.Time) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok || now.After(entry.expires) {
		return nil, false
	}
	return entry.value, true
}

func (c *locationCache) set(key string, value any, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= maxLocationCacheEntries {
		c.items = map[string]locationCacheEntry{}
	}
	c.items[key] = locationCacheEntry{value: value, expires: now.Add(c.ttl)}
}

// cachedLookup returns the cached slice for key, or calls load and caches its
// result. Errors are never cached, so a transient Speedy failure is retried on
// the next call.
func cachedLookup[T any](c *locationCache, key string, now time.Time, load func() ([]T, error)) ([]T, error) {
	if v, ok := c.get(key, now); ok {
		return v.([]T), nil
	}
	res, err := load()
	if err != nil {
		return nil, err
	}
	c.set(key, res, now)
	return res, nil
}
