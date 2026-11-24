package sim

// Vec2 is a point or direction in a scene-local 640x360 frame.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ObjectState contains only what the renderer needs to draw the three
// participating objects. The temporal tracker intentionally does not consume
// these transforms.
type ObjectState struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Name   string  `json:"name"`
	Center Vec2    `json:"center"`
	Radius float64 `json:"radius"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Angle  float64 `json:"angle"`
}

// HighlightTrack is the entire history payload: a Gaussian center, its shape,
// peak radiance, integrated energy, and the same-eye history sample.
type HighlightTrack struct {
	ObjectID   string        `json:"objectId"`
	Center     Vec2          `json:"center"`
	Stretch    float64       `json:"stretch"`
	Thickness  float64       `json:"thickness"`
	Direction  float64       `json:"direction"`
	Peak       float64       `json:"peak"`
	Energy     float64       `json:"energy"`
	Reflection Vec2          `json:"reflection"`
	History    HistorySample `json:"history"`
}

// HistorySample is deliberately sparse. It never stores a full-frame motion
// vector or a copy of an object transform.
type HistorySample struct {
	Valid           bool    `json:"valid"`
	Center          Vec2    `json:"center"`
	Stretch         float64 `json:"stretch"`
	Thickness       float64 `json:"thickness"`
	Direction       float64 `json:"direction"`
	SampledPeak     float64 `json:"sampledPeak"`
	SampledEnergy   float64 `json:"sampledEnergy"`
	CompositePeak   float64 `json:"compositePeak"`
	CompositeEnergy float64 `json:"compositeEnergy"`
	ResampleWeight  float64 `json:"resampleWeight"`
	SourceEye       int     `json:"sourceEye"`
	SameEye         bool    `json:"sameEye"`
	EnergyCapped    bool    `json:"energyCapped"`
}

// Frame is one complete simulation tick for one eye.
type Frame struct {
	Index   int              `json:"index"`
	Time    float64          `json:"time"`
	Objects []ObjectState    `json:"objects"`
	Tracks  []HighlightTrack `json:"tracks"`
}

// EyeTimeline owns a completely isolated history timeline.
type EyeTimeline struct {
	ID     int     `json:"id"`
	Label  string  `json:"label"`
	Offset float64 `json:"offset"`
	Frames []Frame `json:"frames"`
}

// Audit summarizes invariants checked while building the timeline.
type Audit struct {
	Frames               int     `json:"frames"`
	Tracks               int     `json:"tracks"`
	CrossEyeSamples      int     `json:"crossEyeSamples"`
	EnergyCapViolations  int     `json:"energyCapViolations"`
	FullFrameTimewarps   int     `json:"fullFrameTimewarps"`
	MaxCompositeEnergy   float64 `json:"maxCompositeEnergy"`
	MaxPeak              float64 `json:"maxPeak"`
	MeanResampleWeight   float64 `json:"meanResampleWeight"`
	EyeDirectionDeltaMax float64 `json:"eyeDirectionDeltaMax"`
}

// Timeline is the immutable payload consumed by the browser renderer.
type Timeline struct {
	Version    string         `json:"version"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	FPS        int            `json:"fps"`
	Duration   float64        `json:"duration"`
	FrameCount int            `json:"frameCount"`
	Light      Vec2           `json:"light"`
	Eyes       []EyeTimeline  `json:"eyes"`
	Audit      Audit          `json:"audit"`
	Scenes     []SceneSummary `json:"scenes"`
}

type SceneSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Motion      string `json:"motion"`
	Material    string `json:"material"`
	Accent      string `json:"accent"`
	Description string `json:"description"`
}

type trackKey struct {
	eye    int
	object string
}

type trackRecord struct {
	center     Vec2
	stretch    float64
	thickness  float64
	direction  float64
	peak       float64
	energy     float64
	reflection Vec2
	eye        int
}
