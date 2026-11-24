package normalcone

type Mat3 struct {
	M [3][3]float64
}

func Identity() Mat3 {
	return Mat3{M: [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}}
}

func Reflection(normal Vec3) Mat3 {
	unit, ok := normal.Normalize()
	if !ok {
		return Identity()
	}
	projection := Mat3{M: [3][3]float64{
		{unit.X * unit.X, unit.X * unit.Y, unit.X * unit.Z},
		{unit.Y * unit.X, unit.Y * unit.Y, unit.Y * unit.Z},
		{unit.Z * unit.X, unit.Z * unit.Y, unit.Z * unit.Z},
	}}
	return Identity().Subtract(projection.Scale(2))
}

func (m Mat3) MulVec(v Vec3) Vec3 {
	return Vec3{
		X: m.M[0][0]*v.X + m.M[0][1]*v.Y + m.M[0][2]*v.Z,
		Y: m.M[1][0]*v.X + m.M[1][1]*v.Y + m.M[1][2]*v.Z,
		Z: m.M[2][0]*v.X + m.M[2][1]*v.Y + m.M[2][2]*v.Z,
	}
}

func (m Mat3) Scale(s float64) Mat3 {
	result := Mat3{}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			result.M[row][col] = m.M[row][col] * s
		}
	}
	return result
}

func (m Mat3) Subtract(o Mat3) Mat3 {
	result := Mat3{}
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			result.M[row][col] = m.M[row][col] - o.M[row][col]
		}
	}
	return result
}

func (m Mat3) Determinant() float64 {
	return m.M[0][0]*(m.M[1][1]*m.M[2][2]-m.M[1][2]*m.M[2][1]) -
		m.M[0][1]*(m.M[1][0]*m.M[2][2]-m.M[1][2]*m.M[2][0]) +
		m.M[0][2]*(m.M[1][0]*m.M[2][1]-m.M[1][1]*m.M[2][0])
}

func (m Mat3) TransformNormal(normal Vec3) (Vec3, bool) {
	transformed, ok := m.MulVec(normal).Normalize()
	if !ok {
		return Vec3{}, false
	}
	if m.Determinant() < 0 {
		transformed = transformed.Scale(-1)
	}
	return transformed, true
}

func IsReflection(m Mat3) bool {
	return m.Determinant() < 0
}
