package decallab

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strings"
	"testing"
)

func TestRunReportPassesAllRequiredChecks(t *testing.T) {
	report := Run()
	if !report.Pass {
		for _, check := range report.Checks {
			if check.Required && !check.Pass {
				t.Errorf("required check %s failed: observed=%v threshold=%v %s (%s)",
					check.ID, check.Observed, check.Threshold, check.Unit, check.Evidence)
			}
		}
	}
	if len(report.Scenarios) != 4 {
		t.Fatalf("scenario count = %d, want 4", len(report.Scenarios))
	}
	if report.Summary.ObservationCount == 0 || report.Summary.ShadowEdgeCount == 0 {
		t.Fatalf("report did not record observations/edges: %+v", report.Summary)
	}
}

func TestDecalScaleIsIndependentOfSamplingDensity(t *testing.T) {
	for _, scenario := range BuildScenarios() {
		for _, probe := range scenario.Probes {
			point := scenario.ProbePoint(probe)
			high := EvaluateMaterial(point, scenario.Surface, scenario.Decal, scenario.Detail, scenario.Shadow, 0.002, 0)
			low := EvaluateMaterial(point, scenario.Surface, scenario.Decal, scenario.Detail, scenario.Shadow, 0.080, 2)

			if high.Decal.ProjectedWidthM != scenario.Decal.WidthM ||
				high.Decal.ProjectedHeightM != scenario.Decal.HeightM {
				t.Fatalf("%s/%s nominal scale changed", scenario.ID, probe.ID)
			}
			if math.Abs(high.Decal.SurfaceArcWidthM-low.Decal.SurfaceArcWidthM) > 1e-12 {
				t.Fatalf("%s/%s width depends on LOD/density: %.9f vs %.9f",
					scenario.ID, probe.ID, high.Decal.SurfaceArcWidthM, low.Decal.SurfaceArcWidthM)
			}
			if math.Abs(high.Decal.SurfaceArcHeightM-low.Decal.SurfaceArcHeightM) > 1e-12 {
				t.Fatalf("%s/%s height depends on LOD/density", scenario.ID, probe.ID)
			}
		}
	}
}

func TestDetailNormalLODOnlyDropsHighFrequencies(t *testing.T) {
	anchor := Vec3{Z: 1}
	normal := DefaultDetailNormal(anchor, Vec3{X: 1}, Vec3{Y: 1}, Vec3{Z: 1})
	frame := MakeFrame(anchor, Vec3{Z: 1}, Vec3{Y: 1})
	point := anchor.Add(Vec3{X: 0.013, Y: -0.021})

	lod0 := normal.Evaluate(point, frame, 0, 0.001)
	lod2 := normal.Evaluate(point, frame, 2, 0.08)
	if len(lod0.ActiveWavelengthsM) != 3 {
		t.Fatalf("LOD0 active wavelengths = %d, want 3", len(lod0.ActiveWavelengthsM))
	}
	if len(lod2.ActiveWavelengthsM) != 1 {
		t.Fatalf("LOD2 active wavelengths = %d, want 1", len(lod2.ActiveWavelengthsM))
	}
	if lod0.DominantWavelengthM != lod2.DominantWavelengthM {
		t.Fatalf("dominant wavelength changed: %.4f vs %.4f",
			lod0.DominantWavelengthM, lod2.DominantWavelengthM)
	}
	if normal.LODForSpacing(0.060, 2.2) < 2 {
		t.Fatalf("6 cm spacing did not select coarse-only LOD")
	}
	if angleBetweenDeg(lod0.WorldNormal, lod2.WorldNormal) < 0.01 {
		t.Fatal("LOD switch produced no measurable normal change")
	}
}

func TestShadowEdgeBiasStaysWithinOneStep(t *testing.T) {
	scenario := buildCurvedObliqueScenario()
	for _, direction := range []float64{0, 37, 91, 143} {
		edge := MeasureShadowEdge(scenario.Surface, scenario.Shadow, direction, 0.012)
		if edge.CrossingSamples != 1 {
			t.Fatalf("direction %.1f did not find exactly one crossing: %+v", direction, edge)
		}
		if edge.EdgeAbsBiasM > edge.SampleSpacingM+1e-9 {
			t.Fatalf("edge bias %.5fm exceeds step %.5fm", edge.EdgeAbsBiasM, edge.SampleSpacingM)
		}
		if math.Abs(edge.ExpectedRadiusM-scenario.Shadow.ExpectedEdgeRadiusM(edgeDirection(scenario.Shadow, direction))) > 1e-12 {
			t.Fatal("analytic edge radius mismatch")
		}
	}
}

func edgeDirection(shadow ContactShadow, angle float64) Vec3 {
	return shadow.Direction(angle)
}

func TestMonocularTangentCopyIsMeasurablyWrong(t *testing.T) {
	report := Run()
	maxNormal := 0.0
	maxDecal := 0.0
	detectedScenarios := map[string]bool{}
	for _, scenario := range report.Scenarios {
		for _, copy := range scenario.CopyComparisons {
			if !copy.RayHit {
				t.Fatalf("%s/%s copy ray missed surface", scenario.ID, copy.ProbeID)
			}
			maxNormal = math.Max(maxNormal, copy.NormalCopy.TangentReuseErrorDeg)
			maxDecal = math.Max(maxDecal, copy.DecalCopy.ReprojectionM)
			if copy.Detected {
				detectedScenarios[scenario.ID] = true
			}
		}
	}
	if maxNormal < 2.0 {
		t.Fatalf("maximum tangent-copy normal error = %.3f°, want at least 2°", maxNormal)
	}
	if maxDecal < 0.008 {
		t.Fatalf("maximum decal copy error = %.5fm, want at least 0.008m", maxDecal)
	}
	if len(detectedScenarios) != len(report.Scenarios) {
		t.Fatalf("detected scenarios = %d/%d", len(detectedScenarios), len(report.Scenarios))
	}
}

func TestWritersProduceParseableOutput(t *testing.T) {
	report := Run()
	var jsonBuffer bytes.Buffer
	if err := report.WriteJSON(&jsonBuffer); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(jsonBuffer.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON is not parseable: %v", err)
	}

	var markdown bytes.Buffer
	if err := report.WriteMarkdown(&markdown); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markdown.String(), "世界尺度") {
		t.Fatal("markdown does not contain world-scale report")
	}

	csvWriters := map[string]func(*Report, io.Writer) error{
		"normal": WriteNormalProbesCSV,
		"decal":  WriteDecalScalesCSV,
		"shadow": WriteShadowEdgesCSV,
		"stereo": WriteBinocularCSV,
	}
	for name, writer := range csvWriters {
		var buffer bytes.Buffer
		if err := writer(report, &buffer); err != nil {
			t.Fatalf("%s CSV: %v", name, err)
		}
		if buffer.Len() < 100 || !strings.Contains(buffer.String(), ",") {
			t.Fatalf("%s CSV looks empty", name)
		}
	}
}
