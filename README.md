# 20261125

## 中心凹采样时间闪烁基准（Go 标准库）

纯 Go（仅标准库）的离线仿真，用来比较三种覆盖策略在连续帧里的低采样边缘闪烁：

- `ground-truth`：每像素 4×4 抖动采样的全分辨率参考；
- `foveated-independent`：中心凹 1×1、中环 2×2、外围 4×4 分块，每眼独立采样；
- `foveated-shared`：同样的分块密度，但两只眼共享同一份覆盖缓存（双眼逐像素相同）；
- `foveated-blurred`：在独立分块结果上叠加 5×5 盒式模糊，作为“过度模糊假稳定”对照。

### 场景与运动

- 场景：`thin-triangles` 细三角形、`translucent-leaves` 半透明叶片（alpha over）、`specular-highlights` 低采样高光（加法混合）。
- 运动：`static` 静止相机、`slow-turn` 慢速转头（0.6 px/帧）、`fast-saccade` 快速扫视（7 px/帧）。
- 每只眼带 1.5 px 水平眼间距；分块采样点每帧用确定性 PCG 抖动，模拟时间上的采样不稳定。

### 记录的指标

- 闪烁像素：相邻帧亮度差超过 `tau=0.10` 的像素数，另统计其中落在外围区的数量；
- 亮度时间方差：Welford 在线方差，给出均值、外围均值和最大值；
- 时间序列：每个闪烁像素逐帧亮度写入 `timeseries-left.csv`，报告中列出方差最高的序列；
- RMSE：相对地面真值的颜色误差；
- 锐度：相邻像素亮度梯度均值，以及相对 GT 的比值，用来揭穿“模糊换来的稳定”；
- 双眼一致性：共享模式强制两眼缓冲一致（另有 `interEyeDivergence` 指标可对比独立模式）。

`false_stability=true` 仅作用于模糊对照：相对锐度跌破 0.8 且外围方差确实被压低，
说明稳定性来自细节丢失而不是更好的覆盖。正常分块策略不会被标记。

### 运行

```bash
go test ./...        # 确定性、零闪烁 GT、闪烁、假稳定守卫、双眼一致等测试
go run .             # 120x90、24 帧完整基准，约 16 秒
go run . -quick      # 48x36、8 帧快速冒烟
```

输出在 `artifacts/results/`：

- `summary.csv`：全部 3 场景 × 3 运动 × 4 策略 × 2 眼的汇总指标；
- `report.md`：Markdown 表格与最差时间序列；
- `<scene>/<motion>/<strategy>/`：末帧 PNG、闪烁热力图 PNG、逐像素时间序列 CSV。
