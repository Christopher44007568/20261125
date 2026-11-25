package stereoprobe

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func knownCamera() Camera {
	return Camera{
		Position: Vec3{X: 0.125, Y: -0.25, Z: 1.5},
		Right:    Vec3{X: 1, Y: 0, Z: 0},
		Up:       Vec3{X: 0, Y: 1, Z: 0},
		Forward:  Vec3{X: 0, Y: 0, Z: 1},
	}
}

func perturbedCamera() Camera {
	angle := Vec3{
		X: 0.0035,
		Y: -0.0042,
		Z: 0.0021,
	}
	rotation := RotationXYZ(angle)
	camera := NewCamera(Vec3{X: 0.131, Y: -0.246, Z: 1.497}, rotation)
	return camera
}

func eyeFrusta() [2]Frustum {
	return [2]Frustum{
		{Left: -0.17, Right: 0.09, Bottom: -0.11, Top: 0.075, Near: 0.25, Far: 18},
		{Left: -0.10, Right: 0.16, Bottom: -0.095, Top: 0.085, Near: 0.25, Far: 18},
	}
}

func eyeOffsets() [2]float64 {
	return [2]float64{-eyeBaseline / 2, eyeBaseline / 2}
}

func buildSharedCornerCases(camera Camera, group string) []truthCase {
	frusta := eyeFrusta()
	offsets := eyeOffsets()
	zChoices := []float64{0.25, 18}
	var cases []truthCase
	for eye := 0; eye < 2; eye++ {
		eyePosition := vecAdd(camera.Position, vecScale(camera.Right, offsets[eye]))
		xChoices := []float64{frusta[eye].Left, frusta[eye].Right}
		yChoices := []float64{frusta[eye].Bottom, frusta[eye].Top}
		for depthIndex, depth := range zChoices {
			for yIndex, ySlope := range yChoices {
				for xIndex, nearX := range xChoices {
					local := Vec3{
						X: nearX * depth / frusta[eye].Near,
						Y: ySlope * depth / frusta[eye].Near,
						Z: depth,
					}
					world := vecAdd(eyePosition, vecAdd(
						vecScale(camera.Right, local.X),
						vecAdd(vecScale(camera.Up, local.Y), vecScale(camera.Forward, local.Z)),
					))
					name := "shared_" + group
					if depthIndex == 0 {
						name += "_near"
					} else {
						name += "_far"
					}
					name += "_corner_" + string(rune('a'+xIndex)) + string(rune('a'+yIndex))
					cases = append(cases, truthCase{
						name: name, group: group, camera: camera, eye: eye,
						offset: offsets[eye], frustum: frusta[eye], world: world,
					})
				}
			}
		}
	}
	return cases
}

func buildCenterCases(camera Camera, group string) []truthCase {
	frusta := eyeFrusta()
	offsets := eyeOffsets()
	depths := []float64{0.25, 1.25, 8, 18}
	var cases []truthCase
	for eye := 0; eye < 2; eye++ {
		eyePosition := vecAdd(camera.Position, vecScale(camera.Right, offsets[eye]))
		for _, depth := range depths {
			centerX := (frusta[eye].Left + frusta[eye].Right) * 0.5 * depth
			centerY := (frusta[eye].Bottom + frusta[eye].Top) * 0.5 * depth
			local := Vec3{X: centerX, Y: centerY, Z: depth}
			world := vecAdd(eyePosition, vecAdd(
				vecScale(camera.Right, local.X),
				vecAdd(vecScale(camera.Up, local.Y), vecScale(camera.Forward, local.Z)),
			))
			cases = append(cases, truthCase{
				name: group + "_center_depth_" + trimFloat(depth), group: group,
				camera: camera, eye: eye, offset: offsets[eye],
				frustum: frusta[eye], world: world,
			})
		}
	}
	return cases
}

func buildCullingCases(camera Camera) []truthCase {
	frusta := eyeFrusta()
	offsets := eyeOffsets()
	insideDepth := 1.0
	nearPlaneDepth := frusta[0].Near
	farPlaneDepth := frusta[0].Far
	specs := []struct {
		name   string
		eye    int
		local  Vec3
		culled bool
	}{
		{"left_eye_left_boundary", 0, Vec3{X: frusta[0].Left * insideDepth / frusta[0].Near, Y: 0, Z: insideDepth}, false},
		{"right_eye_right_boundary", 1, Vec3{X: frusta[1].Right * insideDepth / frusta[1].Near, Y: 0, Z: insideDepth}, false},
		{"left_eye_only_point_visible", 0, Vec3{X: -0.155, Y: 0, Z: insideDepth}, false},
		{"left_eye_only_point_culled", 1, Vec3{X: -0.11 * insideDepth / frusta[1].Near, Y: 0, Z: insideDepth}, true},
		{"right_eye_only_point_visible", 1, Vec3{X: 0.145, Y: 0, Z: insideDepth}, false},
		{"right_eye_only_point_culled", 0, Vec3{X: 0.10 * insideDepth / frusta[0].Near, Y: 0, Z: insideDepth}, true},
		{"below_both_eyes", 1, Vec3{X: 0, Y: -0.11 * insideDepth / frusta[1].Near, Z: insideDepth}, true},
		{"above_both_eyes", 0, Vec3{X: 0, Y: 0.09 * insideDepth / frusta[0].Near, Z: insideDepth}, true},
		{"near_plane", 0, Vec3{X: 0, Y: 0, Z: nearPlaneDepth}, false},
		{"far_plane", 1, Vec3{X: 0, Y: 0, Z: farPlaneDepth}, false},
		{"closer_than_near", 0, Vec3{X: 0, Y: 0, Z: nearPlaneDepth - 0.001}, true},
		{"farther_than_far", 1, Vec3{X: 0, Y: 0, Z: farPlaneDepth + 0.001}, true},
	}
	var cases []truthCase
	for _, spec := range specs {
		eyePosition := vecAdd(camera.Position, vecScale(camera.Right, offsets[spec.eye]))
		world := vecAdd(eyePosition, vecAdd(
			vecScale(camera.Right, spec.local.X),
			vecAdd(vecScale(camera.Up, spec.local.Y), vecScale(camera.Forward, spec.local.Z)),
		))
		cases = append(cases, truthCase{
			name: "culling_" + spec.name, group: "single_eye_culling", camera: camera,
			eye: spec.eye, offset: offsets[spec.eye], frustum: frusta[spec.eye],
			world: world, expectCulled: spec.culled,
		})
	}
	return cases
}

func TestAnalyticalProjectionTruth(t *testing.T) {
	collector := newErrorCollector()
	allCases := append(buildSharedCornerCases(knownCamera(), "known_pose"), buildCenterCases(knownCamera(), "known_pose")...)
	for _, tc := range allCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProjectToScreen(tc.camera, tc.offset, tc.frustum, tc.world, screenWidth, screenHeight)
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			if got.Culled {
				t.Fatalf("projection case was unexpectedly culled")
			}
			wantX, wantY, wantDepth := analyticalScreen(tc)
			collector.add(t, tc, "screen_x_px", got.X, wantX, 2e-9)
			collector.add(t, tc, "screen_y_px", got.Y, wantY, 2e-9)
			collector.add(t, tc, "depth_01", got.Depth01, wantDepth, 2e-12)

			unprojected, err := UnprojectEye(tc.frustum, got.X, got.Y, got.Depth01, screenWidth, screenHeight)
			if err != nil {
				t.Fatalf("unproject eye: %v", err)
			}
			wantEye := analyticalEyePoint(tc, wantX, wantY, wantDepth)
			collector.add(t, tc, "unproject_eye_x_m", unprojected.X, wantEye.X, 2e-12)
			collector.add(t, tc, "unproject_eye_y_m", unprojected.Y, wantEye.Y, 2e-12)
			collector.add(t, tc, "unproject_eye_z_m", unprojected.Z, wantEye.Z, 2e-12)

			world, err := EyeToWorld(tc.camera, tc.offset, unprojected)
			if err != nil {
				t.Fatalf("eye to world: %v", err)
			}
			wantWorld := analyticalUnproject(tc, wantX, wantY, wantDepth)
			collector.add(t, tc, "unproject_world_x_m", world.X, wantWorld.X, 2e-12)
			collector.add(t, tc, "unproject_world_y_m", world.Y, wantWorld.Y, 2e-12)
			collector.add(t, tc, "unproject_world_z_m", world.Z, wantWorld.Z, 2e-12)
		})
	}
	collector.report(t)
}

func TestSlightPoseError(t *testing.T) {
	collector := newErrorCollector()
	nominal := knownCamera()
	perturbed := perturbedCamera()
	allCases := append(buildSharedCornerCases(perturbed, "pose_error"), buildCenterCases(perturbed, "pose_error")...)
	for _, tc := range allCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProjectToScreen(tc.camera, tc.offset, tc.frustum, tc.world, screenWidth, screenHeight)
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			if got.Culled {
				return
			}
			wantX, wantY, wantDepth := analyticalScreen(tc)
			collector.add(t, tc, "pose_screen_x_px", got.X, wantX, 3e-9)
			collector.add(t, tc, "pose_screen_y_px", got.Y, wantY, 3e-9)
			collector.add(t, tc, "pose_depth_01", got.Depth01, wantDepth, 2e-12)

			nominalCase := tc
			nominalCase.camera = nominal
			nominalX, nominalY, _ := analyticalScreen(nominalCase)
			shiftX := wantX - nominalX
			shiftY := wantY - nominalY
			if math.Abs(shiftX) < 1e-8 || math.Abs(shiftY) < 1e-8 {
				t.Fatalf("pose perturbation did not move screen point: shift=(%.12f, %.12f)", shiftX, shiftY)
			}
		})
	}
	collector.report(t)
}

func TestSingleEyeCulling(t *testing.T) {
	collector := newErrorCollector()
	for _, tc := range buildCullingCases(knownCamera()) {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ProjectToScreen(tc.camera, tc.offset, tc.frustum, tc.world, screenWidth, screenHeight)
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			collector.add(t, tc, "culling_bool", boolToFloat(got.Culled), boolToFloat(tc.expectCulled), 0)
			if got.Culled != tc.expectCulled {
				t.Fatalf("culled = %v, want %v", got.Culled, tc.expectCulled)
			}
			if got.Culled != analyticalCulled(tc) {
				t.Fatalf("culled = %v disagrees with analytical frustum test", got.Culled)
			}
		})
	}
	collector.report(t)
}

func TestTheoreticalDisparity(t *testing.T) {
	collector := newErrorCollector()
	camera := knownCamera()
	frusta := eyeFrusta()
	offsets := eyeOffsets()
	depths := []float64{0.25, 0.75, 2.5, 8, 18}
	xSlopes := []float64{-0.06, 0.05}
	ySlopes := []float64{-0.06, 0.04}
	for _, depth := range depths {
		for _, xSlope := range xSlopes {
			for _, ySlope := range ySlopes {
				world := vecAdd(camera.Position, Vec3{
					X: xSlope * depth / frusta[0].Near,
					Y: ySlope * depth / frusta[0].Near,
					Z: depth,
				})
				left := truthCase{name: "disparity_left", group: "theoretical_disparity", camera: camera, eye: 0, offset: offsets[0], frustum: frusta[0], world: world}
				right := truthCase{name: "disparity_right", group: "theoretical_disparity", camera: camera, eye: 1, offset: offsets[1], frustum: frusta[1], world: world}
				leftScreen, err := ProjectToScreen(camera, offsets[0], frusta[0], world, screenWidth, screenHeight)
				if err != nil || leftScreen.Culled {
					t.Fatalf("left point not visible: %+v err=%v", leftScreen, err)
				}
				rightScreen, err := ProjectToScreen(camera, offsets[1], frusta[1], world, screenWidth, screenHeight)
				if err != nil || rightScreen.Culled {
					t.Fatalf("right point not visible: %+v err=%v", rightScreen, err)
				}
				tc := truthCase{name: "disparity_depth_" + trimFloat(depth), group: "theoretical_disparity", camera: camera, eye: 2}
				collector.add(t, tc, "theoretical_horizontal_disparity_px", leftScreen.X-rightScreen.X, analyticalDisparity(left, right), 3e-9)
			}
		}
	}
	collector.report(t)
}

func trimFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", value), "0"), ".")
}

func boolToFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
