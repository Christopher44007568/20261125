package decallab

import "math"

type Light struct {
	ID        string  `json:"id"`
	Direction Vec3    `json:"direction_toward_light"`
	Color     RGB     `json:"color"`
	Intensity float64 `json:"intensity"`
	Shininess float64 `json:"shininess"`
}

type LightingRig struct {
	Mode          string `json:"mode"`
	Shared        Light  `json:"shared"`
	LeftOverride  Light  `json:"left_override"`
	RightOverride Light  `json:"right_override"`
}

func (l LightingRig) ForEye(eye Eye) Light {
	if l.Mode == "eye_specific" {
		if eye == EyeLeft {
			return l.LeftOverride
		}
		return l.RightOverride
	}
	return l.Shared
}

type LightingSample struct {
	Eye            Eye     `json:"eye"`
	LightID        string  `json:"light_id"`
	LightDirection Vec3    `json:"light_direction"`
	ViewDirection  Vec3    `json:"view_direction"`
	Diffuse        float64 `json:"diffuse"`
	Specular       float64 `json:"specular"`
	Radiance       RGB     `json:"radiance"`
	Alpha          float64 `json:"alpha"`
}

func ShadeSurface(normal, view Vec3, material Material, light Light, alpha float64) LightingSample {
	n := normal.Normalize()
	v := view.Normalize()
	l := light.Direction.Normalize()
	ndl := math.Max(0, n.Dot(l))
	half := l.Add(v).Normalize()
	ndh := math.Max(0, n.Dot(half))
	spec := math.Pow(ndh, math.Max(1, light.Shininess)) * light.Intensity
	diffuse := ndl * light.Intensity
	ambient := 0.16
	base := material.Albedo.Scale(ambient + diffuse)
	lit := RGB{
		R: base.R*light.Color.R + spec*light.Color.R,
		G: base.G*light.Color.G + spec*light.Color.G,
		B: base.B*light.Color.B + spec*light.Color.B,
	}
	return LightingSample{
		Eye:            "",
		LightID:        light.ID,
		LightDirection: l,
		ViewDirection:  v,
		Diffuse:        diffuse,
		Specular:       spec,
		Radiance:       lit,
		Alpha:          alpha,
	}
}

type EyeLightingDifference struct {
	LightDirectionAngleDeg float64 `json:"light_direction_angle_deg"`
	DiffuseDifference      float64 `json:"diffuse_difference"`
	SpecularDifference     float64 `json:"specular_difference"`
	RadianceDistance       float64 `json:"radiance_distance"`
	MaxChannelDifference   float64 `json:"max_channel_difference"`
}

func CompareLighting(left, right LightingSample) EyeLightingDifference {
	dr := left.Radiance.R - right.Radiance.R
	dg := left.Radiance.G - right.Radiance.G
	db := left.Radiance.B - right.Radiance.B
	return EyeLightingDifference{
		LightDirectionAngleDeg: angleBetweenDeg(left.LightDirection, right.LightDirection),
		DiffuseDifference:      math.Abs(left.Diffuse - right.Diffuse),
		SpecularDifference:     math.Abs(left.Specular - right.Specular),
		RadianceDistance:       math.Sqrt(dr*dr + dg*dg + db*db),
		MaxChannelDifference:   math.Max(math.Abs(dr), math.Max(math.Abs(dg), math.Abs(db))),
	}
}
