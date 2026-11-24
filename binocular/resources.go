package binocular

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
)

const (
	FrameWidth  = 32
	FrameHeight = 24
	BytesPerPx  = 4
)

func cleanColor(eye Eye) []byte {
	data := make([]byte, FrameWidth*FrameHeight*BytesPerPx)
	base := byte(17)
	if eye == EyeRight {
		base = 91
	}
	for y := 0; y < FrameHeight; y++ {
		for x := 0; x < FrameWidth; x++ {
			i := (y*FrameWidth + x) * BytesPerPx
			data[i] = base + byte(x*3+y*5)
			data[i+1] = base + byte(x*7+y*2)
			data[i+2] = base + byte(x+y*11)
			data[i+3] = 0xff
		}
	}
	return data
}

func cleanDepth(eye Eye) []byte {
	data := make([]byte, FrameWidth*FrameHeight*BytesPerPx)
	seed := byte(0x2a)
	if eye == EyeRight {
		seed = 0x9d
	}
	for i := range data {
		data[i] = seed
	}
	return data
}

// CleanFrame is the canonical first frame of an empty session.
func CleanFrame() Frame {
	return Frame{Left: cleanColor(EyeLeft), Right: cleanColor(EyeRight)}
}

type frameBuffer struct {
	mu       sync.RWMutex
	id       string
	kind     ResourceKind
	eye      Eye
	data     []byte
	released bool
}

func newFrameBuffer(id string, kind ResourceKind, eye Eye, data []byte) *frameBuffer {
	return &frameBuffer{id: id, kind: kind, eye: eye, data: data}
}

func (b *frameBuffer) write(off int, p []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.released {
		return fmt.Errorf("%w: buffer %s", ErrSessionClosed, b.id)
	}
	if off < 0 || off > len(b.data) || len(p) > len(b.data)-off {
		return fmt.Errorf("write outside buffer %s: off=%d len=%d size=%d", b.id, off, len(p), len(b.data))
	}
	copy(b.data[off:], p)
	return nil
}

func (b *frameBuffer) release() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	already := b.released
	for i := range b.data {
		b.data[i] = 0
	}
	b.released = true
	return !already
}

func (b *frameBuffer) releasedState() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.released
}

func (b *frameBuffer) snapshot() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]byte, len(b.data))
	copy(out, b.data)
	return out
}

type resourceSet struct {
	mu        sync.Mutex
	textures  []*frameBuffer
	depths    []*frameBuffer
	histories []*frameBuffer
	cache     map[string]string
}

func newResourceSet(sessionID uint64) *resourceSet {
	r := &resourceSet{cache: make(map[string]string)}
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		prefix := fmt.Sprintf("s%d", sessionID)
		r.textures = append(r.textures,
			newFrameBuffer(prefix+":tex:"+string(eye)+":color", ResourceTexture, eye, cleanColor(eye)),
			newFrameBuffer(prefix+":tex:"+string(eye)+":intermediate", ResourceTexture, eye, make([]byte, FrameWidth*FrameHeight*BytesPerPx)),
		)
		r.depths = append(r.depths,
			newFrameBuffer(prefix+":depth:"+string(eye), ResourceDepth, eye, cleanDepth(eye)),
		)
		r.histories = append(r.histories,
			newFrameBuffer(prefix+":history:"+string(eye), ResourceHistory, eye, make([]byte, FrameWidth*FrameHeight*BytesPerPx)),
		)
	}
	return r
}

func (r *resourceSet) bufferFor(kind ResourceKind, eye Eye) (*frameBuffer, error) {
	if !eye.Valid() {
		return nil, fmt.Errorf("invalid eye %q", eye)
	}
	var list []*frameBuffer
	switch kind {
	case ResourceTexture, ResourceDepth, ResourceHistory:
		if kind == ResourceTexture {
			list = r.textures
		} else if kind == ResourceDepth {
			list = r.depths
		} else {
			list = r.histories
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownBuffer, kind)
	}
	for _, b := range list {
		if b.eye == eye && (kind != ResourceTexture || len(b.id) > 0) {
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrUnknownBuffer, kind, eye)
}

func (r *resourceSet) colorSnapshots() ([]byte, []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var left, right *frameBuffer
	for _, b := range r.textures {
		if b.kind != ResourceTexture || b.data == nil {
			continue
		}
		if b.eye == EyeLeft && len(b.id) > 0 {
			left = b
			break
		}
	}
	for _, b := range r.textures {
		if b.kind != ResourceTexture || b.data == nil {
			continue
		}
		if b.eye == EyeRight {
			right = b
			break
		}
	}
	var l, rr []byte
	if left != nil {
		l = left.snapshot()
	}
	if right != nil {
		rr = right.snapshot()
	}
	return l, rr
}

func (r *resourceSet) putCache(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cache == nil {
		r.cache = make(map[string]string)
	}
	r.cache[key] = value
}

func (r *resourceSet) cacheSize() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.cache)
}

func (r *resourceSet) historyFrames() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.histories)
}

func (r *resourceSet) release() ([]ReleaseCategory, []ObjectRef) {
	r.mu.Lock()
	defer r.mu.Unlock()

	released := func(list []*frameBuffer) (int, int, []string) {
		ok, failed := 0, 0
		ids := make([]string, 0, len(list))
		for _, b := range list {
			ids = append(ids, b.id)
			if b.release() {
				ok++
			} else {
				failed++
			}
		}
		sort.Strings(ids)
		return ok, failed, ids
	}

	texOK, texFail, texIDs := released(r.textures)
	depthOK, depthFail, depthIDs := released(r.depths)
	histOK, histFail, histIDs := released(r.histories)

	keys := make([]string, 0, len(r.cache))
	for k := range r.cache {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		delete(r.cache, k)
	}
	cacheOK := len(keys)
	cacheFail := 0
	if len(r.cache) != 0 {
		cacheFail = len(r.cache)
	}

	categories := []ReleaseCategory{
		{Name: "textures", Discovered: len(r.textures), Released: texOK, Failed: texFail, OK: texFail == 0, Objects: texIDs},
		{Name: "depth", Discovered: len(r.depths), Released: depthOK, Failed: depthFail, OK: depthFail == 0, Objects: depthIDs},
		{Name: "history", Discovered: len(r.histories), Released: histOK, Failed: histFail, OK: histFail == 0, Objects: histIDs},
		{Name: "cache_keys", Discovered: len(keys), Released: cacheOK, Failed: cacheFail, OK: cacheFail == 0, Objects: keys, Keys: keys},
	}
	return categories, r.unreleasedLocked()
}

func (r *resourceSet) unreleasedLocked() []ObjectRef {
	var out []ObjectRef
	add := func(list []*frameBuffer) {
		for _, b := range list {
			if !b.releasedState() {
				out = append(out, ObjectRef{Kind: string(b.kind), ID: b.id, Detail: "buffer not released"})
			}
		}
	}
	add(r.textures)
	add(r.depths)
	add(r.histories)
	keys := make([]string, 0, len(r.cache))
	for k := range r.cache {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, ObjectRef{Kind: "cache_key", ID: k, Detail: "cache entry not cleared"})
	}
	return out
}

func (r *resourceSet) unreleased() []ObjectRef {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.unreleasedLocked()
}

func hashBytes(data []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return fmt.Sprintf("%016x", h.Sum64())
}

func CompareFrames(want, got Frame) FirstFrameDiff {
	pairs := []struct {
		eye  Eye
		want []byte
		got  []byte
	}{
		{eye: EyeLeft, want: want.Left, got: got.Left},
		{eye: EyeRight, want: want.Right, got: got.Right},
	}
	result := FirstFrameDiff{Equal: true}
	for _, p := range pairs {
		side := FrameSideDiff{
			Eye:          p.eye,
			BaselineHash: hashBytes(p.want),
			ObservedHash: hashBytes(p.got),
			Equal:        true,
		}
		n := len(p.want)
		if len(p.got) < n {
			n = len(p.got)
		}
		for i := 0; i < n; i++ {
			if p.want[i] != p.got[i] {
				side.DiffBytes++
			}
		}
		side.DiffBytes += len(p.want) - n
		side.DiffBytes += len(p.got) - n
		for i := 0; i < n; i += BytesPerPx {
			end := i + BytesPerPx
			if end > n {
				end = n
			}
			changed := false
			for j := i; j < end; j++ {
				if p.want[j] != p.got[j] {
					changed = true
					break
				}
			}
			if changed {
				side.DiffPixels++
			}
		}
		side.Equal = side.DiffBytes == 0
		if !side.Equal {
			result.Equal = false
		}
		result.DiffBytes += side.DiffBytes
		result.Sides = append(result.Sides, side)
	}
	return result
}
