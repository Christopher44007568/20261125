package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	outDir := flag.String("out", "artifacts", "输出目录")
	meshPath := flag.String("mesh", "", "输入网格 JSON（缺省时自动生成三薄片扇形零件）")
	autoYes := flag.Bool("y", false, "跳过交互，自动确认修复")
	noRepair := flag.Bool("no-repair", false, "只输出缺陷覆盖图，不执行修复")
	flag.Parse()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	srcPath := *meshPath
	if srcPath == "" {
		srcPath = filepath.Join(*outDir, "three_sheet_fan.json")
		fmt.Println("== 生成自包含多面体数据:", srcPath)
		if err := SaveMesh(GenerateFanMesh(), srcPath); err != nil {
			fatal(err)
		}
	}
	mesh, err := LoadMesh(srcPath)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("== 读取网格: %d 顶点 / %d 面\n", len(mesh.Vertices), len(mesh.Faces))
	topo := Analyze(mesh)
	coverage := CoverageText(mesh, topo)
	fmt.Print("\n" + coverage)
	writeFile(filepath.Join(*outDir, "defect_coverage.txt"), coverage)
	subtitle := fmt.Sprintf("修复前: %d 顶点 / %d 面 / %d 边 | 边界 %d | 流形 %d | 非流形 %d · 每面法线 N00…N%02d",
		len(mesh.Vertices), len(mesh.Faces), len(topo.Edges),
		topo.BoundaryCount, topo.ManifoldCount, topo.NonManifoldCount, len(mesh.Faces)-1)
	coverageSVG := RenderScene(mesh, renderOptions{
		Title: "缺陷覆盖图 · 三薄片共享脊边", Subtitle: subtitle, Mode: "coverage", Topo: topo,
		StatusLines: []string{"发现非流形边，等待确认", "候选: 按薄片拆分脊边", "每个面保留原几何与来源"},
		StatusColor: "#dc2626",
	})
	writeFile(filepath.Join(*outDir, "defect_coverage.svg"), coverageSVG)
	beforeSVG := RenderScene(mesh, renderOptions{
		Title: "修复前 · 拓扑边与面法线", Subtitle: subtitle, Mode: "before", Topo: topo,
		StatusLines: []string{"等待确认，尚未修改", "候选: 按薄片拆分脊边", "每个面保留原几何与来源"},
		StatusColor: "#dc2626",
	})
	writeFile(filepath.Join(*outDir, "before.svg"), beforeSVG)
	fmt.Println("== 已输出缺陷覆盖图: defect_coverage.txt / defect_coverage.svg / before.svg")
	report := &Report{
		Schema:     "meshfix.report.v1",
		SourceMesh: srcPath,
		Before:     summarize(mesh, topo),
	}
	if topo.NonManifoldCount == 0 {
		report.Status = "无需修复"
		report.FaceMappings = BuildFaceMappings(mesh, mesh, topo, topo)
		report.Safety = buildSafety(mesh, mesh, report.FaceMappings)
		finishReport(*outDir, mesh, report, coverage, coverageSVG, beforeSVG, "")
		fmt.Println("== 网格流形闭合，无需修复")
		return
	}
	if *noRepair {
		report.Status = "待确认（--no-repair）"
		report.FaceMappings = BuildFaceMappings(mesh, mesh, topo, topo)
		report.Safety = buildSafety(mesh, mesh, report.FaceMappings)
		finishReport(*outDir, mesh, report, coverage, coverageSVG, beforeSVG, "")
		fmt.Println("== 按 --no-repair 要求停止；确认后可重新运行以执行拆分")
		return
	}
	if !*autoYes && !confirmRepair(topo) {
		report.Status = "用户取消，未修改"
		report.FaceMappings = BuildFaceMappings(mesh, mesh, topo, topo)
		report.Safety = buildSafety(mesh, mesh, report.FaceMappings)
		finishReport(*outDir, mesh, report, coverage, coverageSVG, beforeSVG, "")
		fmt.Println("== 已取消：未改动任何面")
		return
	}
	repaired, splits := SplitNonManifoldEdges(mesh, topo)
	repairedPath := filepath.Join(*outDir, "three_sheet_fan.repaired.json")
	if *meshPath != "" {
		repairedPath = strings.TrimSuffix(srcPath, ".json") + ".repaired.json"
	}
	if err := SaveMesh(repaired, repairedPath); err != nil {
		fatal(err)
	}
	topoAfter := Analyze(repaired)
	mappings := BuildFaceMappings(mesh, repaired, topo, topoAfter)
	subAfter := fmt.Sprintf("修复后: %d 顶点 / %d 面 / %d 边 | 边界 %d | 流形 %d | 非流形 %d · 拆分脊边顶点已错位展示",
		len(repaired.Vertices), len(repaired.Faces), len(topoAfter.Edges),
		topoAfter.BoundaryCount, topoAfter.ManifoldCount, topoAfter.NonManifoldCount)
	afterSVG := RenderScene(repaired, renderOptions{
		Title: "修复后 · 拓扑边与面法线", Subtitle: subAfter, Mode: "after", Topo: topoAfter,
		Splits: splits, OrigVerts: len(mesh.Vertices),
		StatusLines: []string{"已确认并拆分脊边", "复制顶点 " + itoa(len(repaired.Vertices)-len(mesh.Vertices)) + " 个", "0 面被删除"},
		StatusColor: "#15803d",
	})
	writeFile(filepath.Join(*outDir, "after.svg"), afterSVG)
	report.Status = "已确认并修复"
	report.RepairConfirmed = true
	report.RepairedMesh = repairedPath
	report.After = summarize(repaired, topoAfter)
	report.Splits = splits
	report.FaceMappings = mappings
	report.Safety = buildSafety(mesh, repaired, mappings)
	prov := FaceProvenanceDoc{
		Schema: "meshfix.face-provenance.v1", Status: "repaired",
		SourceMesh: srcPath, RepairedMesh: repairedPath, FaceMappings: mappings,
	}
	if err := writeJSON(filepath.Join(*outDir, "face_provenance.json"), prov); err != nil {
		fatal(err)
	}
	finishReport(*outDir, mesh, report, coverage, coverageSVG, beforeSVG, afterSVG)
	fmt.Printf("\n== 修复完成: 拆分非流形边 %d 条，复制顶点 %d 个，删除面 0\n",
		len(splits), len(repaired.Vertices)-len(mesh.Vertices))
	fmt.Printf("== 修复后拓扑: 边界=%d 流形=%d 非流形=%d\n",
		topoAfter.BoundaryCount, topoAfter.ManifoldCount, topoAfter.NonManifoldCount)
	fmt.Println("== 面来源对应: face_provenance.json（几何/来源/法线全部保持）")
}

func confirmRepair(t *Topology) bool {
	fmt.Printf("\n检测到 %d 条非流形边。修复将按薄片分量拆分脊边（复制端点，不删面）。\n", t.NonManifoldCount)
	fmt.Print("确认执行修复? [y/N] ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

func finishReport(outDir string, m *Mesh, rep *Report, coverage, coverageSVG, beforeSVG, afterSVG string) {
	if err := writeJSON(filepath.Join(outDir, "report.json"), rep); err != nil {
		fatal(err)
	}
	html := RenderHTML(m, rep, coverage, coverageSVG, beforeSVG, afterSVG)
	writeFile(filepath.Join(outDir, "report.html"), html)
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
