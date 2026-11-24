package main

import (
	"bufio"
	"os"
	"strconv"
)

func WriteOBJ(path string, m *Mesh) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.WriteString("# zero-dependency seam stitch output\n"); err != nil {
		return err
	}
	for _, p := range m.Pos {
		line := "v " + strconv.FormatFloat(p.X, 'f', 6, 64) + " " +
			strconv.FormatFloat(p.Y, 'f', 6, 64) + " " +
			strconv.FormatFloat(p.Z, 'f', 6, 64) + "\n"
		if _, err := w.WriteString(line); err != nil {
			return err
		}
	}
	for _, t := range m.Tri {
		line := "f " + strconv.Itoa(t.A+1) + " " + strconv.Itoa(t.B+1) + " " + strconv.Itoa(t.C+1) + "\n"
		if _, err := w.WriteString(line); err != nil {
			return err
		}
	}
	return w.Flush()
}
