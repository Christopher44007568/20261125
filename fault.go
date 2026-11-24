package xrpath

const (
	FaultPoseDrift             = "pose_drift"
	FaultResourceMissing       = "resource_missing"
	FaultVisibilityDrop        = "visibility_drop"
	FaultReprojectionStale     = "reprojection_stale"
	FaultLeftImagePixel        = "left_image_pixel_corruption"
	FaultRightImagePixel       = "right_image_pixel_corruption"
	FaultLifecycleSkipPause    = "lifecycle_skip_pause_frame"
	FaultParametersStale       = "parameters_stale_revision"
	FaultTrackingWrongKeyframe = "tracking_wrong_keyframe"
	FaultHashOnlyMask          = "hash_only_mask"
)

type Fault struct {
	Frame  int    `json:"frame"`
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
}

func CanonicalFaultMatrix() []Fault {
	return []Fault{
		{Frame: FaultPoseFrame, Kind: FaultPoseDrift, Target: "head"},
		{Frame: FaultResourceFrame, Kind: FaultResourceMissing, Target: "tex-crate-blue"},
		{Frame: FaultVisibilityFrame, Kind: FaultVisibilityDrop, Target: "occluder-wall"},
		{Frame: FaultReprojectFrame, Kind: FaultReprojectionStale, Target: "left"},
		{Frame: FaultImageFrame, Kind: FaultLeftImagePixel, Target: "left"},
		{Frame: FaultLifecycleFrame, Kind: FaultLifecycleSkipPause, Target: "lifecycle"},
		{Frame: FaultParametersFrame, Kind: FaultParametersStale, Target: "parameters"},
		{Frame: 56, Kind: FaultTrackingWrongKeyframe, Target: "kf-atrium-07"},
	}
}
