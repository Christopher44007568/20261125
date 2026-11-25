package colorlab

import (
	"math"
	"testing"
)

func TestSRGBToRec2020Matrix(t *testing.T) {
	want := Matrix3{
		{0.6274038959346986, 0.32928303837788364, 0.04331306568741721},
		{0.06909728935823205, 0.9195403950754591, 0.011362315566309174},
		{0.016391438875150252, 0.08801330787722576, 0.8955952532476237},
	}
	got := SRGBToRec2020Matrix()
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if math.Abs(got[row][col]-want[row][col]) > 1e-12 {
				t.Errorf("matrix[%d][%d] = %.15g, want %.15g",
					row, col, got[row][col], want[row][col])
			}
		}
	}
}

func TestMatrixInverseIdentity(t *testing.T) {
	product := SRGBToRec2020Matrix().Mul(Rec2020ToSRGBMatrix())
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			want := 0.0
			if row == col {
				want = 1.0
			}
			if math.Abs(product[row][col]-want) > 1e-12 {
				t.Errorf("product[%d][%d] = %.15g, want %.15g",
					row, col, product[row][col], want)
			}
		}
	}
}

func TestPrimaryConversionKnownValue(t *testing.T) {
	got := Convert(RGB{1, 0, 0}, SRGB, Rec2020)
	want := RGB{
		linearToRec2020(0.6274038959346986),
		linearToRec2020(0.06909728935823205),
		linearToRec2020(0.016391438875150252),
	}
	for _, delta := range channelDeltas(got, want) {
		if delta > 1e-12 {
			t.Fatalf("sRGB red -> Rec.2020 = %+v, want %+v", got, want)
		}
	}
}
