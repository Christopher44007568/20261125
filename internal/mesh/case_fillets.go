package mesh

import "math"

// AdjacentFillets creates an ogee-shaped extrusion. Two tangent arc strips
// share the same vertical tangent at z=0, making the seam intentionally hard
// for a classifier that only looks at normal continuity.
func AdjacentFillets() *Mesh {
	m := &Mesh{}
	pieces := AppendPieces(
		LinePiece(-32, 18, -8, 18, Plane),
		ArcPiece(-8, 10, 8, math.Pi/2, 0, 11, Fillet),
		ArcPiece(8, 10, 8, math.Pi, 1.5*math.Pi, 11, Fillet),
		LinePiece(8, 2, 32, 2, Plane),
		LinePiece(32, 2, 32, 0, Plane),
		LinePiece(32, 0, -32, 0, Plane),
		LinePiece(-32, 0, -32, 18, Plane),
	)
	ExtrudeProfile(m, pieces, -58, 58)
	return m
}
