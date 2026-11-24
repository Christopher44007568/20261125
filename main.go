// Command stereocheck builds asymmetric (off-center) stereo projection
// matrices from per-eye intrinsics/extrinsics, validates them against an
// independent matrix-free geometric reference, and verifies that the
// disparity sign never flips while IPD, convergence distance and aspect
// ratio are varied. Standard library only.
package main

import (
	"fmt"
	"os"
)

const (
	angTolDeg = 1e-6 // angular tolerance between matrix ray and reference ray
	posTolM   = 1e-6 // positional tolerance at near/far/screen planes
)

type config struct {
	ipd, conv float64
	aw, ah    int // aspect ratio
	in        Intrinsics
}

func makeConfig(ipd, conv float64, aw, ah int) config {
	h := 800
	w := h * aw / ah
	return config{
		ipd: ipd, conv: conv, aw: aw, ah: ah,
		in: Intrinsics{
			Fx: 1120, Fy: 1055, // same-ish focal, but...
			Cx: float64(w)/2 + 13, Cy: float64(h)/2 - 7, // ...off-center principal point
			W: w, H: h,
			Near: 0.05, Far: 300,
		},
	}
}

func main() {
	var configs []config
	for _, ipd := range []float64{0.058, 0.064, 0.072} {
		for _, conv := range []float64{0.5, 1.5, 8.0} {
			for _, asp := range [][2]int{{16, 9}, {4, 3}, {1, 1}} {
				configs = append(configs, makeConfig(ipd, conv, asp[0], asp[1]))
			}
		}
	}

	var failures []string
	globalMaxAng := 0.0
	nearSign, farSign := 0, 0 // must stay constant across all configs

	for _, c := range configs {
		fmt.Printf("== ipd=%.3fm conv=%.2fm aspect=%d:%d res=%dx%d cx=%.1f cy=%.1f\n",
			c.ipd, c.conv, c.aw, c.ah, c.in.W, c.in.H, c.in.Cx, c.in.Cy)

		eyes := [2]Eye{}
		for i, x := range []float64{-c.ipd / 2, +c.ipd / 2} {
			name := []string{"L", "R"}[i]
			e, err := NewEye(name, x, c.in, c.conv)
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			eyes[i] = e
			fmt.Printf("  eye %s @x=%+.4f  frustum L%+.5f R%+.5f B%+.5f T%+.5f (asym h=%+.5f v=%+.5f)\n",
				name, x, e.Frust.L, e.Frust.R, e.Frust.B, e.Frust.T,
				(e.Frust.R+e.Frust.L)/2, (e.Frust.T+e.Frust.B)/2)
			fmt.Printf("  Proj[%s]:\n%s", name, e.Proj)
		}

		for _, e := range eyes {
			r := e.CheckEye(c.conv)
			if r.MaxAngleDeg > globalMaxAng {
				globalMaxAng = r.MaxAngleDeg
			}
			fmt.Printf("  check[%s]: maxAng=%.3e deg  nearErr=%.3em  farErr=%.3em  edgeErr=%.3em\n",
				e.Name, r.MaxAngleDeg, r.MaxNearErr, r.MaxFarErr, r.MaxEdgeErr)
			if r.MaxAngleDeg > angTolDeg || r.MaxNearErr > posTolM ||
				r.MaxFarErr > posTolM || r.MaxEdgeErr > posTolM {
				msg := fmt.Sprintf("cfg ipd=%.3f conv=%.2f asp=%d:%d eye %s: %s",
					c.ipd, c.conv, c.aw, c.ah, e.Name, r.WorstSample)
				failures = append(failures, msg)
			}
		}

		// Disparity of a world point on the rig axis at 0.5x, 1x, 2x the
		// convergence distance. Convention: disp = ndcX_L - ndcX_R,
		// so nearer-than-screen => positive, farther => negative.
		for _, tc := range []struct {
			tag  string
			z    float64
			want int
		}{
			{"near(0.5*conv)", 0.5 * c.conv, +1},
			{"at  (1.0*conv)", 1.0 * c.conv, 0},
			{"far (2.0*conv)", 2.0 * c.conv, -1},
		} {
			p := Vec3{0, 0, -tc.z}
			d := eyes[0].project(p).X - eyes[1].project(p).X
			got := sign(d, 1e-9)
			fmt.Printf("  disparity %-15s z=%7.3fm: %+.6e (want sign %2d)\n", tc.tag, tc.z, d, tc.want)
			if got != tc.want {
				failures = append(failures, fmt.Sprintf(
					"cfg ipd=%.3f conv=%.2f asp=%d:%d: DISPARITY SIGN FLIP %s disp=%.3e want sign %d",
					c.ipd, c.conv, c.aw, c.ah, tc.tag, d, tc.want))
			}
			switch tc.want {
			case +1:
				if nearSign == 0 {
					nearSign = got
				} else if got != nearSign {
					failures = append(failures, fmt.Sprintf("cfg ipd=%.3f conv=%.2f asp=%d:%d: near-disparity sign changed across configs", c.ipd, c.conv, c.aw, c.ah))
				}
			case -1:
				if farSign == 0 {
					farSign = got
				} else if got != farSign {
					failures = append(failures, fmt.Sprintf("cfg ipd=%.3f conv=%.2f asp=%d:%d: far-disparity sign changed across configs", c.ipd, c.conv, c.aw, c.ah))
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("================ SUMMARY ================")
	fmt.Printf("configs tested:      %d (x2 eyes)\n", len(configs))
	fmt.Printf("max angular error:   %.3e deg (tol %.0e)\n", globalMaxAng, angTolDeg)
	fmt.Printf("near-disparity sign: %+d constant=%v\n", nearSign, nearSign == +1)
	fmt.Printf("far-disparity sign:  %+d constant=%v\n", farSign, farSign == -1)
	if len(failures) == 0 {
		fmt.Println("failure samples:     none")
	} else {
		fmt.Printf("failure samples:     %d\n", len(failures))
		for _, f := range failures {
			fmt.Println("  FAIL:", f)
		}
		os.Exit(1)
	}
}

func sign(v, eps float64) int {
	switch {
	case v > eps:
		return +1
	case v < -eps:
		return -1
	default:
		return 0
	}
}
