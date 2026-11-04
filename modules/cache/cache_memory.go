package cache

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

type memoryItem struct {
	key, value string
	expires    time.Time
}

// memoryCache is a bounded, concurrent LRU. Expiration is evaluated on access;
// the maximum entry count bounds memory growth without a background goroutine.
type memoryCache struct {
	mu      sync.Mutex
	items   map[string]*list.Element
	order   *list.List
	max     int
	now     func() time.Time
}

func NewMemoryStringCache(maxEntries int) (StringCache, error) {
	if maxEntries <= 0 {
		return nil, fmt.Errorf("memory cache MAX_ENTRIES must be positive")
	}
	return &memoryCache{
		items: make(map[string]*list.Element),
		order: list.New(),
		max:   maxEntries,
		now:   time.Now,
	}, nil
}

func (c *memoryCache) Get(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	node := c.items[key]
	if node == nil {
		return "", false, nil
	}
	item := node.Value.(memoryItem)
	if !c.now().Before(item.expires) {
		c.remove(node)
		return "", false, nil
	}
	c.order.MoveToFront(node)
	return item.value, true, nil
}

func (c *memoryCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item := memoryItem{key: key, value: value, expires: c.now().Add(ttl)}
	if node := c.items[key]; node != nil {
		node.Value = item
		c.order.MoveToFront(node)
		return nil
	}
	if c.order.Len() >= c.max {
		c.remove(c.order.Back())
	}
	c.items[key] = c.order.PushFront(item)
	return nil
}

func (c *memoryCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if node := c.items[key]; node != nil {
		c.remove(node)
	}
	return nil
}

func (c *memoryCache) remove(node *list.Element) {
	if node == nil {
		return
	}
	delete(c.items, node.Value.(memoryItem).key)
	c.order.Remove(node)
}
