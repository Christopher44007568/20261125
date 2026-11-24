package main

import "math"

const (
	imgW = 128
	imgH = 80
	ipd  = 0.064 // interpupillary distance, meters
)

// Scenario perturbs the two eyes relative to each other.
type Scenario struct {
	Name         string
	YawL, YawR   float64 // per-eye view yaw (vergence / head-turn)
	GazeL, GazeR Vec3    // per-eye foveal gaze directions (unit, pre-yaw)
}

func scenarios() []Scenario {
	fwd := Vec3{0, 0, 1}
	off := yaw(fwd, 0.10) // gaze shifted ~5.7 deg right
	return []Scenario{
		{Name: "base", YawL: 0, YawR: 0, GazeL: fwd, GazeR: fwd},
		// Strabismus: right eye deviates outward by 3 deg.
		{Name: "strabismus", YawL: 0, YawR: -0.052, GazeL: fwd, GazeR: fwd},
		// Rapid head turn: eyes sample the rotation 8 ms apart at 250 deg/s.
		{Name: "headturn", YawL: 0, YawR: 0.035, GazeL: fwd, GazeR: fwd},
		// Different gaze regions: foveal partitions land on different content.
		{Name: "gaze", YawL: 0, YawR: 0, GazeL: fwd, GazeR: off},
	}
}

func (s Scenario) cameras() (camL, camR Camera) {
	fov := 50 * math.Pi / 180
	camL = Camera{Origin: Vec3{-ipd / 2, 0, 0}, Yaw: s.YawL, FovY: fov}
	camR = Camera{Origin: Vec3{+ipd / 2, 0, 0}, Yaw: s.YawR, FovY: fov}
	return
}

// Image is a linear RGB float image.
type Image struct {
	W, H int
	Pix  []Vec3
}

func newImage(w, h int) *Image { return &Image{W: w, H: h, Pix: make([]Vec3, w*h)} }

// probePixels are the pixels whose light paths / T-curves get recorded:
// image center (through the glow sphere) and an off-center smoke pixel.
var probePixels = [][2]int{{imgW / 2, imgH / 2}, {imgW / 3, imgH / 2}}

// Strategy renders a stereo pair plus probe curves.
type Strategy struct {
	Name   string
	Render func(sc Scene, s Scenario) (imgL, imgR *Image, probes [2][2]MarchResult)
}

func renderEye(sc Scene, cam Camera, gaze Vec3, foveal bool, img *Image, probes *[2][2]MarchResult, eyeIdx int) {
	gazeWorld := yaw(gaze, cam.Yaw)
	for py := 0; py < img.H; py++ {
		for px := 0; px < img.W; px++ {
			d := cam.Ray(px, py, img.W, img.H)
			step := stepFull
			if foveal {
				step = fovealStep(d, gazeWorld)
			}
			pi := probeIndex(px, py)
			rec := pi >= 0
			r := march(sc, cam.Origin, d, step, rec)
			img.Pix[py*img.W+px] = r.Color
			if rec {
				probes[eyeIdx][pi] = r
			}
		}
	}
}

func probeIndex(px, py int) int {
	for i, pp := range probePixels {
		if px == pp[0] && py == pp[1] {
			return i
		}
	}
	return -1
}

func strategies() []Strategy {
	full := Strategy{Name: "full"}
	full.Render = func(sc Scene, s Scenario) (imgL, imgR *Image, probes [2][2]MarchResult) {
		camL, camR := s.cameras()
		imgL, imgR = newImage(imgW, imgH), newImage(imgW, imgH)
		renderEye(sc, camL, s.GazeL, false, imgL, &probes, 0)
		renderEye(sc, camR, s.GazeR, false, imgR, &probes, 1)
		return
	}

	shared := Strategy{Name: "shared"}
	shared.Render = func(sc Scene, s Scenario) (imgL, imgR *Image, probes [2][2]MarchResult) {
		// One coarse march from the cyclopean eye; the result (radiance and
		// transmittance) is copied to both eyes. Cheap, but it discards all
		// binocular disparity in the medium.
		fov := 50 * math.Pi / 180
		camC := Camera{Origin: Vec3{0, 0, 0}, Yaw: 0.5 * (s.YawL + s.YawR), FovY: fov}
		imgL, imgR = newImage(imgW, imgH), newImage(imgW, imgH)
		for py := 0; py < imgH; py++ {
			for px := 0; px < imgW; px++ {
				d := camC.Ray(px, py, imgW, imgH)
				rec := probeIndex(px, py) >= 0
				r := march(sc, camC.Origin, d, stepShared, rec)
				imgL.Pix[py*imgW+px] = r.Color
				imgR.Pix[py*imgW+px] = r.Color
				if rec {
					pi := probeIndex(px, py)
					probes[0][pi] = r
					probes[1][pi] = r
				}
			}
		}
		return
	}

	foveated := Strategy{Name: "foveated"}
	foveated.Render = func(sc Scene, s Scenario) (imgL, imgR *Image, probes [2][2]MarchResult) {
		camL, camR := s.cameras()
		imgL, imgR = newImage(imgW, imgH), newImage(imgW, imgH)
		renderEye(sc, camL, s.GazeL, true, imgL, &probes, 0)
		renderEye(sc, camR, s.GazeR, true, imgR, &probes, 1)
		return
	}

	return []Strategy{full, shared, foveated}
}

// DiffStats summarizes per-eye image difference.
type DiffStats struct {
	MAE  float64 // mean absolute error over RGB
	Max  float64 // max absolute error
	RMSE float64
}

func diffImages(a, b *Image) (DiffStats, *Image) {
	var sum, sumSq, max float64
	d := newImage(a.W, a.H)
	for i := range a.Pix {
		dr := math.Abs(a.Pix[i].X - b.Pix[i].X)
		dg := math.Abs(a.Pix[i].Y - b.Pix[i].Y)
		db := math.Abs(a.Pix[i].Z - b.Pix[i].Z)
		d.Pix[i] = Vec3{dr, dg, db}
		e := (dr + dg + db) / 3
		sum += e
		sumSq += e * e
		if e > max {
			max = e
		}
	}
	n := float64(len(a.Pix))
	return DiffStats{MAE: sum / n, Max: max, RMSE: math.Sqrt(sumSq / n)}, d
}
