package main

import "math"

type SpecularIntegral struct {
	BareEnergy          float64
	MeanOutTransmission float64
}

func viewFromDegrees(degrees float64) Vec3 {
	radians := degrees * math.Pi / 180
	return normalize(Vec3{X: math.Sin(radians), Y: 0, Z: math.Cos(radians)})
}

func integrateSpecular(view Vec3, baseF0, roughness, coatF0 float64) SpecularIntegral {
	normal := Vec3{Z: 1}
	alpha := ggxAlpha(roughness)

	bareEnergy := 0.0
	transmissionWeight := 0.0

	for index := 0; index < sampleCount; index++ {
		light, pdf := ggxSample(view, alpha, index)
		if pdf <= 0 || light.Z <= 0 {
			continue
		}
		weight := ggxBRDF(view, light, normal, baseF0, alpha) * light.Z / pdf
		if math.IsNaN(weight) || math.IsInf(weight, 0) {
			continue
		}
		outTransmission := 1 - schlick(light.Z, coatF0)
		bareEnergy += weight
		transmissionWeight += weight * outTransmission
	}

	bareEnergy /= sampleCount
	meanOutTransmission := cosineMeanTransmission(coatF0)
	if bareEnergy > 1e-6 {
		meanOutTransmission = transmissionWeight / sampleCount / bareEnergy
	}

	return SpecularIntegral{
		BareEnergy:          bareEnergy,
		MeanOutTransmission: meanOutTransmission,
	}
}

func cosineMeanTransmission(coatF0 float64) float64 {
	total := 0.0
	for index := 0; index < sampleCount; index++ {
		light := cosineHemisphereSample(index)
		total += 1 - schlick(light.Z, coatF0)
	}
	return total / sampleCount
}

func evaluateMaterial(m Material, viewDegrees float64, blackBase bool, lights int, normalizedSeparately bool) Observation {
	view := viewFromDegrees(viewDegrees)
	normal := Vec3{Z: 1}

	baseF0 := m.BaseF0
	diffuseAlbedo := m.DiffuseAlbedo
	if blackBase {
		// A black base is a perfect absorber under the coating.
		baseF0 = 0
		diffuseAlbedo = 0
	}

	inTransmission := 1 - schlick(dot(view, normal), m.CoatF0)
	specular := integrateSpecular(view, baseF0, m.BaseRoughness, m.CoatF0)
	diffuseOutTransmission := cosineMeanTransmission(m.CoatF0)
	specularOutTransmission := 0.0
	if baseF0 > 0 {
		specularOutTransmission = specular.MeanOutTransmission
	}

	diffuseEnergy := 0.0
	specularEnergy := 0.0
	var causes []string

	if normalizedSeparately {
		if baseF0 > 0 {
			// This normalizes the base lobe before sharing the transmitted budget.
			specularEnergy = inTransmission * specular.MeanOutTransmission
		}
		diffuseEnergy = inTransmission * diffuseAlbedo * diffuseOutTransmission
		if diffuseAlbedo > 0 && specular.BareEnergy > 0 {
			causes = append(causes, "diffuse keeps full albedo instead of the 1-E_spec residual")
		}
		if baseF0 > 0 && specular.BareEnergy > 0 {
			causes = append(causes, "specular lobe is rescaled to unit directional albedo")
		}
	} else {
		if baseF0 > 0 {
			specularEnergy = inTransmission * specular.BareEnergy * specular.MeanOutTransmission
		}
		diffuseEnergy = inTransmission * diffuseAlbedo * diffuseOutTransmission *
			(1 - specular.BareEnergy)
	}

	coatEnergy := schlick(dot(view, normal), m.CoatF0)
	singlePixel := coatEnergy + specularEnergy + diffuseEnergy
	totalEnergy := singlePixel * float64(lights)
	upperBound := float64(lights)
	excess := math.Max(0, totalEnergy-upperBound)
	if excess > 1e-3 && len(causes) == 0 {
		causes = append(causes, "numerical or model error: no lobe-budget violation identified")
	}

	return Observation{
		Mode:             modeName(normalizedSeparately),
		Material:         m.Name,
		ViewDegrees:      viewDegrees,
		BlackBase:        blackBase,
		Lights:           lights,
		BaseF0:           baseF0,
		CoatF0:           m.CoatF0,
		InTransmission:   inTransmission,
		OutTransmission:  specularOutTransmission,
		MeanTransmission: diffuseOutTransmission,
		DiffuseEnergy:    diffuseEnergy * float64(lights),
		SpecularEnergy:   specularEnergy * float64(lights),
		CoatEnergy:       coatEnergy * float64(lights),
		TotalEnergy:      totalEnergy,
		InputUpperBound:  upperBound,
		Excess:           excess,
		Causes:           causes,
	}
}
