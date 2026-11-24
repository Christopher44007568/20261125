package stereorollback

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
)

// ResourceID identifies a GPU-style resource in the simulated pool.
type ResourceID string

// ResourceKind classifies a resource.
type ResourceKind string

const (
	KindColorAttachment ResourceKind = "color_attachment"
	KindDepthAttachment ResourceKind = "depth_attachment"
	KindEyeTarget       ResourceKind = "eye_target"
	KindSharedTexture   ResourceKind = "shared_texture"
	KindSharedDepth     ResourceKind = "shared_depth"
	KindSharedMaterial  ResourceKind = "shared_material"
	KindSharedHistory   ResourceKind = "shared_history"
	KindFallbackImage   ResourceKind = "fallback_image"
)

// LeaseID is an opaque reference-count token returned by Acquire.
type LeaseID uint64

// Lease is one reference-count hold on a resource.
type Lease struct {
	ID         LeaseID    `json:"lease_id"`
	ResourceID ResourceID `json:"resource_id"`
	Owner      string     `json:"owner"`
}

// ResourceDesc describes a resource to create.
type ResourceDesc struct {
	ID     ResourceID
	Kind   ResourceKind
	Shared bool
	Width  int
	Height int
	Pixels []byte
}

// ResourceInfo is an immutable, JSON-friendly resource snapshot.
type ResourceInfo struct {
	ID            ResourceID     `json:"id"`
	Kind          ResourceKind   `json:"kind"`
	Shared        bool           `json:"shared"`
	Generation    int            `json:"generation"`
	Width         int            `json:"width,omitempty"`
	Height        int            `json:"height,omitempty"`
	Checksum      string         `json:"checksum,omitempty"`
	Corrupt       bool           `json:"corrupt,omitempty"`
	Quarantined   bool           `json:"quarantined,omitempty"`
	State         string         `json:"state"`
	RefCount      int            `json:"ref_count"`
	Users         []string       `json:"users"`
	HolderRefs    map[string]int `json:"holder_refs,omitempty"`
	CreatedFrame  uint64         `json:"created_frame"`
	ReleasedFrame uint64         `json:"released_frame,omitempty"`
}

type resource struct {
	desc          ResourceDesc
	generation    int
	checksum      string
	corrupt       bool
	quarantined   bool
	creating      bool
	createdFrame  uint64
	releasedFrame uint64

	holderRefs map[string]int
	leases     map[LeaseID]string
}

func checksumBytes(pixels []byte) string {
	sum := sha256.Sum256(pixels)
	return hex.EncodeToString(sum[:])
}

// ResourcePool is a ref-counted registry of shared and eye-private resources.
// Destroy never succeeds while another holder keeps a lease, which is the
// invariant that protects other users of shared resources during rollback.
type ResourcePool struct {
	mu      sync.Mutex
	next    LeaseID
	frame   uint64
	active  map[ResourceID]*resource
	history []ResourceInfo
}

// NewResourcePool creates an empty pool.
func NewResourcePool() *ResourcePool {
	return &ResourcePool{active: make(map[ResourceID]*resource)}
}

// SetFrame tags subsequent creates/releases with the logical frame number.
func (p *ResourcePool) SetFrame(frame uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frame = frame
}

// Create registers a fully initialised resource.
func (p *ResourcePool) Create(desc ResourceDesc) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.active[desc.ID]; exists {
		return ResourceInfo{}, fmt.Errorf("resource %q already exists", desc.ID)
	}
	pixels := append([]byte(nil), desc.Pixels...)
	r := &resource{
		desc:         desc,
		generation:   1,
		checksum:     checksumBytes(pixels),
		createdFrame: p.frame,
		holderRefs:   make(map[string]int),
		leases:       make(map[LeaseID]string),
	}
	r.desc.Pixels = pixels
	p.active[desc.ID] = r
	return p.infoLocked(r), nil
}

// BeginPartial registers a resource that is still being created. A failed
// attachment creator uses it so rollback can prove partial objects are removed.
func (p *ResourcePool) BeginPartial(desc ResourceDesc) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.active[desc.ID]; exists {
		return ResourceInfo{}, fmt.Errorf("resource %q already exists", desc.ID)
	}
	r := &resource{
		desc:         desc,
		generation:   1,
		checksum:     checksumBytes(desc.Pixels),
		creating:     true,
		createdFrame: p.frame,
		holderRefs:   make(map[string]int),
		leases:       make(map[LeaseID]string),
	}
	p.active[desc.ID] = r
	return p.infoLocked(r), nil
}

// Acquire adds one reference-count lease for owner.
func (p *ResourcePool) Acquire(id ResourceID, owner string) (Lease, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok || r.quarantined {
		return Lease{}, fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	p.next++
	leaseID := p.next
	r.holderRefs[owner]++
	r.leases[leaseID] = owner
	return Lease{ID: leaseID, ResourceID: id, Owner: owner}, nil
}

// Release gives back one lease. Releasing an unknown lease is an error so
// double-release bugs are visible instead of silently corrupting the count.
func (p *ResourcePool) Release(lease Lease) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[lease.ResourceID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownResource, lease.ResourceID)
	}
	owner, ok := r.leases[lease.ID]
	if !ok || owner != lease.Owner {
		return fmt.Errorf("%w: id=%d resource=%s", ErrUnknownLease, lease.ID, lease.ResourceID)
	}
	delete(r.leases, lease.ID)
	r.holderRefs[owner]--
	if r.holderRefs[owner] <= 0 {
		delete(r.holderRefs, owner)
	}
	return nil
}

// RefCount returns the total outstanding leases for id.
func (p *ResourcePool) RefCount(id ResourceID) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return 0
	}
	total := 0
	for _, n := range r.holderRefs {
		total += n
	}
	return total
}

// Users returns the sorted distinct holders of id.
func (p *ResourcePool) Users(id ResourceID) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return nil
	}
	users := make([]string, 0, len(r.holderRefs))
	for u := range r.holderRefs {
		users = append(users, u)
	}
	sort.Strings(users)
	return users
}

// Destroy removes a resource. It refuses with ErrResourceBusy when other
// leases are still outstanding.
func (p *ResourcePool) Destroy(id ResourceID) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return ResourceInfo{}, fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	if total := p.refCountLocked(r); total > 0 {
		return p.infoLocked(r), fmt.Errorf("%w: %s has %d refs (%v)",
			ErrResourceBusy, id, total, p.usersLocked(r))
	}
	r.releasedFrame = p.frame
	info := p.infoLocked(r)
	p.history = append(p.history, info)
	delete(p.active, id)
	return info, nil
}

// Corrupt flips a byte of a resource's pixels without updating its checksum.
func (p *ResourcePool) Corrupt(id ResourceID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	if len(r.desc.Pixels) == 0 {
		r.corrupt = true
		r.checksum = "expected-good"
		return nil
	}
	r.desc.Pixels[0] ^= 0xFF
	r.corrupt = true
	return nil
}

// Verify recomputes the checksum and returns ErrSharedTextureCorrupt on mismatch.
func (p *ResourcePool) Verify(id ResourceID) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return ResourceInfo{}, fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	actual := checksumBytes(r.desc.Pixels)
	if r.corrupt || actual != r.checksum {
		r.corrupt = true
		return p.infoLocked(r), fmt.Errorf("%w: %s checksum mismatch", ErrSharedTextureCorrupt, id)
	}
	r.corrupt = false
	return p.infoLocked(r), nil
}

// Checksums returns the stored (expected) and current (actual) checksums.
func (p *ResourcePool) Checksums(id ResourceID) (expected, actual string, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, exists := p.active[id]
	if !exists {
		return "", "", false
	}
	return r.checksum, checksumBytes(r.desc.Pixels), true
}

// Restore replaces a resource's pixels with a known-good snapshot, clears the
// corrupt flag and bumps its generation. Holders and leases are untouched, so
// every existing user keeps working with the restored shared resource.
func (p *ResourcePool) Restore(id ResourceID, pixels []byte) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return ResourceInfo{}, fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	r.desc.Pixels = append([]byte(nil), pixels...)
	r.checksum = checksumBytes(r.desc.Pixels)
	r.corrupt = false
	r.quarantined = false
	r.creating = false
	r.generation++
	return p.infoLocked(r), nil
}

// Quarantine marks a corrupt resource as quarantined. Existing holder counts
// are preserved for audit; new Acquire calls are refused until Restore.
func (p *ResourcePool) Quarantine(id ResourceID) (ResourceInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return ResourceInfo{}, fmt.Errorf("%w: %s", ErrUnknownResource, id)
	}
	r.quarantined = true
	r.corrupt = true
	return p.infoLocked(r), nil
}

// Info returns the snapshot of one active resource.
func (p *ResourcePool) Info(id ResourceID) (ResourceInfo, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.active[id]
	if !ok {
		return ResourceInfo{}, false
	}
	return p.infoLocked(r), true
}

// List returns sorted snapshots of all active resources.
func (p *ResourcePool) List() []ResourceInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	ids := make([]ResourceID, 0, len(p.active))
	for id := range p.active {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]ResourceInfo, 0, len(ids))
	for _, id := range ids {
		out = append(out, p.infoLocked(p.active[id]))
	}
	return out
}

func (p *ResourcePool) refCountLocked(r *resource) int {
	total := 0
	for _, n := range r.holderRefs {
		total += n
	}
	return total
}

func (p *ResourcePool) usersLocked(r *resource) []string {
	users := make([]string, 0, len(r.holderRefs))
	for u := range r.holderRefs {
		users = append(users, u)
	}
	sort.Strings(users)
	return users
}

func (p *ResourcePool) infoLocked(r *resource) ResourceInfo {
	holderRefs := make(map[string]int, len(r.holderRefs))
	for u, n := range r.holderRefs {
		holderRefs[u] = n
	}
	state := "ready"
	switch {
	case r.quarantined:
		state = "quarantined"
	case r.creating:
		state = "creating"
	case r.corrupt:
		state = "corrupt"
	}
	return ResourceInfo{
		ID:            r.desc.ID,
		Kind:          r.desc.Kind,
		Shared:        r.desc.Shared,
		Generation:    r.generation,
		Width:         r.desc.Width,
		Height:        r.desc.Height,
		Checksum:      r.checksum,
		Corrupt:       r.corrupt,
		Quarantined:   r.quarantined,
		State:         state,
		RefCount:      p.refCountLocked(r),
		Users:         p.usersLocked(r),
		HolderRefs:    holderRefs,
		CreatedFrame:  r.createdFrame,
		ReleasedFrame: r.releasedFrame,
	}
}
