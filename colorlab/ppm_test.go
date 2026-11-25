package colorlab

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestWritePPM(t *testing.T) {
	img := NewImage(2, 1)
	img.Set(0, 0, RGB{1, 0.5, 0})
	img.Set(1, 0, RGB{-1, 2, math.NaN()})

	var buf bytes.Buffer
	if err := img.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}
	wantHeader := "P6\n2 1\n255\n"
	if !bytes.HasPrefix(buf.Bytes(), []byte(wantHeader)) {
		t.Fatalf("header = %q, want prefix %q", buf.Bytes(), wantHeader)
	}
	wantPixels := []byte{255, 128, 0, 0, 255, 0}
	if got := buf.Bytes()[len(wantHeader):]; !bytes.Equal(got, wantPixels) {
		t.Fatalf("pixels = %v, want %v", got, wantPixels)
	}
}

func TestWritePPMFile(t *testing.T) {
	img := NewImage(1, 1)
	img.Set(0, 0, RGB{0.25, 0.5, 0.75})
	path := filepath.Join(t.TempDir(), "test.ppm")
	if err := img.WritePPMFile(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte("P6\n1 1\n255\n"), 64, 128, 191)
	if !bytes.Equal(data, want) {
		t.Fatalf("file = %v, want %v", data, want)
	}
}

func TestQuantize8(t *testing.T) {
	tests := []struct {
		in   float64
		want byte
	}{
		{-1, 0},
		{0, 0},
		{0.5, 128},
		{1, 255},
		{2, 255},
		{math.NaN(), 0},
	}
	for _, tt := range tests {
		if got := quantize8(tt.in); got != tt.want {
			t.Errorf("quantize8(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
