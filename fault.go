package stereorollback

// Stage is a point in the per-eye pipeline at which a failure can be raised.
type Stage string

const (
	// StageAttachment is render attachment (color/depth target) creation.
	StageAttachment Stage = "attachment_creation"
	// StageMaterial is material parameter validation and binding.
	StageMaterial Stage = "material_parameters"
	// StageReprojection is the async reprojection / time-warp step.
	StageReprojection Stage = "reprojection"
	// StageSharedValidation is the shared-texture integrity check.
	StageSharedValidation Stage = "shared_texture_validation"
)

// FaultType names an injected fault.
type FaultType string

const (
	FaultNone FaultType = ""

	// FaultAttachmentCreate injects a failure while creating attachments.
	FaultAttachmentCreate FaultType = "attachment_create_failed"
	// FaultMaterialParameter injects an invalid material parameter.
	FaultMaterialParameter FaultType = "material_parameter_error"
	// FaultReprojectionTimeout injects a reprojection deadline overrun.
	FaultReprojectionTimeout FaultType = "reprojection_timeout"
	// FaultSharedTextureCorrupt flips the contents/checksum of a shared texture.
	FaultSharedTextureCorrupt FaultType = "shared_texture_corrupt"
)

// Stage returns the pipeline stage that owns f.
func (f FaultType) Stage() Stage {
	switch f {
	case FaultAttachmentCreate:
		return StageAttachment
	case FaultMaterialParameter:
		return StageMaterial
	case FaultReprojectionTimeout:
		return StageReprojection
	case FaultSharedTextureCorrupt:
		return StageSharedValidation
	default:
		return ""
	}
}

// FailureScope separates left-eye, right-eye and shared-resource incidents so
// each can be handled (and reported) through its own path.
type FailureScope string

const (
	ScopeLeftEye        FailureScope = "left_eye_failure"
	ScopeRightEye       FailureScope = "right_eye_failure"
	ScopeSharedResource FailureScope = "shared_texture_corruption"
)

// ScopeForEye maps an eye to its dedicated failure scope.
func ScopeForEye(eye Eye) FailureScope {
	if eye == EyeLeft {
		return ScopeLeftEye
	}
	return ScopeRightEye
}
