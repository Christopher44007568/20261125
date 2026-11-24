package stereorollback

import "fmt"

type eyeOutcome struct {
	surface EyeSurface
	failure *FailureRecord
}

// ExecuteEye runs one isolated eye attempt for a numbered frame. On failure
// the attempt has already been rolled back and the returned surface is the
// declared placeholder. It lets callers drive a single eye directly.
func (p *Pipeline) ExecuteEye(frame uint64, eye Eye) (EyeSurface, *FailureRecord) {
	p.pool.SetFrame(frame)
	out := p.renderEye(frame, eye)
	return out.surface, out.failure
}

// RunFrame renders one stereo frame. Eye attempts are isolated: a failure in
// one eye is rolled back locally and replaced by its declared placeholder,
// while the other eye still renders and the shared pool stays intact.
func (p *Pipeline) RunFrame(frame uint64) FrameRecord {
	p.pool.SetFrame(frame)
	rec := FrameRecord{Index: frame}

	if id, scheduled := p.script.CorruptAt(frame); scheduled {
		_ = p.pool.Corrupt(id)
	}

	// Shared-texture integrity is a distinct, non-eye-specific path.
	if _, err := p.pool.Verify(ResSharedAtlas); err != nil {
		corruption, fallback := p.handleSharedCorruption(frame, ResSharedAtlas)
		rec.SharedCorruption = corruption
		rec.Left = fallback.Left
		rec.Right = fallback.Right
		fb := fallback
		rec.FallbackFrame = &fb
		rec.FullSuccess = false
		return rec
	}

	left := p.renderEye(frame, EyeLeft)
	right := p.renderEye(frame, EyeRight)
	rec.Left = left.surface
	rec.Right = right.surface

	failures := make([]FailureRecord, 0, 2)
	if left.failure != nil {
		failures = append(failures, *left.failure)
	}
	if right.failure != nil {
		failures = append(failures, *right.failure)
	}
	if len(failures) > 0 {
		rec.Failures = failures
		if len(failures) == 1 {
			only := failures[0]
			rec.Failure = &only
			fb := newEyeFallbackFrame(frame, only.Eye, rec.Left, rec.Right)
			rec.FallbackFrame = &fb
		} else {
			fb := newEyeFallbackFrame(frame, "", rec.Left, rec.Right)
			fb.Kind = "both_eyes_placeholder"
			fb.Scope = ""
			fb.Note = "both eyes failed; each shows its own placeholder"
			rec.FallbackFrame = &fb
		}
	}

	rec.FullSuccess = rec.Left.Status == StatusRendered && rec.Right.Status == StatusRendered
	return rec
}

func (p *Pipeline) renderEye(frame uint64, eye Eye) eyeOutcome {
	tx := newEyeTx(frame, eye)
	fault := p.script.FaultFor(frame, eye)

	// Stage 1: attachment creation -----------------------------------------
	attachID := ResourceID(fmt.Sprintf("attach.%s.f%d", eye.Label(), frame))
	if fault == FaultAttachmentCreate {
		_, _ = p.pool.BeginPartial(ResourceDesc{
			ID: attachID, Kind: KindColorAttachment, Shared: false,
			Width: p.width, Height: p.height,
		})
		tx.created = append(tx.created, attachID)
		stageErr := &StageError{Eye: eye, Stage: StageAttachment, Fault: fault, Err: ErrAttachmentCreate}
		return p.failEye(tx, stageErr)
	}
	if _, err := p.pool.Create(ResourceDesc{
		ID: attachID, Kind: KindColorAttachment, Shared: false,
		Width: p.width, Height: p.height,
	}); err != nil {
		stageErr := &StageError{Eye: eye, Stage: StageAttachment, Fault: FaultAttachmentCreate, Err: err}
		return p.failEye(tx, stageErr)
	}
	tx.created = append(tx.created, attachID)

	for _, sharedID := range []ResourceID{ResSharedAtlas, ResSharedDepth, ResSharedMaterial} {
		lease, err := p.pool.Acquire(sharedID, OwnerEye(eye))
		if err != nil {
			stageErr := &StageError{Eye: eye, Stage: StageAttachment, Fault: FaultAttachmentCreate, Err: err}
			return p.failEye(tx, stageErr)
		}
		tx.addLease(lease)
	}

	// Stage 2: material parameters ------------------------------------------
	params := p.materials[eye]
	if fault == FaultMaterialParameter {
		p.materials[eye] = injectedBadMaterial(ResSharedAtlas)
		params = p.materials[eye]
		tx.changedParams = append(tx.changedParams, "roughness")
	}
	if field, err := params.Validate(); err != nil {
		stageErr := &StageError{Eye: eye, Stage: StageMaterial, Fault: FaultMaterialParameter,
			Err: ErrMaterialParameter, Field: field}
		return p.failEye(tx, stageErr)
	}

	// Stage 3: reprojection --------------------------------------------------
	elapsed := p.script.ElapsedFor(frame, eye, 4)
	if fault == FaultReprojectionTimeout {
		elapsed = p.budget + 42
	}
	if fault == FaultReprojectionTimeout || elapsed > p.budget {
		stageErr := &StageError{Eye: eye, Stage: StageReprojection, Fault: FaultReprojectionTimeout,
			Err: ErrReprojectionTimeout, ElapsedMS: elapsed, BudgetMS: p.budget}
		return p.failEye(tx, stageErr)
	}

	img := eyeImage(eye, frame)
	surface := renderedSurface(eye, img, attachID)
	p.commit(tx)
	return eyeOutcome{surface: surface}
}

func (p *Pipeline) failEye(tx *eyeTx, cause *StageError) eyeOutcome {
	rb := p.rollback(tx)
	failure := &FailureRecord{
		Frame:        tx.frame,
		Scope:        ScopeForEye(tx.eye),
		Eye:          tx.eye,
		FailureStage: cause.Stage,
		Fault:        cause.Fault,
		Message:      cause.Error(),
		Field:        cause.Field,
		ElapsedMS:    cause.ElapsedMS,
		BudgetMS:     cause.BudgetMS,
		Rollback:     rb,
	}
	surface := placeholderSurface(tx.eye, p.placeholders[tx.eye], cause.Stage, "")
	return eyeOutcome{surface: surface, failure: failure}
}
