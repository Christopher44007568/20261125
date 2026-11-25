package spectralresample

import (
	"math"
	"math/rand"
	"strings"
)

// Family groups patches for separate spectral-mismatch reporting. The brief
// explicitly requires grey, skin and orange targets to be checked on their
// own so equal average colour cannot hide spectral failures.
const (
	FamilyGrey   = "grey"
	FamilySkin   = "skin"
	FamilyOrange = "orange"
	FamilyOther  = "other"
)

// ClassifyFamily maps a ColorChecker label to its inspection family.
func ClassifyFamily(label string) string {
	switch {
	case containsAny(label, "neutral", "white", "black"):
		return FamilyGrey
	case containsAny(label, "skin"):
		return FamilySkin
	case containsAny(label, "orange"):
		return FamilyOrange
	default:
		return FamilyOther
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

// PatchResult gathers every computed quantity for one patch pair.
type PatchResult struct {
	Index          int
	Label          string
	Family         string
	Base           Spectrum
	Metamer        Spectrum
	D65BaseXYZ     [3]float64
	D65MetaXYZ     [3]float64
	LEDBaseXYZ     [3]float64
	LEDMetaXYZ     [3]float64
	D65BaseRGB     RGB
	D65MetaRGB     RGB
	LEDBaseRGB     RGB
	LEDMetaRGB     RGB
	DeltaED65      float64
	DeltaELED      float64
	GainBaseRGB    RGB
	GainMetaRGB    RGB
	DeltaEGainBase float64
	DeltaEGainMeta float64
	GainPairSplit  float64
}

// PipelineConfig bundles runtime choices.
type PipelineConfig struct {
	DataDir string
	LED     []LEDEmitter
	LEDName string
	Metamer MetamerConfig
}

// Pipeline holds loaded reference data and computed results.
type Pipeline struct {
	Config   PipelineConfig
	CMF      CMF
	D65      Illuminant
	LED      Illuminant
	Results  []PatchResult
	D65White [3]float64
	LEDWhite [3]float64
}

// NewPipeline loads CMF, D65 and the ColorChecker table, then builds one
// metamer pair per patch and evaluates both under D65 and the LED.
func NewPipeline(cfg PipelineConfig) (*Pipeline, error) {
	cmf, err := LoadCMF(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	d65, err := LoadD65(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	led := NewGaussianLED(cfg.LEDName, cfg.LED)
	names, patches, err := LoadReflectanceCSV(cfg.DataDir + "/colorchecker_ohta_5nm.csv")
	if err != nil {
		return nil, err
	}

	p := &Pipeline{
		Config:   cfg,
		CMF:      cmf,
		D65:      d65,
		LED:      led,
		D65White: cmf.WhitePoint(d65),
		LEDWhite: cmf.WhitePoint(led),
	}
	rng := rand.New(rand.NewSource(cfg.Metamer.Seed))
	for i, base := range patches {
		family := ClassifyFamily(names[i])
		pair := cmf.BuildMetamerPair(d65, base, names[i], family, cfg.Metamer, rng)
		p.Results = append(p.Results, p.evaluate(i, names[i], family, pair))
	}
	return p, nil
}

func (p *Pipeline) evaluate(index int, label, family string, pair MetamerPair) PatchResult {
	cmf := p.CMF
	d65Base := cmf.Tristimulus(p.D65, pair.Base)
	d65Meta := cmf.Tristimulus(p.D65, pair.Metamer)
	ledBase := cmf.Tristimulus(p.LED, pair.Base)
	ledMeta := cmf.Tristimulus(p.LED, pair.Metamer)

	// D65 comparison uses the native D65 white point.
	labD65Base := XYZToLab(d65Base, p.D65White)
	labD65Meta := XYZToLab(d65Meta, p.D65White)

	// LED comparison chromatically adapts each stimulus to the D65
	// reference white via Bradford, so the reported difference isolates the
	// reflectance mismatch rather than the illuminant white shift.
	adaptedBase := ChromaticAdaptation(ledBase, p.LEDWhite, p.D65White)
	adaptedMeta := ChromaticAdaptation(ledMeta, p.LEDWhite, p.D65White)
	labLEDBase := XYZToLab(adaptedBase, p.D65White)
	labLEDMeta := XYZToLab(adaptedMeta, p.D65White)

	// Naive reference pipeline: render under D65, then apply a single RGB
	// channel gain calibrated on the white point. This is the "multiply by
	// a colour-temperature factor" approach the brief warns against.
	gain := rgbWhiteGain(cmf, p.D65, p.LED)
	gainBase := applyRGBGain(SRGB8(XYZToLinearSRGB(d65Base)), gain)
	gainMeta := applyRGBGain(SRGB8(XYZToLinearSRGB(d65Meta)), gain)
	// The shortcut predicts LED colour from D65 colour. Its error against
	// the true spectral LED rendering, and its failure to distinguish the
	// metamer pair, are both quantified.
	gainDeltaBase := rgbEuclidean(gainBase, rgbFrom8(SRGB8(XYZToLinearSRGB(adaptedBase))))
	gainDeltaMeta := rgbEuclidean(gainMeta, rgbFrom8(SRGB8(XYZToLinearSRGB(adaptedMeta))))
	gainPairSplit := rgbEuclidean(gainBase, gainMeta)

	return PatchResult{
		Index:          index,
		Label:          label,
		Family:         family,
		Base:           pair.Base,
		Metamer:        pair.Metamer,
		D65BaseXYZ:     d65Base,
		D65MetaXYZ:     d65Meta,
		LEDBaseXYZ:     ledBase,
		LEDMetaXYZ:     ledMeta,
		D65BaseRGB:     rgbFrom8(SRGB8(XYZToLinearSRGB(d65Base))),
		D65MetaRGB:     rgbFrom8(SRGB8(XYZToLinearSRGB(d65Meta))),
		LEDBaseRGB:     rgbFrom8(SRGB8(XYZToLinearSRGB(adaptedBase))),
		LEDMetaRGB:     rgbFrom8(SRGB8(XYZToLinearSRGB(adaptedMeta))),
		DeltaED65:      DeltaE2000(labD65Base, labD65Meta),
		DeltaELED:      DeltaE2000(labLEDBase, labLEDMeta),
		GainBaseRGB:    gainBase,
		GainMetaRGB:    gainMeta,
		DeltaEGainBase: gainDeltaBase,
		DeltaEGainMeta: gainDeltaMeta,
		GainPairSplit:  gainPairSplit,
	}
}

// rgbWhiteGain derives the per-channel scalar gain that maps the D65 white
// point rendering to the LED white point rendering. It is exactly the
// single-factor-per-channel shortcut being tested against the full pipeline.
func rgbWhiteGain(cmf CMF, d65, led Illuminant) [3]float64 {
	grid := Grid()
	white := Spectrum{Wavelengths: grid, Values: make([]float64, len(grid))}
	for i := range white.Values {
		white.Values[i] = 1
	}
	d65RGB := XYZToLinearSRGB(cmf.Tristimulus(d65, white))
	ledAdapted := ChromaticAdaptation(cmf.Tristimulus(led, white), cmf.WhitePoint(led), cmf.WhitePoint(d65))
	ledRGB := XYZToLinearSRGB(ledAdapted)
	var gain [3]float64
	for i := range gain {
		gain[i] = ledRGB[i] / d65RGB[i]
	}
	return gain
}

func applyRGBGain(v [3]int, gain [3]float64) RGB {
	var out [3]int
	for i := 0; i < 3; i++ {
		out[i] = int(math.Round(Clamp01(float64(v[i])/255*gain[i]) * 255))
	}
	return rgbFrom8(out)
}

func rgbEuclidean(a, b RGB) float64 {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// ByFamily returns results belonging to one inspection family, in order.
func (p *Pipeline) ByFamily(family string) []PatchResult {
	var out []PatchResult
	for _, result := range p.Results {
		if result.Family == family {
			out = append(out, result)
		}
	}
	return out
}
