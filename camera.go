package decallab

import "math"

type Eye string

const (
	EyeLeft  Eye = "left"
	EyeRight Eye = "right"
)

var Eyes = []Eye{EyeLeft, EyeRight}

type PixelCoord struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Camera struct {
	Eye       Eye     `json:"eye"`
	Position  Vec3    `json:"position"`
	Forward   Vec3    `json:"forward"`
	RightAxis Vec3    `json:"right_axis"`
	Up        Vec3    `json:"up"`
	TanHalfX  float64 `json:"tan_half_x"`
	TanHalfY  float64 `json:"tan_half_y"`
	Width     int     `json:"width_px"`
	Height    int     `json:"height_px"`
}

type Rig struct {
	IPDM    float64        `json:"ipd_m"`
	Width   int            `json:"width_px"`
	Height  int            `json:"height_px"`
	Cameras map[Eye]Camera `json:"cameras"`
	Target  Vec3           `json:"target"`
}

func NewCamera(eye Eye, position, target Vec3, fovYDeg float64, width, height int) Camera {
	forward := target.Sub(position).Normalize()
	worldUp := Vec3{Y: 1}
	right := forward.Cross(worldUp).Normalize()
	if right.Norm() < 1e-6 {
		right = Vec3{X: 1}
	}
	up := right.Cross(forward).Normalize()
	tanHalfY := math.Tan(fovYDeg * math.Pi / 360)
	aspect := float64(width) / float64(height)
	return Camera{
		Eye:       eye,
		Position:  position,
		Forward:   forward,
		RightAxis: right,
		Up:        up,
		TanHalfX:  tanHalfY * aspect,
		TanHalfY:  tanHalfY,
		Width:     width,
		Height:    height,
	}
}

// NewStereoRig places the eyes along world X. RigCenter can be shifted laterally
// to create an oblique viewing angle without rotating the physical decal scale.
func NewStereoRig(rigCenter, target Vec3, ipdM, fovYDeg float64, width, height int) Rig {
	leftPos := rigCenter.Add(Vec3{X: -ipdM / 2})
	rightPos := rigCenter.Add(Vec3{X: ipdM / 2})
	return Rig{
		IPDM:   ipdM,
		Width:  width,
		Height: height,
		Target: target,
		Cameras: map[Eye]Camera{
			EyeLeft:  NewCamera(EyeLeft, leftPos, target, fovYDeg, width, height),
			EyeRight: NewCamera(EyeRight, rightPos, target, fovYDeg, width, height),
		},
	}
}

func (c Camera) Project(point Vec3) (PixelCoord, bool) {
	relative := point.Sub(c.Position)
	depth := relative.Dot(c.Forward)
	if depth <= 0.02 {
		return PixelCoord{}, false
	}
	cameraX := relative.Dot(c.RightAxis)
	cameraY := relative.Dot(c.Up)
	ndcX := cameraX / (depth * c.TanHalfX)
	ndcY := cameraY / (depth * c.TanHalfY)
	if ndcX < -1.25 || ndcX > 1.25 || ndcY < -1.25 || ndcY > 1.25 {
		return PixelCoord{
			X: (ndcX*0.5 + 0.5) * float64(c.Width),
			Y: (0.5 - ndcY*0.5) * float64(c.Height),
		}, false
	}
	return PixelCoord{
		X: (ndcX*0.5 + 0.5) * float64(c.Width),
		Y: (0.5 - ndcY*0.5) * float64(c.Height),
	}, true
}

func (c Camera) PixelToNDC(px, py float64) (ndcX, ndcY float64) {
	return 2*px/float64(c.Width) - 1, 1 - 2*py/float64(c.Height)
}

func (c Camera) RayForPixel(px, py float64) Ray {
	ndcX, ndcY := c.PixelToNDC(px, py)
	direction := c.Forward.
		Add(c.RightAxis.Scale(ndcX * c.TanHalfX)).
		Add(c.Up.Scale(ndcY * c.TanHalfY)).
		Normalize()
	return Ray{Origin: c.Position, Direction: direction}
}

func (c Camera) ViewDirection(point Vec3) Vec3 {
	return point.Sub(c.Position).Normalize()
}

type StereoProjection struct {
	LeftPixel             PixelCoord `json:"left_pixel"`
	RightPixel            PixelCoord `json:"right_pixel"`
	LeftVisible           bool       `json:"left_visible"`
	RightVisible          bool       `json:"right_visible"`
	HorizontalDisparityPx float64    `json:"horizontal_disparity_px"`
	VerticalDifferencePx  float64    `json:"vertical_difference_px"`
	BinocularAngleDeg     float64    `json:"binocular_angle_deg"`
}

func (r Rig) ProjectStereo(point Vec3) StereoProjection {
	left := r.Cameras[EyeLeft]
	right := r.Cameras[EyeRight]
	lp, lv := left.Project(point)
	rp, rv := right.Project(point)
	leftDir := point.Sub(left.Position).Normalize()
	rightDir := point.Sub(right.Position).Normalize()
	return StereoProjection{
		LeftPixel:             lp,
		RightPixel:            rp,
		LeftVisible:           lv,
		RightVisible:          rv,
		HorizontalDisparityPx: rp.X - lp.X,
		VerticalDifferencePx:  rp.Y - lp.Y,
		BinocularAngleDeg:     angleBetweenDeg(leftDir, rightDir),
	}
}

// SurfacePixelFootprintM estimates the world distance between adjacent pixel
// rays after they intersect the same surface. The result grows at grazing angles.
func SurfacePixelFootprintM(camera Camera, pixel PixelCoord, surface Surface) (xSpacing, ySpacing float64, ok bool) {
	centerRay := camera.RayForPixel(pixel.X, pixel.Y)
	centerHit, _, found := IntersectNearest(centerRay, []Surface{surface})
	if !found {
		return 0, 0, false
	}
	_ = centerHit

	rayX0 := camera.RayForPixel(pixel.X-0.5, pixel.Y)
	rayX1 := camera.RayForPixel(pixel.X+0.5, pixel.Y)
	hitX0, okX0 := surface.Intersect(rayX0)
	hitX1, okX1 := surface.Intersect(rayX1)
	if okX0 && okX1 {
		xSpacing = hitX0.Point.Distance(hitX1.Point)
	}
	rayY0 := camera.RayForPixel(pixel.X, pixel.Y-0.5)
	rayY1 := camera.RayForPixel(pixel.X, pixel.Y+0.5)
	hitY0, okY0 := surface.Intersect(rayY0)
	hitY1, okY1 := surface.Intersect(rayY1)
	if okY0 && okY1 {
		ySpacing = hitY0.Point.Distance(hitY1.Point)
	}
	if xSpacing <= 0 || ySpacing <= 0 {
		return 0, 0, false
	}
	return xSpacing, ySpacing, true
}
