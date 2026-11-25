package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func downsample(src *Canvas, w, h int) *Canvas {
	out := NewCanvas(w, h)
	out.Downsample(src)
	return out
}

func frontCamera(aspect float64) Camera {
	return NewCamera(V(0.28, 0.30, 5.15), V(0, 0.20, 0), 38, aspect)
}

func renderAll(w, h int, outDir string, quick bool) error {
	scene := buildScene()
	cam := frontCamera(float64(w) / float64(h))
	ss := 2
	if quick {
		ss = 1
	}
	rw, rh := w*ss, h*ss

	fmt.Println("rendering shell...")
	shell := renderShell(&scene, cam, rw, rh)
	fmt.Println("rendering strand...")
	strand := renderStrand(&scene, cam, rw, rh, float64(ss))
	gs, gt := calibrateFurPair(shell, strand)
	fmt.Printf("fur calibration: shell x%.3f, strand x%.3f\n", gs, gt)
	fmt.Println("compositing hybrid...")
	mixed := compositeHybrid(shell, strand)

	modes := []struct {
		name string
		sub  string
		img  *Canvas
	}{
		{"shell", "FAR LOD / 64 LAYERS / DARK ROOTS", shell},
		{"strand", "NEAR LOD / BRIGHT TIPS", strand},
		{"hybrid", "SINGLE-LAYER DEPTH BLEND", mixed},
	}
	for _, m := range modes {
		lo := downsample(m.img, w, h)
		beauty := addBackground(lo)
		drawFace(beauty, cam, &scene)
		applyBloom(beauty)
		applyFinish(beauty)
		drawCaption(beauty, strings.ToUpper(m.name)+" / "+m.sub, "")
		path := filepath.Join(outDir, m.name+".png")
		if err := savePNG(path, beauty); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	return nil
}

func main() {
	w := flag.Int("w", 1200, "output width")
	h := flag.Int("h", 800, "output height")
	outDir := flag.String("out", "artifacts", "output directory")
	quick := flag.Bool("quick", false, "disable supersampling")
	flag.Parse()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := renderAll(*w, *h, *outDir, *quick); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := renderSideDiff(filepath.Join(*outDir, "side-profile-diff.png")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", filepath.Join(*outDir, "side-profile-diff.png"))
}
