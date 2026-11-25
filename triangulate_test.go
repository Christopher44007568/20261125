package main

import "testing"

func TestSquareWithHoleEarClipping(t *testing.T) {
	domain := TrimDomain{
		Outer: []Vec2{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
		Holes: [][]Vec2{{{4, 4}, {4, 6}, {6, 6}, {6, 4}}},
	}
	mesh, err := TriangulateDomain(domain)
	if err != nil {
		t.Fatalf("triangulate hole domain: %v", err)
	}
	if got, want := len(mesh.Triangles), 9; got != want {
		t.Fatalf("triangle count = %d, want %d (one bridge Steiner point)", got, want)
	}
	if bad := ValidateTriangles(mesh, domain); len(bad) != 0 {
		t.Fatalf("expected no out-of-bounds triangles, got %+v", bad)
	}
}

func TestDemoScenesProduceValidMeshes(t *testing.T) {
	for _, scene := range DemoScenes() {
		t.Run(scene.ID, func(t *testing.T) {
			loops, err := ExtractLoops(scene.Curves, attachEps)
			if err != nil {
				t.Fatalf("extract loops: %v", err)
			}
			if len(loops) < 2 {
				t.Fatalf("extracted %d loops, want at least outer and trim loop", len(loops))
			}
			domain, err := BuildTrimDomain(loops)
			if err != nil {
				t.Fatalf("classify domain: %v", err)
			}
			mesh, err := TriangulateDomain(domain)
			if err != nil {
				t.Fatalf("ear clipping: %v", err)
			}
			if len(mesh.Triangles) == 0 {
				t.Fatal("no triangles produced")
			}
			seen := make(map[[3]int]bool)
			for _, tri := range mesh.Triangles {
				ids := [3]int{tri[0], tri[1], tri[2]}
				for i := 0; i < 3; i++ {
					for j := i + 1; j < 3; j++ {
						if ids[j] < ids[i] {
							ids[i], ids[j] = ids[j], ids[i]
						}
					}
				}
				if seen[ids] {
					t.Fatalf("duplicate triangle %v", tri)
				}
				seen[ids] = true
				a, b, c := trianglePoints(mesh.Vertices, tri)
				if orient2(a, b, c) <= areaEps {
					t.Fatalf("degenerate or reversed triangle %v: %v %v %v", tri, a, b, c)
				}
			}
			if bad := ValidateTriangles(mesh, domain); len(bad) != 0 {
				for _, item := range bad {
					t.Errorf("invalid tri %d %v at %v: %v", item.Index, item.Triangle, item.Centroid, item.Reasons)
				}
			}

			naive, err := TriangulateSimple(domain.Outer)
			if err != nil {
				t.Fatalf("naive outer triangulation: %v", err)
			}
			if bad := ValidateTriangles(naive, domain); len(bad) == 0 {
				t.Fatal("outer-only mesh unexpectedly has no hole violations")
			}

			if scene.ID == "self_intersecting" && domain.CrossingCount != 1 {
				t.Fatalf("crossing count = %d, want 1", domain.CrossingCount)
			}
		})
	}
}

func TestNURBSCircleEndpointsAndRadius(t *testing.T) {
	curve := QuarterArcNURBS("q", v2(10, 20), 5, 0)
	start := curve.Eval(0)
	end := curve.Eval(1)
	if dist(start, v2(15, 20)) > 1e-8 {
		t.Fatalf("bad start %v", start)
	}
	if dist(end, v2(10, 25)) > 1e-8 {
		t.Fatalf("bad end %v", end)
	}
	mid := curve.Eval(0.5)
	if got := dist(mid, v2(10, 20)); got < 4.9 || got > 5.1 {
		t.Fatalf("mid radius = %.6f, want about 5; point %v", got, mid)
	}
}
