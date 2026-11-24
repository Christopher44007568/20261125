package decallab

type Report struct {
	SchemaVersion string           `json:"schema_version"`
	GoVersion     string           `json:"go_version"`
	Question      string           `json:"question"`
	Conclusion    string           `json:"conclusion"`
	Pass          bool             `json:"pass"`
	Summary       ReportSummary    `json:"summary"`
	Scenarios     []ScenarioReport `json:"scenarios"`
	Checks        []Check          `json:"checks"`
}

type ReportSummary struct {
	ScenarioCount       int     `json:"scenario_count"`
	ProbeCount          int     `json:"probe_count"`
	ObservationCount    int     `json:"observation_count"`
	ShadowEdgeCount     int     `json:"shadow_edge_count"`
	CheckCount          int     `json:"check_count"`
	PassedCheckCount    int     `json:"passed_check_count"`
	FailedCheckCount    int     `json:"failed_check_count"`
	MaxCopyNormalDeg    float64 `json:"max_copy_normal_error_deg"`
	MaxCopyDecalErrorM  float64 `json:"max_copy_decal_error_m"`
	MaxShadowEdgeBiasM  float64 `json:"max_shadow_edge_bias_m"`
	MaxBinocularPixelDx float64 `json:"max_binocular_pixel_dx"`
	MaxObliquityDeg     float64 `json:"max_obliquity_deg"`
	MaxLightingDiff     float64 `json:"max_lighting_radiance_distance"`
}

type Check struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Pass            bool    `json:"pass"`
	Required        bool    `json:"required"`
	NegativeControl bool    `json:"negative_control"`
	ScenarioID      string  `json:"scenario_id"`
	Observed        float64 `json:"observed"`
	Threshold       float64 `json:"threshold"`
	Unit            string  `json:"unit"`
	Evidence        string  `json:"evidence"`
}

type SurfaceInfo struct {
	ID       string      `json:"id"`
	Kind     SurfaceKind `json:"kind"`
	Material Material    `json:"material"`
}

type ScenarioReport struct {
	ID              string                  `json:"id"`
	Title           string                  `json:"title"`
	Tags            []string                `json:"tags"`
	Surface         SurfaceInfo             `json:"surface"`
	Decal           Decal                   `json:"decal"`
	Detail          DetailNormal            `json:"detail_normal"`
	Rig             Rig                     `json:"rig"`
	Lighting        LightingRig             `json:"lighting"`
	Projections     []ProbeProjectionRecord `json:"projections"`
	Observations    []ObservationRecord     `json:"observations"`
	CopyComparisons []CopyComparisonRecord  `json:"copy_comparisons"`
	ShadowEdges     []ShadowEdgeRecord      `json:"shadow_edges"`
	LODStates       []LODStateRecord        `json:"lod_states"`
	LODSwitches     []LODSwitchRecord       `json:"lod_switches"`
	Backside        *BacksideRecord         `json:"backside,omitempty"`
	Cache           CacheSummary            `json:"shared_cache"`
	Caches          []CacheSummary          `json:"caches"`
	Stats           ScenarioStats           `json:"stats"`
}

type ProbeProjectionRecord struct {
	ProbeID        string           `json:"probe_id"`
	CanonicalPoint Vec3             `json:"canonical_point"`
	Projection     StereoProjection `json:"projection"`
}

type ScenarioStats struct {
	ProbeCount             int     `json:"probe_count"`
	MaxViewObliquityDeg    float64 `json:"max_view_obliquity_deg"`
	MaxFootprintM          float64 `json:"max_world_sample_spacing_m"`
	MaxCurvatureStretchPct float64 `json:"max_curvature_stretch_pct"`
	MaxBinocularAngleDeg   float64 `json:"max_binocular_angle_deg"`
	MaxPixelDisparityPx    float64 `json:"max_pixel_disparity_px"`
	MaxLightingDifference  float64 `json:"max_lighting_difference"`
	SharedCacheReusePct    float64 `json:"shared_cache_reuse_pct"`
}

type ObservationRecord struct {
	ProbeID          string         `json:"probe_id"`
	Density          string         `json:"density"`
	Mode             SamplingMode   `json:"mode"`
	Eye              Eye            `json:"eye"`
	Pixel            PixelCoord     `json:"pixel"`
	PixelInFrustum   bool           `json:"pixel_in_frustum"`
	RayHit           bool           `json:"ray_hit"`
	CanonicalPoint   Vec3           `json:"canonical_point"`
	HitPoint         Vec3           `json:"hit_point"`
	HitErrorM        float64        `json:"hit_error_m"`
	SpacingM         float64        `json:"spacing_m"`
	LOD              int            `json:"lod"`
	ViewObliquityDeg float64        `json:"view_obliquity_deg"`
	FootprintXM      float64        `json:"footprint_x_m"`
	FootprintYM      float64        `json:"footprint_y_m"`
	CacheKey         string         `json:"cache_key,omitempty"`
	CacheShared      bool           `json:"cache_shared"`
	Sample           MaterialSample `json:"sample"`
	Lighting         LightingSample `json:"lighting"`
}

type CopyComparisonRecord struct {
	ProbeID               string                `json:"probe_id"`
	Density               string                `json:"density"`
	RayHit                bool                  `json:"ray_hit"`
	LeftPixel             PixelCoord            `json:"left_pixel"`
	RightSamePixel        PixelCoord            `json:"right_same_pixel"`
	LeftPoint             Vec3                  `json:"left_point"`
	RightSamePixelPoint   Vec3                  `json:"right_same_pixel_point"`
	ParallaxShiftM        float64               `json:"parallax_shift_m"`
	SourceLOD             int                   `json:"source_lod"`
	DestinationLOD        int                   `json:"destination_lod"`
	DecalCopy             CopiedDecalError      `json:"decal_copy"`
	NormalCopy            CopiedNormalError     `json:"normal_copy"`
	SourceShadowAlpha     float64               `json:"source_shadow_alpha"`
	CorrectShadowAlpha    float64               `json:"correct_shadow_alpha"`
	CopiedShadowAlphaDiff float64               `json:"copied_shadow_alpha_diff"`
	LightingDifference    EyeLightingDifference `json:"lighting_difference"`
	CopiedRadianceError   float64               `json:"copied_radiance_error"`
	Detected              bool                  `json:"detected"`
}

type LODStateRecord struct {
	ProbeID             string       `json:"probe_id"`
	LOD                 int          `json:"lod"`
	Point               Vec3         `json:"point"`
	ArcWidthM           float64      `json:"arc_width_m"`
	ArcHeightM          float64      `json:"arc_height_m"`
	DominantWavelengthM float64      `json:"dominant_wavelength_m"`
	PerturbationDeg     float64      `json:"perturbation_deg"`
	Normal              NormalSample `json:"normal"`
}

type LODSwitchRecord struct {
	ProbeID                string  `json:"probe_id"`
	FromLOD                int     `json:"from_lod"`
	ToLOD                  int     `json:"to_lod"`
	NormalAngleDeltaDeg    float64 `json:"normal_angle_delta_deg"`
	PerturbationDeltaDeg   float64 `json:"perturbation_delta_deg"`
	DominantWavelengthSame bool    `json:"dominant_wavelength_same"`
	ArcScaleDifferencePct  float64 `json:"arc_scale_difference_pct"`
}

type BacksideRecord struct {
	Ray       Ray     `json:"ray"`
	Hit       bool    `json:"hit"`
	Side      string  `json:"side"`
	Point     Vec3    `json:"point"`
	U         float64 `json:"u"`
	V         float64 `json:"v"`
	WidthM    float64 `json:"width_m"`
	HeightM   float64 `json:"height_m"`
	HitErrorM float64 `json:"hit_error_m"`
}

type CacheSummary struct {
	DensityLabel       string  `json:"density_label"`
	CellSizeM          float64 `json:"cell_size_m"`
	Entries            int     `json:"entries"`
	Hits               int     `json:"hits"`
	Misses             int     `json:"misses"`
	ReusePct           float64 `json:"reuse_pct"`
	MaxSharedPointErrM float64 `json:"max_shared_point_error_m"`
	MaxSharedNormalDeg float64 `json:"max_shared_normal_error_deg"`
	MaxArcScaleDiffPct float64 `json:"max_arc_scale_difference_pct"`
	MaxShadowAlphaDiff float64 `json:"max_shadow_alpha_difference"`
}
