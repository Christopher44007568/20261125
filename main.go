package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

type Material struct {
	Name          string
	Metal         bool
	BaseF0        float64
	DiffuseAlbedo float64
	BaseRoughness float64
	CoatF0        float64
}

type Observation struct {
	Mode             string
	Material         string
	ViewDegrees      float64
	BlackBase        bool
	Lights           int
	BaseF0           float64
	CoatF0           float64
	InTransmission   float64
	OutTransmission  float64
	MeanTransmission float64
	DiffuseEnergy    float64
	SpecularEnergy   float64
	CoatEnergy       float64
	TotalEnergy      float64
	InputUpperBound  float64
	Excess           float64
	Causes           []string
}

func main() {
	materials := buildMaterials()
	views := []float64{0, 60, 89}

	fmt.Println("clearcoat BRDF energy audit")
	fmt.Printf("sample count per lobe: %d\n", sampleCount)
	fmt.Println()

	var observations []Observation
	for _, material := range materials {
		for _, view := range views {
			for _, black := range []bool{false, true} {
				observations = append(observations, evaluateMaterial(material, view, black, 1, false))
				observations = append(observations, evaluateMaterial(material, view, black, 1, true))
				observations = append(observations, evaluateMaterial(material, view, black, 3, false))
				observations = append(observations, evaluateMaterial(material, view, black, 3, true))
			}
		}
	}

	if err := writeCSV(observations, "energy_audit.csv"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	printSummary(observations)
	printViolations(observations)
}

func buildMaterials() []Material {
	return []Material{
		{Name: "rough_metal", Metal: true, BaseF0: 0.95, DiffuseAlbedo: 0, BaseRoughness: 0.42, CoatF0: 0.04},
		{Name: "clear_dielectric", Metal: false, BaseF0: 0.04, DiffuseAlbedo: 0.65, BaseRoughness: 0.28, CoatF0: 0.04},
		{Name: "energy_limit", Metal: false, BaseF0: 0.04, DiffuseAlbedo: 0.96, BaseRoughness: 0.18, CoatF0: 0.04},
	}
}

func modeName(buggy bool) string {
	if buggy {
		return "separate_lobe_normalization"
	}
	return "shared_budget"
}

func printSummary(observations []Observation) {
	fmt.Println("single-light energy (multi-light is the same linear BRDF sum)")
	fmt.Println("mode,material,view,black,F0_base,F0_coat,T_in,T_out_mean,diffuse,spec,coat,total,bound")
	for _, row := range observations {
		if row.Lights != 1 || row.BlackBase {
			continue
		}
		fmt.Printf("%s,%s,%.0f,false,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.0f\n",
			row.Mode, row.Material, row.ViewDegrees, row.BaseF0, row.CoatF0,
			row.InTransmission, row.OutTransmission, row.DiffuseEnergy,
			row.SpecularEnergy, row.CoatEnergy, row.TotalEnergy, row.InputUpperBound)
	}

	fmt.Println()
	fmt.Println("black substrate (three equal unit-irradiance lights)")
	fmt.Println("mode,material,view,T_in,T_out_mean,coat_total,total,bound")
	for _, row := range observations {
		if row.Lights != 3 || !row.BlackBase {
			continue
		}
		fmt.Printf("%s,%s,%.0f,%.4f,%.4f,%.4f,%.4f,%.0f\n",
			row.Mode, row.Material, row.ViewDegrees, row.InTransmission,
			row.OutTransmission, row.CoatEnergy, row.TotalEnergy, row.InputUpperBound)
	}
}

func printViolations(observations []Observation) {
	count := 0
	fmt.Println()
	fmt.Println("pixels above input upper bound")
	for _, row := range observations {
		if row.Excess <= 1e-3 {
			continue
		}
		count++
		black := "colored"
		if row.BlackBase {
			black = "black"
		}
		fmt.Printf("%s | %s | %.0f° | %s base | %d light(s): total %.4f > %.0f, excess %.4f; %s\n",
			row.Mode, row.Material, row.ViewDegrees, black, row.Lights,
			row.TotalEnergy, row.InputUpperBound, row.Excess, strings.Join(row.Causes, "; "))
	}
	if count == 0 {
		fmt.Println("none")
	}
}

func writeCSV(observations []Observation, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"mode", "material", "view_degrees", "black_base", "lights", "base_f0", "coat_f0",
		"in_transmission", "spec_out_transmission", "diffuse_mean_transmission",
		"diffuse_energy", "spec_energy", "coat_energy", "total_energy",
		"input_upper_bound", "excess", "causes",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, row := range observations {
		record := []string{
			row.Mode, row.Material, fmt.Sprintf("%.0f", row.ViewDegrees),
			fmt.Sprintf("%t", row.BlackBase), fmt.Sprintf("%d", row.Lights),
			fmt.Sprintf("%.6f", row.BaseF0), fmt.Sprintf("%.6f", row.CoatF0),
			fmt.Sprintf("%.6f", row.InTransmission),
			fmt.Sprintf("%.6f", row.OutTransmission),
			fmt.Sprintf("%.6f", row.MeanTransmission),
			fmt.Sprintf("%.6f", row.DiffuseEnergy),
			fmt.Sprintf("%.6f", row.SpecularEnergy),
			fmt.Sprintf("%.6f", row.CoatEnergy),
			fmt.Sprintf("%.6f", row.TotalEnergy),
			fmt.Sprintf("%.6f", row.InputUpperBound),
			fmt.Sprintf("%.6f", row.Excess),
			strings.Join(row.Causes, " | "),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
