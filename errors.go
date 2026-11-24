package stereorollback

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors used by the simulated pipeline and resource pool.
var (
	// ErrAttachmentCreate is returned when an attachment cannot be created.
	ErrAttachmentCreate = errors.New("attachment creation failed")
	// ErrMaterialParameter is returned when material parameters are invalid.
	ErrMaterialParameter = errors.New("material parameter error")
	// ErrReprojectionTimeout wraps context.DeadlineExceeded.
	ErrReprojectionTimeout = fmt.Errorf("reprojection timed out: %w", context.DeadlineExceeded)
	// ErrSharedTextureCorrupt is returned on a checksum mismatch.
	ErrSharedTextureCorrupt = errors.New("shared texture is corrupt")

	// ErrResourceBusy refuses destruction while other holders remain.
	ErrResourceBusy = errors.New("resource still has users")
	// ErrUnknownResource is returned for an unknown resource id/lease.
	ErrUnknownResource = errors.New("unknown resource")
	// ErrUnknownLease is returned when a lease token is not recognised.
	ErrUnknownLease = errors.New("unknown lease")
)

// StageError pairs a pipeline stage with the underlying fault.
type StageError struct {
	Eye   Eye
	Stage Stage
	Fault FaultType
	Err   error

	Field     string
	ElapsedMS int
	BudgetMS  int
}

func (e *StageError) Error() string {
	who := "shared"
	if e.Eye.Valid() {
		who = string(e.Eye) + " eye"
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: stage=%s fault=%s field=%s: %v",
			who, e.Stage, e.Fault, e.Field, e.Err)
	}
	if e.BudgetMS > 0 || e.ElapsedMS > 0 {
		return fmt.Sprintf("%s: stage=%s fault=%s elapsed=%dms budget=%dms: %v",
			who, e.Stage, e.Fault, e.ElapsedMS, e.BudgetMS, e.Err)
	}
	return fmt.Sprintf("%s: stage=%s fault=%s: %v", who, e.Stage, e.Fault, e.Err)
}

func (e *StageError) Unwrap() error { return e.Err }
