package main

import (
	"fmt"
	"math"
	"os"
)

func vecSum(c Vec3) float64 { return c.X + c.Y + c.Z }

func writeReport(path string, r *EnergyReport, gstats []GrazingStats, fibers, hairPx int) {
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format+"\n", args...)
	}

	w("Hair highlight energy comparison")
	w("================================")
	w("All values are summed LINEAR radiance over the shared hair-pixel mask.")
	w("fibers=%d hairPixels=%d (same geometry, light and camera for all models)",
		fibers, hairPx)
	w("")

	w("Total reflected energy (RGB channel sums)")
	w("-----------------------------------------")
	for _, name := range []string{"single", "kajiya_kay", "double"} {
		c := r.Total[name]
		w("%-11s R=%10.2f G=%10.2f B=%10.2f  sum=%10.2f  mean/pixel=%.5f",
			name, c.X, c.Y, c.Z, vecSum(c), vecSum(c)/float64(hairPx))
	}
	w("")

	w("Highlight-only energy (luminance of pure specular terms, diffuse removed)")
	w("-------------------------------------------------------------------------")
	specBase := r.SpecEnergy["single"]
	for _, name := range []string{"single", "kajiya_kay", "double"} {
		d := r.SpecEnergy[name] - specBase
		pct := 100 * d / specBase
		w("%-11s specSum=%10.2f  dSpec=%+10.2f (%+.2f%%)",
			name, r.SpecEnergy[name], d, pct)
	}
	w("")

	base := vecSum(r.Total["single"])
	w("Energy difference relative to single-scatter baseline")
	w("-----------------------------------------------------")
	for _, name := range []string{"single", "kajiya_kay", "double"} {
		c := r.Total[name]
		rel := r.Relative[name]
		d := vecSum(c) - base
		pct := 100 * d / base
		w("%-11s dSum=%+10.2f  (%+.2f%%)  channel ratios R=%.4f G=%.4f B=%.4f",
			name, d, pct, rel.X, rel.Y, rel.Z)
	}
	w("")

	w("Pairwise absolute energy difference (L1 over hair pixels)")
	w("---------------------------------------------------------")
	for _, key := range []string{"single-kajiya_kay", "single-double", "kajiya_kay-double"} {
		w("%-20s L1=%10.2f  mean/pixel=%.5f",
			key, r.PairwiseL1[key], r.PairwiseL1[key]/float64(hairPx))
	}
	w("")

	w("Highlight centroid (world x,y on scalp) and screen pixel position")
	w("-----------------------------------------------------------------")
	for _, name := range []string{"single", "kajiya_kay", "double"} {
		c := r.Centroid[name]
		px := r.CentroidPx[name]
		w("%-11s world=(%+.4f,%+.4f) screen=(%.1f,%.1f)",
			name, c[0], c[1], px[0], px[1])
	}
	w("")

	w("Highlight centroid displacement (pixels)")
	w("----------------------------------------")
	ref := r.CentroidPx["single"]
	for _, name := range []string{"kajiya_kay", "double"} {
		p := r.CentroidPx[name]
		dx, dy := p[0]-ref[0], p[1]-ref[1]
		w("single -> %-11s dx=%+.2f dy=%+.2f |d|=%.2f px",
			name, dx, dy, math.Hypot(dx, dy))
	}
	w("")

	w("Grazing-angle diagnostic (fiber axis = x; sigmaX/sigmaY > 1 means")
	w("highlightTravel = std-dev of each fiber's highlight centroid along the")
	w("fiber axis. Hair models travel with the tangent wave; the isotropic")
	w("plastic control stays locked at one screen band, so travel ~ 0.)")
	w("------------------------------------------------------------------")
	for _, st := range gstats {
		w("%-11s centroid=(%.1f,%.1f) highlightTravelX=%.2f bandSigmaY=%.2f",
			st.Model, st.CentroidX, st.CentroidY, st.SigmaX, st.SigmaY)
	}
}
