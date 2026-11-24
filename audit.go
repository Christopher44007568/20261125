package texturelife

import (
	"fmt"
	"sort"
)

// IssueCode classifies lifecycle invariant failures.
type IssueCode string

const (
	IssueLeak                  IssueCode = "leak"
	IssueDanglingReference     IssueCode = "dangling-reference"
	IssueReferenceMismatch     IssueCode = "reference-count-mismatch"
	IssueReleasedWithRefs      IssueCode = "released-with-references"
	IssueInvalidRetirement     IssueCode = "invalid-retirement"
	IssueUnreleasedAfterRetire IssueCode = "unreleased-after-retire"
)

// Issue is one audit finding. TextureID and Frame identify the affected
// resource and, where applicable, the frame that exposed the problem.
type Issue struct {
	Code      IssueCode
	TextureID TextureID
	Frame     Frame
	Detail    string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s: texture=%d frame=%d: %s", i.Code, i.TextureID, i.Frame, i.Detail)
}

// Reference is one frame-to-texture edge in the simulated reference graph.
type Reference struct {
	Frame     Frame
	TextureID TextureID
	Count     int
}

// DanglingRef describes a frame edge whose texture was released or vanished.
type DanglingRef struct {
	Frame     Frame
	TextureID TextureID
	Count     int
	Detail    string
}

// Peak stores the maximum observed live resource count, bytes, and reference
// count. Count/Bytes and RefCount have separate timestamps because they may
// peak at different points in a frame transition.
type Peak struct {
	Count      int
	Bytes      int64
	CountFrame Frame
	BytesFrame Frame
	RefCount   int
	RefFrame   Frame
}

// ScopePeak stores the maximum live values for one eye or texture kind.
type ScopePeak struct {
	Count    int
	Bytes    int64
	RefCount int
	Frame    Frame
}

// Report is a point-in-time audit. Textures includes released resources so the
// complete lifetime metadata remains inspectable. Leaks contains every live
// resource for FinalAudit, and only unreferenced, non-retiring allocations for
// Snapshot.
type Report struct {
	Textures       []Texture
	Live           []Texture
	Leaks          []Texture
	References     []Reference
	Dangling       []DanglingRef
	Issues         []Issue
	LiveCount      int
	LiveBytes      int64
	LiveReferences int
	RetiringCount  int
	CreatedCount   int
	Peak           Peak
	PeakByEye      map[Eye]ScopePeak
	PeakByKind     map[Kind]ScopePeak
}

// Healthy reports whether the audit found no lifecycle violations. Leaks are
// always violations. A Snapshot of an active device can legitimately contain
// live textures, but a FinalAudit should be healthy only after shutdown.
func (r Report) Healthy() bool {
	return len(r.Leaks) == 0 && len(r.Dangling) == 0 && len(r.Issues) == 0
}

// Snapshot audits the current reference graph. It reports orphaned
// allocations as leaks but does not treat active, referenced resources as
// leaks.
func (tr *Tracker) Snapshot() Report {
	return tr.audit(false)
}

// FinalAudit audits a completed simulation and treats every still-live
// resource as a leak.
func (tr *Tracker) FinalAudit() Report {
	return tr.audit(true)
}

func (tr *Tracker) audit(final bool) Report {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	report := Report{
		LiveCount:      tr.liveCount,
		LiveBytes:      tr.liveBytes,
		LiveReferences: tr.liveRefs,
		CreatedCount:   len(tr.order),
		Peak: Peak{
			Count:      tr.peakCount,
			Bytes:      tr.peakBytes,
			CountFrame: tr.peakCountFrame,
			BytesFrame: tr.peakBytesFrame,
			RefCount:   tr.peakRefs,
			RefFrame:   tr.peakRefsFrame,
		},
		PeakByEye:  make(map[Eye]ScopePeak, len(tr.peakByEye)),
		PeakByKind: make(map[Kind]ScopePeak, len(tr.peakByKind)),
	}
	for eye, peak := range tr.peakByEye {
		report.PeakByEye[eye] = ScopePeak{
			Count: peak.count, Bytes: peak.bytes, RefCount: peak.refCount, Frame: peak.frame,
		}
	}
	for kind, peak := range tr.peakByKind {
		report.PeakByKind[kind] = ScopePeak{
			Count: peak.count, Bytes: peak.bytes, RefCount: peak.refCount, Frame: peak.frame,
		}
	}

	expectedRefs := make(map[TextureID]int)
	for frame, frameRefs := range tr.refs {
		for id, count := range frameRefs {
			report.References = append(report.References, Reference{
				Frame: frame, TextureID: id, Count: count,
			})
			expectedRefs[id] += count
		}
	}
	sort.Slice(report.References, func(i, j int) bool {
		if report.References[i].Frame != report.References[j].Frame {
			return report.References[i].Frame < report.References[j].Frame
		}
		return report.References[i].TextureID < report.References[j].TextureID
	})

	for _, id := range tr.order {
		rec := tr.records[id]
		report.Textures = append(report.Textures, rec.Texture)
		if !rec.Alive() {
			if expectedRefs[id] > 0 || rec.RefCount > 0 {
				detail := fmt.Sprintf("released at frame %d with %d live references", rec.ReleasedAt, expectedRefs[id])
				report.Issues = append(report.Issues, Issue{
					Code: IssueReleasedWithRefs, TextureID: id, Frame: rec.ReleasedAt, Detail: detail,
				})
			}
			continue
		}

		report.Live = append(report.Live, rec.Texture)
		if rec.Retiring() {
			report.RetiringCount++
			if rec.RefCount == 0 {
				report.Issues = append(report.Issues, Issue{
					Code: IssueUnreleasedAfterRetire, TextureID: id, Frame: rec.RetireFrame,
					Detail: "retirement was requested with no references, but the texture is still live",
				})
			}
			if rec.RetireFrame < rec.CreatedFrame {
				report.Issues = append(report.Issues, Issue{
					Code: IssueInvalidRetirement, TextureID: id, Frame: rec.RetireFrame,
					Detail: fmt.Sprintf("retirement frame precedes creation frame %d", rec.CreatedFrame),
				})
			}
		}
		if expectedRefs[id] != rec.RefCount {
			report.Issues = append(report.Issues, Issue{
				Code: IssueReferenceMismatch, TextureID: id, Frame: rec.LastUsedFrame,
				Detail: fmt.Sprintf("frame edges hold %d references, metadata says %d", expectedRefs[id], rec.RefCount),
			})
		}
		if final || (rec.RefCount == 0 && !rec.Retiring()) {
			report.Leaks = append(report.Leaks, rec.Texture)
		}
	}

	for frame, frameRefs := range tr.refs {
		for id, count := range frameRefs {
			rec, ok := tr.records[id]
			if !ok || !rec.Alive() {
				detail := "texture metadata is missing"
				if ok {
					detail = fmt.Sprintf("texture was released at frame %d", rec.ReleasedAt)
				}
				report.Dangling = append(report.Dangling, DanglingRef{
					Frame: frame, TextureID: id, Count: count, Detail: detail,
				})
			}
		}
	}
	sort.Slice(report.Dangling, func(i, j int) bool {
		if report.Dangling[i].Frame != report.Dangling[j].Frame {
			return report.Dangling[i].Frame < report.Dangling[j].Frame
		}
		return report.Dangling[i].TextureID < report.Dangling[j].TextureID
	})
	report.Dangling = uniqueDangling(report.Dangling)
	for _, dangling := range report.Dangling {
		report.Issues = append(report.Issues, Issue{
			Code: IssueDanglingReference, TextureID: dangling.TextureID, Frame: dangling.Frame,
			Detail: dangling.Detail,
		})
	}
	for _, leak := range report.Leaks {
		report.Issues = append(report.Issues, Issue{
			Code: IssueLeak, TextureID: leak.ID, Frame: leak.LastUsedFrame,
			Detail: fmt.Sprintf("%s %s %s texture is still live (%d bytes)", leak.Eye, leak.Kind, leak.Size, leak.Bytes),
		})
	}
	sort.Slice(report.Leaks, func(i, j int) bool { return report.Leaks[i].ID < report.Leaks[j].ID })
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Code != report.Issues[j].Code {
			return report.Issues[i].Code < report.Issues[j].Code
		}
		if report.Issues[i].TextureID != report.Issues[j].TextureID {
			return report.Issues[i].TextureID < report.Issues[j].TextureID
		}
		return report.Issues[i].Frame < report.Issues[j].Frame
	})
	return report
}

func uniqueDangling(refs []DanglingRef) []DanglingRef {
	if len(refs) == 0 {
		return nil
	}
	out := refs[:1]
	for _, ref := range refs[1:] {
		last := &out[len(out)-1]
		if last.Frame == ref.Frame && last.TextureID == ref.TextureID {
			last.Count += ref.Count
			continue
		}
		out = append(out, ref)
	}
	return out
}
