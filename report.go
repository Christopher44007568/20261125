package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeReports(out string, d reportData) error {
	j, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	j = append(j, '\n')
	if err := os.WriteFile(filepath.Join(out, "metrics.json"), j, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "report.html"), []byte(reportHTML(d)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("README.md", []byte(reportREADME(d)), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote artifacts/metrics.json, artifacts/report.html, README.md")
	return nil
}

func reportREADME(d reportData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# BRep-style sparse-cloud surface preview\n\n")
	fmt.Fprintf(&b, "纯 Go 标准库的确定性曲面拟合演示。输入为随机顺序的稀疏扫描点云，")
	fmt.Fprintf(&b, "拟合结果是带裁剪边界和四块面缝的 B-spline BRep 风格预览壳。\n\n")
	fmt.Fprintf(&b, "## Run\n\n```bash\ngo run . -out artifacts\n```\n\n")
	fmt.Fprintf(&b, "也可直接打开 `artifacts/report.html`，或查看单张 SVG。生成过程不需要浏览器、字体包、图像库或网络。\n\n")
	fmt.Fprintf(&b, "## Gallery\n\n")
	fmt.Fprintf(&b, "[打开完整视觉报告](artifacts/report.html)\n\n")
	fmt.Fprintf(&b, "![fitted mesh](artifacts/02_fitted_mesh.svg)\n\n")
	fmt.Fprintf(&b, "![normal error heatmap](artifacts/03_normal_error.svg)\n\n")
	fmt.Fprintf(&b, "## Outputs\n\n")
	fmt.Fprintf(&b, "- `artifacts/01_pointcloud.svg`：稀疏散点图；清洁点青色、噪声橙色、遮挡空洞紫色。\n")
	fmt.Fprintf(&b, "- `artifacts/02_fitted_mesh.svg`：拟合网格、等参线、四象限面缝和粗边界。\n")
	fmt.Fprintf(&b, "- `artifacts/03_normal_error.svg`：逐面法线角误差热图、直方图与 p95 标尺。\n")
	fmt.Fprintf(&b, "- `artifacts/04_view_a.svg`、`05_view_b.svg`：两个旋转视图，实线拟合轮廓与虚线参考轮廓叠合。\n")
	fmt.Fprintf(&b, "- `artifacts/06_convergence.svg`：控制网分辨率阶梯与独立验收指标。\n")
	fmt.Fprintf(&b, "- `artifacts/metrics.json`：全部验收数值。\n\n")
	fmt.Fprintf(&b, "## Method\n\n")
	fmt.Fprintf(&b, "1. 合成参考壳生成随机稀疏扫描；24 个点加入横向/高度离群噪声，三块椭圆区域整体缺失作为扫描遮挡。\n")
	fmt.Fprintf(&b, "2. 从点云估计裁剪椭圆，不读取参考面。对高度场做 clamped 三次张量积 B-spline 最小二乘。\n")
	fmt.Fprintf(&b, "3. 法方程加入二阶差分平滑惩罚，并使用保守 Tukey IRLS 抑制噪声。噪声和空洞都不直接从拟合集合中手工删除。\n")
	fmt.Fprintf(&b, "4. 拟合完成后才启用稠密参考网格审计：高度、法线、平均曲率、4 mm 高频波纹、投影轮廓 IoU 和边界 Chamfer。\n")
	fmt.Fprintf(&b, "5. 逐级加密控制网；只有轮廓、曲率和波纹同步改善时才接受更高分辨率。\n\n")
	fmt.Fprintf(&b, "## Selected result\n\n")
	fmt.Fprintf(&b, "| 指标 | 数值 |\n|---|---:|\n")
	fmt.Fprintf(&b, "| 控制网 | **%s**, degree 3 |\n", d.Fit.ControlNet)
	fmt.Fprintf(&b, "| 输入 | %d clean + %d noise, %d occlusion voids |\n", d.Input.Clean, d.Input.Noise, d.Input.OcclusionVoids)
	fmt.Fprintf(&b, "| 控制点/内点残差 | %.3f mm RMS |\n", d.Fit.InlierRMSmm)
	fmt.Fprintf(&b, "| 稠密高度误差 | %.3f mm RMS（最大 %.3f mm） |\n", d.Audit.DenseRMSEmm, d.Audit.DenseMaxAbsmm)
	fmt.Fprintf(&b, "| 法线误差 | %.2f° RMS，%.2f° p95，%.2f° max |\n", d.Audit.NormalRMSdeg, d.Audit.NormalP95deg, d.Audit.NormalMaxdeg)
	fmt.Fprintf(&b, "| 平均曲率误差 | %.1f%% RMS |\n", d.Audit.CurvatureRelPct)
	fmt.Fprintf(&b, "| 4 mm 波纹能量 | %.1f%% |\n", d.Audit.RippleRelPct)
	fmt.Fprintf(&b, "| 轮廓 IoU / Chamfer | %.5f / %.3f mm |\n", d.Audit.SilhouetteIoU, d.Audit.SilhouetteChamferMM)
	fmt.Fprintf(&b, "\n继续加密到 `14 × 10` 只把稠密高度误差降低 %.1f%%，但曲率误差变化 %+.1f%%、波纹变化 %+.1f%%，轮廓 IoU 变化 %+.5f。", d.Selection.HeightGainFromNextPct, d.Selection.CurvatureChangePct, d.Selection.RippleChangePct, d.Selection.SilhouetteChange)
	fmt.Fprintf(&b, "因此选择 `%s`，而不是用更低控制残差换取不可见波纹。\n\n", d.Selection.Selected)
	fmt.Fprintf(&b, "## Acceptance\n\n")
	fmt.Fprintf(&b, "`go test ./...` 会检查 B-spline 单位分解、平面恢复、鲁棒拟合的稠密误差阈值、轮廓 IoU、法线 p95、曲率误差和波纹上限，并验证加密阶梯的收敛趋势。\n")
	fmt.Fprintf(&b, "\n参考面只用于扫描合成和拟合后的稠密审计；拟合函数只接收 `[]ScanPoint`。遮挡掩码用于显示，不传给拟合器。\n")
	return b.String()
}
