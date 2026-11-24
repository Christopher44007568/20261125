package xrpath

import "image"

const SchemaVersion = "xrpath.frame/v1"

const (
	SubsystemSequence     = "frame_sequence"
	SubsystemLifecycle    = "lifecycle"
	SubsystemTracking     = "tracking_state"
	SubsystemParameters   = "parameters"
	SubsystemPose         = "pose"
	SubsystemResource     = "resource_state"
	SubsystemVisibility   = "visible_set"
	SubsystemReprojection = "reprojection_source"
	SubsystemLeftImage    = "left_image"
	SubsystemRightImage   = "right_image"
	SubsystemFrameHash    = "frame_hash"
)

type Pose struct {
	Position        Vec3    `json:"position"`
	Orientation     Quat    `json:"orientation"`
	Yaw             float64 `json:"yaw_rad"`
	Pitch           float64 `json:"pitch_rad"`
	Roll            float64 `json:"roll_rad"`
	LinearVelocity  Vec3    `json:"linear_velocity"`
	AngularVelocity Vec3    `json:"angular_velocity"`
}

type Parameters struct {
	Revision           int     `json:"revision"`
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	FOVDeg             float64 `json:"fov_deg"`
	Near               float64 `json:"near"`
	Far                float64 `json:"far"`
	IPD                float64 `json:"ipd"`
	Exposure           float64 `json:"exposure"`
	ReprojectionPolicy string  `json:"reprojection_policy"`
	MaxResourceBytes   int64   `json:"max_resource_bytes"`
}

type ResourceState struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Revision       int      `json:"revision"`
	SHA256         string   `json:"sha256"`
	Status         string   `json:"status"`
	Resident       bool     `json:"resident"`
	Bytes          int64    `json:"bytes"`
	ReferenceCount int      `json:"reference_count"`
	UsedBy         []string `json:"used_by"`
}

type ScreenRect struct {
	MinX    int     `json:"min_x"`
	MinY    int     `json:"min_y"`
	MaxX    int     `json:"max_x"`
	MaxY    int     `json:"max_y"`
	CenterX float64 `json:"center_x"`
	CenterY float64 `json:"center_y"`
}

type VisibleObject struct {
	ID          string     `json:"id"`
	ResourceID  string     `json:"resource_id"`
	Center      Vec3       `json:"center"`
	HalfExtents Vec3       `json:"half_extents"`
	Depth       float64    `json:"depth"`
	Screen      ScreenRect `json:"screen"`
	Pixels      int        `json:"pixels"`
}

type VisibilitySet struct {
	Left  []VisibleObject `json:"left"`
	Right []VisibleObject `json:"right"`
}

type ReprojectionSource struct {
	Eye                string  `json:"eye"`
	Mode               string  `json:"mode"`
	SourceFrame        int     `json:"source_frame"`
	SourceKeyframe     string  `json:"source_keyframe,omitempty"`
	SourcePosition     Vec3    `json:"source_position"`
	SourceOrientation  Quat    `json:"source_orientation"`
	SourceImageSHA256  string  `json:"source_image_sha256"`
	WarpMeshSHA256     string  `json:"warp_mesh_sha256"`
	Confidence         float64 `json:"confidence"`
	ValidPixelRatio    float64 `json:"valid_pixel_ratio"`
	DisocclusionPixels int     `json:"disocclusion_pixels"`
	Reason             string  `json:"reason"`
}

type ImageRef struct {
	Eye         string `json:"eye"`
	Path        string `json:"path"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PixelSHA256 string `json:"pixel_sha256"`
	PNGSHA256   string `json:"png_sha256"`
}

type ImagePair struct {
	Left  ImageRef `json:"left"`
	Right ImageRef `json:"right"`
}

type TrackingState struct {
	Status           string  `json:"status"`
	Quality          float64 `json:"quality"`
	MatchedKeyframe  string  `json:"matched_keyframe,omitempty"`
	LastGoodFrame    int     `json:"last_good_frame"`
	RelocalizedFrame int     `json:"relocalized_frame,omitempty"`
	FeatureCount     int     `json:"feature_count"`
}

type Event struct {
	Frame     int     `json:"frame"`
	AtSeconds float64 `json:"at_seconds"`
	Type      string  `json:"type"`
	Detail    string  `json:"detail"`
	Revision  int     `json:"revision,omitempty"`
}

type FrameRecord struct {
	Schema       string                `json:"schema"`
	Frame        int                   `json:"frame"`
	MotionFrame  int                   `json:"motion_frame"`
	AtSeconds    float64               `json:"at_seconds"`
	Phase        string                `json:"phase"`
	Lifecycle    string                `json:"lifecycle"`
	Pose         Pose                  `json:"pose"`
	Tracking     TrackingState         `json:"tracking"`
	Parameters   Parameters            `json:"parameters"`
	Events       []Event               `json:"events"`
	Resources    []ResourceState       `json:"resources"`
	Visibility   VisibilitySet         `json:"visibility"`
	Reprojection [2]ReprojectionSource `json:"reprojection"`
	Images       ImagePair             `json:"images"`
	FrameSHA256  string                `json:"frame_sha256"`
}

type Manifest struct {
	SchemaVersion   string         `json:"schema_version"`
	FrameCount      int            `json:"frame_count"`
	FrameRateHz     int            `json:"frame_rate_hz"`
	FixedPathSHA256 string         `json:"fixed_path_sha256"`
	Parameters      Parameters     `json:"parameters"`
	Phases          []PhaseSummary `json:"phases"`
	Events          []Event        `json:"events"`
}

type PhaseSummary struct {
	Name       string `json:"name"`
	FirstFrame int    `json:"first_frame"`
	LastFrame  int    `json:"last_frame"`
}

type RunResult struct {
	Manifest Manifest
	Records  []FrameRecord
	Images   map[int]*StereoImage `json:"-"`
}

type StereoImage struct {
	Left  *RenderedEye
	Right *RenderedEye
}

type RenderedEye struct {
	Eye       string
	Image     *image.RGBA
	PNGBytes  []byte
	Depth     []float64
	ObjectIDs []string
	Hits      []VisibleObject
}
