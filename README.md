# 反向参数曲面缝拼接（Go，零外部依赖）

这个示例构造两片共享同一条空间边的参数曲面：A 片的共享边参数沿 canonical `q=0..1`
正向采样，B 片使用原生参数 `t=0..1`，但在几何上映射为 `q=1-t`，因此两边参数方向相反。
两侧采样密度不同（默认 A 为 4 段、B 为 18 段），在曲边和 T-junction 处会产生放大可见的细缝。

## 运行

```bash
go run . -out artifacts
go test ./...
```

常用参数：

```bash
go run . -seg-a 3 -seg-b 22 -bow 2.6 -supersample 2
```

只使用 Go 标准库；`go.mod` 中没有外部依赖。

## 产物

- `artifacts/normal_view.png`：普通正交视图和缝合报告卡。
- `artifacts/zoom_10x.png`：以普通视图比例的 10 倍放大裂缝最大位置，左为修复前，右为修复后。
- `artifacts/crack_heatmap.png`：canonical `q` 上的裂缝距离热图、定量曲线和色标。
- `artifacts/stitch_report.json`：统一采样、新增三角形、边 incidence、锐边法线和裂缝指标。
- `artifacts/stitched_mesh.obj`：修复后的位置焊接网格。

## 实现要点

1. **边界方向统一**：`UnifyBoundarySamples` 把 A 的正向 `q` 与 B 的 `q=1-t`
   映射到同一条主边 `SharedEdge(q)`，合并采样点并按弦差自适应加密。
2. **缝边三角形补齐**：`RefineBoundaryChain` 找到每条旧边界边唯一的邻接三角形，
   按统一采样插入新顶点，并用同一个对顶点生成扇形三角形替换旧三角形。
   这不是只改顶点表：旧的粗边和 T-junction 被真实的新边/新三角形替代。
3. **位置焊接与拓扑检查**：`StitchPatches` 让两侧统一采样点引用相同位置 ID。
   `CheckWatertight` 逐条统计无向边的三角形引用数和绕向；接缝边必须恰好有两个、
   绕向相反的邻接三角形。
4. **锐边法线保留**：`ComputeCornerNormals` 按平滑组和二面角建立角点法线并查集。
   接缝位置虽然共享位置 ID，但当两侧法线夹角超过 `hard_angle_deg` 或属于不同组时，
   法线不会跨缝平均。默认模型的法线跃变约 `45°`。
5. **裂缝量化**：`MeasureCracks` 在密集 canonical `q` 上计算两条边界折线的对应距离
   和点到折线距离。默认参数修复前最大裂缝约 `0.196 mm`，修复后两侧共享同一顶点，
   裂缝宽度为 `0`。

## 水密口径

两片矩形曲面只共享一条边，因此缝合后仍是一张“带完整外轮廓的曲面片”：

- `watertight` / `seam_watertight = true`：目标接缝上的每条边都有两个反向绕向的三角形，
  不存在 T-junction 裂缝边。
- `closed_manifold = false`：曲面片其余四条外边界是预期的单边轮廓，
  报告在 `outer_boundary_edges` 中单独计数。

测试 `TestWeldingVerticesAloneDoesNotMakeTJunctionsWatertight` 专门构造了只焊接位置、
但没有分裂粗边和补三角形的 T-junction，并验证水密检查会拒绝这种“假缝合”。
