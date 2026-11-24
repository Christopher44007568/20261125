package normalcone

import (
	"fmt"
	"math"
)

type PolicyStats struct {
	TotalSamples      int
	VisibleSamples    int
	FalseCulls        int
	FalseCullRate     float64
	BoundaryDecisions []bool
	BoundaryStable    bool
}

type ScenarioReport struct {
	Scenario              Scenario
	GroundTruthCone       Cone
	Conservative          PolicyStats
	AggressiveWrongVertex PolicyStats
	AggressiveFirstNormal PolicyStats
}

func Evaluate(scenarios []Scenario) []ScenarioReport {
	reports := make([]ScenarioReport, len(scenarios))
	for index, scenario := range scenarios {
		reports[index] = evaluateScenario(scenario)
	}
	return reports
}

func evaluateScenario(scenario Scenario) ScenarioReport {
	report := ScenarioReport{Scenario: scenario, GroundTruthCone: scenario.Reference}
	report.Conservative = evaluatePolicy(scenario, conservativeDecision)
	report.AggressiveWrongVertex = evaluatePolicy(scenario, wrongAxisDecision)
	report.AggressiveFirstNormal = evaluatePolicy(scenario, firstNormalDecision)
	return report
}

func evaluatePolicy(scenario Scenario, decision func(Scenario, Vec3) bool) PolicyStats {
	stats := PolicyStats{}
	for _, camera := range scenario.CameraSamples {
		stats.TotalSamples++
		visible := groundTruthVisible(scenario, camera)
		if visible {
			stats.VisibleSamples++
		}
		if visible && decision(scenario, camera) {
			stats.FalseCulls++
		}
	}

	first := decision(scenario, scenario.BoundaryViews[0])
	stats.BoundaryDecisions = []bool{first}
	stats.BoundaryStable = true
	for _, camera := range scenario.BoundaryViews[1:] {
		current := decision(scenario, camera)
		stats.BoundaryDecisions = append(stats.BoundaryDecisions, current)
		if current != first {
			stats.BoundaryStable = false
		}
	}

	if stats.VisibleSamples > 0 {
		stats.FalseCullRate = float64(stats.FalseCulls) / float64(stats.VisibleSamples)
	}
	return stats
}

func groundTruthVisible(scenario Scenario, camera Vec3) bool {
	for _, normal := range scenario.Normals {
		if !Unit(normal) {
			continue
		}
		visible, ok := ActuallyVisible(normal, scenario.SurfacePoint, camera)
		if ok && visible {
			return true
		}
	}
	return false
}

func conservativeDecision(scenario Scenario, camera Vec3) bool {
	if !scenario.Reference.Valid {
		return false
	}
	return ConservativeCull(scenario.Reference, scenario.SurfacePoint, camera)
}

func wrongAxisDecision(scenario Scenario, camera Vec3) bool {
	if !scenario.Reference.Valid {
		return false
	}
	return AggressiveCull(scenario.WrongAxis, scenario.SurfacePoint, camera)
}

func firstNormalDecision(scenario Scenario, camera Vec3) bool {
	if len(scenario.Normals) == 0 || !Unit(scenario.Normals[0]) {
		return false
	}
	return AggressiveCull(scenario.WrongAxis, scenario.SurfacePoint, camera)
}

func FormatTextReport(reports []ScenarioReport) string {
	output := "scenario             visible  conservative false-kill  aggressive wrong-axis  aggressive degenerate\n"
	for _, report := range reports {
		output += fmt.Sprintf(
			"%-20s %8d %18.4f%% %21.4f%% %24.4f%%\n",
			report.Scenario.Name,
			report.Conservative.VisibleSamples,
			report.Conservative.FalseCullRate*100,
			report.AggressiveWrongVertex.FalseCullRate*100,
			report.AggressiveFirstNormal.FalseCullRate*100,
		)
	}

	output += "\nreference cones\n"
	for _, report := range reports {
		cone := report.GroundTruthCone
		if !cone.Valid {
			output += fmt.Sprintf("%-20s invalid: %s\n", report.Scenario.Name, cone.Reason)
			continue
		}
		output += fmt.Sprintf(
			"%-20s axis=(% .6f % .6f % .6f) half-angle=%8.4f deg\n",
			report.Scenario.Name,
			cone.Axis.X,
			cone.Axis.Y,
			cone.Axis.Z,
			cone.HalfAngle*180/math.Pi,
		)
	}

	output += "\nboundary determinism (false means retain; repeated per boundary camera)\n"
	for _, report := range reports {
		output += fmt.Sprintf(
			"%-20s conservative=%v stable=%t  wrong-axis=%v stable=%t\n",
			report.Scenario.Name,
			report.Conservative.BoundaryDecisions,
			report.Conservative.BoundaryStable,
			report.AggressiveWrongVertex.BoundaryDecisions,
			report.AggressiveWrongVertex.BoundaryStable,
		)
	}
	return output
}
