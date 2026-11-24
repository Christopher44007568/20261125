package stereofault

import (
	"fmt"
	"io"
	"strings"
)

// WriteReport writes the full audit report to w.
func (r *Renderer) WriteReport(w io.Writer) {
	var b strings.Builder

	b.WriteString("=== Stereo layer fault timeline ===\n")
	header := fmt.Sprintf("%-5s | %-22s | %-22s | %-22s | %-6s | %-6s\n",
		"frame", "world", "ui", "hand", "cmds", "black")
	b.WriteString(header)
	b.WriteString(strings.Repeat("-", len(header)-1) + "\n")

	for _, s := range r.Timeline {
		cells := map[LayerID]string{}
		for _, ls := range s.Layers {
			cells[ls.Layer] = fmt.Sprintf("%s/%s", ls.Status, faultLabel(ls.Fault))
		}
		fmt.Fprintf(&b, "%-5d | %-22s | %-22s | %-22s | %-6d | %-6v\n",
			s.Frame, cells[World], cells[UI], cells[Hand], s.Commands, s.Black)
	}

	b.WriteString("\n=== Events ===\n")
	if len(r.Events) == 0 {
		b.WriteString("(none)\n")
	}
	for _, e := range r.Events {
		fmt.Fprintf(&b, "f%-3d eye=%s %-19s layer=%-5s fault=%-17s %s\n",
			e.Frame, e.eyeLabel(), e.Kind, e.Layer, e.Fault, e.Note)
	}

	b.WriteString("\n=== Cumulative per-layer frame counts ===\n")
	fmt.Fprintf(&b, "%-6s | %-14s | %-17s | %-13s | %s\n",
		"layer", "healthy", "placeholder-frames", "hidden-frames", "failed-frames")
	for _, id := range r.order {
		l := r.layers[id]
		fmt.Fprintf(&b, "%-6s | %-14d | %-17d | %-13d | %d\n",
			id, l.HealthyFrames, l.PlaceholderFrames, l.HiddenFrames, l.FailFrames)
	}

	b.WriteString("\n=== Final service state ===\n")
	for _, id := range r.order {
		l := r.layers[id]
		state := "in service"
		if !l.Healthy {
			state = "OUT OF SERVICE (" + l.Fault.String() + ")"
		}
		fmt.Fprintf(&b, "%-6s %s\n", id, state)
	}

	io.WriteString(w, b.String())
}

func faultLabel(f Fault) string {
	if f == FaultNone {
		return "ok"
	}
	return f.String()
}
