package stereorollback

// RetainedResource describes a shared resource that survived one eye's
// rollback because other users still hold leases.
type RetainedResource struct {
	ID         ResourceID   `json:"id"`
	Kind       ResourceKind `json:"kind"`
	Shared     bool         `json:"shared"`
	Generation int          `json:"generation"`
	RefCount   int          `json:"ref_count"`
	Users      []string     `json:"users"`
	OtherUsers []string     `json:"other_users"`
	Reason     string       `json:"reason"`
}

// RollbackRecord is the audit trail of a single-eye rollback. It lists the
// eye-private objects that were destroyed, the leases that were released and
// the shared resources that were retained for other users.
type RollbackRecord struct {
	Frame             uint64             `json:"frame"`
	Eye               Eye                `json:"eye"`
	Scope             FailureScope       `json:"scope"`
	DestroyedPrivate  []ResourceID       `json:"destroyed_private_resources"`
	ReleasedLeases    []Lease            `json:"released_leases"`
	RetainedResources []RetainedResource `json:"retained_resources"`
	RestoredParams    []string           `json:"restored_material_params,omitempty"`
	Note              string             `json:"note"`
}

// RollbackReport is the audit report of a single-eye rollback. It is an
// alias for RollbackRecord under the vocabulary used in the feedback.
type RollbackReport = RollbackRecord

// FailureRecord captures the failure stage and fault for one incident.
type FailureRecord struct {
	Frame        uint64          `json:"frame"`
	Scope        FailureScope    `json:"scope"`
	Eye          Eye             `json:"eye,omitempty"`
	FailureStage Stage           `json:"failure_stage"`
	Fault        FaultType       `json:"fault"`
	Message      string          `json:"message"`
	Field        string          `json:"field,omitempty"`
	ElapsedMS    int             `json:"elapsed_ms,omitempty"`
	BudgetMS     int             `json:"budget_ms,omitempty"`
	Rollback     *RollbackRecord `json:"rollback,omitempty"`
}

// CorruptionRecord is the separate handling path for a corrupt shared texture.
// The corrupt texture is quarantined and recreated while every user is migrated
// to the replacement; no other user is dropped and unrelated resources survive.
type CorruptionRecord struct {
	Frame                uint64       `json:"frame"`
	Scope                FailureScope `json:"scope"`
	ResourceID           ResourceID   `json:"resource_id"`
	OldGeneration        int          `json:"old_generation"`
	ExpectedChecksum     string       `json:"expected_checksum"`
	ActualChecksum       string       `json:"actual_checksum"`
	ReplacementID        ResourceID   `json:"replacement_id"`
	NewGeneration        int          `json:"new_generation"`
	AffectedUsers        []string     `json:"affected_users"`
	MigratedUsers        []string     `json:"migrated_users"`
	DroppedUsers         []string     `json:"dropped_users"`
	QuarantinedResources []ResourceID `json:"quarantined_resources"`
	UnaffectedResources  []ResourceID `json:"unaffected_resources"`
	Note                 string       `json:"note"`
}

// FrameRecord is the per-frame outcome used by reports and assertions.
type FrameRecord struct {
	Index       uint64     `json:"frame"`
	Left        EyeSurface `json:"left"`
	Right       EyeSurface `json:"right"`
	FullSuccess bool       `json:"full_success"`
	// Failure is set when exactly one eye failed (the central isolation case).
	Failure *FailureRecord `json:"failure,omitempty"`
	// Failures contains every eye failure in the frame, if any.
	Failures         []FailureRecord   `json:"failures,omitempty"`
	FallbackFrame    *FallbackFrame    `json:"fallback_frame,omitempty"`
	SharedCorruption *CorruptionRecord `json:"shared_corruption,omitempty"`
}

// HealthyEye returns the preserved eye and true when exactly one eye failed
// and the other surface is a real render.
func (f FrameRecord) HealthyEye() (EyeSurface, bool) {
	if f.Left.Status == StatusRendered && f.Right.Status != StatusRendered {
		return f.Left, true
	}
	if f.Right.Status == StatusRendered && f.Left.Status != StatusRendered {
		return f.Right, true
	}
	return EyeSurface{}, false
}
