# Fur LOD: shell / strand / hybrid

同一套毛皮几何与 groom 的纯 Go 软渲染演示。实现只使用 Go 标准库，模块没有第三方依赖。

## 输出

- `artifacts/shell.png`：64 层 shell 毛皮，远看 LOD。
- `artifacts/strand.png`：显式发丝，近看 LOD，深根、亮梢。
- `artifacts/hybrid.png`：按相机深度在 strand 与 shell 间过渡的单层混合。
- `artifacts/side-profile-diff.png`：同侧视机位的 alpha 轮廓三栏差分，附 IoU、XOR 与面积差。

## 运行

```bash
go run . -w 1200 -h 800 -out artifacts
```

默认使用 2x 超采样。快速预览可加 `-quick` 关闭超采样。

```bash
go test ./...
go list -m all
```

## 单层切换

`hybrid.go` 不会把 shell 与 strand 的 alpha 相加。混合时先只看该像素上真正有覆盖的表示，再对其 alpha 与颜色做归一化：

```text
ws = (1 - w) * present(shell)
wt =       w * present(strand)
alpha = (shell.a * ws + strand.a * wt) / (ws + wt)
```

因此：

- 过渡区最多只有一份覆盖，不会出现双层重影或额外变暗。
- 其中一侧有空洞时，另一侧有效的毛尖不会被稀释成背景幽灵。
- 深度 LOD 权重叠加低频毛束扰动，避免平直几何分界。
- 混合前对两条毛皮缓冲做一次仅作用于 RGB 的共享反照率校准，alpha 与轮廓不变。

## 毛尖与根部

- 壳层与发丝共用 `fur.go` 中的部位方向、长度、根/中/梢色和 rim light。
- 毛尖亮边来自高度加权的 tip color、轮廓 backlight、发丝末端高光与最终 bloom。
- 根部遮挡来自先把深色 undercoat 光线追踪写入画布，再由按深度排序的发丝覆盖；shell 的底层同样是不透明暗根层。

## 侧面差分

侧视相机独立渲染两张 alpha 掩膜，以 90% 覆盖轮廓比较，统一裁切到联合包围盒。差分图以琥珀色表示 shell-only、青色表示 strand-only、灰色表示重合，并输出 `IoU / XOR / area delta`。当前 1200x800 结果为 IoU `0.937`、XOR `6.3%`、面积差 `-5.4%`。
