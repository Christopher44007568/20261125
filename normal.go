package decallab

import "math"

type NormalOctave struct {
	ID           string  `json:"id"`
	WavelengthM  float64 `json:"wavelength_m"`
	AmplitudeM   float64 `json:"amplitude_m"`
	DirectionDeg float64 `json:"direction_deg"`
	PhaseRad     float64 `json:"phase_rad"`
}

type DetailNormal struct {
	ID      string         `json:"id"`
	Anchor  Vec3           `json:"anchor"`
	AxisU   Vec3           `json:"axis_u"`
	AxisV   Vec3           `json:"axis_v"`
	Normal  Vec3           `json:"normal"`
	Octaves []NormalOctave `json:"octaves"`
	MaxLOD  int            `json:"max_lod"`
}

func DefaultDetailNormal(anchor Vec3, axisU, axisV, normal Vec3) DetailNormal {
	return DetailNormal{
		ID:     "world_meter_detail",
		Anchor: anchor,
		AxisU:  axisU,
		AxisV:  axisV,
		Normal: normal,
		Octaves: []NormalOctave{
			{ID: "coarse_weave", WavelengthM: 0.120, AmplitudeM: 0.0018, DirectionDeg: 18, PhaseRad: 0.35},
			{ID: "medium_grain", WavelengthM: 0.048, AmplitudeM: 0.0008, DirectionDeg: 64, PhaseRad: 1.40},
			{ID: "fine_pores", WavelengthM: 0.020, AmplitudeM: 0.00028, DirectionDeg: -27, PhaseRad: 2.60},
		},
		MaxLOD: 2,
	}
}

// ActiveOctaveCount maps LOD 0..2 to high/medium/low detail. Higher LOD drops
// the finest octaves but never rescales the wavelengths that remain.
func (n DetailNormal) ActiveOctaveCount(lod int) int {
	if lod < 0 {
		lod = 0
	}
	if lod > n.MaxLOD {
		lod = n.MaxLOD
	}
	count := len(n.Octaves) - lod
	if count < 1 {
		count = 1
	}
	if count > len(n.Octaves) {
		count = len(n.Octaves)
	}
	return count
}

func (n DetailNormal) LODForSpacing(spacingM, nyquistFactor float64) int {
	if nyquistFactor <= 0 {
		nyquistFactor = 2.2
	}
	lod := 0
	for lod <= n.MaxLOD {
		active := n.ActiveOctaveCount(lod)
		required := nyquistFactor * spacingM
		ok := true
		for i := 0; i < active; i++ {
			if n.Octaves[i].WavelengthM < required {
				ok = false
				break
			}
		}
		if ok || lod == n.MaxLOD {
			return lod
		}
		lod++
	}
	return n.MaxLOD
}

type NormalSample struct {
	DetailID             string    `json:"detail_id"`
	LOD                  int       `json:"lod"`
	SamplePoint          Vec3      `json:"sample_point"`
	TangentNormal        Vec3      `json:"tangent_normal"`
	WorldNormal          Vec3      `json:"world_normal"`
	GeometricNormal      Vec3      `json:"geometric_normal"`
	PerturbationDeg      float64   `json:"perturbation_deg"`
	DominantWavelengthM  float64   `json:"dominant_wavelength_m"`
	ActiveWavelengthsM   []float64 `json:"active_wavelengths_m"`
	ActiveFrequenciesCPM []float64 `json:"active_frequencies_cycles_per_meter"`
	UndersampledOctaves  []string  `json:"undersampled_octaves"`
}

func (n DetailNormal) Evaluate(point Vec3, frame Frame, lod int, spacingM float64) NormalSample {
	activeCount := n.ActiveOctaveCount(lod)
	gradient := Vec3{}
	wavelengths := make([]float64, 0, activeCount)
	frequencies := make([]float64, 0, activeCount)
	undersampled := []string{}

	for i := 0; i < activeCount; i++ {
		octave := n.Octaves[i]
		rad := octave.DirectionDeg * math.Pi / 180
		direction := n.AxisU.Scale(math.Cos(rad)).Add(n.AxisV.Scale(math.Sin(rad))).Normalize()
		q := point.Sub(n.Anchor).Dot(direction)
		frequency := 1 / octave.WavelengthM
		angularFrequency := 2 * math.Pi * frequency
		derivative := octave.AmplitudeM * angularFrequency * math.Cos(angularFrequency*q+octave.PhaseRad)
		gradient = gradient.Add(direction.Scale(derivative))
		wavelengths = append(wavelengths, octave.WavelengthM)
		frequencies = append(frequencies, frequency)
		if spacingM > 0 && octave.WavelengthM < 2.2*spacingM {
			undersampled = append(undersampled, octave.ID)
		}
	}

	dhDT := gradient.Dot(frame.Tangent)
	dhDB := gradient.Dot(frame.Bitangent)
	tangentNormal := Vec3{X: -dhDT, Y: -dhDB, Z: 1}.Normalize()
	worldNormal := frame.Tangent.Scale(tangentNormal.X).
		Add(frame.Bitangent.Scale(tangentNormal.Y)).
		Add(frame.Normal.Scale(tangentNormal.Z)).
		Normalize()

	return NormalSample{
		DetailID:             n.ID,
		LOD:                  lod,
		SamplePoint:          point,
		TangentNormal:        tangentNormal,
		WorldNormal:          worldNormal,
		GeometricNormal:      frame.Normal,
		PerturbationDeg:      angleBetweenDeg(worldNormal, frame.Normal),
		DominantWavelengthM:  n.Octaves[0].WavelengthM,
		ActiveWavelengthsM:   wavelengths,
		ActiveFrequenciesCPM: frequencies,
		UndersampledOctaves:  undersampled,
	}
}

type CopiedNormalError struct {
	SourceTangentNormal  Vec3    `json:"source_tangent_normal"`
	UsedTangentNormal    Vec3    `json:"used_tangent_normal"`
	CorrectWorldNormal   Vec3    `json:"correct_world_normal"`
	CopiedWorldNormal    Vec3    `json:"copied_world_normal"`
	TangentReuseErrorDeg float64 `json:"tangent_reuse_error_deg"`
	WorldCopyErrorDeg    float64 `json:"world_copy_error_deg"`
}

// ReuseTangentAtDestination applies the source tangent value in the destination
// surface frame. This is the smallest plausible "copy one eye's tangent result"
// mistake and is still wrong when the eye rays reach different curved fragments.
func ReuseTangentAtDestination(source NormalSample, destinationFrame Frame, correct NormalSample) CopiedNormalError {
	tn := source.TangentNormal
	copiedWorld := destinationFrame.Tangent.Scale(tn.X).
		Add(destinationFrame.Bitangent.Scale(tn.Y)).
		Add(destinationFrame.Normal.Scale(tn.Z)).
		Normalize()
	return CopiedNormalError{
		SourceTangentNormal:  source.TangentNormal,
		UsedTangentNormal:    tn,
		CorrectWorldNormal:   correct.WorldNormal,
		CopiedWorldNormal:    copiedWorld,
		TangentReuseErrorDeg: angleBetweenDeg(copiedWorld, correct.WorldNormal),
		WorldCopyErrorDeg:    angleBetweenDeg(source.WorldNormal, correct.WorldNormal),
	}
}
