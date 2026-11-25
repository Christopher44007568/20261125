// Command extractdata converts the upstream colour-science Python data
// snapshots (kept under thirdparty/) into the compact CSV tables embedded by
// the spectral pipeline under data/. It is a build-time helper only.
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	lambdaMin = 380.0
	lambdaMax = 780.0
	step      = 5.0
)

type point struct {
	lambda float64
	values []float64
}

func wavelengths() []float64 {
	n := int((lambdaMax-lambdaMin)/step) + 1
	out := make([]float64, n)
	for i := range out {
		out[i] = lambdaMin + float64(i)*step
	}
	return out
}

// extractDict reads `key: (v1, v2, v3),` tuple lines between startLine and
// the first line containing "}," after the start marker.
func extractTuples(path, marker string) ([]point, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entryRe := regexp.MustCompile(`^\s*(\d+):\s*(\(.*\)|[-\d.eE+ ]+),?\s*$`)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var out []point
	active := false
	for scanner.Scan() {
		line := scanner.Text()
		if !active {
			if strings.Contains(line, marker) {
				active = true
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "}") {
			break
		}
		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lambda, _ := strconv.ParseFloat(m[1], 64)
		body := strings.Trim(m[2], "() ")
		parts := strings.Split(body, ",")
		p := point{lambda: lambda, values: make([]float64, 0, len(parts))}
		for _, raw := range parts {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, fmt.Errorf("%s: bad value %q: %w", path, raw, err)
			}
			p.values = append(p.values, v)
		}
		out = append(out, p)
	}
	if !active {
		return nil, fmt.Errorf("%s: marker %q not found", path, marker)
	}
	return out, scanner.Err()
}

func interp(grid []point, lambda float64) []float64 {
	i := sort.Search(len(grid), func(i int) bool { return grid[i].lambda >= lambda })
	if i <= 0 {
		return append([]float64(nil), grid[0].values...)
	}
	if i >= len(grid) {
		return append([]float64(nil), grid[len(grid)-1].values...)
	}
	a, b := grid[i-1], grid[i]
	t := (lambda - a.lambda) / (b.lambda - a.lambda)
	out := make([]float64, len(a.values))
	for k := range out {
		out[k] = a.values[k] + t*(b.values[k]-a.values[k])
	}
	return out
}

func writeCSV(path string, header []string, grid []point, cols int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, strings.Join(header, ","))
	for _, lambda := range wavelengths() {
		values := interp(grid, lambda)
		fields := []string{strconv.FormatFloat(lambda, 'f', 0, 64)}
		for k := 0; k < cols; k++ {
			fields = append(fields, strconv.FormatFloat(values[k], 'g', -1, 64))
		}
		fmt.Fprintln(w, strings.Join(fields, ","))
	}
	return w.Flush()
}

func main() {
	cmf, err := extractTuples("thirdparty/cmfs.py", `"CIE 1931 2 Degree Standard Observer": {`)
	if err != nil {
		fail(err)
	}
	if err := writeCSV("data/cie1931_2deg_5nm.csv", []string{"wavelength_nm", "xbar", "ybar", "zbar"}, cmf, 3); err != nil {
		fail(err)
	}

	d65, err := extractTuples("thirdparty/d65_sds.py", `'D65': {`)
	if err != nil {
		fail(err)
	}
	if err := writeCSV("data/d65_5nm.csv", []string{"wavelength_nm", "power"}, d65, 1); err != nil {
		fail(err)
	}

	fmt.Println("cmf rows:", len(cmf), "d65 rows:", len(d65))

	names, patches, err := extractColorChecker("thirdparty/colorchecker_sds.py")
	if err != nil {
		fail(err)
	}
	if err := writeReflectance("data/colorchecker_ohta_5nm.csv", names, patches); err != nil {
		fail(err)
	}
	fmt.Println("colorchecker patches:", len(patches))
}

// extractColorChecker parses the Ohta 1997 ColorChecker block, which is the
// second top-level dict of wavelength: scalar entries in the snapshot.
func extractColorChecker(path string) ([]string, map[string][]point, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	entryRe := regexp.MustCompile(`^\s*(\d+):\s*([-\d.eE+ ]+),?\s*$`)
	nameRe := regexp.MustCompile(`^\s*"([^"]+)":\s*\{\s*$`)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	patches := map[string][]point{}
	var order []string
	current := ""
	seenOhta := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "DATA_COLORCHECKER_N_OHTA") {
			seenOhta = true
		}
		if !seenOhta {
			continue
		}
		if line == "}" {
			break
		}
		if m := nameRe.FindStringSubmatch(line); m != nil {
			current = m[1]
			order = append(order, current)
			patches[current] = nil
			continue
		}
		if current == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "}") {
			current = ""
			continue
		}
		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lambda, _ := strconv.ParseFloat(m[1], 64)
		value, err := strconv.ParseFloat(strings.TrimSpace(m[2]), 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: bad reflectance %q: %w", path, m[2], err)
		}
		patches[current] = append(patches[current], point{lambda: lambda, values: []float64{value}})
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return order, patches, nil
}

func writeReflectance(path string, names []string, patches map[string][]point) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	header := append([]string{"wavelength_nm"}, names...)
	fmt.Fprintln(w, strings.Join(header, ","))
	for _, lambda := range wavelengths() {
		fields := []string{strconv.FormatFloat(lambda, 'f', 0, 64)}
		for _, name := range names {
			value := interp(patches[name], lambda)[0]
			fields = append(fields, strconv.FormatFloat(value, 'g', -1, 64))
		}
		fmt.Fprintln(w, strings.Join(fields, ","))
	}
	return w.Flush()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "extractdata:", err)
	os.Exit(1)
}
