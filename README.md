# 20261125

## 双眼透明材质不一致因果分类器（Go，零外部依赖）

针对 VR/立体渲染中“一只眼透明材质正确、另一只眼错误”的问题，用**已知深度、颜色、
覆盖率的标记像素**建立证据驱动的分类器。只使用 Go 标准库，无需安装任何额外组件。

### 诊断流程

1. **特征提取**（`internal/features`）：逐标记、逐眼计算颜色距离、深度/顶点偏差、
   覆盖率与边缘覆盖率偏差、采样缺口、绘制顺序错位、等深度排序歧义、视差/双眼视差
   误差、Alpha 混合反演拟合、预乘 Alpha 误差。
2. **四原因独立打分**（`internal/classifier`）：排序、视差、Alpha、抗锯齿各有独立
   加权证据与触发阈值（默认 0.55），同一眼可同时命中多个原因（多类叠加）。
3. **阶段定位**：视差 → 顶点阶段；排序/Alpha → 图层阶段；抗锯齿 → 采样阶段。
4. **聚合报告**（`internal/report`）：按原因统计双眼命中、标记清单、定位阶段、
   预算降级因子，并输出 text 或 JSON。

### 关键设计约束

- **总像素差不参与因果判定**：`totalPixelDiff` 仅作为上下文展示。测试
  `TestEqualPixelDiffDifferentCauses` 构造像素差完全相同但原因不同的两个标记，
  证明分类结果来自证据而非差异量。
- **单眼正确/单眼错误**：每只眼独立打分，报告标记 `asymmetric`（见 `one_eye` 场景）。
- **多类叠加**：守卫规则只压制弱证据，强覆盖率/采样证据可与排序/视差共存
  （见 `multi` 场景）。
- **预算降级不臆造原因**：深度预载关闭、过载超限、MSAA 削减、Alpha-to-Coverage
  等只作为已命中原因的降级因子附加；干净帧即使处于降级状态也保持零标签。

### 目录结构

```
main.go                     CLI：generate / analyze / list
internal/model/             标记像素模型、四原因与三阶段枚举、阈值
internal/features/          逐眼证据信号提取
internal/classifier/        四原因独立打分、多标签、守卫与降级因子
internal/report/            帧级聚合与 text/json 报告
internal/synth/             确定性合成场景（9 种，含故障注入）
```

### 使用

```bash
go test ./...                       # 全部测试（含覆盖率）
go run . list                       # 列出内置场景
go run . generate -scenario multi   # 生成合成 case JSON（默认 stdout）
go run . generate -scenario budget -out frame.json
cat frame.json | go run . analyze                    # 中文文字报告
go run . analyze -in frame.json -format json         # 机器可读 JSON
go run . analyze -in frame.json -fire 0.6            # 自定义触发阈值
```

输入 JSON 为 `{"name": "...", "markers": [...]}`，每个 marker 携带期望值
（`expectedDepth/expectedColor/expectedCoverage/expectedSamples/expectedOrder`）、
预算状态（`budget`）和双眼观测（`eyes`，含绘制顺序、顶点深度、视差偏移、采样数等）。

### 内置场景

`clean`、`sorting`、`parallax`、`alpha`、`aa`、`multi`（排序+Alpha、视差+AA 叠加）、
`one_eye`（仅右眼故障）、`budget`（预算降级帧）、`equal_diff_different_cause`
（等像素差、不同因果）。
