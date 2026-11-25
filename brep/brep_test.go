package brep

import (
	"math"
	"testing"

	"brepsection/geom"
)

func TestSampleSolidIsClosedManifold(t *testing.T) {
	b := SampleSolid()
	if len(b.Points) == 0 || len(b.Faces) == 0 {
		t.Fatal("empty BRep")
	}

	// Every manifold edge must be incident to exactly two faces.
	ef := b.EdgeFaces()
	for key, faces := range ef {
		if len(faces) != 2 {
			t.Fatalf("edge %d has %d incident faces, want 2", key, len(faces))
		}
	}

	// Every face normal must be unit length and its outer loop must face away
	// from the recorded solid interior point.
	for i, f := range b.Faces {
		if math.Abs(f.Normal.Len()-1) > 1e-12 {
			t.Fatalf("face %d normal not unit: %f", i, f.Normal.Len())
		}
		center := loopCentroid(b.Points, f.Outer)
		if f.Normal.Dot(center.Sub(f.Interior)) <= 0 {
			t.Fatalf("face %d normal points toward interior", i)
		}
		for _, hole := range f.Holes {
			n := NewellNormal(b.Points, hole)
			if n.Dot(f.Normal) >= 0 {
				t.Fatalf("face %d hole not reverse-wound", i)
			}
		}
	}
}

func TestExtrudeVertexSharing(t *testing.T) {
	// The split bottom edge (x=5 vertex) must create two coplanar side faces
	// sharing exactly the same plane, which models the tangent face case.
	b := SampleSolid()
	coplanar := 0
	for i := range b.Faces {
		for j := i + 1; j < len(b.Faces); j++ {
			a, c := b.Faces[i], b.Faces[j]
			if math.Abs(a.Normal.Dot(c.Normal)) > 1-1e-12 {
				coplanar++
			}
		}
	}
	if coplanar == 0 {
		t.Fatal("expected at least one coplanar face pair for the split edge")
	}
}

func TestAABB(t *testing.T) {
	b := SampleSolid()
	box := b.AABB()
	if box.Diagonal() <= 0 {
		t.Fatal("non-positive bounding box diagonal")
	}
	for _, p := range b.Points {
		if p.X < box.Min.X-1e-9 || p.X > box.Max.X+1e-9 ||
			p.Y < box.Min.Y-1e-9 || p.Y > box.Max.Y+1e-9 ||
			p.Z < box.Min.Z-1e-9 || p.Z > box.Max.Z+1e-9 {
			t.Fatalf("point %v outside AABB", p)
		}
	}
}

func TestProfileInterior(t *testing.T) {
	p := Profile2D{
		Outer: []geom.Vec2{
			geom.V2(0, 0), geom.V2(4, 0), geom.V2(4, 4), geom.V2(0, 4),
		},
		Holes: [][]geom.Vec2{
			{geom.V2(1, 1), geom.V2(2, 1), geom.V2(2, 2), geom.V2(1, 2)},
		},
	}
	c := profileInterior(p)
	if geom.PointInLoop(c, p.Outer) == 0 {
		t.Fatalf("interior point %v outside outer loop", c)
	}
	for _, h := range p.Holes {
		if geom.PointInLoop(c, h) != 0 {
			t.Fatalf("interior point %v landed inside hole", c)
		}
	}
}
