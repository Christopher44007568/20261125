package stereoprobe

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	globalMetrics = make(map[string][]truthSample)
	os.Exit(m.Run())
}

const (
	screenWidth  = 1280
	screenHeight = 720
	eyeBaseline  = 0.064
)

type truthCase struct {
	name         string
	group        string
	camera       Camera
	eye          int
	offset       float64
	frustum      Frustum
	world        Vec3
	expectCulled bool
}

type truthSample struct {
	name   string
	group  string
	eye    int
	values map[string]float64
}

type metricStats struct {
	name  string
	min   float64
	max   float64
	mean  float64
	p95   float64
	worst truthSample
}

type errorCollector struct {
	metrics map[string][]truthSample
}

var globalMetrics = map[string][]truthSample{}

func newErrorCollector() *errorCollector {
	return &errorCollector{metrics: make(map[string][]truthSample)}
}

func (c *errorCollector) add(t *testing.T, tc truthCase, metric string, got, want float64, tolerance float64) {
	t.Helper()
	err := math.Abs(got - want)
	sample := truthSample{
		name:   tc.name,
		group:  tc.group,
		eye:    tc.eye,
		values: map[string]float64{"error": err, "got": got, "want": want, "tolerance": tolerance},
	}
	c.metrics[metric] = append(c.metrics[metric], sample)
	globalMetrics[metric] = append(globalMetrics[metric], sample)
	if err > tolerance {
		t.Fatalf("%s %s = %.12f, want %.12f, error %.3e exceeds %.3e", tc.name, metric, got, want, err, tolerance)
	}
}

func (c *errorCollector) report(t *testing.T) string {
	t.Helper()
	names := make([]string, 0, len(c.metrics))
	for name := range c.metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	var lines []string
	lines = append(lines, "metric,count,min,mean,p95,max,worst_case,worst_eye,got,want,error")
	for _, name := range names {
		samples := c.metrics[name]
		stats := summarize(name, samples)
		lines = append(lines, fmt.Sprintf("%s,%d,%.3e,%.3e,%.3e,%.3e,%s,eye%d,%.9f,%.9f,%.3e",
			name, len(samples), stats.min, stats.mean, stats.p95, stats.max,
			strings.ReplaceAll(stats.worst.name, ",", "_"), stats.worst.eye,
			stats.worst.values["got"], stats.worst.values["want"], stats.max))
	}
	report := strings.Join(lines, "\n")
	names = make([]string, 0, len(globalMetrics))
	for name := range globalMetrics {
		names = append(names, name)
	}
	sort.Strings(names)

	var globalLines []string
	globalLines = append(globalLines, "metric,count,min,mean,p95,max,worst_case,worst_eye,got,want,error")
	for _, name := range names {
		samples := globalMetrics[name]
		stats := summarize(name, samples)
		globalLines = append(globalLines, fmt.Sprintf("%s,%d,%.3e,%.3e,%.3e,%.3e,%s,eye%d,%.9f,%.9f,%.3e",
			name, len(samples), stats.min, stats.mean, stats.p95, stats.max,
			strings.ReplaceAll(stats.worst.name, ",", "_"), stats.worst.eye,
			stats.worst.values["got"], stats.worst.values["want"], stats.max))
	}
	globalReport := strings.Join(globalLines, "\n")
	if path := os.Getenv("STEREO_TRUTH_REPORT"); path != "" {
		if err := os.WriteFile(path, []byte(globalReport+"\n"), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}
	t.Log("\n" + report)
	return report
}

func summarize(name string, samples []truthSample) metricStats {
	values := append([]truthSample(nil), samples...)
	sort.Slice(values, func(i, j int) bool {
		return values[i].values["error"] < values[j].values["error"]
	})
	stats := metricStats{name: name, min: values[0].values["error"], worst: values[len(values)-1], max: values[len(values)-1].values["error"]}
	var sum float64
	for _, sample := range values {
		sum += sample.values["error"]
	}
	stats.mean = sum / float64(len(values))
	p95Index := int(math.Ceil(0.95*float64(len(values)))) - 1
	stats.p95 = values[p95Index].values["error"]
	return stats
}

func analyticalScreen(tc truthCase) (float64, float64, float64) {
	eyePosition := vecAdd(tc.camera.Position, vecScale(tc.camera.Right, tc.offset))
	relative := vecSub(tc.world, eyePosition)
	x := dot(relative, tc.camera.Right)
	y := dot(relative, tc.camera.Up)
	z := dot(relative, tc.camera.Forward)
	f := tc.frustum
	eyeDepth := z
	ndcX := (2*f.Near*x/eyeDepth - f.Right - f.Left) / (f.Right - f.Left)
	ndcY := (2*f.Near*y/eyeDepth - f.Top - f.Bottom) / (f.Top - f.Bottom)
	ndcZ := (-(f.Far+f.Near)/(f.Far-f.Near) + 2*f.Far*f.Near/(eyeDepth*(f.Far-f.Near)))
	return (ndcX + 1) * 0.5 * screenWidth, (1 - ndcY) * 0.5 * screenHeight, 1 - (ndcZ+1)*0.5
}

func analyticalEyePoint(tc truthCase, screenX, screenY, depth01 float64) Vec3 {
	ndcX := 2*screenX/screenWidth - 1
	ndcY := 1 - 2*screenY/screenHeight
	ndcZ := 2*depth01 - 1
	f := tc.frustum
	eyeZ := -2 * f.Far * f.Near / (ndcZ*(f.Far+f.Near) + (f.Far - f.Near))
	eyeX := eyeZ * ((f.Right-f.Left)*ndcX + f.Left + f.Right) / (2 * f.Near)
	eyeY := eyeZ * ((f.Top-f.Bottom)*ndcY + f.Bottom + f.Top) / (2 * f.Near)
	return Vec3{X: eyeX, Y: eyeY, Z: eyeZ}
}

func analyticalUnproject(tc truthCase, screenX, screenY, depth01 float64) Vec3 {
	eyePoint := analyticalEyePoint(tc, screenX, screenY, depth01)
	eyePosition := vecAdd(tc.camera.Position, vecScale(tc.camera.Right, tc.offset))
	return vecAdd(eyePosition, vecAdd(
		vecScale(tc.camera.Right, eyePoint.X),
		vecAdd(vecScale(tc.camera.Up, eyePoint.Y), vecScale(tc.camera.Forward, -eyePoint.Z)),
	))
}

func analyticalDisparity(left, right truthCase) float64 {
	leftX, _, _ := analyticalScreen(left)
	rightX, _, _ := analyticalScreen(right)
	return leftX - rightX
}

func analyticalCulled(tc truthCase) bool {
	eyePosition := vecAdd(tc.camera.Position, vecScale(tc.camera.Right, tc.offset))
	relative := vecSub(tc.world, eyePosition)
	x := dot(relative, tc.camera.Right)
	y := dot(relative, tc.camera.Up)
	z := dot(relative, tc.camera.Forward)
	depth := z
	f := tc.frustum
	if depth < f.Near || depth > f.Far {
		return true
	}
	epsilon := 1e-12
	if x < f.Left*depth/f.Near-epsilon || x > f.Right*depth/f.Near+epsilon {
		return true
	}
	return y < f.Bottom*depth/f.Near-epsilon || y > f.Top*depth/f.Near+epsilon
}

func vecAdd(a, b Vec3) Vec3 { return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
func vecSub(a, b Vec3) Vec3 { return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }
func vecScale(a Vec3, s float64) Vec3 {
	return Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}
