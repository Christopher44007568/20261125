package main

import "fmt"

// Eye bundles one eye's rig placement with its derived transforms.
type Eye struct {
	Name  string
	X     float64 // world X offset in meters (rotation-free rig)
	Frust Frustum
	Proj  Mat4 // camera -> clip
	View  Mat4 // world -> camera (pure translation)
	InvPV Mat4 // clip -> world
}

func NewEye(name string, x float64, in Intrinsics, conv float64) (Eye, error) {
	fr := EyeFrustum(in, x, conv)
	proj := PerspectiveOffCenter(fr)
	view := Translation(-x, 0, 0)
	invPV, ok := Invert4(Mul4(proj, view))
	if !ok {
		return Eye{}, fmt.Errorf("eye %s: singular projection", name)
	}
	return Eye{Name: name, X: x, Frust: fr, Proj: proj, View: view, InvPV: invPV}, nil
}

// unproject maps an NDC point back to world space via the inverted matrix.
func (e Eye) unproject(nx, ny, nz float64) Vec3 {
	p := e.InvPV.MulV(Vec4{nx, ny, nz, 1})
	return Vec3{p.X / p.W, p.Y / p.W, p.Z / p.W}
}

// project maps a world point to NDC via the forward matrix pipeline.
func (e Eye) project(w Vec3) Vec3 {
	c := Mul4(e.Proj, e.View).MulV(Vec4{w.X, w.Y, w.Z, 1})
	return Vec3{c.X / c.W, c.Y / c.W, c.Z / c.W}
}

// refRay is the INDEPENDENT geometric reference: it never touches any
// matrix. The near-plane corner for NDC (nx,ny) is interpolated from the
// frustum bounds in camera space, then the ray is built in world space
// with plain vector arithmetic from the eye position.
func (e Eye) refRay(nx, ny float64) (origin, dir Vec3) {
	f := e.Frust
	x := f.L + (nx+1)/2*(f.R-f.L)
	y := f.B + (ny+1)/2*(f.T-f.B)
	origin = Vec3{e.X, 0, 0}
	dir = Vec3{x, y, -f.N}.Norm() // camera axes == world axes (parallel rig)
	return origin, dir
}

// rayAt intersects a ray with the plane z = planeZ (planeZ negative).
func rayAt(origin, dir Vec3, planeZ float64) Vec3 {
	t := (planeZ - origin.Z) / dir.Z
	return origin.Add(dir.Scale(t))
}

// CheckResult aggregates the worst deviations for one eye.
type CheckResult struct {
	MaxAngleDeg float64 // matrix ray vs geometric ray
	MaxNearErr  float64 // meters, near-plane intersection
	MaxFarErr   float64 // meters, far-plane intersection
	MaxEdgeErr  float64 // meters, virtual-screen (convergence plane) hit
	WorstSample string
}

// CheckEye compares matrix-based unprojection against the geometric
// reference at the 4 screen corners, 4 edge midpoints and the center.
func (e Eye) CheckEye(conv float64) CheckResult {
	samples := []struct{ nx, ny float64 }{
		{-1, -1}, {1, -1}, {1, 1}, {-1, 1}, // corners
		{0, -1}, {1, 0}, {0, 1}, {-1, 0}, // edge midpoints
		{0, 0}, // center
	}
	var res CheckResult
	fail := func(format string, a ...interface{}) {
		if res.WorstSample == "" {
			res.WorstSample = fmt.Sprintf(format, a...)
		}
	}
	for _, s := range samples {
		origin, dir := e.refRay(s.nx, s.ny)

		// Matrix path: unproject near (ndc z=-1) and far (ndc z=+1).
		pNear := e.unproject(s.nx, s.ny, -1)
		pFar := e.unproject(s.nx, s.ny, 1)
		mDir := pFar.Sub(pNear).Norm()

		if ang := AngleDeg(mDir, dir); ang > res.MaxAngleDeg {
			res.MaxAngleDeg = ang
			if ang > 1e-6 {
				fail("eye %s ndc(%+.0f,%+.0f): angle %.3e deg", e.Name, s.nx, s.ny, ang)
			}
		}
		if d := pNear.Sub(rayAt(origin, dir, -e.Frust.N)).Len(); d > res.MaxNearErr {
			res.MaxNearErr = d
		}
		if d := pFar.Sub(rayAt(origin, dir, -e.Frust.F)).Len(); d > res.MaxFarErr {
			res.MaxFarErr = d
		}
		// Screen edge: march the matrix ray to the convergence plane and
		// compare with the reference ray's hit on the same plane.
		mEdge := rayAt(pNear, mDir, -conv)
		rEdge := rayAt(origin, dir, -conv)
		if d := mEdge.Sub(rEdge).Len(); d > res.MaxEdgeErr {
			res.MaxEdgeErr = d
		}
	}
	return res
}
