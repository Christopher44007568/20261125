package main

const (
	tierNear uint8 = iota + 1
	tierMid
	tierFar
	tierTransition
)

type GridSet struct {
	NearU int
	NearV int
	MidU  int
	MidV  int
	FarU  int
	FarV  int
}

type LevelConfig struct {
	ID     string
	Label  string
	File   string
	Tier   uint8
	Nu     int
	Nv     int
	Budget float64
}

type Camera struct {
	Position Vec3
	Target   Vec3
	Up       Vec3
	Yaw      float64
	Pitch    float64
	Distance float64
	Focal    float64
	Width    int
	Height   int
}

func defaultGrids() GridSet {
	return GridSet{
		NearU: 96,
		NearV: 64,
		MidU:  48,
		MidV:  32,
		FarU:  24,
		FarV:  16,
	}
}

func defaultLevels() []LevelConfig {
	grids := defaultGrids()
	return []LevelConfig{
		{
			ID:     "near",
			Label:  "NEAR",
			File:   "artifacts/near.ppm",
			Tier:   tierNear,
			Nu:     grids.NearU,
			Nv:     grids.NearV,
			Budget: 0.0005,
		},
		{
			ID:     "mid",
			Label:  "MID",
			File:   "artifacts/mid.ppm",
			Tier:   tierMid,
			Nu:     grids.MidU,
			Nv:     grids.MidV,
			Budget: 0.002,
		},
		{
			ID:     "far",
			Label:  "FAR",
			File:   "artifacts/far.ppm",
			Tier:   tierFar,
			Nu:     grids.FarU,
			Nv:     grids.FarV,
			Budget: 0.008,
		},
	}
}

func newCamera(distance float64, width, height int) Camera {
	return Camera{
		Position: Vec3{},
		Target:   Vec3{},
		Up:       Vec3{Z: 1},
		Yaw:      0.72,
		Pitch:    0.42,
		Distance: distance,
		Focal:    1050,
		Width:    width,
		Height:   height,
	}
}
