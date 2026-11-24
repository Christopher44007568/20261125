// Package notify models deterministic notification commits around scanout.
//
// The central rule is that arrival time decides which frame may own an update,
// while monotonic revisions decide whether a snapshot is newer. A committed
// frame is snapshotted and never edited by late input; late input is staged
// for a later frame.
package notify

import (
	"fmt"
	"math"
	"time"
)

// Epsilon is used for deterministic floating-point change detection.
const Epsilon = 1e-9

// Vec2 is a position in display-local logical pixels.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Add returns v+other.
func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Sub returns v-other.
func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

// Distance returns the Euclidean distance to other.
func (v Vec2) Distance(other Vec2) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	return math.Hypot(dx, dy)
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= Epsilon
}

// State is one immutable-ish version of a system notification.
//
// Closed is an explicit tombstone. It must survive later frame snapshots so
// that an older open snapshot cannot make the notification reappear.
type State struct {
	ID        string        `json:"id"`
	Revision  int64         `json:"revision"`
	Closed    bool          `json:"closed"`
	Position  Vec2          `json:"position"`
	Opacity   float64       `json:"opacity"`
	UpdatedAt time.Duration `json:"updated_at_ns"`
}

// Visible reports whether the notification should produce visible pixels.
func (s State) Visible() bool {
	return !s.Closed && s.Opacity > Epsilon
}

func (s State) validate() error {
	if s.ID == "" {
		return fmt.Errorf("notification id is empty")
	}
	if s.Revision <= 0 {
		return fmt.Errorf("notification %q revision must be positive, got %d", s.ID, s.Revision)
	}
	if math.IsNaN(s.Opacity) || s.Opacity < 0 || s.Opacity > 1 {
		return fmt.Errorf("notification %q opacity must be in [0,1], got %v", s.ID, s.Opacity)
	}
	if math.IsNaN(s.Position.X) || math.IsNaN(s.Position.Y) {
		return fmt.Errorf("notification %q position must not be NaN", s.ID)
	}
	return nil
}

func cloneStates(in map[string]State) map[string]State {
	out := make(map[string]State, len(in))
	for id, state := range in {
		out[id] = state
	}
	return out
}
