package notify

// FloatDiff records an opacity scalar transition.
type FloatDiff struct {
	Changed bool    `json:"changed"`
	Old     float64 `json:"old"`
	New     float64 `json:"new"`
	Delta   float64 `json:"delta"`
}

// Vec2Diff records a position transition.
type Vec2Diff struct {
	Changed  bool    `json:"changed"`
	Old      Vec2    `json:"old"`
	New      Vec2    `json:"new"`
	Delta    Vec2    `json:"delta"`
	Distance float64 `json:"distance"`
}

// BoolDiff records a boolean transition.
type BoolDiff struct {
	Changed bool `json:"changed"`
	Old     bool `json:"old"`
	New     bool `json:"new"`
}

// Diff is the semantic difference between two notification versions.
type Diff struct {
	ID              string    `json:"id"`
	FromRevision    int64     `json:"from_revision"`
	ToRevision      int64     `json:"to_revision"`
	RevisionChanged bool      `json:"revision_changed"`
	Position        Vec2Diff  `json:"position"`
	Opacity         FloatDiff `json:"opacity"`
	Closed          BoolDiff  `json:"closed"`
}

// DiffStates compares two versions. The direction is old -> next.
func DiffStates(old State, next State) Diff {
	positionDelta := next.Position.Sub(old.Position)
	positionDistance := old.Position.Distance(next.Position)
	positionChanged := positionDistance > Epsilon

	opacityDelta := next.Opacity - old.Opacity
	opacityChanged := !nearlyEqual(old.Opacity, next.Opacity)
	revisionChanged := old.Revision != next.Revision

	return Diff{
		ID:              next.ID,
		FromRevision:    old.Revision,
		ToRevision:      next.Revision,
		RevisionChanged: revisionChanged,
		Position: Vec2Diff{
			Changed:  positionChanged,
			Old:      old.Position,
			New:      next.Position,
			Delta:    positionDelta,
			Distance: positionDistance,
		},
		Opacity: FloatDiff{
			Changed: opacityChanged,
			Old:     old.Opacity,
			New:     next.Opacity,
			Delta:   opacityDelta,
		},
		Closed: BoolDiff{
			Changed: old.Closed != next.Closed,
			Old:     old.Closed,
			New:     next.Closed,
		},
	}
}

// IsEmpty reports whether no semantic field or revision changed.
func (d Diff) IsEmpty() bool {
	return !d.RevisionChanged && !d.Position.Changed && !d.Opacity.Changed && !d.Closed.Changed
}

// Kinds returns machine-readable changed categories.
func (d Diff) Kinds() []string {
	kinds := make([]string, 0, 3)
	if d.Position.Changed {
		kinds = append(kinds, "position")
	}
	if d.Opacity.Changed {
		kinds = append(kinds, "opacity")
	}
	if d.Closed.Changed {
		kinds = append(kinds, "closed")
	}
	return kinds
}

// HasPositionChange reports a position delta above floating-point epsilon.
func (d Diff) HasPositionChange() bool { return d.Position.Changed }

// HasOpacityChange reports an opacity delta above floating-point epsilon.
func (d Diff) HasOpacityChange() bool { return d.Opacity.Changed }

// HasCloseChange reports a transition between open and closed.
func (d Diff) HasCloseChange() bool { return d.Closed.Changed }
