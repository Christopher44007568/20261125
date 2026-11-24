package geomlife

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type ManualClock struct {
	mu sync.RWMutex
	t  time.Time
}

func NewManualClock() *ManualClock {
	return &ManualClock{t: time.Date(2026, time.November, 25, 0, 0, 0, 0, time.UTC)}
}

func NewManualClockAt(t time.Time) *ManualClock {
	return &ManualClock{t: t}
}

func (c *ManualClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.t
}

func (c *ManualClock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
	return c.t
}

func (c *ManualClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}
