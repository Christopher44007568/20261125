package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG, OBJ, and JSON outputs")
	renderOBJ := flag.String("render-obj", "", "parse an OBJ and render its top view, then exit")
	pngOut := flag.String("png", "", "PNG path used with -render-obj")
	flag.Parse()

	if *renderOBJ != "" {
		if err := renderOBJFile(*renderOBJ, *pngOut); err != nil {
			fmt.Fprintln(os.Stderr, "render OBJ failed:", err)
			os.Exit(1)
		}
		return
	}

	report, err := Run(*outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pipeline failed:", err)
		os.Exit(1)
	}

	allValid := true
	for _, c := range report.Cases {
		fmt.Printf("%-18s loops=%d tris=%d bad_trim=%d | outer_only_tris=%d bad_outer=%d | obj F=%d boundary=%d\n",
			c.ID, c.ExtractedLoops, c.CorrectTriangles, c.CorrectInvalid,
			c.NaiveTriangles, c.NaiveInvalid, c.OBJFaces, c.OBJBoundaryEdges)
		if c.CorrectInvalid != 0 || c.OBJReloadInvalid != 0 || c.NaiveInvalid == 0 || c.OBJFaces != c.CorrectTriangles {
			allValid = false
		}
	}
	fmt.Println("wrote OBJ, PNG overlays, OBJ re-renders, and", filepath.Join(*outDir, "trim_report.json"))
	if !allValid {
		fmt.Fprintln(os.Stderr, "verification mismatch: inspect trim_report.json")
		os.Exit(1)
	}
}

func renderOBJFile(objPath, pngPath string) error {
	obj, err := ReadOBJ(objPath)
	if err != nil {
		return err
	}
	if pngPath == "" {
		ext := filepath.Ext(objPath)
		pngPath = strings.TrimSuffix(objPath, ext) + "_reload.png"
	}
	img := RenderOBJReloadPNG(obj, "OBJ RELOAD", "STANDALONE PARSE AND TOP-VIEW RENDER")
	if err := os.MkdirAll(filepath.Dir(pngPath), 0o755); err != nil && filepath.Dir(pngPath) != "." {
		return err
	}
	if err := WritePNG(pngPath, img); err != nil {
		return err
	}
	fmt.Printf("read %d vertices and %d faces from %s\nwrote %s\n", len(obj.Vertices), len(obj.Triangles), objPath, pngPath)
	return nil
}
