package stereoprobe

import (
	"errors"
	"math"
)

type Vec3 struct {
	X, Y, Z float64
}

type Vec4 struct {
	X, Y, Z, W float64
}

type Mat4 [16]float64

type Frustum struct {
	Left, Right, Bottom, Top, Near, Far float64
}

type Camera struct {
	Position Vec3
	Right    Vec3
	Up       Vec3
	Forward  Vec3
}

type EyeRig struct {
	Camera     Camera
	EyeOffsetX float64
}

type ScreenPoint struct {
	X, Y, Depth01 float64
	Culled        bool
}

var (
	ErrInvalidFrustum = errors.New("frustum must have positive near/far distances and ordered bounds")
	ErrInvalidCamera  = errors.New("camera basis must be orthonormal")
	ErrInvalidDepth   = errors.New("depth must be in [0, 1]")
)

func (f Frustum) Validate() error {
	if !(f.Near > 0 && f.Far > f.Near && f.Left < f.Right && f.Bottom < f.Top) {
		return ErrInvalidFrustum
	}
	return nil
}

func (c Camera) Validate() error {
	if !orthonormal(c.Right, c.Up, c.Forward) {
		return ErrInvalidCamera
	}
	return nil
}

func NewCamera(position Vec3, rotation Mat4) Camera {
	return Camera{
		Position: position,
		Right:    Vec3{rotation[0], rotation[1], rotation[2]},
		Up:       Vec3{rotation[4], rotation[5], rotation[6]},
		Forward:  Vec3{rotation[8], rotation[9], rotation[10]},
	}
}

func RotationXYZ(radians Vec3) Mat4 {
	cx, sx := math.Cos(radians.X), math.Sin(radians.X)
	cy, sy := math.Cos(radians.Y), math.Sin(radians.Y)
	cz, sz := math.Cos(radians.Z), math.Sin(radians.Z)
	return Mat4{
		cy * cz, -cy * sz, sy, 0,
		cx*sz + sx*sy*cz, cx*cz - sx*sy*sz, -sx * cy, 0,
		sx*sz - cx*sy*cz, sx*cz + cx*sy*sz, cx * cy, 0,
		0, 0, 0, 1,
	}
}

func FrustumMatrix(f Frustum) (Mat4, error) {
	if err := f.Validate(); err != nil {
		return Mat4{}, err
	}
	width := f.Right - f.Left
	height := f.Top - f.Bottom
	depth := f.Far - f.Near
	return Mat4{
		2 * f.Near / width, 0, (f.Right + f.Left) / width, 0,
		0, 2 * f.Near / height, (f.Top + f.Bottom) / height, 0,
		0, 0, -(f.Far + f.Near) / depth, -2 * f.Far * f.Near / depth,
		0, 0, -1, 0,
	}, nil
}

func EyeView(camera Camera, eyeOffsetX float64) (Mat4, Vec3, error) {
	if err := camera.Validate(); err != nil {
		return Mat4{}, Vec3{}, err
	}
	eye := add(camera.Position, scale(camera.Right, eyeOffsetX))
	view := Mat4{
		camera.Right.X, camera.Right.Y, camera.Right.Z, -dot(camera.Right, eye),
		camera.Up.X, camera.Up.Y, camera.Up.Z, -dot(camera.Up, eye),
		-camera.Forward.X, -camera.Forward.Y, -camera.Forward.Z, dot(camera.Forward, eye),
		0, 0, 0, 1,
	}
	return view, eye, nil
}

func ViewProjection(camera Camera, eyeOffsetX float64, frustum Frustum) (Mat4, Mat4, Vec3, error) {
	view, eye, err := EyeView(camera, eyeOffsetX)
	if err != nil {
		return Mat4{}, Mat4{}, Vec3{}, err
	}
	projection, err := FrustumMatrix(frustum)
	if err != nil {
		return Mat4{}, Mat4{}, Vec3{}, err
	}
	return multiply(projection, view), projection, eye, nil
}

func ProjectToScreen(camera Camera, eyeOffsetX float64, frustum Frustum, world Vec3, width, height int) (ScreenPoint, error) {
	if err := camera.Validate(); err != nil {
		return ScreenPoint{}, err
	}
	if err := frustum.Validate(); err != nil {
		return ScreenPoint{}, err
	}
	if width <= 0 || height <= 0 {
		return ScreenPoint{}, errors.New("screen dimensions must be positive")
	}
	eyePosition := add(camera.Position, scale(camera.Right, eyeOffsetX))
	relative := sub(world, eyePosition)
	x := dot(relative, camera.Right)
	y := dot(relative, camera.Up)
	eyeDepth := dot(relative, camera.Forward)
	if analyticalCullingGeometry(frustum, x, y, eyeDepth) {
		return ScreenPoint{Culled: true}, nil
	}
	ndcX := (2*frustum.Near*x/eyeDepth - frustum.Right - frustum.Left) / (frustum.Right - frustum.Left)
	ndcY := (2*frustum.Near*y/eyeDepth - frustum.Top - frustum.Bottom) / (frustum.Top - frustum.Bottom)
	ndcZ := (-(frustum.Far+frustum.Near)/(frustum.Far-frustum.Near) +
		2*frustum.Far*frustum.Near/(eyeDepth*(frustum.Far-frustum.Near)))
	return ScreenPoint{
		X:       (ndcX + 1) * 0.5 * float64(width),
		Y:       (1 - ndcY) * 0.5 * float64(height),
		Depth01: 1 - (ndcZ+1)*0.5,
	}, nil
}

func analyticalCullingGeometry(frustum Frustum, x, y, eyeDepth float64) bool {
	if eyeDepth < frustum.Near || eyeDepth > frustum.Far {
		return true
	}
	leftBound := frustum.Left * eyeDepth / frustum.Near
	rightBound := frustum.Right * eyeDepth / frustum.Near
	bottomBound := frustum.Bottom * eyeDepth / frustum.Near
	topBound := frustum.Top * eyeDepth / frustum.Near
	epsilon := 1e-12
	if x < leftBound-epsilon || x > rightBound+epsilon {
		return true
	}
	return y < bottomBound-epsilon || y > topBound+epsilon
}

func EyeFromClip(frustum Frustum, clip Vec4) (Vec3, error) {
	if err := frustum.Validate(); err != nil {
		return Vec3{}, err
	}
	width := frustum.Right - frustum.Left
	height := frustum.Top - frustum.Bottom
	depth := frustum.Far - frustum.Near
	eyeZ := -2 * frustum.Far * frustum.Near / (clip.Z*(frustum.Far+frustum.Near) + clip.W*depth)
	eyeX := eyeZ * (width*clip.X/clip.W + frustum.Right + frustum.Left) / (2 * frustum.Near)
	eyeY := eyeZ * (height*clip.Y/clip.W + frustum.Top + frustum.Bottom) / (2 * frustum.Near)
	return Vec3{X: eyeX, Y: eyeY, Z: eyeZ}, nil
}

func UnprojectEye(frustum Frustum, screenX, screenY, depth01 float64, width, height int) (Vec3, error) {
	if width <= 0 || height <= 0 {
		return Vec3{}, errors.New("screen dimensions must be positive")
	}
	if depth01 < 0 || depth01 > 1 {
		return Vec3{}, ErrInvalidDepth
	}
	ndcX := 2*screenX/float64(width) - 1
	ndcY := 1 - 2*screenY/float64(height)
	ndcZ := 2*depth01 - 1
	return EyeFromClip(frustum, Vec4{X: ndcX, Y: ndcY, Z: ndcZ, W: 1})
}

func EyeToWorld(camera Camera, eyeOffsetX float64, eyePoint Vec3) (Vec3, error) {
	if err := camera.Validate(); err != nil {
		return Vec3{}, err
	}
	eye := add(camera.Position, scale(camera.Right, eyeOffsetX))
	return add(eye, add(scale(camera.Right, eyePoint.X), add(scale(camera.Up, eyePoint.Y), scale(camera.Forward, -eyePoint.Z)))), nil
}

func multiply(a, b Mat4) Mat4 {
	var out Mat4
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			out[row*4+col] = 0
			for k := 0; k < 4; k++ {
				out[row*4+col] += a[row*4+k] * b[k*4+col]
			}
		}
	}
	return out
}

func orthonormal(right, up, forward Vec3) bool {
	basis := [3]Vec3{right, up, forward}
	for i, axis := range basis {
		if math.Abs(length(axis)-1) > 1e-10 {
			return false
		}
		for j := i + 1; j < len(basis); j++ {
			if math.Abs(dot(axis, basis[j])) > 1e-10 {
				return false
			}
		}
	}
	return true
}

func dot(a, b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func length(a Vec3) float64 { return math.Sqrt(dot(a, a)) }
func scale(a Vec3, s float64) Vec3 {
	return Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}
func add(a, b Vec3) Vec3 {
	return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func sub(a, b Vec3) Vec3 {
	return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}
