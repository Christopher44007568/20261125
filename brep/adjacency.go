package brep

// edgeKey orders two vertex indices into a direction-independent map key.
func edgeKey(a, b int) uint64 {
	if a > b {
		a, b = b, a
	}
	return (uint64(uint32(a)) << 32) | uint64(uint32(b))
}

// edgeFaces maps each undirected manifold edge to the (one or two) faces
// incident to it.
type edgeFaces map[uint64][]int

func (b *BRep) buildEdgeFaces() edgeFaces {
	ef := edgeFaces{}
	register := func(faceIdx int, loop []int) {
		m := len(loop)
		for i := 0; i < m; i++ {
			key := edgeKey(loop[i], loop[(i+1)%m])
			ef[key] = append(ef[key], faceIdx)
		}
	}
	for i, f := range b.Faces {
		register(i, f.Outer)
		for _, hole := range f.Holes {
			register(i, hole)
		}
	}
	return ef
}

// EdgeFaces is cached lazily by section extraction.
func (b *BRep) EdgeFaces() edgeFaces {
	if b.edgeCache == nil {
		b.edgeCache = b.buildEdgeFaces()
	}
	return b.edgeCache
}
