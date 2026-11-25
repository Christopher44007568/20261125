# 发丝 LOD 稳定过渡演示（纯 Go 标准库）

程序用 CPU 软渲染一条固定的由近到远相机路径，输出连续帧缩略图、三角形计数、
平均亮度/发丝亮度曲线，以及帧间跳变报告。没有使用图形 SDK 或第三方依赖。

## 运行

```bash
go run .
go test ./...
```

常用参数：

```bash
go run . -frames 72 -width 320 -height 180 \
  -out artifacts/hair-lod -contact-cols 6
```

## 产物

默认写入 `artifacts/hair-lod/`：

- `frames/frame_0000.png` 到 `frame_0071.png`：连续帧缩略图。
- `contact_sheet.png`：带帧号的总览联系表。
- `metrics.csv`：相机距离、LOD 状态、混合系数、三角形数、Rec.709 平均亮度、
  发丝亮度、发丝覆盖率和相邻帧差值。
- `triangle_counts.csv`：每帧头发与场景三角形数。
- `brightness_curve.svg`：整帧平均亮度和发丝像素亮度曲线，色带为过渡窗口。
- `triangle_counts.svg`：每帧头发三角形数及三级 LOD 参考线。
- `metrics_chart.png`：上述两类曲线的纯标准库 PNG 合并预览。
- `stability_report.json` / `summary.txt`：最大亮度、覆盖率帧阶跃等汇总。

## 稳定 LOD 策略

- 发丝按 `12 个发束片（clump）→ 每片 4 个发束（lock）→ 每束 6 根发丝`
  建立稳定父子层级。
- 子级路径不是独立随机消失，而是在父级路径上叠加确定性偏移；过渡时对位置、
  宽度和颜色一起做 `smoothstep` geomorph，收敛后才切换拓扑。
- 距离窗口为 `2.35–3.45` 和 `3.90–5.05`，窗口较宽且端点导数为零。
- 固定随机种子只用于生成初始发型；运行期没有逐发丝随机剔除。
- 另有一层低多边形发壳保持整体轮廓，避免细碎缎带在远景产生空洞。

当前层级的缎带三角形为 LOD0 `6912`、LOD1 `672`、LOD2 `96`；加上恒定的
`672` 三角形发壳后，指标中的头发三角形分别为 `7584 / 1344 / 768`。

## 实现概览

- `math3d.go`：向量、固定相机路径和透视投影。
- `hair.go`：确定性层级头发模型与父子路径。
- `lod.go`：距离驱动的邻级 geomorph 状态机。
- `render.go` / `scene.go`：标准库 `image` 上的深度缓冲三角形光栅器、
  2× SSAA、头肩模型和相机面向发束。
- `metrics.go` / `chart_*.go`：指标、CSV、JSON 和 SVG 图表。
