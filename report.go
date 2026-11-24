package geomlife

import "time"

type Report struct {
	GeneratedAt time.Time       `json:"generated_at"`
	OK          bool            `json:"ok"`
	Canonical   CanonicalReport `json:"canonical"`
	Parallel    ParallelReport  `json:"parallel"`
	Canary      CanaryReport    `json:"canary"`
}

func RunSimulation() Report {
	canonical := RunCanonicalScenario()
	parallel := RunParallelScenario()
	canary := RunGeometryCanary()
	return Report{
		GeneratedAt: time.Now().UTC(),
		OK:          canonical.OK && parallel.OK && canary.OK,
		Canonical:   canonical,
		Parallel:    parallel,
		Canary:      canary,
	}
}
