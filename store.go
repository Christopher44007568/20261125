package xrpath

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
)

const (
	FramesFile   = "frames.jsonl"
	ManifestFile = "manifest.json"
	CompareJSON  = "comparison.json"
	CompareMD    = "comparison.md"
	FaultJSON    = "fault-injections.json"
	FaultMD      = "fault-injections.md"
)

func SaveRun(dir string, run RunResult) error {
	imageDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return err
	}

	manifestData, err := json.MarshalIndent(run.Manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), manifestData, 0o644); err != nil {
		return err
	}

	framesPath := filepath.Join(dir, FramesFile)
	file, err := os.Create(framesPath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	for _, record := range run.Records {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			return err
		}

		stereo := run.Images[record.Frame]
		if stereo == nil {
			return fmt.Errorf("missing rendered images for frame %d", record.Frame)
		}
		for _, eye := range []*RenderedEye{stereo.Left, stereo.Right} {
			if eye == nil || len(eye.PNGBytes) == 0 {
				return fmt.Errorf("missing PNG bytes for frame %d", record.Frame)
			}
			name := formatFrameName("images/"+eye.Eye, record.Frame)
			if err := os.WriteFile(filepath.Join(dir, name), eye.PNGBytes, 0o644); err != nil {
				return err
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	return file.Close()
}

func LoadRecords(dir string) ([]FrameRecord, error) {
	file, err := os.Open(filepath.Join(dir, FramesFile))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	records := []FrameRecord{}
	line := 0
	for scanner.Scan() {
		line++
		var record FrameRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", FramesFile, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func LoadManifest(dir string) (Manifest, error) {
	var manifest Manifest
	data, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func VerifyStoredImages(dir string, records []FrameRecord) []Mismatch {
	mismatches := []Mismatch{}
	for _, record := range records {
		for _, ref := range []ImageRef{record.Images.Left, record.Images.Right} {
			path := filepath.Join(dir, ref.Path)
			data, err := os.ReadFile(path)
			if err != nil {
				mismatches = append(mismatches, Mismatch{
					Frame: record.Frame, Subsystem: imageSubsystem(ref.Eye), Item: ref.Eye,
					Field: "image_file", Detail: err.Error(),
				})
				continue
			}
			sum := sha256.Sum256(data)
			if got := hex.EncodeToString(sum[:]); got != ref.PNGSHA256 {
				mismatches = append(mismatches, Mismatch{
					Frame: record.Frame, Subsystem: imageSubsystem(ref.Eye), Item: ref.Eye,
					Field: "png_sha256", Expected: ref.PNGSHA256, Actual: got,
				})
			}

			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				mismatches = append(mismatches, Mismatch{
					Frame: record.Frame, Subsystem: imageSubsystem(ref.Eye), Item: ref.Eye,
					Field: "png_decode", Detail: err.Error(),
				})
				continue
			}
			rgba := image.NewRGBA(img.Bounds())
			draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
			if got := bytesSHA256(rgba.Pix); got != ref.PixelSHA256 {
				mismatches = append(mismatches, Mismatch{
					Frame: record.Frame, Subsystem: imageSubsystem(ref.Eye), Item: ref.Eye,
					Field: "decoded_pixel_sha256", Expected: ref.PixelSHA256, Actual: got,
					Detail: "stored image pixels do not match frame record",
				})
			}
		}
	}
	return mismatches
}

func imageSubsystem(eye string) string {
	if eye == "right" {
		return SubsystemRightImage
	}
	return SubsystemLeftImage
}
