package sim

import (
	"math"
	"reflect"
	"testing"
)

func TestTimelineAuditAndSparseHistory(t *testing.T) {
	timeline := BuildTimeline(4)
	expectedFrames := timeline.FrameCount * len(timeline.Eyes)
	expectedTracks := expectedFrames * 3

	if got := timeline.Audit.Frames; got != expectedFrames {
		t.Fatalf("audited frames = %d, want %d", got, expectedFrames)
	}
	if got := timeline.Audit.Tracks; got != expectedTracks {
		t.Fatalf("audited tracks = %d, want %d", got, expectedTracks)
	}
	if timeline.Audit.CrossEyeSamples != 0 {
		t.Fatalf("cross-eye samples = %d, want 0", timeline.Audit.CrossEyeSamples)
	}
	if timeline.Audit.EnergyCapViolations != 0 {
		t.Fatalf("energy cap violations = %d, want 0", timeline.Audit.EnergyCapViolations)
	}
	if timeline.Audit.FullFrameTimewarps != 0 {
		t.Fatalf("full frame timewarps = %d, want 0", timeline.Audit.FullFrameTimewarps)
	}
	if timeline.Audit.MaxCompositeEnergy > 1+1e-9 {
		t.Fatalf("composite energy = %.6f, want <= 1", timeline.Audit.MaxCompositeEnergy)
	}
	if timeline.Audit.MeanResampleWeight < 0.9 {
		t.Fatalf("mean resample weight = %.6f, want >= 0.9", timeline.Audit.MeanResampleWeight)
	}

	for _, eye := range timeline.Eyes {
		for frameIndex, frame := range eye.Frames {
			for _, track := range frame.Tracks {
				if !track.History.SameEye || track.History.SourceEye != eye.ID {
					t.Fatalf("frame %d eye %d track %s leaked stereo history",
						frameIndex, eye.ID, track.ObjectID)
				}
				if frameIndex == 0 {
					if track.History.Valid {
						t.Fatalf("frame 0 unexpectedly has history for %s", track.ObjectID)
					}
					continue
				}
				prior := trackForTest(t, eye.Frames[frameIndex-1].Tracks, track.ObjectID)
				if !track.History.Valid {
					t.Fatalf("frame %d missing history for %s", frameIndex, track.ObjectID)
				}
				if track.History.Center != prior.Center {
					t.Fatalf("frame %d history center = %+v, want prior center %+v",
						frameIndex, track.History.Center, prior.Center)
				}
				if track.History.Stretch != prior.Stretch {
					t.Fatalf("frame %d history stretch = %.4f, want %.4f",
						frameIndex, track.History.Stretch, prior.Stretch)
				}
			}
		}
	}
}

func TestCompositeEnergyIsConvexAndBounded(t *testing.T) {
	timeline := BuildTimeline(5)
	for _, eye := range timeline.Eyes {
		for _, frame := range eye.Frames {
			for _, track := range frame.Tracks {
				history := track.History
				if !history.Valid {
					continue
				}
				limit := math.Max(track.Energy, history.SampledEnergy)
				if history.CompositeEnergy > limit+1e-9 {
					t.Fatalf("%s energy compounded: current %.6f sampled %.6f composite %.6f",
						track.ObjectID, track.Energy, history.SampledEnergy, history.CompositeEnergy)
				}
				if history.CompositePeak > math.Max(track.Peak, history.SampledPeak)+1e-9 {
					t.Fatalf("%s peak compounded past both inputs", track.ObjectID)
				}
				if history.CompositeEnergy > 1 || history.CompositePeak > 1 {
					t.Fatalf("%s exceeded normalized radiance", track.ObjectID)
				}
			}
		}
	}
}

func TestThreeMotionSignatures(t *testing.T) {
	timeline := BuildTimeline(6)
	left := timeline.Eyes[0].Frames
	metal := motionRange(left, "rough-metal")
	sphere := motionRange(left, "mirror-sphere")
	glass := motionRange(left, "glass")

	if metal.trackX < 25 || metal.trackY < 15 {
		t.Fatalf("metal highlight should orbit under rotation, got x=%.2f y=%.2f",
			metal.trackX, metal.trackY)
	}
	if sphere.objectX < 125 || sphere.trackX < 100 {
		t.Fatalf("sphere should translate with a moving reflection, got object=%.2f track=%.2f",
			sphere.objectX, sphere.trackX)
	}
	if glass.objectX < 60 || glass.objectY < 20 || glass.trackX < 60 {
		t.Fatalf("glass should move as an object, got object=(%.2f,%.2f) track=%.2f",
			glass.objectX, glass.objectY, glass.trackX)
	}
	if timeline.Audit.EyeDirectionDeltaMax < 0.1 {
		t.Fatalf("stereo reflection delta = %.4f rad, want at least 0.1", timeline.Audit.EyeDirectionDeltaMax)
	}
}

func TestStereoHistoriesRemainDistinct(t *testing.T) {
	timeline := BuildTimeline(5)
	left := timeline.Eyes[0].Frames
	right := timeline.Eyes[1].Frames
	distinct := 0

	for frameIndex := 1; frameIndex < len(left); frameIndex++ {
		for _, objectID := range []string{"rough-metal", "mirror-sphere", "glass"} {
			leftTrack := trackForTest(t, left[frameIndex].Tracks, objectID)
			rightTrack := trackForTest(t, right[frameIndex].Tracks, objectID)
			if leftTrack.History.Center != rightTrack.History.Center ||
				leftTrack.History.SourceEye != rightTrack.History.SourceEye {
				distinct++
			}
		}
	}
	if distinct < 50 {
		t.Fatalf("only %d stereo frames remained distinct", distinct)
	}
}

func TestTimelineDeterministic(t *testing.T) {
	first := BuildTimeline(2)
	second := BuildTimeline(2)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("timeline is not deterministic")
	}
}

type motionSummary struct {
	objectX float64
	objectY float64
	trackX  float64
	trackY  float64
}

func motionRange(frames []Frame, objectID string) motionSummary {
	firstObject := objectForTest(frames[0].Objects, objectID)
	firstTrack := trackByID(frames[0].Tracks, objectID)
	minObjectX, maxObjectX := firstObject.Center.X, firstObject.Center.X
	minObjectY, maxObjectY := firstObject.Center.Y, firstObject.Center.Y
	minTrackX, maxTrackX := firstTrack.Center.X, firstTrack.Center.X
	minTrackY, maxTrackY := firstTrack.Center.Y, firstTrack.Center.Y

	for _, frame := range frames[1:] {
		object := objectForTest(frame.Objects, objectID)
		track := trackByID(frame.Tracks, objectID)
		minObjectX = math.Min(minObjectX, object.Center.X)
		maxObjectX = math.Max(maxObjectX, object.Center.X)
		minObjectY = math.Min(minObjectY, object.Center.Y)
		maxObjectY = math.Max(maxObjectY, object.Center.Y)
		minTrackX = math.Min(minTrackX, track.Center.X)
		maxTrackX = math.Max(maxTrackX, track.Center.X)
		minTrackY = math.Min(minTrackY, track.Center.Y)
		maxTrackY = math.Max(maxTrackY, track.Center.Y)
	}
	return motionSummary{
		objectX: maxObjectX - minObjectX,
		objectY: maxObjectY - minObjectY,
		trackX:  maxTrackX - minTrackX,
		trackY:  maxTrackY - minTrackY,
	}
}

func trackByID(tracks []HighlightTrack, objectID string) HighlightTrack {
	for _, track := range tracks {
		if track.ObjectID == objectID {
			return track
		}
	}
	panic("track " + objectID + " not found")
}

func trackForTest(t *testing.T, tracks []HighlightTrack, objectID string) HighlightTrack {
	t.Helper()
	for _, track := range tracks {
		if track.ObjectID == objectID {
			return track
		}
	}
	t.Fatalf("track %s not found", objectID)
	return HighlightTrack{}
}

func objectForTest(objects []ObjectState, objectID string) ObjectState {
	for _, object := range objects {
		if object.ID == objectID {
			return object
		}
	}
	panic("object " + objectID + " not found")
}
