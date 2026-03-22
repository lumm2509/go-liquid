package liquid

import (
	"sync"
	"sync/atomic"
	"time"
)

// TemplateCache stores parsed templates keyed by an arbitrary string.
// Parsing is O(source length); rendering is cheap. In servers that render
// the same template on every request, use TemplateCache to pay the parse
// cost once and render many times.
//
// The cache is sharded into 16 independent buckets to reduce mutex contention
// under concurrent load. Each shard has its own RWMutex; reads never contend
// across shards.
//
// TemplateCache is safe for concurrent use.
//
// Example:
//
//	cache := liquid.NewTemplateCache(env)
//	tmpl, err := cache.Get("product", productTemplateSource)
//	output, err := tmpl.Render(assigns, nil)
type TemplateCache struct {
	shards  [16]cacheShard
	env     *Environment
	// MaxSize is the approximate maximum number of templates to keep.
	// 0 = unlimited. The limit is enforced per shard (MaxSize/16, min 1),
	// so the actual maximum can be up to MaxSize entries in a perfectly
	// uniform key distribution — treat this as a soft cap, not a hard limit.
	MaxSize int
	TTL     time.Duration // 0 = no expiration
}

type cacheShard struct {
	mu    sync.RWMutex
	cache map[string]*cachedTemplate
	_     [56]byte // padding to prevent false sharing between shards
}

type cachedTemplate struct {
	tmpl       *Template
	insertedAt time.Time
	lastAccess atomic.Int64 // Unix seconds; updated atomically on hit — no lock needed
}

// NewTemplateCache creates a TemplateCache that uses env for all Parse calls.
// Pass nil to use a fresh default Environment per entry.
func NewTemplateCache(env *Environment) *TemplateCache {
	c := &TemplateCache{env: env}
	for i := range c.shards {
		c.shards[i].cache = make(map[string]*cachedTemplate)
	}
	return c
}

// shardFor returns the shard index for a given key using FNV-1a.
func shardFor(key string) int {
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return int(h & 15)
}

// Get returns the cached template for key. If no entry exists (or it has expired),
// it parses source and stores the result.
func (c *TemplateCache) Get(key, source string) (*Template, error) {
	s := &c.shards[shardFor(key)]

	// Fast path: read lock only — no write lock on cache hit.
	s.mu.RLock()
	entry, ok := s.cache[key]
	s.mu.RUnlock()

	if ok {
		if c.TTL > 0 && time.Since(entry.insertedAt) > c.TTL {
			// Expired — fall through to write path
		} else {
			entry.lastAccess.Store(time.Now().Unix())
			return entry.tmpl, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check under write lock
	if entry, ok = s.cache[key]; ok {
		if c.TTL == 0 || time.Since(entry.insertedAt) <= c.TTL {
			entry.lastAccess.Store(time.Now().Unix())
			return entry.tmpl, nil
		}
		delete(s.cache, key)
	}

	env := c.env
	if env == nil {
		env = NewEnvironment()
	}
	t, err := ParseWithEnv(source, env, nil)
	if err != nil {
		return nil, err
	}

	// Per-shard size limit: MaxSize / 16, minimum 1.
	if c.MaxSize > 0 {
		limit := c.MaxSize / 16
		if limit < 1 {
			limit = 1
		}
		if len(s.cache) >= limit {
			evictLRUFromShard(s)
		}
	}

	entry = &cachedTemplate{tmpl: t, insertedAt: time.Now()}
	entry.lastAccess.Store(entry.insertedAt.Unix())
	s.cache[key] = entry
	return t, nil
}

// evictLRUFromShard removes the least recently accessed entry from shard s.
// Must be called with s.mu held for writing. O(n/16) — fast in practice.
func evictLRUFromShard(s *cacheShard) {
	var lruKey string
	var lruTime int64 = 1<<63 - 1
	for k, e := range s.cache {
		if t := e.lastAccess.Load(); t < lruTime {
			lruTime = t
			lruKey = k
		}
	}
	if lruKey != "" {
		delete(s.cache, lruKey)
	}
}

// Invalidate removes a single entry from the cache.
func (c *TemplateCache) Invalidate(key string) {
	s := &c.shards[shardFor(key)]
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}

// Flush removes all entries from all shards.
func (c *TemplateCache) Flush() {
	for i := range c.shards {
		s := &c.shards[i]
		s.mu.Lock()
		s.cache = make(map[string]*cachedTemplate)
		s.mu.Unlock()
	}
}

// Len returns the total number of templates currently in the cache.
// Safe for concurrent use.
func (c *TemplateCache) Len() int {
	total := 0
	for i := range c.shards {
		s := &c.shards[i]
		s.mu.RLock()
		total += len(s.cache)
		s.mu.RUnlock()
	}
	return total
}
