package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type OBJMesh struct {
	Name      string
	Vertices  []Vec3
	Triangles [][3]int
}

func WriteTriangulatedOBJ(path, name string, mesh *Mesh2, patch NURBSSurface) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "# Pure-Go trimmed NURBS patch: %s\n", name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "o %s\n", objSafeName(name)); err != nil {
		return err
	}
	for _, p := range mesh.Vertices {
		q := patch.EvalUV(p.X/100.0, p.Y/100.0)
		if _, err := fmt.Fprintf(w, "v %.9f %.9f %.9f\n", q.X, q.Y, q.Z); err != nil {
			return err
		}
	}
	for _, tri := range mesh.Triangles {
		if _, err := fmt.Fprintf(w, "f %d %d %d\n", tri[0]+1, tri[1]+1, tri[2]+1); err != nil {
			return err
		}
	}
	return w.Flush()
}

func objSafeName(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func ReadOBJ(path string) (*OBJMesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	obj := &OBJMesh{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "o":
			if len(fields) > 1 {
				obj.Name = fields[1]
			}
		case "v":
			if len(fields) < 4 {
				return nil, fmt.Errorf("%s:%d: invalid vertex", path, lineNo)
			}
			xs, err1 := strconv.ParseFloat(fields[1], 64)
			ys, err2 := strconv.ParseFloat(fields[2], 64)
			zs, err3 := strconv.ParseFloat(fields[3], 64)
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("%s:%d: invalid float", path, lineNo)
			}
			obj.Vertices = append(obj.Vertices, Vec3{X: xs, Y: ys, Z: zs})
		case "f":
			if len(fields) < 4 {
				continue
			}
			indices := make([]int, 0, len(fields)-1)
			for _, token := range fields[1:] {
				first := strings.Split(token, "/")[0]
				idx, err := strconv.Atoi(first)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: invalid face index %q", path, lineNo, first)
				}
				if idx < 0 {
					idx = len(obj.Vertices) + idx
				} else {
					idx--
				}
				if idx < 0 || idx >= len(obj.Vertices) {
					return nil, fmt.Errorf("%s:%d: face index out of range", path, lineNo)
				}
				indices = append(indices, idx)
			}
			for i := 1; i+1 < len(indices); i++ {
				obj.Triangles = append(obj.Triangles, [3]int{indices[0], indices[i], indices[i+1]})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(obj.Vertices) == 0 || len(obj.Triangles) == 0 {
		return nil, fmt.Errorf("%s contains no mesh", path)
	}
	return obj, nil
}

type objEdge struct {
	a, b int
}

func sortedEdge(a, b int) objEdge {
	if a > b {
		a, b = b, a
	}
	return objEdge{a, b}
}

func OBJBoundaryEdges(obj *OBJMesh) [][2]Vec2 {
	counts := make(map[objEdge]int)
	for _, tri := range obj.Triangles {
		edges := [3]objEdge{sortedEdge(tri[0], tri[1]), sortedEdge(tri[1], tri[2]), sortedEdge(tri[2], tri[0])}
		for _, e := range edges {
			counts[e]++
		}
	}
	var keys []objEdge
	for e, count := range counts {
		if count == 1 {
			keys = append(keys, e)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].a != keys[j].a {
			return keys[i].a < keys[j].a
		}
		return keys[i].b < keys[j].b
	})
	segments := make([][2]Vec2, 0, len(keys))
	for _, e := range keys {
		a := obj.Vertices[e.a]
		b := obj.Vertices[e.b]
		segments = append(segments, [2]Vec2{v2(a.X, a.Y), v2(b.X, b.Y)})
	}
	return segments
}
