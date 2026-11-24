# 20261125

Repository initialized for 20261125.

## stereovol — 双眼体积采样对比工具

纯 Go 标准库实现，无第三方依赖。场景包含薄烟层（z=2.0–2.25，噪声调制）、
雾墙（z=4–6）与透明发光球（z=3.2），对三种采样策略做立体对比：

- `full`：逐眼全采样（dt=0.02），作为参考真值
- `shared`：共享低采样（dt=0.16），从中央眼步进一次、透射率直接复制给双眼
- `foveated`：中心分区采样，注视锥内 dt=0.02、外周 dt=0.16，逐眼独立注视点

四种双眼差异场景：`base`、`strabismus`（右眼外偏 3°）、`headturn`
（两眼 8ms 采样间隔的快速转头）、`gaze`（右眼注视点偏移 ~5.7°）。

运行：

```sh
go run . -out artifacts            # 全部场景
go run . -scenario headturn        # 单个场景
```

每个场景输出到 `artifacts/<scenario>/`：

- `{full,shared,foveated}_{L,R}.png`：每眼渲染图
- `*_diff.png`：双眼差异图（增益 8x）
- `transmittance_curves.csv`：探针像素逐眼的 T(t) 透射曲线（长表：series,t,T）
- `report.txt`：双眼差异指标（MAE/RMSE/Max）、相对参考的误差、
  每眼探针光路（原点/方向）、透明度步长修正自检

关键结论（见各 report.txt）：共享低采样的双眼差异恒为 0（视差被抹除），
且相对逐眼参考的误差在转头/斜视时显著增大；中心分区采样以约 1/3–1/2
的误差逼近全采样。步长改变时必须做透明度修正
`alpha2 = 1 - (1-alpha1)^(dt2/dt1)`（见 `render.go` 的 `CorrectAlpha`），
直接复制会把 sigma=6 的薄烟单步不透明度从 0.617 错当成 0.113。
