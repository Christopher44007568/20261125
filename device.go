package texturelife

import "fmt"

// EyeSet is the color, depth, and history texture group for one eye.
type EyeSet struct {
	Eye     Eye
	Size    Size
	Color   TextureID
	Depth   TextureID
	History TextureID
}

// IDs returns the set's texture IDs in color, depth, history order.
func (s EyeSet) IDs() []TextureID {
	return []TextureID{s.Color, s.Depth, s.History}
}

// Device simulates a stereo viewport and owns the current frame references for
// its eye sets. A frame passed to a mutation must be newer than the device's
// current frame, so the transition can acquire new references before dropping
// old ones. Device itself is intended to be driven by one goroutine; the
// underlying Tracker is concurrency-safe.
type Device struct {
	tracker *Tracker

	initialized  bool
	currentFrame Frame
	viewport     Size
	eyes         map[Eye]EyeSet
}

// NewDevice creates a device backed by tracker.
func NewDevice(tracker *Tracker) *Device {
	if tracker == nil {
		tracker = NewTracker()
	}
	return &Device{tracker: tracker, eyes: make(map[Eye]EyeSet)}
}

// Tracker returns the device's resource tracker.
func (d *Device) Tracker() *Tracker {
	return d.tracker
}

// Frame returns the frame currently owning the device's eye references.
func (d *Device) Frame() Frame {
	return d.currentFrame
}

// Viewport returns the active viewport size.
func (d *Device) Viewport() Size {
	return d.viewport
}

// EyeSet returns the active texture set for eye.
func (d *Device) EyeSet(eye Eye) (EyeSet, bool) {
	set, ok := d.eyes[eye]
	return set, ok
}

// Initialize allocates both eyes and acquires their textures in frame.
func (d *Device) Initialize(frame Frame, size Size) error {
	if frame == 0 {
		return ErrInvalidFrame
	}
	if !size.valid() {
		return ErrInvalidSize
	}
	if d.initialized {
		return ErrDeviceInitialized
	}
	if d.currentFrame != 0 && frame <= d.currentFrame {
		return fmt.Errorf("%w: current=%d new=%d", ErrFrameNotAdvancing, d.currentFrame, frame)
	}

	sets := make(map[Eye]EyeSet, 2)
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		set, err := d.createAndAcquireSet(frame, eye, size)
		if err != nil {
			d.rollbackSets(frame, sets)
			return err
		}
		sets[eye] = set
	}

	d.eyes = sets
	d.viewport = size
	d.currentFrame = frame
	d.initialized = true
	return nil
}

// Resize rebuilds every active eye at size. New references are acquired in
// frame before the old buffers are retired and their old references released.
func (d *Device) Resize(frame Frame, size Size) error {
	if err := d.validateTransition(frame); err != nil {
		return err
	}
	if !size.valid() {
		return ErrInvalidSize
	}
	eyes := d.activeEyes()
	if len(eyes) == 0 {
		return ErrNoActiveEyes
	}

	newSets := make(map[Eye]EyeSet, len(eyes))
	for _, eye := range eyes {
		set, err := d.createAndAcquireSet(frame, eye, size)
		if err != nil {
			d.rollbackSets(frame, newSets)
			return err
		}
		newSets[eye] = set
	}

	for _, eye := range eyes {
		d.retireSet(frame, d.eyes[eye])
	}
	oldFrame := d.currentFrame
	for _, eye := range eyes {
		d.releaseSet(oldFrame, frame, d.eyes[eye])
	}

	d.eyes = newSets
	d.viewport = size
	d.currentFrame = frame
	return nil
}

// Rebuild is the successful device-rebuild path.
func (d *Device) Rebuild(frame Frame, size Size) error {
	return d.Resize(frame, size)
}

// DestroyEye retires one eye's buffers while keeping the other eye active. The
// surviving eye is re-acquired in frame before the old frame references drop.
func (d *Device) DestroyEye(frame Frame, eye Eye) error {
	if err := d.validateTransition(frame); err != nil {
		return err
	}
	target, ok := d.eyes[eye]
	if !ok {
		return fmt.Errorf("%w: %s", ErrEyeNotActive, eye)
	}

	kept := d.activeEyes()
	acquired := make([]TextureID, 0, len(kept)*3)
	for _, keptEye := range kept {
		if keptEye == eye {
			continue
		}
		for _, id := range d.eyes[keptEye].IDs() {
			if err := d.tracker.Acquire(frame, id); err != nil {
				for _, acquiredID := range acquired {
					_ = d.tracker.ReleaseAt(frame, frame, acquiredID)
				}
				return err
			}
			acquired = append(acquired, id)
		}
	}

	d.retireSet(frame, target)
	oldFrame := d.currentFrame
	for _, keptEye := range kept {
		if keptEye != eye {
			d.releaseSet(oldFrame, frame, d.eyes[keptEye])
		}
	}
	d.releaseSet(oldFrame, frame, target)

	delete(d.eyes, eye)
	d.currentFrame = frame
	if len(d.eyes) == 0 {
		d.initialized = false
		d.viewport = Size{}
	}
	return nil
}

// SimulateRebuildFailure allocates the first active eye's candidate buffers in
// frame, then fails before commit. Candidates are retired and their frame
// references released, leaving the active viewport and old buffers untouched.
func (d *Device) SimulateRebuildFailure(frame Frame, size Size) error {
	if err := d.validateTransition(frame); err != nil {
		return err
	}
	if !size.valid() {
		return ErrInvalidSize
	}
	eyes := d.activeEyes()
	if len(eyes) == 0 {
		return ErrNoActiveEyes
	}

	candidate, err := d.createAndAcquireSet(frame, eyes[0], size)
	if err != nil {
		return err
	}
	d.retireSet(frame, candidate)
	d.releaseSet(frame, frame, candidate)
	return fmt.Errorf("%w: allocated and rolled back %s buffers", ErrRebuildFailed, eyes[0])
}

// Shutdown retires all active eyes and drops their current-frame references.
// It does not release references owned by other frames.
func (d *Device) Shutdown(frame Frame) error {
	if err := d.validateTransition(frame); err != nil {
		return err
	}
	eyes := d.activeEyes()
	if len(eyes) == 0 {
		return ErrNoActiveEyes
	}

	for _, eye := range eyes {
		d.retireSet(frame, d.eyes[eye])
	}
	oldFrame := d.currentFrame
	for _, eye := range eyes {
		d.releaseSet(oldFrame, frame, d.eyes[eye])
	}
	clear(d.eyes)
	d.initialized = false
	d.currentFrame = frame
	d.viewport = Size{}
	return nil
}

func (d *Device) validateTransition(frame Frame) error {
	if frame == 0 {
		return ErrInvalidFrame
	}
	if !d.initialized {
		return ErrDeviceNotReady
	}
	if frame <= d.currentFrame {
		return fmt.Errorf("%w: current=%d new=%d", ErrFrameNotAdvancing, d.currentFrame, frame)
	}
	return nil
}

func (d *Device) activeEyes() []Eye {
	eyes := make([]Eye, 0, len(d.eyes))
	if _, ok := d.eyes[EyeLeft]; ok {
		eyes = append(eyes, EyeLeft)
	}
	if _, ok := d.eyes[EyeRight]; ok {
		eyes = append(eyes, EyeRight)
	}
	return eyes
}

func (d *Device) createAndAcquireSet(frame Frame, eye Eye, size Size) (EyeSet, error) {
	set := EyeSet{Eye: eye, Size: size}
	created := make([]TextureID, 0, 3)
	for _, kind := range []Kind{KindColor, KindDepth, KindHistory} {
		id, err := d.tracker.Create(frame, Desc{Kind: kind, Eye: eye, Size: size})
		if err != nil {
			d.retireIDs(frame, created)
			return EyeSet{}, err
		}
		created = append(created, id)
	}
	set.Color, set.Depth, set.History = created[0], created[1], created[2]

	acquired := make([]TextureID, 0, len(created))
	for _, id := range created {
		if err := d.tracker.Acquire(frame, id); err != nil {
			for _, acquiredID := range acquired {
				_ = d.tracker.ReleaseAt(frame, frame, acquiredID)
			}
			d.retireIDs(frame, created)
			return EyeSet{}, err
		}
		acquired = append(acquired, id)
	}
	return set, nil
}

func (d *Device) rollbackSets(frame Frame, sets map[Eye]EyeSet) {
	for _, set := range sets {
		d.retireSet(frame, set)
		d.releaseSet(frame, frame, set)
	}
}

func (d *Device) retireSet(frame Frame, set EyeSet) {
	d.retireIDs(frame, set.IDs())
}

func (d *Device) retireIDs(frame Frame, ids []TextureID) {
	for _, id := range ids {
		_ = d.tracker.Retire(frame, id)
	}
}

func (d *Device) releaseSet(owner, at Frame, set EyeSet) {
	for _, id := range set.IDs() {
		_ = d.tracker.ReleaseAt(owner, at, id)
	}
}
