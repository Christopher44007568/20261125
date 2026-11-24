package binocular

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Device owns the current binocular session and all retired sessions. A new
// Reconnect always creates a new session ID, generation, resource set, private
// state, and write-lease epoch.
type Device struct {
	mu         sync.Mutex
	nextID     uint64
	generation uint64
	current    *Session
	retired    []*Session
	baseline   Frame
}

func NewDevice() *Device {
	d := &Device{baseline: CleanFrame()}
	session, _ := d.newSessionLocked()
	d.current = session
	return d
}

func (d *Device) CurrentSession() *Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current
}

func (d *Device) Baseline() Frame {
	return Frame{Left: append([]byte(nil), d.baseline.Left...), Right: append([]byte(nil), d.baseline.Right...)}
}

func (d *Device) newSessionLocked() (*Session, time.Duration) {
	start := time.Now()
	d.nextID++
	d.generation++
	s := newSession(d.nextID, d.generation, d.baseline)
	return s, time.Since(start)
}

// Disconnect stops every queued/running task in the current binocular session.
// Left-eye, right-eye, and full-device failures all invalidate the whole
// binocular session because no frame can be completed without both eyes.
func (d *Device) Disconnect(scope DisconnectScope) (ReleaseReport, error) {
	if !scope.Valid() {
		return ReleaseReport{}, ErrInvalidScope
	}
	d.mu.Lock()
	s := d.current
	d.current = nil
	d.mu.Unlock()
	if s == nil {
		return ReleaseReport{}, ErrNoSession
	}
	report := s.Close(scope)
	d.mu.Lock()
	d.retired = append(d.retired, s)
	d.mu.Unlock()
	return report, nil
}

func (d *Device) Reconnect() (*Session, time.Duration, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.current != nil {
		return nil, 0, ErrSessionActive
	}
	s, setup := d.newSessionLocked()
	d.current = s
	return s, setup, nil
}

func (d *Device) RetiredSessions() []*Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]*Session(nil), d.retired...)
}

// UnreleasedObjects reports leaks from every retired session. Resources owned
// by the current active session are intentionally excluded.
func (d *Device) UnreleasedObjects() []ObjectRef {
	d.mu.Lock()
	retired := append([]*Session(nil), d.retired...)
	d.mu.Unlock()
	var out []ObjectRef
	for _, s := range retired {
		out = append(out, s.UnreleasedObjects()...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Session != out[j].Session {
			return out[i].Session < out[j].Session
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (d *Device) String() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.current == nil {
		return "device(disconnected)"
	}
	return fmt.Sprintf("device(session=%d,generation=%d)", d.current.id, d.current.generation)
}
