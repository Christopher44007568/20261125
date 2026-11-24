package main

import "testing"

func TestFresnelBoundaries(t *testing.T) {
	if got := schlick(1, 0.04); got != 0.04 {
		t.Fatalf("normal fresnel = %v, want 0.04", got)
	}
	if got := schlick(0, 0.04); got != 1 {
		t.Fatalf("grazing fresnel = %v, want 1", got)
	}
}

func TestSharedBudgetRespectsAllLightBounds(t *testing.T) {
	for _, material := range buildMaterials() {
		for _, view := range []float64{0, 60, 89} {
			for _, black := range []bool{false, true} {
				for _, lights := range []int{1, 3} {
					row := evaluateMaterial(material, view, black, lights, false)
					if row.TotalEnergy > row.InputUpperBound+0.02 {
						t.Fatalf("%s view %.0f black %v lights %d = %.4f > %.4f",
							material.Name, view, black, lights,
							row.TotalEnergy, row.InputUpperBound)
					}
				}
			}
		}
	}
}

func TestSeparateNormalizationViolatesDielectricBound(t *testing.T) {
	material := buildMaterials()[1]
	row := evaluateMaterial(material, 0, false, 1, true)
	if row.Excess <= 0.1 {
		t.Fatalf("expected substantial overshoot, got excess %.4f", row.Excess)
	}
	if len(row.Causes) != 2 {
		t.Fatalf("expected diffuse and specular causes, got %v", row.Causes)
	}
}

func TestBlackBaseOnlyReflectsAtCoating(t *testing.T) {
	material := buildMaterials()[1]
	row := evaluateMaterial(material, 60, true, 3, false)
	if row.BaseF0 != 0 || row.DiffuseEnergy != 0 || row.SpecularEnergy != 0 {
		t.Fatalf("black base leaked sub-coat energy: %+v", row)
	}
	if row.TotalEnergy > row.InputUpperBound {
		t.Fatalf("black coating total %.4f exceeds bound %.4f",
			row.TotalEnergy, row.InputUpperBound)
	}
}
