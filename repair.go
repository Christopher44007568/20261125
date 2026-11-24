package main

// SplitGroup 描述一条非流形边拆分后某个薄片分量拿到的顶点映射。
type SplitGroup struct {
	Index        int            `json:"index"`
	FaceIDs      []string       `json:"face_ids"`
	FaceIndices  []int          `json:"face_indices"`
	NewVertexMap map[string]int `json:"new_vertex_map"` // 原顶点 -> 拆分后顶点
	ReusedSpine  bool           `json:"reused_spine"`   // 第一组复用原脊边端点
}

type SplitRecord struct {
	EdgeID      string       `json:"edge_id"`
	U           int          `json:"u"`
	V           int          `json:"v"`
	GeometryKey string       `json:"geometry_key"`
	FaceIDs     []string     `json:"face_ids"`
	Groups      []SplitGroup `json:"groups"`
}

// faceComponents 以“非流形边不连接面”为原则，把面集划分为薄片连通分量。
// 返回每个面所属分量编号（按首次出现顺序编号）。
func faceComponents(m *Mesh, t *Topology) []int {
	adj := make([][]int, len(m.Faces))
	for _, e := range t.Edges {
		if e.Status == StatusNonManifold {
			continue
		}
		for i := 0; i < len(e.FaceIndices); i++ {
			for j := i + 1; j < len(e.FaceIndices); j++ {
				a, b := e.FaceIndices[i], e.FaceIndices[j]
				adj[a] = append(adj[a], b)
				adj[b] = append(adj[b], a)
			}
		}
	}
	comp := make([]int, len(m.Faces))
	for i := range comp {
		comp[i] = -1
	}
	ncomp := 0
	for start := range m.Faces {
		if comp[start] != -1 {
			continue
		}
		queue := []int{start}
		comp[start] = ncomp
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, nb := range adj[cur] {
				if comp[nb] == -1 {
					comp[nb] = ncomp
					queue = append(queue, nb)
				}
			}
		}
		ncomp++
	}
	return comp
}

// SplitNonManifoldEdges 对每条非流形边执行边界拆分：
// 第一个薄片分量复用原端点，其余分量各复制一份端点顶点。
// 不删除、不重绕任何面，面 ID 与来源保持不变。
func SplitNonManifoldEdges(m *Mesh, t *Topology) (*Mesh, []SplitRecord) {
	repaired := &Mesh{
		Schema:      m.Schema,
		Name:        m.Name,
		Description: m.Description,
		Units:       m.Units,
		Vertices:    append([]Vec3(nil), m.Vertices...),
		Faces:       make([]Face, len(m.Faces)),
	}
	for i, f := range m.Faces {
		repaired.Faces[i] = Face{
			ID:         f.ID,
			Vertices:   append([]int(nil), f.Vertices...),
			Provenance: f.Provenance,
		}
	}
	comp := faceComponents(m, t)
	// dup[comp][origVertex] = 新顶点索引，保证同一分量内多处拆分共享复制结果
	dup := map[int]map[int]int{}
	var records []SplitRecord
	for _, e := range t.NonManifoldEdges {
		rec := SplitRecord{
			EdgeID:      e.ID,
			U:           e.U,
			V:           e.V,
			GeometryKey: e.GeometryKey,
			FaceIDs:     append([]string(nil), e.Faces...),
		}
		// 该边触及的分量，按分量编号排序，首个分量复用原脊边
		compSet := map[int]bool{}
		var compOrder []int
		for _, fi := range e.FaceIndices {
			c := comp[fi]
			if !compSet[c] {
				compSet[c] = true
				compOrder = append(compOrder, c)
			}
		}
		for gi, c := range compOrder {
			group := SplitGroup{
				Index:        gi,
				NewVertexMap: map[string]int{},
				ReusedSpine:  gi == 0,
			}
			for fi, f := range repaired.Faces {
				if comp[fi] != c {
					continue
				}
				group.FaceIDs = append(group.FaceIDs, f.ID)
				group.FaceIndices = append(group.FaceIndices, fi)
			}
			for _, endpoint := range [2]int{e.U, e.V} {
				if gi == 0 {
					group.NewVertexMap[itoa(endpoint)] = endpoint
					continue
				}
				if dup[c] == nil {
					dup[c] = map[int]int{}
				}
				nv, ok := dup[c][endpoint]
				if !ok {
					nv = len(repaired.Vertices)
					repaired.Vertices = append(repaired.Vertices, repaired.Vertices[endpoint])
					dup[c][endpoint] = nv
				}
				group.NewVertexMap[itoa(endpoint)] = nv
			}
			if gi > 0 {
				for _, fi := range group.FaceIndices {
					f := &repaired.Faces[fi]
					for i, vi := range f.Vertices {
						if nv, ok := dup[c][vi]; ok {
							f.Vertices[i] = nv
						}
					}
				}
			}
			rec.Groups = append(rec.Groups, group)
		}
		records = append(records, rec)
	}
	return repaired, records
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [8]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
