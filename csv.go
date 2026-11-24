package decallab

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

func WriteNormalProbesCSV(report *Report, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"scenario_id", "probe_id", "density", "mode", "eye", "lod",
		"point_x_m", "point_y_m", "point_z_m",
		"tangent_nx", "tangent_ny", "tangent_nz",
		"world_nx", "world_ny", "world_nz",
		"geometric_nx", "geometric_ny", "geometric_nz",
		"perturbation_deg", "dominant_wavelength_m", "active_wavelengths_m",
		"spacing_m", "undersampled_octaves",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, observation := range scenario.Observations {
			normal := observation.Sample.Normal
			row := []string{
				scenario.ID, observation.ProbeID, observation.Density, string(observation.Mode), string(observation.Eye),
				strconv.Itoa(observation.LOD),
				ft(observation.HitPoint.X), ft(observation.HitPoint.Y), ft(observation.HitPoint.Z),
				ft(normal.TangentNormal.X), ft(normal.TangentNormal.Y), ft(normal.TangentNormal.Z),
				ft(normal.WorldNormal.X), ft(normal.WorldNormal.Y), ft(normal.WorldNormal.Z),
				ft(normal.GeometricNormal.X), ft(normal.GeometricNormal.Y), ft(normal.GeometricNormal.Z),
				ft(normal.PerturbationDeg), ft(normal.DominantWavelengthM),
				joinFloats(normal.ActiveWavelengthsM), ft(observation.SpacingM),
				strings.Join(normal.UndersampledOctaves, "|"),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func WriteDecalScalesCSV(report *Report, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"scenario_id", "probe_id", "density", "mode", "eye",
		"u", "v", "in_bounds", "edge_distance_m",
		"nominal_width_m", "nominal_height_m",
		"surface_arc_width_m", "surface_arc_height_m", "curvature_stretch_pct",
		"hit_error_m", "cache_key", "cache_shared",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, observation := range scenario.Observations {
			decal := observation.Sample.Decal
			row := []string{
				scenario.ID, observation.ProbeID, observation.Density, string(observation.Mode), string(observation.Eye),
				ft(decal.U), ft(decal.V), strconv.FormatBool(decal.InBounds), ft(decal.EdgeDistanceM),
				ft(decal.ProjectedWidthM), ft(decal.ProjectedHeightM),
				ft(decal.SurfaceArcWidthM), ft(decal.SurfaceArcHeightM), ft(decal.CurvatureStretchPct),
				ft(observation.HitErrorM), observation.CacheKey, strconv.FormatBool(observation.CacheShared),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func WriteShadowEdgesCSV(report *Report, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"scenario_id", "shadow_id", "density", "direction_deg", "sample_spacing_m",
		"expected_radius_m", "sampled_radius_m", "edge_bias_m", "edge_abs_bias_m",
		"edge_error_pct", "transition_10_90_m", "crossing_samples",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, edge := range scenario.ShadowEdges {
			row := []string{
				scenario.ID, edge.ShadowID, edge.DensityLabel, ft(edge.DirectionDeg), ft(edge.SampleSpacingM),
				ft(edge.ExpectedRadiusM), ft(edge.SampledRadiusM), ft(edge.EdgeBiasM), ft(edge.EdgeAbsBiasM),
				ft(edge.EdgeErrorPct), ft(edge.Transition1090M), strconv.Itoa(edge.CrossingSamples),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func WriteBinocularCSV(report *Report, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{
		"scenario_id", "probe_id", "density", "ray_hit",
		"left_px_x", "left_px_y", "right_same_px_x", "right_same_px_y",
		"parallax_shift_m", "decal_reprojection_error_m", "decal_scale_error_pct",
		"tangent_reuse_normal_error_deg", "world_copy_normal_error_deg",
		"shadow_copy_alpha_diff", "light_direction_angle_deg",
		"lighting_radiance_distance", "copied_radiance_error", "detected",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}
	for _, scenario := range report.Scenarios {
		for _, copy := range scenario.CopyComparisons {
			row := []string{
				scenario.ID, copy.ProbeID, copy.Density, strconv.FormatBool(copy.RayHit),
				ft(copy.LeftPixel.X), ft(copy.LeftPixel.Y), ft(copy.RightSamePixel.X), ft(copy.RightSamePixel.Y),
				ft(copy.ParallaxShiftM), ft(copy.DecalCopy.ReprojectionM), ft(copy.DecalCopy.ScaleErrorPct),
				ft(copy.NormalCopy.TangentReuseErrorDeg), ft(copy.NormalCopy.WorldCopyErrorDeg),
				ft(copy.CopiedShadowAlphaDiff), ft(copy.LightingDifference.LightDirectionAngleDeg),
				ft(copy.LightingDifference.RadianceDistance), ft(copy.CopiedRadianceError),
				strconv.FormatBool(copy.Detected),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func ft(value float64) string {
	if math.IsInf(value, 0) {
		return "inf"
	}
	if math.IsNaN(value) {
		return "nan"
	}
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func joinFloats(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.4f", value))
	}
	return strings.Join(parts, "|")
}
