package main

import (
	"container/list"
	"fmt"
	"sort"
)

type CacheStats struct {
	Hits      int `json:"hits"`
	Misses    int `json:"misses"`
	Evictions int `json:"evictions"`
	Entries   int `json:"entries"`
	Capacity  int `json:"capacity"`
}

type cacheEntry struct {
	key      CacheKey
	prepared *PreparedUniforms
}

// UniformCache is an LRU cache of prepared dynamic uniform blocks. A caller
// that wants eye isolation must pass a key containing the eye and the complete
// dynamic fingerprint. The cache itself never merges keys or inspects values.
type UniformCache struct {
	capacity int
	order    *list.List
	items    map[CacheKey]*list.Element
	stats    CacheStats
}

func newUniformCache(capacity int) (*UniformCache, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("cache capacity must be positive, got %d", capacity)
	}
	return &UniformCache{
		capacity: capacity,
		order:    list.New(),
		items:    make(map[CacheKey]*list.Element, capacity),
		stats:    CacheStats{Capacity: capacity},
	}, nil
}

func (c *UniformCache) get(key CacheKey) (*PreparedUniforms, bool) {
	element, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}
	c.order.MoveToFront(element)
	c.stats.Hits++
	return element.Value.(*cacheEntry).prepared, true
}

func (c *UniformCache) put(key CacheKey, prepared *PreparedUniforms) {
	if prepared == nil {
		return
	}
	if element, ok := c.items[key]; ok {
		element.Value.(*cacheEntry).prepared = prepared
		c.order.MoveToFront(element)
		return
	}
	element := c.order.PushFront(&cacheEntry{key: key, prepared: prepared})
	c.items[key] = element
	for len(c.items) > c.capacity {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(*cacheEntry)
		delete(c.items, entry.key)
		c.order.Remove(oldest)
		c.stats.Evictions++
	}
}

func (c *UniformCache) snapshotStats() CacheStats {
	stats := c.stats
	stats.Entries = len(c.items)
	return stats
}

type CacheSnapshotEntry struct {
	Key          CacheKey `json:"key"`
	PreparedHash string   `json:"prepared_hash"`
	Eye          Eye      `json:"eye"`
	Instance     string   `json:"instance"`
}

func (c *UniformCache) snapshot() []CacheSnapshotEntry {
	entries := make([]CacheSnapshotEntry, 0, len(c.items))
	for element := c.order.Front(); element != nil; element = element.Next() {
		entry := element.Value.(*cacheEntry)
		entries = append(entries, CacheSnapshotEntry{
			Key:          entry.key,
			PreparedHash: entry.prepared.DynamicHash,
			Eye:          entry.prepared.Eye,
			Instance:     entry.prepared.Instance,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Eye != entries[j].Eye {
			return entries[i].Eye < entries[j].Eye
		}
		return entries[i].Instance < entries[j].Instance
	})
	return entries
}
