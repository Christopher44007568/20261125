package eyebw

import (
	"container/list"
	"sync"
)

// Cache is a byte-budgeted LRU. It models cache residency, not a particular
// GPU cache hierarchy. A hit returns logical bytes to the stage but no bus
// transfer; a miss is counted once by the calling Meter.
type Cache struct {
	mu       sync.Mutex
	capacity uint64
	size     uint64
	ll       *list.List
	items    map[string]*list.Element
}

type cacheEntry struct {
	key  string
	size uint64
}

func NewCache(capacityBytes uint64) *Cache {
	return &Cache{
		capacity: capacityBytes,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *Cache) Capacity() uint64 {
	if c == nil {
		return 0
	}
	return c.capacity
}

func (c *Cache) Size() uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *Cache) Get(key string) (uint64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return 0, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*cacheEntry).size, true
}

// Put stores a write-through resident copy. Eviction does not add transfer
// bytes because the write path was already counted.
func (c *Cache) Put(key string, size uint64) {
	if c == nil || key == "" || size == 0 || size > c.capacity {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		c.size -= entry.size
		entry.size = size
		c.size += size
		c.ll.MoveToFront(el)
	} else {
		el := c.ll.PushFront(&cacheEntry{key: key, size: size})
		c.items[key] = el
		c.size += size
	}
	for c.size > c.capacity && c.ll.Len() > 0 {
		oldest := c.ll.Back()
		if oldest == nil {
			break
		}
		entry := c.ll.Remove(oldest).(*cacheEntry)
		delete(c.items, entry.key)
		c.size -= entry.size
	}
}

type Meter struct {
	mu       sync.Mutex
	Frame    int
	Pipeline PipelineID
	Cache    *Cache
	Accesses []Access
}

func NewMeter(frame int, pipeline PipelineID, cache *Cache) *Meter {
	return &Meter{Frame: frame, Pipeline: pipeline, Cache: cache}
}

func (m *Meter) Snapshot() []Access {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Access(nil), m.Accesses...)
}

func (m *Meter) record(a Access) Access {
	if m == nil || a.LogicalBytes == 0 {
		return a
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a.Frame = m.Frame
	if a.Pipeline == "" {
		a.Pipeline = m.Pipeline
	}
	m.Accesses = append(m.Accesses, a)
	return a
}

func (m *Meter) Read(stage StageID, eye EyeID, class DataClass, key string, bytes uint64) Access {
	return m.read(stage, eye, class, key, "", bytes, true)
}

func (m *Meter) ReadReason(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64) Access {
	return m.read(stage, eye, class, key, reason, bytes, true)
}

func (m *Meter) ReadUncached(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64) Access {
	return m.read(stage, eye, class, key, reason, bytes, false)
}

func (m *Meter) read(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64, cacheable bool) Access {
	a := Access{
		Stage:        stage,
		Eye:          eye,
		Class:        class,
		Direction:    DirRead,
		Key:          key,
		Reason:       reason,
		LogicalBytes: bytes,
		Cacheable:    cacheable,
	}
	if bytes == 0 {
		return a
	}
	if cacheable && m.Cache != nil && key != "" {
		if _, hit := m.Cache.Get(key); hit {
			a.CacheHit = true
			a.TransferBytes = 0
			return m.record(a)
		}
		m.Cache.Put(key, bytes)
	}
	a.TransferBytes = bytes
	return m.record(a)
}

func (m *Meter) Write(stage StageID, eye EyeID, class DataClass, key string, bytes uint64) Access {
	return m.write(stage, eye, class, key, "", bytes, true)
}

func (m *Meter) WriteReason(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64) Access {
	return m.write(stage, eye, class, key, reason, bytes, true)
}

func (m *Meter) WriteUncached(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64) Access {
	return m.write(stage, eye, class, key, reason, bytes, false)
}

func (m *Meter) write(stage StageID, eye EyeID, class DataClass, key, reason string, bytes uint64, cacheable bool) Access {
	a := Access{
		Stage:         stage,
		Eye:           eye,
		Class:         class,
		Direction:     DirWrite,
		Key:           key,
		Reason:        reason,
		LogicalBytes:  bytes,
		TransferBytes: bytes,
		Cacheable:     cacheable,
	}
	if bytes == 0 {
		return a
	}
	// Write-through accounting: the write itself is the transfer. Populating
	// the cache cannot create a second transfer.
	if cacheable && m.Cache != nil && key != "" {
		m.Cache.Put(key, bytes)
	}
	return m.record(a)
}
