package analysis

import (
	"testing"

	"meshscope/internal/mesh"
)

func testCases() map[string]*mesh.Mesh {
	return map[string]*mesh.Mesh{
		"fillets": mesh.AdjacentFillets(),
		"hole":    mesh.ObliqueHole(),
		"planar":  mesh.NearPlanarPanel(),
	}
}

func TestClassifyGeneratedCases(t *testing.T) {
	for name, m := range testCases() {
		t.Run(name, func(t *testing.T) {
			res := Classify(m)
			correct, total := 0, 0
			got := map[mesh.Feature]int{}
			for i, tri := range m.Tris {
				got[res.Faces[i].Kind]++
				if tri.Hint == res.Faces[i].Kind {
					correct++
				}
				total++
			}
			accuracy := float64(correct) / float64(total)
			t.Logf("accuracy=%.2f%% regions=%d boundaries=%d confidence=%.2f",
				accuracy*100, len(res.Regions), len(res.Boundaries), res.MeanConf)
			if accuracy < 0.97 {
				t.Fatalf("classification accuracy %.2f%% is below 97%%", accuracy*100)
			}
			switch name {
			case "fillets":
				if got[mesh.Fillet] == 0 {
					t.Fatal("adjacent fillet sample produced no fillet region")
				}
			case "hole":
				if got[mesh.Hole] == 0 || got[mesh.Cylinder] == 0 {
					t.Fatal("oblique hole sample must contain hole and reference cylinder")
				}
			case "planar":
				if got[mesh.Plane] != total {
					t.Fatal("near-planar sample should classify as planes")
				}
				if res.MaxAbsDev <= 0.02 {
					t.Fatal("near-planar form error was not retained in audit data")
				}
			}
			if len(res.Boundaries) == 0 {
				t.Fatal("expected confidence boundaries")
			}
		})
	}
}

func TestAdjacentFilletsStaySeparate(t *testing.T) {
	m := mesh.AdjacentFillets()
	res := Classify(m)
	filletRegions := 0
	for _, r := range res.Regions {
		if r.Kind == mesh.Fillet {
			filletRegions++
		}
	}
	if filletRegions < 2 {
		t.Fatalf("tangent fillet pair merged into %d region(s)", filletRegions)
	}
	for i, tri := range m.Tris {
		if tri.Hint == mesh.Fillet && res.Faces[i].Kind != mesh.Fillet {
			t.Fatalf("fillet facet %d classified as %s", i, res.Faces[i].Kind)
		}
	}
}

func TestBoundaryConfidenceIsReported(t *testing.T) {
	m := mesh.ObliqueHole()
	res := Classify(m)
	if len(res.Callouts) == 0 {
		t.Fatal("no boundary confidence callouts")
	}
	foundHoleRim := false
	for _, c := range res.Callouts {
		if c.KindA == mesh.Hole || c.KindB == mesh.Hole {
			foundHoleRim = true
			if c.Confidence < 0.8 {
				t.Fatalf("sharp elliptical rim confidence unexpectedly low: %.2f", c.Confidence)
			}
		}
	}
	if !foundHoleRim {
		t.Fatal("hole/plane boundary confidence is missing")
	}
	if res.DatumValid == false {
		t.Fatal("audit datum plane is missing")
	}
}
