package xrpath

import (
	"fmt"
	"sort"
)

type WorldObject struct {
	ID              string
	MeshResource    string
	TextureResource string
	Min             Vec3
	Max             Vec3
	Color           [3]byte
	Checkered       bool
}

func (o WorldObject) Center() Vec3 {
	return V((o.Min.X+o.Max.X)/2, (o.Min.Y+o.Max.Y)/2, (o.Min.Z+o.Max.Z)/2)
}

func (o WorldObject) HalfExtents() Vec3 {
	return V((o.Max.X-o.Min.X)/2, (o.Max.Y-o.Min.Y)/2, (o.Max.Z-o.Min.Z)/2)
}

type ResourceSpec struct {
	ID    string
	Kind  string
	Bytes int64
}

type World struct {
	Objects   []WorldObject
	Resources []ResourceSpec
}

func DefaultWorld() World {
	objects := []WorldObject{
		{
			ID: "floor-grid", MeshResource: "mesh-floor", TextureResource: "tex-grid",
			Min: V(-8, -0.20, -14), Max: V(8, 0, 10), Color: [3]byte{72, 78, 90}, Checkered: true,
		},
		{
			ID: "crate-blue", MeshResource: "mesh-crate-blue", TextureResource: "tex-crate-blue",
			Min: V(-1.75, 0, -2.65), Max: V(-0.85, 0.95, -1.75), Color: [3]byte{64, 132, 224},
		},
		{
			ID: "pillar-red", MeshResource: "mesh-pillar-red", TextureResource: "tex-pillar-red",
			Min: V(1.62, 0, -3.55), Max: V(2.38, 2.25, -2.85), Color: [3]byte{214, 82, 72},
		},
		{
			ID: "beacon-green", MeshResource: "mesh-beacon-green", TextureResource: "tex-beacon-green",
			Min: V(-2.95, 0, -6.55), Max: V(-2.25, 1.45, -5.85), Color: [3]byte{72, 210, 132}, Checkered: true,
		},
		{
			ID: "occluder-wall", MeshResource: "mesh-wall", TextureResource: "tex-wall",
			Min: V(-0.10, 0, -4.38), Max: V(1.20, 2.05, -4.02), Color: [3]byte{176, 126, 76},
		},
		{
			ID: "reveal-relic", MeshResource: "mesh-relic", TextureResource: "tex-relic",
			Min: V(0.22, 0.30, -6.72), Max: V(0.88, 1.22, -6.08), Color: [3]byte{214, 190, 64}, Checkered: true,
		},
		{
			ID: "reloc-anchor", MeshResource: "mesh-anchor", TextureResource: "tex-anchor",
			Min: V(-3.55, 0, -4.95), Max: V(-2.85, 2.45, -4.25), Color: [3]byte{178, 96, 232},
		},
		{
			ID: "back-wall", MeshResource: "mesh-back-wall", TextureResource: "tex-back-wall",
			Min: V(-6, 0, -9.2), Max: V(6, 3.2, -8.9), Color: [3]byte{58, 64, 78},
		},
	}

	specs := []ResourceSpec{
		{ID: "kf-atrium-07", Kind: "relocalization_keyframe", Bytes: 184320},
		{ID: "warp-mesh-auto", Kind: "warp_mesh", Bytes: 12288},
		{ID: "warp-mesh-depth-motion", Kind: "warp_mesh", Bytes: 24576},
	}
	for _, obj := range objects {
		specs = append(specs,
			ResourceSpec{ID: obj.MeshResource, Kind: "mesh", Bytes: 4096 + int64(len(obj.ID)*137)},
			ResourceSpec{ID: obj.TextureResource, Kind: "texture", Bytes: 16384 + int64(len(obj.ID)*911)},
		)
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID < objects[j].ID })
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return World{Objects: objects, Resources: specs}
}

func (w World) ObjectByID(id string) (WorldObject, bool) {
	for _, obj := range w.Objects {
		if obj.ID == id {
			return obj, true
		}
	}
	return WorldObject{}, false
}

func rayAABB(origin, dir, min, max Vec3) (float64, Vec3, bool) {
	tNear := 0.0
	tFar := 1e9
	normal := V(0, 0, 0)

	axes := []struct {
		o, d, lo, hi float64
		n            Vec3
	}{
		{origin.X, dir.X, min.X, max.X, V(1, 0, 0)},
		{origin.Y, dir.Y, min.Y, max.Y, V(0, 1, 0)},
		{origin.Z, dir.Z, min.Z, max.Z, V(0, 0, 1)},
	}

	for _, ax := range axes {
		if absFloat(ax.d) < 1e-12 {
			if ax.o < ax.lo || ax.o > ax.hi {
				return 0, Vec3{}, false
			}
			continue
		}
		t0 := (ax.lo - ax.o) / ax.d
		t1 := (ax.hi - ax.o) / ax.d
		entryNormal := ax.n.Scale(-1)
		if t0 > t1 {
			t0, t1 = t1, t0
			entryNormal = ax.n
		}
		if t0 > tNear {
			tNear = t0
			normal = entryNormal
		}
		if t1 < tFar {
			tFar = t1
		}
		if tNear > tFar {
			return 0, Vec3{}, false
		}
	}
	if tFar < 0 {
		return 0, Vec3{}, false
	}
	if tNear < 0 {
		tNear = 0
	}
	if normal.Length() == 0 {
		normal = V(0, 1, 0)
	}
	return tNear, normal, true
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func resourceContentHash(spec ResourceSpec, revision int) string {
	return canonicalSHA256(struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Bytes    int64  `json:"bytes"`
		Revision int    `json:"revision"`
	}{spec.ID, spec.Kind, spec.Bytes, revision})
}

func formatFrameName(prefix string, frame int) string {
	return fmt.Sprintf("%s-%04d.png", prefix, frame)
}
