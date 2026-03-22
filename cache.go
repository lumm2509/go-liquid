package liquid

import (
	"container/list"
	"sync"
	"time"
)

// TemplateCache stores parsed templates keyed by an arbitrary string.
// Parsing is O(source length); rendering is cheap. In servers that render
// the same template on every request, use TemplateCache to pay the parse
// cost once and render many times.
//
// TemplateCache is safe for concurrent use.
//
// Example:
//
//	cache := liquid.NewTemplateCache(env)
//	tmpl, err := cache.Get("product", productTemplateSource)
//	output, err := tmpl.Render(assigns, nil)
type TemplateCache struct {
	mu      sync.RWMutex
	cache   map[string]*cachedTemplate
	env     *Environment
	order   *list.List              // LRU order; Front() = oldest (least recently used)
	index   map[string]*list.Element // key → list element for O(1) removal/promotion
	MaxSize int                     // 0 = unlimited
	TTL     time.Duration           // 0 = no expiration
}

type cachedTemplate struct {
	tmpl       *Template
	insertedAt time.Time
}

// NewTemplateCache creates a TemplateCache that uses env for all Parse calls.
// Pass nil to use a fresh default Environment per entry.
func NewTemplateCache(env *Environment) *TemplateCache {
	return &TemplateCache{
		cache: make(map[string]*cachedTemplate),
		env:   env,
		order: list.New(),
		index: make(map[string]*list.Element),
	}
}

// Get returns the cached template for key. If no entry exists (or it has expired),
// it parses source using the cache's Environment and stores the result.
// Subsequent calls with the same key return the cached template regardless
// of the source argument (unless expired).
func (c *TemplateCache) Get(key, source string) (*Template, error) {
	// Fast path: check under read lock
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if ok {
		// Check TTL expiry
		if c.TTL > 0 && time.Since(entry.insertedAt) > c.TTL {
			// Expired: fall through to write path
		} else {
			// LRU promotion: move to back under write lock
			c.mu.Lock()
			if el, exists := c.index[key]; exists {
				c.order.MoveToBack(el)
			}
			c.mu.Unlock()
			return entry.tmpl, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock
	if entry, ok = c.cache[key]; ok {
		if c.TTL == 0 || time.Since(entry.insertedAt) <= c.TTL {
			// Still valid — promote and return
			if el, exists := c.index[key]; exists {
				c.order.MoveToBack(el)
			}
			return entry.tmpl, nil
		}
		// Expired — remove the stale entry before re-parsing
		if el, exists := c.index[key]; exists {
			c.order.Remove(el)
			delete(c.index, key)
		}
		delete(c.cache, key)
	}

	env := c.env
	if env == nil {
		env = NewEnvironment()
	}
	t, err := ParseWithEnv(source, env, nil)
	if err != nil {
		return nil, err
	}

	if c.MaxSize > 0 && len(c.cache) >= c.MaxSize {
		// Evict LRU entry (Front = least recently used)
		front := c.order.Front()
		if front != nil {
			oldest := front.Value.(string)
			c.order.Remove(front)
			delete(c.index, oldest)
			delete(c.cache, oldest)
		}
	}

	c.cache[key] = &cachedTemplate{tmpl: t, insertedAt: time.Now()}
	e := c.order.PushBack(key)
	c.index[key] = e
	return t, nil
}

// Invalidate removes a single entry from the cache in O(1).
func (c *TemplateCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.index[key]; ok {
		c.order.Remove(e)
		delete(c.index, key)
		delete(c.cache, key)
	}
}

// Flush removes all entries from the cache.
func (c *TemplateCache) Flush() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedTemplate)
	c.order = list.New()
	c.index = make(map[string]*list.Element)
	c.mu.Unlock()
}

// Len returns the number of templates currently in the cache.
// Safe for concurrent use.
func (c *TemplateCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
