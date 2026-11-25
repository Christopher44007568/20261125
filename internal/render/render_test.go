package render

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPPMHeader(t *testing.T) {
	c := NewCanvas(7, 5, RGB{R: 12, G: 34, B: 56})
	path := filepath.Join(t.TempDir(), "test.ppm")
	if err := c.WritePPM(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := []byte("P6\n7 5\n255\n")
	if !bytes.HasPrefix(data, header) {
		t.Fatalf("bad PPM header: %q", data[:min(len(data), len(header))])
	}
	if len(data) != len(header)+7*5*3 {
		t.Fatalf("bad PPM payload length: %d", len(data))
	}
}
