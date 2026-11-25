# Hair guide LOD

一个只使用 Go 标准库的毛发聚类与连续 LOD 示例。默认从 20,000 根完整发丝构建
512 个 guides，并输出完整数据、层级 LOD、覆盖率检查和误差色图。

## 运行

```bash
go test ./...
go run ./cmd/hairlod -out out
```

常用参数：

```bash
go run ./cmd/hairlod \
  -strands 20000 -guides 512 -points 32 -seed 20261125 \
  -width 720 -height 720 -analysis-size 200 -dolly-samples 25 \
  -error-distance 18 -out out
```

没有外部模块或图形库依赖。

## 实现

- **确定性发丝生成**：在头皮球冠上采样发根，避开面部区域，生成带垂坠和波动的完整发丝。
- **guide 聚类**：每根发丝取根部、1/3、2/3 和发尾四个控制点组成 12 维特征，使用加权
  k-means++ 聚类。每个 guide 的真实代表发丝是距离加权簇中心最近的成员，不虚构平均发丝。
- **密度权重**：guide 质量等于簇内完整发丝质量之和，`densityWeight = guideMass / totalMass`；
  各层总质量严格守恒。
- **层次 LOD**：512 -> 256 -> 128 -> 64 -> 32 逐级合并，每个较粗 guide 仍保留其上一层成员，
  因而可以追踪代表发丝和原始来源。
- **连续切换**：连续 LOD 坐标是 `q = log2(cameraDistance / focusDistance)`。每帧同时计算
  相邻两层密度场，并按 `q` 的小数部分做线性混合，级别边界处不会硬切。
- **屏幕空间重建**：发丝质量沿屏幕曲线累积为密度场；guide 线宽有上限，使远距轮廓保持
  可读的边界和发尾纹理，而不是把全部发丝压成一个点或单块。
- **误差色图**：在同一机位渲染完整发丝和连续 LOD，对局部密度先做平滑，再输出 alpha
  覆盖率绝对误差。红/黄色表示覆盖差异较大的区域，同时统计 IoU、P95 和最大误差。

## 输出

| 文件 | 内容 |
| --- | --- |
| `out/strands.json` | 完整 20,000 根发丝，每根 32 个点，附根部和密度质量 |
| `out/guides.json` | 512 个 guides 的代表发丝、成员 ID、源数量、质量和密度权重 |
| `out/lod.json` | 所有 LOD 层的代表发丝与权重、dolly 覆盖率、误差统计和图片索引 |
| `out/full_strands.png` | 完整发丝近景 |
| `out/guides.png` | 512 个 guide/代表发丝近景 |
| `out/lod.png` | `-error-distance` 处的连续 LOD |
| `out/full_reference.png` | 同机位的完整发丝参考图 |
| `out/error_heatmap.png` | 完整发丝与 LOD 的覆盖率误差色图 |
| `out/coverage_strip.png` | 从左到右五个后退距离的 LOD 帧 |
| `out/coverage_curve.png` | 归一化覆盖率 `C` 与轮廓 IoU `I` 随距离变化曲线 |
| `out/coverage_curve.json` | 每个采样距离的层选择、混合系数、覆盖率和 IoU |

默认 20,000 根时 `strands.json` 约 40 MB；这是完整点数据的预期大小。只需检查 LOD 时可
使用较小的 `-strands`/`-guides` 参数。

## 连续性判定

`AnalyzeDolly` 在 5 到 32 的镜头距离上采样，并要求：

- 相邻采样帧归一化覆盖率变化不超过 `0.08`；
- 最低覆盖率不低于完整发丝的 `0.55`；
- 远距轮廓 IoU 不低于 `0.45`；
- 远距覆盖率包围盒宽高均至少 4 像素。

默认配置（20k strands / 512 guides / 25 dolly samples）会输出：

```text
guides: [20000 512 256 128 64 32]
dolly:  coverage ratio 0.882..0.906, max step 0.051, far IoU 0.906, continuous=true
```
