package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"final-sorter/sorter"
)

type input struct {
	Config      inputConfig         `json:"config"`
	Submissions []sorter.Submission `json:"submissions"`
}

type inputConfig struct {
	ReorderWindow *uint64        `json:"reorder_window"`
	TimeoutMS     *int64         `json:"timeout_ms"`
	LatestFrame   *uint64        `json:"latest_frame"`
	QueueOrder    []sorter.Queue `json:"queue_order"`
	EyeOrder      []sorter.Eye   `json:"eye_order"`
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "final-sorter:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("final-sorter", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputPath := flags.String("in", "-", "input JSON path, or - for stdin")
	outputPath := flags.String("out", "-", "output JSON path, or - for stdout")
	pretty := flags.Bool("pretty", true, "indent output JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	data, err := readInput(*inputPath, stdin)
	if err != nil {
		return err
	}
	var request input
	if err := decodeOne(data, &request); err != nil {
		return fmt.Errorf("decode input: %w", err)
	}

	cfg, latest, err := buildConfig(request.Config)
	if err != nil {
		return err
	}

	var report sorter.Report
	if latest == nil {
		report, err = sorter.Sort(request.Submissions, cfg)
	} else {
		report, err = sorter.SortAt(request.Submissions, *latest, cfg)
	}
	if err != nil {
		return fmt.Errorf("sort submissions: %w", err)
	}

	output, err := encodeReport(report, *pretty)
	if err != nil {
		return err
	}
	return writeOutput(*outputPath, stdout, output)
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return nil, errors.New("input is empty")
		}
		return data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

func decodeOne(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func buildConfig(raw inputConfig) (sorter.Config, *uint64, error) {
	cfg := sorter.Config{
		QueueOrder: raw.QueueOrder,
		EyeOrder:   raw.EyeOrder,
	}
	if raw.ReorderWindow != nil {
		cfg.ReorderWindow = *raw.ReorderWindow
	}
	if raw.TimeoutMS != nil {
		const maxMilliseconds = int64((1<<63 - 1) / int64(time.Millisecond))
		if *raw.TimeoutMS < 0 {
			return sorter.Config{}, nil, errors.New("config.timeout_ms must not be negative")
		}
		if *raw.TimeoutMS > maxMilliseconds {
			return sorter.Config{}, nil, errors.New("config.timeout_ms is too large")
		}
		cfg.Timeout = time.Duration(*raw.TimeoutMS) * time.Millisecond
	}
	return cfg, raw.LatestFrame, nil
}

func encodeReport(report sorter.Report, pretty bool) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(report); err != nil {
		return nil, fmt.Errorf("encode report: %w", err)
	}
	return buffer.Bytes(), nil
}

func writeOutput(path string, stdout io.Writer, data []byte) error {
	if path == "-" {
		if _, err := stdout.Write(data); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
		return nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
