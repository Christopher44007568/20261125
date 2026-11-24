package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	scenario := flag.String("scenario", "all", "base|strabismus|headturn|gaze|all")
	flag.Parse()

	sc := Scene{}
	strats := strategies()
	for _, s := range scenarios() {
		if *scenario != "all" && *scenario != s.Name {
			continue
		}
		if err := runScenario(sc, s, strats, *outDir); err != nil {
			log.Fatal(err)
		}
	}
}

func runScenario(sc Scene, s Scenario, strats []Strategy, outDir string) error {
	dir := filepath.Join(outDir, s.Name)
	if err := ensureDir(dir); err != nil {
		return err
	}
	report := []string{
		fmt.Sprintf("scenario=%s yawL=%.4f yawR=%.4f rad", s.Name, s.YawL, s.YawR),
		fmt.Sprintf("image=%dx%d ipd=%.3fm steps: full=%.3f coarse=%.3f fovea=%.3f periph=%.3f tMax=%.1f",
			imgW, imgH, ipd, dtFull, dtCoarse, dtFovea, dtPeriph, tMax),
		"",
	}
	cw := &curveWriter{}
	eyes := map[string][2]*Image{}
	for _, st := range strats {
		imgL, imgR, probes := st.Render(sc, s)
		eyes[st.Name] = [2]*Image{imgL, imgR}

		lr, diff := diffImages(imgL, imgR)
		report = append(report,
			fmt.Sprintf("[%s] binocular L-vs-R: MAE=%.5f RMSE=%.5f Max=%.5f",
				st.Name, lr.MAE, lr.RMSE, lr.Max))

		for eye := 0; eye < 2; eye++ {
			for pi := range probePixels {
				r := probes[eye][pi]
				series := fmt.Sprintf("%s/%s/eye%d/probe%d", s.Name, st.Name, eye, pi)
				cw.add(series, r)
				report = append(report,
					fmt.Sprintf("  probe eye%d px%v: samples=%d T_end=%.4f path=(%s)",
						eye, probePixels[pi], len(r.TCurve), lastT(r), pathSummary(st.Name, s, eye, pi)))
			}
		}

		if err := savePNG(imgL, outPath(dir, st.Name+"_L.png"), 1.0); err != nil {
			return err
		}
		if err := savePNG(imgR, outPath(dir, st.Name+"_R.png"), 1.0); err != nil {
			return err
		}
		if err := savePNG(diff, outPath(dir, st.Name+"_diff.png"), 8.0); err != nil {
			return err
		}
	}

	// Accuracy vs the per-eye full-sampling reference.
	report = append(report, "")
	ref := eyes["full"]
	for _, name := range []string{"shared", "foveated"} {
		for eye := 0; eye < 2; eye++ {
			st, _ := diffImages(eyes[name][eye], ref[eye])
			report = append(report,
				fmt.Sprintf("[%s] eye%d vs full reference: MAE=%.5f RMSE=%.5f Max=%.5f",
					name, eye, st.MAE, st.RMSE, st.Max))
		}
	}

	report = append(report, "", alphaCorrectionCheck())

	if err := cw.save(outPath(dir, "transmittance_curves.csv")); err != nil {
		return err
	}
	if err := writeReport(outPath(dir, "report.txt"), report); err != nil {
		return err
	}
	fmt.Printf("scenario %-10s -> %s\n", s.Name, dir)
	return nil
}

func lastT(r MarchResult) float64 {
	if len(r.TCurve) == 0 {
		return 1
	}
	return r.TCurve[len(r.TCurve)-1]
}

// pathSummary describes the ray origin/direction that actually produced a
// probe curve. The shared strategy marches from the cyclopean eye, so both
// eyes report the mid-eye path (this is exactly the copy being flagged).
func pathSummary(strategy string, s Scenario, eye, pi int) string {
	camL, camR := s.cameras()
	cam := camL
	if eye == 1 {
		cam = camR
	}
	if strategy == "shared" {
		cam.Origin = Vec3{}
		cam.Yaw = 0.5 * (s.YawL + s.YawR)
	}
	pp := probePixels[pi]
	d := cam.Ray(pp[0], pp[1], imgW, imgH)
	return fmt.Sprintf("o=(%.3f,%.3f,%.3f) d=(%.3f,%.3f,%.3f)",
		cam.Origin.X, cam.Origin.Y, cam.Origin.Z, d.X, d.Y, d.Z)
}

// alphaCorrectionCheck verifies CorrectAlpha against direct exp evaluation
// and shows what error a naive copy would introduce.
func alphaCorrectionCheck() string {
	const sigma = 6.0 // smoke peak density
	aFine := alphaFor(sigma, dtFull)
	aCoarseDirect := alphaFor(sigma, dtCoarse)
	aCoarseCorrected := CorrectAlpha(aFine, dtFull, dtCoarse)
	err := math.Abs(aCoarseDirect - aCoarseCorrected)
	naiveErr := math.Abs(aFine - aCoarseDirect)
	return fmt.Sprintf("alpha correction: sigma=%.1f alpha(dt=%.2f)=%.4f alpha(dt=%.2f)=%.4f corrected=%.4f (residual=%.2e, naive-copy error=%.4f)",
		sigma, dtFull, aFine, dtCoarse, aCoarseDirect, aCoarseCorrected, err, naiveErr)
}
