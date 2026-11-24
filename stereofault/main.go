package stereofault

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunDemo executes the fault/recovery scenario and writes the audit report
// to stdout and to reportPath.
func RunDemo(reportPath string) *Renderer {
	const w, h = 10, 8

	world := NewWorldLayer(Rect{X0: 0, Y0: 0, X1: w, Y1: h})
	ui := NewUILayer(Rect{X0: 0, Y0: 0, X1: w, Y1: 2})
	hand := NewHandLayer(Rect{X0: 6, Y0: 4, X1: w, Y1: h})

	// Back-to-front compositing order: world, hand, then UI on top.
	r := NewRenderer(w, h, world, hand, ui)

	// Scheduled inject/repair actions keyed by frame number.
	injectAt := map[int]func(){
		2: func() { world.Inject(FaultMissingTexture) },
		4: func() { ui.Inject(FaultBadDepthFormat) },
		6: func() { hand.Inject(FaultBadProjection) },
	}
	repairAt := map[int]func(){
		5:  func() { world.Repair() },
		8:  func() { ui.Repair() },
		10: func() { hand.Repair() },
	}

	for range 12 {
		next := r.Frame + 1
		if fn, ok := injectAt[next]; ok {
			fn()
		}
		if fn, ok := repairAt[next]; ok {
			fn()
		}
		r.RenderFrame()
	}

	fmt.Println("Answer to the opening question: no. One broken layer does")
	fmt.Println("not black out the stereo picture. The failed layer is replaced")
	fmt.Println("by its declared placeholder or hidden, while world/UI/hand keep")
	fmt.Println("compositing independently for both eyes.")
	fmt.Println()
	r.WriteReport(os.Stdout)

	if reportPath != "" {
		f, err := os.Create(reportPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "report file: %v\n", err)
			return r
		}
		defer f.Close()
		r.WriteReport(f)
	}
	return r
}

func Main() {
	out := filepath.Join("..", "artifacts", "stereo-fault-report.txt")
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	RunDemo(out)
}
