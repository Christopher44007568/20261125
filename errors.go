package texturelife

import "errors"

var (
	ErrInvalidFrame      = errors.New("frame must be greater than zero")
	ErrInvalidEye        = errors.New("invalid eye")
	ErrInvalidKind       = errors.New("invalid texture kind")
	ErrInvalidSize       = errors.New("texture size must be positive")
	ErrTextureAbsent     = errors.New("texture does not exist")
	ErrTextureGone       = errors.New("texture has already been released")
	ErrNotReferenced     = errors.New("frame does not reference texture")
	ErrDeviceInitialized = errors.New("device is already initialized")
	ErrDeviceNotReady    = errors.New("device is not active")
	ErrNoActiveEyes      = errors.New("device has no active eyes")
	ErrEyeNotActive      = errors.New("eye is not active")
	ErrFrameNotAdvancing = errors.New("new frame must be greater than the device's current frame")
	ErrRebuildFailed     = errors.New("simulated device rebuild failed")
)
