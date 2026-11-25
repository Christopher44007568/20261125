package scene

import (
	"meshscope/internal/analysis"
	"meshscope/internal/mesh"
)

type MeshData struct {
	Positions  []float64 `json:"positions"`
	Normals    []float64 `json:"normals"`
	Features   []int     `json:"features"`
	Confidence []float64 `json:"confidence"`
	Deviation  []float64 `json:"deviation"`
}

type FeatureStat struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Regions    int     `json:"regions"`
	Triangles  int     `json:"triangles"`
	Area       float64 `json:"area"`
	Confidence float64 `json:"confidence"`
	Color      string  `json:"color"`
}

type BoundaryLine struct {
	A     [3]float64 `json:"a"`
	B     [3]float64 `json:"b"`
	N     [3]float64 `json:"n"`
	KindA string     `json:"kindA"`
	KindB string     `json:"kindB"`
	Conf  float64    `json:"confidence"`
}

type Callout struct {
	Position   [3]float64 `json:"position"`
	N          [3]float64 `json:"normal"`
	Label      string     `json:"label"`
	KindA      string     `json:"kindA"`
	KindB      string     `json:"kindB"`
	Confidence float64    `json:"confidence"`
}

type Probe struct {
	Position [3]float64 `json:"position"`
	N        [3]float64 `json:"normal"`
	Value    float64    `json:"value"`
	Kind     string     `json:"kind"`
}

type Datum struct {
	Normal [3]float64 `json:"normal"`
	Offset float64    `json:"offset"`
}

type Camera struct {
	Azimuth     float64    `json:"azimuth"`
	Elevation   float64    `json:"elevation"`
	Distance    float64    `json:"distance"`
	Target      [3]float64 `json:"target"`
	FieldOfView float64    `json:"fov"`
}

type Finding struct {
	Tone  string `json:"tone"`
	Title string `json:"title"`
	Value string `json:"value"`
}

type Summary struct {
	Triangles    int     `json:"triangles"`
	Regions      int     `json:"regions"`
	MeanConf     float64 `json:"meanConfidence"`
	BoundaryConf float64 `json:"boundaryConfidence"`
	RMSError     float64 `json:"rmsError"`
	MaxError     float64 `json:"maxError"`
}

type Scene struct {
	ID          string         `json:"id"`
	Index       string         `json:"index"`
	Title       string         `json:"title"`
	TitleCN     string         `json:"titleCN"`
	Subtitle    string         `json:"subtitle"`
	Code        string         `json:"code"`
	Severity    string         `json:"severity"`
	Description string         `json:"description"`
	Camera      Camera         `json:"camera"`
	Mesh        MeshData       `json:"mesh"`
	Features    []FeatureStat  `json:"features"`
	Boundaries  []BoundaryLine `json:"boundaries"`
	Callouts    []Callout      `json:"callouts"`
	Probes      []Probe        `json:"probes"`
	Datum       *Datum         `json:"datum,omitempty"`
	Summary     Summary        `json:"summary"`
	Findings    []Finding      `json:"findings"`
	AuditNote   string         `json:"auditNote"`
}

type Registry struct {
	Order []string
	ByID  map[string]*Scene
}

type SceneMeta struct {
	ID        string  `json:"id"`
	Index     string  `json:"index"`
	Title     string  `json:"title"`
	TitleCN   string  `json:"titleCN"`
	Subtitle  string  `json:"subtitle"`
	Code      string  `json:"code"`
	Severity  string  `json:"severity"`
	Triangles int     `json:"triangles"`
	MeanConf  float64 `json:"meanConfidence"`
	MaxError  float64 `json:"maxError"`
}

func BuildRegistry() *Registry {
	scenes := []*Scene{
		buildAdjacentFillets(),
		buildObliqueHole(),
		buildNearPlanar(),
	}
	r := &Registry{ByID: map[string]*Scene{}}
	for _, s := range scenes {
		r.Order = append(r.Order, s.ID)
		r.ByID[s.ID] = s
	}
	return r
}

func List(r *Registry) []SceneMeta {
	out := make([]SceneMeta, 0, len(r.Order))
	for _, id := range r.Order {
		s := r.ByID[id]
		out = append(out, SceneMeta{
			ID: s.ID, Index: s.Index, Title: s.Title, TitleCN: s.TitleCN,
			Subtitle: s.Subtitle, Code: s.Code, Severity: s.Severity,
			Triangles: s.Summary.Triangles, MeanConf: s.Summary.MeanConf,
			MaxError: s.Summary.MaxError,
		})
	}
	return out
}

func buildScene(base *Scene, m *mesh.Mesh, res analysis.Result) *Scene {
	base.Mesh = expand(m, res)
	base.Boundaries = boundaryLines(res)
	base.Callouts = callouts(res)
	base.Probes = probes(res)
	if res.DatumValid {
		base.Datum = &Datum{Normal: vec(res.DatumNormal), Offset: round(res.DatumOffset, 4)}
	}
	base.Features = features(res)
	base.Summary = Summary{
		Triangles: len(m.Tris), Regions: len(res.Regions),
		MeanConf: round(res.MeanConf, 3), BoundaryConf: round(res.BoundaryAvg, 3),
		RMSError: round(res.RMSDev, 4), MaxError: round(res.MaxAbsDev, 4),
	}
	b := m.Bounds()
	base.Camera.Target = vec(b.Center)
	if base.Camera.Distance <= 0 {
		base.Camera.Distance = b.Radius * 3.15
	}
	if base.Camera.FieldOfView <= 0 {
		base.Camera.FieldOfView = 30
	}
	return base
}
