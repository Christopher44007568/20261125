package stereorollback

import (
	"errors"
	"sort"
	"strconv"
)

type eyeTx struct {
	eye           Eye
	frame         uint64
	created       []ResourceID
	leases        []Lease
	touchedShared map[ResourceID]bool
	changedParams []string
}

func newEyeTx(frame uint64, eye Eye) *eyeTx {
	return &eyeTx{frame: frame, eye: eye, touchedShared: map[ResourceID]bool{}}
}

func (tx *eyeTx) addLease(lease Lease) {
	tx.leases = append(tx.leases, lease)
	tx.touchedShared[lease.ResourceID] = true
}

// releaseLeases gives back only this attempt's lease tokens. It never removes
// another owner's hold because each Release targets one specific lease id.
func (p *Pipeline) releaseLeases(tx *eyeTx) []Lease {
	released := make([]Lease, 0, len(tx.leases))
	for _, lease := range tx.leases {
		if err := p.pool.Release(lease); err == nil {
			released = append(released, lease)
		}
	}
	return released
}

// destroyPrivate deletes resources created by this attempt. Shared resources
// are protected by an explicit guard and by the pool's busy-ref refusal.
func (p *Pipeline) destroyPrivate(tx *eyeTx, rb *RollbackRecord) {
	for _, id := range tx.created {
		info, known := p.pool.Info(id)
		if known && info.Shared {
			rb.RetainedResources = append(rb.RetainedResources, RetainedResource{
				ID: id, Kind: info.Kind, Shared: true, Generation: info.Generation,
				RefCount: info.RefCount, Users: info.Users,
				Reason: "refusing to destroy a shared resource during single-eye rollback",
			})
			continue
		}
		if _, err := p.pool.Destroy(id); err != nil {
			if errors.Is(err, ErrResourceBusy) {
				if latest, ok := p.pool.Info(id); ok {
					rb.RetainedResources = append(rb.RetainedResources, RetainedResource{
						ID: id, Kind: latest.Kind, Shared: latest.Shared,
						Generation: latest.Generation, RefCount: latest.RefCount,
						Users: latest.Users, Reason: "destroy refused: resource still busy",
					})
				}
			}
			continue
		}
		rb.DestroyedPrivate = append(rb.DestroyedPrivate, id)
	}
}

func (p *Pipeline) retainedShared(tx *eyeTx) []RetainedResource {
	idSet := map[ResourceID]bool{}
	for id := range tx.touchedShared {
		idSet[id] = true
	}
	for _, id := range []ResourceID{ResSharedAtlas, ResSharedDepth, ResSharedMaterial, ResSharedHistory} {
		idSet[id] = true
	}
	ids := make([]ResourceID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	out := make([]RetainedResource, 0, len(ids))
	owner := OwnerEye(tx.eye)
	for _, id := range ids {
		info, ok := p.pool.Info(id)
		if !ok {
			continue
		}
		other := make([]string, 0, len(info.Users))
		for _, u := range info.Users {
			if u != owner {
				other = append(other, u)
			}
		}
		reason := "shared resource retained: not owned by the failed eye transaction"
		if info.RefCount > 0 {
			reason = "shared resource retained: other holders keep " + strconv.Itoa(info.RefCount) + " ref(s)"
		}
		out = append(out, RetainedResource{
			ID: id, Kind: info.Kind, Shared: info.Shared, Generation: info.Generation,
			RefCount: info.RefCount, Users: info.Users, OtherUsers: other, Reason: reason,
		})
	}
	return out
}

// rollback undoes exactly one failed eye attempt: release its leases, restore
// its material mutation, destroy its private attachments, and snapshot the
// shared resources that survived. The opposite eye and all other users are
// untouched.
func (p *Pipeline) rollback(tx *eyeTx) *RollbackRecord {
	rb := &RollbackRecord{Frame: tx.frame, Eye: tx.eye, Scope: ScopeForEye(tx.eye)}

	rb.ReleasedLeases = p.releaseLeases(tx)

	if len(tx.changedParams) > 0 {
		p.materials[tx.eye] = DefaultMaterialParams(ResSharedAtlas)
		rb.RestoredParams = append(rb.RestoredParams, tx.changedParams...)
		sort.Strings(rb.RestoredParams)
	}

	p.destroyPrivate(tx, rb)
	rb.RetainedResources = append(rb.RetainedResources, p.retainedShared(tx)...)
	rb.Note = "rolled back only the failed eye attempt; shared pool and other users preserved"
	return rb
}

// commit is the happy-path cleanup of an eye attempt.
func (p *Pipeline) commit(tx *eyeTx) {
	p.releaseLeases(tx)
	for _, id := range tx.created {
		_, _ = p.pool.Destroy(id)
	}
}
