package sim

import "math"

const (
	fps           = 60
	eyeSeparation = 0.30
)

var sceneLight = vec3{x: -0.44, y: 0.58, z: 0.69}.unit()

func BuildTimeline(seconds float64) Timeline {
	frameCount := int(math.Round(seconds*fps)) + 1
	eyes := []EyeTimeline{
		buildEye(0, "L", -eyeSeparation, frameCount),
		buildEye(1, "R", eyeSeparation, frameCount),
	}

	audit := auditTimeline(eyes)
	return Timeline{
		Version:    "highlight-track-v1",
		Width:      int(sceneWidth),
		Height:     int(sceneHeight),
		FPS:        fps,
		Duration:   seconds,
		FrameCount: frameCount,
		Light:      Vec2{X: sceneLight.x, Y: sceneLight.y},
		Eyes:       eyes,
		Audit:      audit,
		Scenes:     sceneSummaries(),
	}
}

func buildEye(id int, label string, offset float64, frameCount int) EyeTimeline {
	frames := make([]Frame, 0, frameCount)
	previous := make(map[trackKey]trackRecord)

	for index := 0; index < frameCount; index++ {
		t := float64(index) / fps
		objects, tracks := sampleScene(t, offset)
		for trackIndex := range tracks {
			track := &tracks[trackIndex]
			key := trackKey{eye: id, object: track.ObjectID}
			prior, ok := previous[key]
			track.History = resampleHistory(*track, prior, ok, id)
			previous[key] = recordTrack(*track, id)
		}
		frames = append(frames, Frame{
			Index:   index,
			Time:    t,
			Objects: objects,
			Tracks:  tracks,
		})
	}

	return EyeTimeline{ID: id, Label: label, Offset: offset, Frames: frames}
}

func sampleScene(t, eyeOffset float64) ([]ObjectState, []HighlightTrack) {
	metal, metalTrack := sampleMetal(t, eyeOffset)
	sphere, sphereTrack := sampleSphere(t, eyeOffset)
	glass, glassTrack := sampleGlass(t, eyeOffset)

	return []ObjectState{metal, sphere, glass},
		[]HighlightTrack{metalTrack, sphereTrack, glassTrack}
}

func sampleMetal(t, eyeOffset float64) (ObjectState, HighlightTrack) {
	angle := 0.58 + 1.08*t + 0.08*math.Sin(2.1*t)
	center := Vec2{
		X: sceneWidth*0.5 - eyeOffset*10,
		Y: sceneHeight*0.5 + 3,
	}
	orbit := rotate(Vec2{X: 0.34, Y: 0.17}, angle*0.72)
	highlight := add(center, Vec2{
		X: orbit.X*pixelsPerMeter + eyeOffset*7,
		Y: orbit.Y * pixelsPerMeter,
	})
	stretch := 58 + 36*(0.5+0.5*math.Sin(angle*1.7+0.4))
	thickness := 17 + 8*(0.5+0.5*math.Cos(angle*1.15))
	peak := 0.79 + 0.16*(0.5+0.5*math.Sin(angle*0.92+1.1))
	direction := angle*0.82 + 0.3 + eyeOffset*0.12

	object := ObjectState{
		ID: "rough-metal", Kind: "rough-metal", Name: "ROUGH METAL",
		Center: center, Radius: 102, Width: 218, Height: 218, Angle: angle,
	}
	track := HighlightTrack{
		ObjectID: object.ID, Center: highlight, Stretch: stretch,
		Thickness: thickness, Direction: direction, Peak: peak,
		Energy:     gaussianEnergy(peak, stretch, thickness),
		Reflection: reflection2(center, highlight),
	}
	return object, track
}

func sampleSphere(t, eyeOffset float64) (ObjectState, HighlightTrack) {
	centerWorld := vec3{
		x: 0.94 * math.Sin(0.94*t+0.35),
		y: 0.10 * math.Cos(1.28*t),
		z: 0,
	}
	radiusWorld := 0.86
	eye := vec3{x: eyeOffset, y: 0, z: 6.4}
	normal := specularNormal(sceneLight, eye, centerWorld)
	highlightWorld := centerWorld.add(normal.scale(radiusWorld))

	center := screenPoint(centerWorld.x, centerWorld.y, eyeOffset*0.32)
	highlight := screenPoint(highlightWorld.x, highlightWorld.y, eyeOffset*0.32)
	direction := math.Atan2(normal.y, normal.x) + math.Pi*0.5
	stretch := 14 + 5*(0.5+0.5*math.Sin(1.8*t+0.6))
	thickness := 7.5 + 2.2*(0.5+0.5*math.Cos(1.2*t))
	peak := 0.92 + 0.06*math.Sin(0.83*t+0.2)

	object := ObjectState{
		ID: "mirror-sphere", Kind: "mirror-sphere", Name: "MIRROR SPHERE",
		Center: center, Radius: radiusWorld * pixelsPerMeter,
		Width: 2 * radiusWorld * pixelsPerMeter, Height: 2 * radiusWorld * pixelsPerMeter,
		Angle: 0,
	}
	track := HighlightTrack{
		ObjectID: object.ID, Center: highlight, Stretch: stretch,
		Thickness: thickness, Direction: direction, Peak: peak,
		Energy:     gaussianEnergy(peak, stretch, thickness),
		Reflection: reflection2(center, highlight),
	}
	return object, track
}

func sampleGlass(t, eyeOffset float64) (ObjectState, HighlightTrack) {
	centerWorld := vec2motion(t)
	countershift := eyeOffset * 0.28
	center := screenPoint(centerWorld.X, centerWorld.Y, countershift)
	angle := 0.17 * math.Sin(0.63*t-0.2)

	localOffset := rotate(Vec2{X: 0.20, Y: 0.46}, angle*1.35)
	highlight := add(center, Vec2{
		X: localOffset.X*pixelsPerMeter + eyeOffset*5,
		Y: -localOffset.Y * pixelsPerMeter,
	})
	stretch := 38 + 27*(0.5+0.5*math.Sin(1.62*t+angle*2.2))
	thickness := 13 + 5*(0.5+0.5*math.Cos(1.17*t+0.5))
	peak := 0.63 + 0.15*(0.5+0.5*math.Cos(0.88*t+0.8))
	direction := -0.48 + 0.34*math.Sin(0.92*t+angle)

	object := ObjectState{
		ID: "glass", Kind: "glass", Name: "GLASS",
		Center: center, Radius: 74, Width: 150, Height: 196, Angle: angle,
	}
	track := HighlightTrack{
		ObjectID: object.ID, Center: highlight, Stretch: stretch,
		Thickness: thickness, Direction: direction, Peak: peak,
		Energy:     gaussianEnergy(peak, stretch, thickness),
		Reflection: reflection2(center, highlight),
	}
	return object, track
}

func vec2motion(t float64) Vec2 {
	return Vec2{
		X: 0.57 * math.Sin(0.72*t+0.2),
		Y: 0.20 * math.Sin(1.04*t-0.35),
	}
}

func sceneSummaries() []SceneSummary {
	return []SceneSummary{
		{
			ID: "rough-metal", Name: "ROUGH METAL", Motion: "ROTATION",
			Material: "anisotropic brushed alloy", Accent: "#ffd37a",
			Description: "Rotation stretches one broad lobe while the peak trades height for area.",
		},
		{
			ID: "mirror-sphere", Name: "MIRROR SPHERE", Motion: "TRANSLATION",
			Material: "polished chrome", Accent: "#b9e8ff",
			Description: "A translating sphere changes the half-vector without moving its object transform.",
		},
		{
			ID: "glass", Name: "GLASS", Motion: "OBJECT MOTION",
			Material: "thick dielectric", Accent: "#b8ffdd",
			Description: "A sliding pane bends a narrow transmitted lobe across its rim.",
		},
	}
}
