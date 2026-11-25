package main

type ImageSet struct {
	DirectOuter string `json:"direct_outer_png"`
	Trimmed     string `json:"trimmed_png"`
	OBJReload   string `json:"obj_reload_png"`
}

type CaseReport struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Description        string            `json:"description"`
	TrimCurveCount     int               `json:"trim_curve_count"`
	ExtractedLoops     int               `json:"extracted_loops"`
	LoopVertexCounts   []int             `json:"loop_vertex_counts"`
	LoopCurveIDs       [][]string        `json:"loop_curve_ids"`
	CrossingCount      int               `json:"self_crossing_count"`
	Repaired           bool              `json:"repaired"`
	RepairDescription  string            `json:"repair_description,omitempty"`
	OuterVertexCount   int               `json:"outer_vertex_count"`
	HoleVertexCounts   []int             `json:"hole_vertex_counts"`
	CorrectTriangles   int               `json:"correct_triangle_count"`
	CorrectInvalid     int               `json:"correct_invalid_count"`
	CorrectInvalidTris []InvalidTriangle `json:"correct_invalid_triangles,omitempty"`
	NaiveTriangles     int               `json:"naive_outer_triangle_count"`
	NaiveInvalid       int               `json:"naive_invalid_count"`
	NaiveInvalidTris   []InvalidTriangle `json:"naive_invalid_triangles,omitempty"`
	OBJPath            string            `json:"obj_path"`
	OBJVertices        int               `json:"obj_vertices"`
	OBJFaces           int               `json:"obj_faces"`
	OBJBoundaryEdges   int               `json:"obj_boundary_edges"`
	OBJReloadInvalid   int               `json:"obj_reload_invalid_count"`
	Images             ImageSet          `json:"images"`
}

type Report struct {
	StandardLibraryOnly bool         `json:"standard_library_only"`
	CoordinateSystem    string       `json:"coordinate_system"`
	Cases               []CaseReport `json:"cases"`
}
