package main

import "math"

type Scenario struct {
	ID    string
	Title string
	A     Groom
	B     Groom
}

func rad(deg float64) float64 {
	return deg * math.Pi / 180
}

func buildScenarios() ([]Scenario, error) {
	curly, err := buildGroom("Groom A · Curly 卷发", Pose{
		Yaw:       rad(-11),
		Pitch:     rad(2.5),
		Roll:      rad(1.2),
		Translate: V(-0.018, 0.004, 0),
	}, HairStyle{
		Count:      63,
		Levels:     13,
		Length:     0.58,
		LengthVar:  0.10,
		CurlAmp:    0.092,
		Turns:      2.75,
		Bulge:      0.095,
		TipFlip:    0.035,
		Width:      1.20,
		PhaseShift: 0.18,
	})
	if err != nil {
		return nil, err
	}
	straight, err := buildGroom("Groom B · Straight 直发", Pose{
		Yaw:       rad(9),
		Pitch:     rad(-3.5),
		Roll:      rad(-0.8),
		Translate: V(0.016, -0.003, 0),
	}, HairStyle{
		Count:      87,
		Levels:     13,
		Length:     0.34,
		LengthVar:  0.09,
		CurlAmp:    0.010,
		Turns:      0.65,
		Bulge:      0.052,
		TipFlip:    -0.012,
		Width:      1.04,
		PhaseShift: -0.22,
	})
	if err != nil {
		return nil, err
	}
	long, err := buildGroom("Groom A · Long 长发", Pose{
		Yaw:       rad(8),
		Pitch:     rad(3),
		Roll:      rad(-1),
		Translate: V(0.014, 0.006, 0),
	}, HairStyle{
		Count:      69,
		Levels:     13,
		Length:     0.76,
		LengthVar:  0.08,
		CurlAmp:    0.045,
		Turns:      1.35,
		Bulge:      0.075,
		TipFlip:    0.045,
		Width:      1.13,
		PhaseShift: 0.32,
	})
	if err != nil {
		return nil, err
	}
	short, err := buildGroom("Groom B · Short 短发", Pose{
		Yaw:       rad(-10),
		Pitch:     rad(-2),
		Roll:      rad(1),
		Translate: V(-0.015, -0.004, 0),
	}, HairStyle{
		Count:      99,
		Levels:     13,
		Length:     0.245,
		LengthVar:  0.11,
		CurlAmp:    0.028,
		Turns:      0.72,
		Bulge:      0.063,
		TipFlip:    0.058,
		Width:      1.00,
		PhaseShift: -0.28,
	})
	if err != nil {
		return nil, err
	}
	return []Scenario{
		{ID: "curly_to_straight", Title: "卷发 → 直发", A: curly, B: straight},
		{ID: "long_to_short", Title: "长发 → 短发", A: long, B: short},
	}, nil
}
