package notify

import (
	"fmt"
	"io"
	"strings"
)

// WriteMarkdown writes a human-readable audit record.
func (r *Report) WriteMarkdown(writer io.Writer) error {
	var out strings.Builder
	overall := "PASS"
	if !r.Pass {
		overall = "FAIL"
	}
	fmt.Fprintf(&out, "# Notification scanout report\n\nOverall: **%s**\n\n", overall)

	for _, scenario := range r.Cases {
		status := "PASS"
		if !scenario.Pass {
			status = "FAIL"
		}
		fmt.Fprintf(&out, "## %s — %s\n\n", scenario.Title, status)
		fmt.Fprintf(&out, "%s\n\n", scenario.Description)

		out.WriteString("| Version | Revision | Closed | Position | Opacity | Visible |\n")
		out.WriteString("| --- | ---: | :---: | ---: | ---: | :---: |\n")
		writeVersionRow(&out, "Old", scenario.Old)
		writeVersionRow(&out, "Committed", scenario.Committed)
		writeVersionRow(&out, "Final", scenario.Final)
		out.WriteString("\n")

		fmt.Fprintf(&out, "- Committed vs old: `%s`\n", diffKindsOrNone(scenario.CommittedDiff))
		fmt.Fprintf(&out, "- Final vs old: `%s`\n", diffKindsOrNone(scenario.FinalDiff))
		fmt.Fprintf(&out, "- Final vs committed: `%s`\n\n", diffKindsOrNone(scenario.CommittedToFinalDiff))

		out.WriteString("Checks:\n\n")
		for _, check := range scenario.Checks {
			if !check.Pass {
				out.WriteString("- [ ] ")
			} else {
				out.WriteString("- [x] ")
			}
			fmt.Fprintf(&out, "**%s** — %s\n", check.Name, check.Detail)
		}
		out.WriteString("\n")

		out.WriteString("| Seq | At | Source | Rev | Timing | Target | Outcome | Changes |\n")
		out.WriteString("| ---: | ---: | --- | ---: | --- | ---: | --- | --- |\n")
		for _, event := range scenario.Events {
			fmt.Fprintf(&out, "| %d | %s | %s | %d | %s | %d | %s | %s |\n",
				event.Seq,
				event.Update.At.String(),
				event.Update.Source,
				event.Update.State.Revision,
				event.Timing,
				event.TargetFrame,
				event.Outcome,
				diffKindsOrNone(event.Change),
			)
		}
		out.WriteString("\n")
	}

	_, err := io.WriteString(writer, out.String())
	return err
}

func writeVersionRow(out *strings.Builder, label string, state State) {
	fmt.Fprintf(out, "| %s | %d | %t | (%.1f, %.1f) | %.2f | %t |\n",
		label,
		state.Revision,
		state.Closed,
		state.Position.X,
		state.Position.Y,
		state.Opacity,
		state.Visible(),
	)
}

func diffKindsOrNone(diff Diff) string {
	kinds := diff.Kinds()
	if len(kinds) == 0 {
		return "none"
	}
	return strings.Join(kinds, ", ")
}
