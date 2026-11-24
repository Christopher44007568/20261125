package glass

import (
	"math"
	"strings"
	"testing"
)

func TestValidationRejectsIncompleteOrAmbiguousInputs(t *testing.T) {
	base := func() Scene {
		return Scene{
			Kind:      SceneNearCoplanar,
			Materials: []Material{{ID: "mat", Color: RGB{R: 0.5}}},
			Layers: []Layer{{
				ID:         "layer",
				MaterialID: "mat",
				Depth:      map[Eye]float64{EyeLeft: 1, EyeRight: 1},
				Opacity:    0.5,
			}},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Scene)
		budget  Budget
		options Options
		want    string
	}{
		{
			name: "no layers",
			mutate: func(scene *Scene) {
				scene.Layers = nil
			},
			budget: Budget{},
			want:   "no layers",
		},
		{
			name: "duplicate layer",
			mutate: func(scene *Scene) {
				scene.Layers = append(scene.Layers, scene.Layers[0])
			},
			budget: Budget{},
			want:   "duplicate layer",
		},
		{
			name: "missing material",
			mutate: func(scene *Scene) {
				scene.Layers[0].MaterialID = "missing"
			},
			budget: Budget{},
			want:   "unknown material",
		},
		{
			name: "missing eye depth",
			mutate: func(scene *Scene) {
				delete(scene.Layers[0].Depth, EyeRight)
			},
			budget: Budget{},
			want:   "no right-eye depth",
		},
		{
			name: "invalid opacity",
			mutate: func(scene *Scene) {
				scene.Layers[0].Opacity = 1.1
			},
			budget: Budget{},
			want:   "opacity",
		},
		{
			name: "negative depth",
			mutate: func(scene *Scene) {
				scene.Layers[0].Depth[EyeLeft] = -1
			},
			budget: Budget{},
			want:   "invalid left-eye depth",
		},
		{
			name: "nan depth",
			mutate: func(scene *Scene) {
				scene.Layers[0].Depth[EyeLeft] = math.NaN()
			},
			budget: Budget{},
			want:   "invalid left-eye depth",
		},
		{
			name: "negative left budget",
			mutate: func(scene *Scene) {
			},
			budget: Budget{Left: -1},
			want:   "left-eye budget",
		},
		{
			name: "invalid epsilon",
			mutate: func(scene *Scene) {
			},
			budget:  Budget{},
			options: Options{DepthEpsilon: -1},
			want:    "depth epsilon",
		},
		{
			name: "invalid background",
			mutate: func(scene *Scene) {
			},
			budget:  Budget{},
			options: Options{Background: PremultipliedRGBA{A: 2}},
			want:    "background alpha",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene := base()
			test.mutate(&scene)
			_, err := AuditScene(scene, test.budget, test.options)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want substring %q", err, test.want)
			}
		})
	}
}

func TestDefaultDepthEpsilonIsReported(t *testing.T) {
	scene := nearCoplanarScene()
	audit, err := AuditScene(scene, DefaultBudget(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if audit.DepthEpsilon != DefaultDepthEpsilon {
		t.Fatalf("depth epsilon = %g, want %g", audit.DepthEpsilon, DefaultDepthEpsilon)
	}
}
