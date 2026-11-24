package main

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

const probeX, probeY = 10, 10

func transferName(t Transfer) string {
	if t == Linear {
		return "linear"
	}
	return "srgb"
}

func writeErrorCSV(path string, errors [][]GammaError) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err := w.Write([]string{"x", "y", "gamma_delta_r", "gamma_delta_g", "gamma_delta_b", "linear_rmse", "valid"}); err != nil {
		return err
	}
	for y, row := range errors {
		for x, e := range row {
			valid := "false"
			if e.Valid {
				valid = "true"
			}
			gamma := func(value float64) string {
				if math.IsNaN(value) {
					return ""
				}
				return formatFloat(value)
			}
			record := []string{
				itoa(x), itoa(y),
				gamma(e.R), gamma(e.G), gamma(e.B),
				formatFloat(e.LinearRMSE), valid,
			}
			if err := w.Write(record); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}

func itoa(v int) string {
	return formatFloat(float64(v))
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(f64(v), 'f', -1, 64)
}

func summarizeLayer(layer Layer, frame LayerFrame) LayerReport {
	var maxAlpha, sumAlpha float64
	for _, row := range frame.Decoded {
		for _, p := range row {
			maxAlpha = math.Max(maxAlpha, p.A)
			sumAlpha += p.A
		}
	}

	input := frame.Input[probeY][probeX]
	decoded := frame.Decoded[probeY][probeX]
	composited := frame.Composited[probeY][probeX]
	spaceValue := 0.0
	if layer.Space == SRGB {
		spaceValue = 1
	}

	return LayerReport{
		Name:          layer.Name,
		Eye:           frame.Eye,
		Space:         transferName(layer.Space),
		Premultiplied: layer.Premul,
		Probe: PixelSummary{
			X:          probeX,
			Y:          probeY,
			Eye:        frame.Eye,
			Input:      [5]float64{input.R, input.G, input.B, input.A, spaceValue},
			Decoded:    [4]float64{decoded.R, decoded.G, decoded.B, decoded.A},
			Composited: [4]float64{composited.R, composited.G, composited.B, composited.A},
		},
		MaxAlpha:  maxAlpha,
		MeanAlpha: sumAlpha / float64(len(frame.Decoded)*len(frame.Decoded[0])),
	}
}

func summarizeErrors(eye int, mode string, errors [][]GammaError) ErrorStats {
	var sumRMSE, maxRMSE, sumGamma float64
	var validCount int

	for _, row := range errors {
		for _, e := range row {
			sumRMSE += e.LinearRMSE
			maxRMSE = math.Max(maxRMSE, e.LinearRMSE)
			if e.Valid {
				sumGamma += (math.Abs(e.R) + math.Abs(e.G) + math.Abs(e.B)) / 3
				validCount++
			}
		}
	}

	pixelCount := float64(len(errors) * len(errors[0]))
	meanGamma := 0.0
	if validCount > 0 {
		meanGamma = sumGamma / float64(validCount)
	}

	return ErrorStats{
		Mode:             mode,
		Eye:              eye,
		MeanLinearRMSE:   sumRMSE / pixelCount,
		MaxLinearRMSE:    maxRMSE,
		MeanGammaDelta:   meanGamma,
		ValidGammaPixels: validCount,
	}
}

func run(outputDir string) (Report, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Report{}, err
	}

	scene := buildScene()
	allRecords, correctFinal := scene.Composite()
	report := Report{
		Width:      scene.Width,
		Height:     scene.Height,
		Background: "linear HDR, straight alpha, Reinhard tonemap only at display encoding",
	}

	for eye := 0; eye < 2; eye++ {
		wrongFinal := scene.CompositeWrongEncodedEye(eye)
		wrongErrors := gammaErrors(correctFinal[eye], wrongFinal, SRGB)
		additionFinal := scene.CompositeDirectAdditionEye(eye)
		additionErrors := gammaErrors(correctFinal[eye], additionFinal, Linear)
		report.ErrorStats = append(
			report.ErrorStats,
			summarizeErrors(eye, "srgb_encoded_over", wrongErrors),
			summarizeErrors(eye, "direct_linear_addition", additionErrors),
		)

		for index, frame := range allRecords[eye] {
			layer := scene.Layers[index]
			base := filepath.Join(outputDir, "eye"+itoa(eye)+"_layer"+itoa(index)+"_"+layer.Name)
			if err := writeLayerCSV(base+".csv", frame.Input, frame.Decoded, frame.Composited); err != nil {
				return Report{}, err
			}
			if err := writePNG(base+"_preview.png", frameImage(frame.Decoded, true)); err != nil {
				return Report{}, err
			}
			report.Layers = append(report.Layers, summarizeLayer(layer, frame))
			report.OutputFiles = append(report.OutputFiles, base+".csv", base+"_preview.png")
		}

		finalCSV := filepath.Join(outputDir, "eye"+itoa(eye)+"_correct_final_linear_premul.csv")
		if err := writePremulCSV(finalCSV, correctFinal[eye]); err != nil {
			return Report{}, err
		}
		errorCSV := filepath.Join(outputDir, "eye"+itoa(eye)+"_gamma_error.csv")
		if err := writeErrorCSV(errorCSV, wrongErrors); err != nil {
			return Report{}, err
		}
		additionErrorCSV := filepath.Join(outputDir, "eye"+itoa(eye)+"_addition_error.csv")
		if err := writeErrorCSV(additionErrorCSV, additionErrors); err != nil {
			return Report{}, err
		}
		correctPNG := filepath.Join(outputDir, "eye"+itoa(eye)+"_correct.png")
		wrongPNG := filepath.Join(outputDir, "eye"+itoa(eye)+"_wrong_gamma.png")
		additionPNG := filepath.Join(outputDir, "eye"+itoa(eye)+"_wrong_addition.png")
		heatPNG := filepath.Join(outputDir, "eye"+itoa(eye)+"_gamma_heatmap.png")
		additionHeatPNG := filepath.Join(outputDir, "eye"+itoa(eye)+"_addition_heatmap.png")
		if err := writePNG(correctPNG, frameImage(correctFinal[eye], false)); err != nil {
			return Report{}, err
		}
		if err := writePNG(wrongPNG, encodedFrameImage(wrongFinal)); err != nil {
			return Report{}, err
		}
		if err := writePNG(additionPNG, frameImage(additionFinal, false)); err != nil {
			return Report{}, err
		}
		if err := writePNG(heatPNG, heatImage(wrongErrors)); err != nil {
			return Report{}, err
		}
		if err := writePNG(additionHeatPNG, heatImage(additionErrors)); err != nil {
			return Report{}, err
		}
		report.OutputFiles = append(
			report.OutputFiles,
			finalCSV, errorCSV, additionErrorCSV,
			correctPNG, wrongPNG, additionPNG, heatPNG, additionHeatPNG,
		)
	}

	jsonPath := filepath.Join(outputDir, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return Report{}, err
	}
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return Report{}, err
	}
	report.OutputFiles = append(report.OutputFiles, jsonPath)
	return report, nil
}
