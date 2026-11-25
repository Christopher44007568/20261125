package hair

const (
	DefaultStrandCount = 20000
	DefaultGuideCount  = 512
	DefaultSeed        = 20261125
)

type Strand struct {
	ID     int
	Points []Vec3
}

type Entry struct {
	ID                  int
	Points              []Vec3
	Weight              float64
	RepresentativeID    int
	RepresentativeLevel int
	SourceIDs           []int
}

type Level struct {
	Index     int
	Label     string
	Entries   []Entry
	TotalMass float64
}

type Asset struct {
	Seed       int64
	Strands    []Strand
	Bounds     Bounds
	Levels     []Level
	GuideCount int
	Camera     CameraConfig
}

type CameraConfig struct {
	FocusDistance float64
	FarDistance   float64
	FOVDegrees    float64
}

func DefaultCamera() CameraConfig {
	return CameraConfig{
		FocusDistance: 5,
		FarDistance:   40,
		FOVDegrees:    38,
	}
}

func (a *Asset) FullLevel() Level {
	return a.Levels[0]
}

func (a *Asset) TotalMass() float64 {
	return float64(len(a.Strands))
}
