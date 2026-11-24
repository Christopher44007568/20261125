package texturelife

import (
	"fmt"
	"math"
	"sync"
)

type textureRecord struct {
	Texture
}

type scopePeak struct {
	count    int
	bytes    int64
	refCount int
	frame    Frame
}

// Tracker owns texture metadata and frame references. All methods are safe for
// concurrent use. Frames are logical time: release operations use their frame
// argument as the release timestamp.
type Tracker struct {
	mu sync.Mutex

	nextID  TextureID
	records map[TextureID]*textureRecord
	order   []TextureID
	refs    map[Frame]map[TextureID]int

	liveCount int
	liveBytes int64
	liveRefs  int

	peakCount      int
	peakBytes      int64
	peakCountFrame Frame
	peakBytesFrame Frame
	peakRefs       int
	peakRefsFrame  Frame

	peakByEye  map[Eye]scopePeak
	peakByKind map[Kind]scopePeak
}

// NewTracker creates an empty resource tracker.
func NewTracker() *Tracker {
	return &Tracker{
		records:    make(map[TextureID]*textureRecord),
		refs:       make(map[Frame]map[TextureID]int),
		peakByEye:  make(map[Eye]scopePeak),
		peakByKind: make(map[Kind]scopePeak),
	}
}

// Create allocates a texture. The returned resource has no frame references;
// callers normally acquire it in the frame that first uses it.
func (tr *Tracker) Create(frame Frame, desc Desc) (TextureID, error) {
	if frame == 0 {
		return 0, ErrInvalidFrame
	}
	if !desc.Eye.valid() {
		return 0, ErrInvalidEye
	}
	if !desc.Kind.valid() {
		return 0, ErrInvalidKind
	}
	if !desc.Size.valid() {
		return 0, ErrInvalidSize
	}
	if int64(desc.Size.Width) > math.MaxInt64/int64(desc.Size.Height) {
		return 0, fmt.Errorf("%w: pixel count overflows int64", ErrInvalidSize)
	}
	pixels := int64(desc.Size.Width) * int64(desc.Size.Height)
	if pixels > math.MaxInt64/BytesPerPixel(desc.Kind) {
		return 0, fmt.Errorf("%w: byte count overflows int64", ErrInvalidSize)
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.nextID++
	rec := &textureRecord{Texture: Texture{
		ID:           tr.nextID,
		Kind:         desc.Kind,
		Eye:          desc.Eye,
		Size:         desc.Size,
		Bytes:        desc.Size.Bytes(desc.Kind),
		CreatedFrame: frame,
	}}
	tr.records[rec.ID] = rec
	tr.order = append(tr.order, rec.ID)
	tr.liveCount++
	tr.liveBytes += rec.Bytes
	tr.updatePeaksLocked(frame)
	return rec.ID, nil
}

// Acquire adds one reference to id for frame and records frame as the latest
// use. A retiring texture stays alive while any frame reference remains.
func (tr *Tracker) Acquire(frame Frame, id TextureID) error {
	if frame == 0 {
		return ErrInvalidFrame
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[id]
	if !ok {
		return fmt.Errorf("%w: %d", ErrTextureAbsent, id)
	}
	if !rec.Alive() {
		return fmt.Errorf("%w: %d", ErrTextureGone, id)
	}

	if tr.refs[frame] == nil {
		tr.refs[frame] = make(map[TextureID]int)
	}
	tr.refs[frame][id]++
	rec.RefCount++
	tr.liveRefs++
	if rec.RefCount > rec.PeakRefCount {
		rec.PeakRefCount = rec.RefCount
	}
	if frame > rec.LastUsedFrame {
		rec.LastUsedFrame = frame
	}
	tr.updatePeaksLocked(frame)
	return nil
}

// Release removes one reference from id owned by frame. The frame is also used
// as the release timestamp. Use ReleaseAt when the owning frame completed in a
// later simulation frame.
func (tr *Tracker) Release(frame Frame, id TextureID) error {
	return tr.ReleaseAt(frame, frame, id)
}

// ReleaseAt removes one reference owned by owner at simulation time at.
// Physical release happens only if this was the final reference and retirement
// was requested.
func (tr *Tracker) ReleaseAt(owner, at Frame, id TextureID) error {
	if owner == 0 || at == 0 {
		return ErrInvalidFrame
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[id]
	if !ok {
		return fmt.Errorf("%w: %d", ErrTextureAbsent, id)
	}
	if !tr.dropReferenceLocked(owner, id) {
		return fmt.Errorf("%w: frame %d, texture %d", ErrNotReferenced, owner, id)
	}
	if rec.RefCount == 0 && rec.Retiring() {
		tr.releaseLocked(rec, at)
	}
	return nil
}

func (tr *Tracker) dropReferenceLocked(frame Frame, id TextureID) bool {
	frameRefs := tr.refs[frame]
	count := frameRefs[id]
	if count == 0 {
		return false
	}
	if count == 1 {
		delete(frameRefs, id)
		if len(frameRefs) == 0 {
			delete(tr.refs, frame)
		}
	} else {
		frameRefs[id] = count - 1
	}
	rec := tr.records[id]
	rec.RefCount--
	tr.liveRefs--
	return true
}

func (tr *Tracker) releaseLocked(rec *textureRecord, frame Frame) {
	rec.ReleasedAt = frame
	tr.liveCount--
	tr.liveBytes -= rec.Bytes
}

// ReleaseFrame drops every reference owned by frame at the same timestamp.
// Releasing an unknown or already empty frame is a no-op.
func (tr *Tracker) ReleaseFrame(frame Frame) {
	tr.ReleaseFrameAt(frame, frame)
}

// ReleaseFrameAt drops every reference owned by owner at simulation time at.
// It is useful when an old frame completes during a newer transition frame.
func (tr *Tracker) ReleaseFrameAt(owner, at Frame) {
	if owner == 0 || at == 0 {
		return
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	frameRefs := tr.refs[owner]
	for id, count := range frameRefs {
		rec := tr.records[id]
		rec.RefCount -= count
		tr.liveRefs -= count
		if rec.RefCount == 0 && rec.Retiring() {
			tr.releaseLocked(rec, at)
		}
	}
	delete(tr.refs, owner)
}

// Retire requests release. A texture with no remaining references is released
// immediately; otherwise release is deferred until the last reference drops.
func (tr *Tracker) Retire(frame Frame, id TextureID) error {
	if frame == 0 {
		return ErrInvalidFrame
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[id]
	if !ok {
		return fmt.Errorf("%w: %d", ErrTextureAbsent, id)
	}
	if !rec.Alive() {
		return fmt.Errorf("%w: %d", ErrTextureGone, id)
	}
	if !rec.Retiring() {
		rec.RetireFrame = frame
	}
	if rec.RefCount == 0 {
		tr.releaseLocked(rec, frame)
	}
	return nil
}

// Texture returns the current immutable metadata for id.
func (tr *Tracker) Texture(id TextureID) (Texture, bool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[id]
	if !ok {
		return Texture{}, false
	}
	return rec.Texture, true
}

func (tr *Tracker) updatePeaksLocked(frame Frame) {
	if tr.liveCount > tr.peakCount {
		tr.peakCount = tr.liveCount
		tr.peakCountFrame = frame
	}
	if tr.liveBytes > tr.peakBytes {
		tr.peakBytes = tr.liveBytes
		tr.peakBytesFrame = frame
	}
	if tr.liveRefs > tr.peakRefs {
		tr.peakRefs = tr.liveRefs
		tr.peakRefsFrame = frame
	}
	tr.updateScopesLocked(frame)
}

func (tr *Tracker) updateScopesLocked(frame Frame) {
	eyeCount := make(map[Eye]int)
	eyeBytes := make(map[Eye]int64)
	eyeRefs := make(map[Eye]int)
	kindCount := make(map[Kind]int)
	kindBytes := make(map[Kind]int64)
	kindRefs := make(map[Kind]int)
	for _, rec := range tr.records {
		if !rec.Alive() {
			continue
		}
		refs := rec.RefCount
		eyeCount[rec.Eye]++
		eyeBytes[rec.Eye] += rec.Bytes
		eyeRefs[rec.Eye] += refs
		kindCount[rec.Kind]++
		kindBytes[rec.Kind] += rec.Bytes
		kindRefs[rec.Kind] += refs
	}
	for eye, count := range eyeCount {
		p := tr.peakByEye[eye]
		if count > p.count {
			p.count = count
			p.frame = frame
		}
		if eyeBytes[eye] > p.bytes {
			p.bytes = eyeBytes[eye]
			p.frame = frame
		}
		if eyeRefs[eye] > p.refCount {
			p.refCount = eyeRefs[eye]
			p.frame = frame
		}
		tr.peakByEye[eye] = p
	}
	for kind, count := range kindCount {
		p := tr.peakByKind[kind]
		if count > p.count {
			p.count = count
			p.frame = frame
		}
		if kindBytes[kind] > p.bytes {
			p.bytes = kindBytes[kind]
			p.frame = frame
		}
		if kindRefs[kind] > p.refCount {
			p.refCount = kindRefs[kind]
			p.frame = frame
		}
		tr.peakByKind[kind] = p
	}
}
