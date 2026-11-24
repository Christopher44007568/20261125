package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"glassaudit/glass"
)

type suiteReport struct {
	Budget             glass.Budget     `json:"budget"`
	Options            glass.Options    `json:"options"`
	StableAcrossScenes bool             `json:"stable_across_all_permutations"`
	Scenarios          []scenarioReport `json:"scenarios"`
}

type scenarioReport struct {
	Kind                    glass.SceneKind `json:"kind"`
	LayerCount              int             `json:"layer_count"`
	SubmissionOrdersTested  int             `json:"submission_orders_tested"`
	StableAcrossSubmissions bool            `json:"stable_across_all_submissions"`
	Audit                   glass.Audit     `json:"audit"`
}

func main() {
	out := flag.String("out", "-", "output path, or - for stdout")
	left := flag.Int("left", glass.DefaultBudget().Left, "left-eye transparent layer budget")
	right := flag.Int("right", glass.DefaultBudget().Right, "right-eye transparent layer budget")
	epsilon := flag.Float64("epsilon", glass.DefaultDepthEpsilon, "depth bucket size")
	flag.Parse()

	report, err := buildSuite(glass.Budget{Left: *left, Right: *right}, glass.Options{DepthEpsilon: *epsilon})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !report.StableAcrossScenes {
		fmt.Fprintln(os.Stderr, "audit evidence changed across submissions")
		os.Exit(1)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *out == "-" {
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildSuite(budget glass.Budget, options glass.Options) (suiteReport, error) {
	report := suiteReport{
		Budget:             budget,
		Options:            options,
		StableAcrossScenes: true,
	}
	for _, scene := range glass.BuiltinScenes() {
		base, err := glass.AuditScene(scene, budget, options)
		if err != nil {
			return suiteReport{}, fmt.Errorf("%s: %w", scene.Kind, err)
		}
		submissions := permutations(scene.Layers)
		stable := true
		for _, submission := range submissions {
			shuffled := scene
			shuffled.Layers = submission
			candidate, err := glass.AuditScene(shuffled, budget, options)
			if err != nil {
				return suiteReport{}, fmt.Errorf("%s: %w", scene.Kind, err)
			}
			if !sameEvidence(base, candidate) {
				stable = false
				break
			}
		}
		if !stable {
			report.StableAcrossScenes = false
		}
		report.Scenarios = append(report.Scenarios, scenarioReport{
			Kind:                    scene.Kind,
			LayerCount:              len(scene.Layers),
			SubmissionOrdersTested:  len(submissions),
			StableAcrossSubmissions: stable,
			Audit:                   base,
		})
	}
	return report, nil
}

func sameEvidence(left, right glass.Audit) bool {
	return left.Scene == right.Scene &&
		left.DepthEpsilon == right.DepthEpsilon &&
		left.Budget == right.Budget &&
		reflect.DeepEqual(left.MaskBitOrder, right.MaskBitOrder) &&
		reflect.DeepEqual(left.Eyes, right.Eyes)
}

func permutations(layers []glass.Layer) [][]glass.Layer {
	result := make([][]glass.Layer, 0, factorial(len(layers)))
	var generate func(int)
	generate = func(index int) {
		if index == len(layers) {
			result = append(result, append([]glass.Layer(nil), layers...))
			return
		}
		for swap := index; swap < len(layers); swap++ {
			layers[index], layers[swap] = layers[swap], layers[index]
			generate(index + 1)
			layers[index], layers[swap] = layers[swap], layers[index]
		}
	}
	generate(0)
	return result
}

func factorial(value int) int {
	result := 1
	for factor := 2; factor <= value; factor++ {
		result *= factor
	}
	return result
}
