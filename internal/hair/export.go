package hair

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

const (
	StrandsSchema = "hairlod/strands-v1"
	GuidesSchema  = "hairlod/guides-v1"
	LODSchema     = "hairlod/lod-v1"
)

type strandRecord struct {
	ID          int          `json:"id"`
	Root        [3]float64   `json:"root"`
	Points      [][3]float64 `json:"points"`
	DensityMass float64      `json:"densityMass"`
}

type StrandDocument struct {
	Schema          string         `json:"schema"`
	Seed            int64          `json:"seed"`
	Count           int            `json:"count"`
	PointsPerStrand int            `json:"pointsPerStrand"`
	Bounds          Bounds         `json:"bounds"`
	Strands         []strandRecord `json:"strands"`
}

type guideRecord struct {
	ID                     int          `json:"id"`
	Level                  int          `json:"level"`
	RepresentativeStrandID int          `json:"representativeStrandId"`
	RepresentativeLevel    int          `json:"representativeLevel"`
	SourceCount            int          `json:"sourceCount"`
	DensityWeight          float64      `json:"densityWeight"`
	Mass                   float64      `json:"mass"`
	Points                 [][3]float64 `json:"representativePoints"`
	SourceIDs              []int        `json:"sourceIds"`
}

type GuideDocument struct {
	Schema        string        `json:"schema"`
	Seed          int64         `json:"seed"`
	SourceStrands int           `json:"sourceStrands"`
	GuideCount    int           `json:"guideCount"`
	LevelCounts   []int         `json:"levelCounts"`
	Guides        []guideRecord `json:"guides"`
}

type lodEntryRecord struct {
	ID                     int          `json:"id"`
	Mass                   float64      `json:"mass"`
	DensityWeight          float64      `json:"densityWeight"`
	RepresentativeStrandID int          `json:"representativeStrandId"`
	RepresentativeLevel    int          `json:"representativeLevel"`
	SourceIDs              []int        `json:"sourceIds,omitempty"`
	Points                 [][3]float64 `json:"representativePoints"`
}

type lodLevelRecord struct {
	Index       int              `json:"index"`
	Label       string           `json:"label"`
	EntryCount  int              `json:"entryCount"`
	TotalMass   float64          `json:"totalMass"`
	EntriesFile string           `json:"entriesFile,omitempty"`
	Entries     []lodEntryRecord `json:"entries,omitempty"`
}

type LODDocument struct {
	Schema        string            `json:"schema"`
	Seed          int64             `json:"seed"`
	Camera        CameraConfig      `json:"camera"`
	Levels        []lodLevelRecord  `json:"levels"`
	Dolly         DollyReport       `json:"dollyCoverage"`
	Error         ErrorReport       `json:"error"`
	ErrorDistance float64           `json:"errorDistance"`
	Outputs       map[string]string `json:"outputs"`
}

func strandPoints(points []Vec3) [][3]float64 {
	out := make([][3]float64, len(points))
	for i, p := range points {
		out[i] = [3]float64{p.X, p.Y, p.Z}
	}
	return out
}

func BuildStrandDocument(asset *Asset) StrandDocument {
	doc := StrandDocument{
		Schema:          StrandsSchema,
		Seed:            asset.Seed,
		Count:           len(asset.Strands),
		PointsPerStrand: len(asset.Strands[0].Points),
		Bounds:          asset.Bounds,
		Strands:         make([]strandRecord, len(asset.Strands)),
	}
	for i, s := range asset.Strands {
		doc.Strands[i] = strandRecord{
			ID:          s.ID,
			Root:        [3]float64{s.Points[0].X, s.Points[0].Y, s.Points[0].Z},
			Points:      strandPoints(s.Points),
			DensityMass: 1,
		}
	}
	return doc
}

func BuildGuideDocument(asset *Asset) GuideDocument {
	doc := GuideDocument{
		Schema:        GuidesSchema,
		Seed:          asset.Seed,
		SourceStrands: len(asset.Strands),
		GuideCount:    len(asset.Levels[1].Entries),
		LevelCounts:   levelCounts(asset),
	}
	doc.Guides = make([]guideRecord, len(asset.Levels[1].Entries))
	total := asset.Levels[1].TotalMass
	for i, e := range asset.Levels[1].Entries {
		doc.Guides[i] = guideRecord{
			ID:                     e.ID,
			Level:                  1,
			RepresentativeStrandID: e.RepresentativeID,
			RepresentativeLevel:    e.RepresentativeLevel,
			SourceCount:            len(e.SourceIDs),
			DensityWeight:          e.Weight / total,
			Mass:                   e.Weight,
			Points:                 strandPoints(e.Points),
			SourceIDs:              e.SourceIDs,
		}
	}
	return doc
}

func BuildLODDocument(asset *Asset, dolly DollyReport, err ErrorReport, errorDistance float64, outputs map[string]string) LODDocument {
	doc := LODDocument{
		Schema:        LODSchema,
		Seed:          asset.Seed,
		Camera:        asset.Camera,
		Dolly:         dolly,
		Error:         err,
		ErrorDistance: errorDistance,
		Outputs:       outputs,
	}
	for _, level := range asset.Levels {
		record := lodLevelRecord{
			Index:      level.Index,
			Label:      level.Label,
			EntryCount: len(level.Entries),
			TotalMass:  level.TotalMass,
		}
		if level.Index == 0 {
			record.EntriesFile = "strands.json"
			doc.Levels = append(doc.Levels, record)
			continue
		}
		record.Entries = make([]lodEntryRecord, len(level.Entries))
		for i, e := range level.Entries {
			record.Entries[i] = lodEntryRecord{
				ID:                     e.ID,
				Mass:                   e.Weight,
				DensityWeight:          e.Weight / level.TotalMass,
				RepresentativeStrandID: e.RepresentativeID,
				RepresentativeLevel:    e.RepresentativeLevel,
				SourceIDs:              e.SourceIDs,
				Points:                 strandPoints(e.Points),
			}
		}
		doc.Levels = append(doc.Levels, record)
	}
	return doc
}

func levelCounts(asset *Asset) []int {
	out := make([]int, 0, len(asset.Levels))
	for _, level := range asset.Levels {
		out = append(out, len(level.Entries))
	}
	return out
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func WritePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
