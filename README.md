# 20261125

Repository initialized for 20261125.

## groomviz

Groom 压缩 / 分块加载演示（纯 Go 标准库，无第三方依赖）。

```sh
go build -o groomviz .
./groomviz                      # 默认: 1400 发束, 压缩 12bit, 低 LOD 6bit/5pts
./groomviz -bits 8 -lod-bits 4  # 更激进的精度
```

输出到 `out/`：

- `original.png` / `compressed.png` / `low_lod.png` — 三个精度档位的渲染，
  左下角有 VRAM 对比条形图；红框 = 缺失块，品红 = 回退发束
- `error_heatmap.png` / `error_heatmap_lod.png` — 压缩 / 低 LOD 的逐点误差热图
- `report.txt` — 显存、RMSE、最大误差、轮廓差异、缺失块列表

不变量：发根与发梢始终以 fp32 原样存储，重载后 `rootDev`/`tipDev` 恒为 0，
轮廓端点逐位不变；中间点按发束包围盒量化，不会超出原始包络。
