# BRep-style sparse-cloud surface preview

纯 Go 标准库的确定性曲面拟合演示。输入为随机顺序的稀疏扫描点云，拟合结果是带裁剪边界和四块面缝的 B-spline BRep 风格预览壳。

## Run

```bash
go run . -out artifacts
```

也可直接打开 `artifacts/report.html`，或查看单张 SVG。生成过程不需要浏览器、字体包、图像库或网络。

## Gallery

[打开完整视觉报告](artifacts/report.html)

![fitted mesh](artifacts/02_fitted_mesh.svg)

![normal error heatmap](artifacts/03_normal_error.svg)

## Outputs

- `artifacts/01_pointcloud.svg`：稀疏散点图；清洁点青色、噪声橙色、遮挡空洞紫色。
- `artifacts/02_fitted_mesh.svg`：拟合网格、等参线、四象限面缝和粗边界。
- `artifacts/03_normal_error.svg`：逐面法线角误差热图、直方图与 p95 标尺。
- `artifacts/04_view_a.svg`、`05_view_b.svg`：两个旋转视图，实线拟合轮廓与虚线参考轮廓叠合。
- `artifacts/06_convergence.svg`：控制网分辨率阶梯与独立验收指标。
- `artifacts/metrics.json`：全部验收数值。

## Method

1. 合成参考壳生成随机稀疏扫描；24 个点加入横向/高度离群噪声，三块椭圆区域整体缺失作为扫描遮挡。
2. 从点云估计裁剪椭圆，不读取参考面。对高度场做 clamped 三次张量积 B-spline 最小二乘。
3. 法方程加入二阶差分平滑惩罚，并使用保守 Tukey IRLS 抑制噪声。噪声和空洞都不直接从拟合集合中手工删除。
4. 拟合完成后才启用稠密参考网格审计：高度、法线、平均曲率、4 mm 高频波纹、投影轮廓 IoU 和边界 Chamfer。
5. 逐级加密控制网；只有轮廓、曲率和波纹同步改善时才接受更高分辨率。

## Selected result

| 指标 | 数值 |
|---|---:|
| 控制网 | **12 × 9**, degree 3 |
| 输入 | 493 clean + 24 noise, 3 occlusion voids |
| 控制点/内点残差 | 0.185 mm RMS |
| 稠密高度误差 | 0.077 mm RMS（最大 0.331 mm） |
| 法线误差 | 0.59° RMS，1.09° p95，2.06° max |
| 平均曲率误差 | 12.8% RMS |
| 4 mm 波纹能量 | 14.9% |
| 轮廓 IoU / Chamfer | 0.99865 / 0.041 mm |

继续加密到 `14 × 10` 只把稠密高度误差降低 7.5%，但曲率误差变化 +6.6%、波纹变化 +10.7%，轮廓 IoU 变化 -0.00002。因此选择 `12 × 9`，而不是用更低控制残差换取不可见波纹。

## Acceptance

`go test ./...` 会检查 B-spline 单位分解、平面恢复、鲁棒拟合的稠密误差阈值、轮廓 IoU、法线 p95、曲率误差和波纹上限，并验证加密阶梯的收敛趋势。

参考面只用于扫描合成和拟合后的稠密审计；拟合函数只接收 `[]ScanPoint`。遮挡掩码用于显示，不传给拟合器。
