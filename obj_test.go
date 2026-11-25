package main

import (
	"path/filepath"
	"testing"
)

func TestOBJWriteReadRoundTrip(t *testing.T) {
	domain := TrimDomain{
		Outer: []Vec2{{10, 10}, {90, 10}, {90, 90}, {10, 90}},
		Holes: [][]Vec2{{{40, 40}, {40, 60}, {60, 60}, {60, 40}}},
	}
	mesh, err := TriangulateDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "roundtrip.obj")
	if err := WriteTriangulatedOBJ(path, "roundtrip", mesh, DemoPatch()); err != nil {
		t.Fatal(err)
	}
	obj, err := ReadOBJ(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(obj.Vertices) != len(mesh.Vertices) {
		t.Fatalf("vertices = %d, want %d", len(obj.Vertices), len(mesh.Vertices))
	}
	if len(obj.Triangles) != len(mesh.Triangles) {
		t.Fatalf("faces = %d, want %d", len(obj.Triangles), len(mesh.Triangles))
	}
	if edges := OBJBoundaryEdges(obj); len(edges) < 8 {
		t.Fatalf("boundary edges = %d, want outer and hole loops", len(edges))
	}
	reloaded := objAsMesh2(obj)
	if bad := ValidateTriangles(reloaded, domain); len(bad) != 0 {
		t.Fatalf("reloaded OBJ has invalid triangles: %+v", bad[:min(3, len(bad))])
	}
}
