# eyebw — VR 眼缓冲带宽台账

纯 Go 标准库实现，用来回答每一帧在颜色（color）、深度（depth）、历史（history）和补绘（repaint）四类数据上到底发生了多少逻辑读写与实际传输，并对比：

- `separate_eye_buffers`：两眼独立 eye-buffer；
- `binocular_atlas`：双眼打包进同一 atlas；
- `single_eye_reuse`：渲染一只眼，另一只眼通过 warp/缓存/补绘生成。

实现不依赖任何额外组件，`go test ./...` 与 `go run ./cmd/eyebw` 即可运行。

## 计量口径

每个阶段访问都记录为一条 `Access`：

- `LogicalBytes`：阶段请求读/写的字节，包含缓存命中的访问；
- `TransferBytes`：真正进入内存/总线/显示链路的字节；
- `CacheHit=true` 的读取，`TransferBytes=0`；
- 写采用 write-through 口径：写本身计一次传输，随后的缓存命中读不再计传输；
- scanout/显示读取标记为不可缓存，即使 cache 中存在同一 front buffer，也仍计显示传输；
- LRU 容量可配置，缓存驱逐后的再次 miss 会重新计一次真实传输。

因此“命中字节”只表示避免了多少传输，不会被重复加进带宽总量。MSAA 按 `width × height × MSAA samples × bytes/sample` 计算多重采样颜色/深度；resolve 后再按单采样 texel 计量。动态注视点使用瓦片最近距离判定中心交集，中心契约内瓦片必须保持 divisor=1、完整 MSAA 和双眼独立渲染/补绘。

## 三条管线的预期结论

| 管线 | 带宽特征 | 质量约束 |
|---|---|---|
| 独立 eye-buffer | 两眼各有完整颜色、深度、历史和 front buffer | 天然双眼独立 |
| 双眼 atlas | 分配/局部性不同；无 gap 时像素读写总量与独立 buffer 相同，atlas 不会凭空消除立体像素 | 天然双眼独立 |
| 单眼复用 | 外围省略第二眼 color/depth/history 渲染，改为 source 读取、warp 输出和空洞/注视点升级补绘 | 中心区必须双眼全速率、全 MSAA、独立渲染或完整补绘 |

安全单眼复用通常降低总传输；如果打开 `SingleEye.AggressiveCenterReuse` 直接复用中心像素，带宽会更低，但 `QualityReport.Passed=false`。`RejectSilentBandwidthSaving` 会拒绝这种“带宽下降、中心画质静默下降”的结果。

对于真实截图，`CompareCenterQuality` 可用标准库 `image` 计算中心区域 PSNR/SSIM；半分辨率后最近邻放大等静默模糊会触发探针违规。

## 运行

```sh
go test ./...
go run ./cmd/eyebw -out artifacts/eyebw
```

常用参数：

- `-events`：额外输出每条 access 到 `events.csv`；高分辨率场景文件可能很大；
- `-matrix`：运行分辨率 `2064×2208 / 3200² / 4096²`、MSAA `1/2/4/8`、开/关注视点的完整矩阵；
- `-strict-budget`：存在预算超限时以状态码 `2` 退出；
- `-out DIR`：报告目录。

默认场景包含 `2400² MSAA4`、`3200² MSAA4`、`4096² MSAA8`、动态 sweep gaze，以及一个预期被质量闸门拦截的激进中心复用陷阱。

## 输出文件

- `report.json`：所有场景、逐帧、三管线、阶段/类别/眼睛聚合、热点、预算和质量报告；
- `frames.csv`：逐帧读写、传输、命中率、预算、热点和补绘 texel；
- `stages.csv`：跨帧阶段热点排名；
- `classes.csv`：颜色、深度、历史、补绘的读/写和缓存命中；
- `summary.md`：中文摘要；
- `events.csv`：仅 `-events` 时生成的原始事件台账。

## 接入真实渲染器

模型只是确定性参考负载；真实引擎/驱动 trace 可以直接使用同一本台账：

```go
cache := eyebw.NewCache(32 * eyebw.MebiByte)
meter := eyebw.NewMeter(frameIndex, eyebw.PipelineAtlas, cache)

meter.Write(eyebw.StageDepthPrepass, eyebw.EyeLeft,
    eyebw.ClassDepth, "depth:left:tile-12-8", depthBytes)
meter.Read(eyebw.StageOpaquePass, eyebw.EyeLeft,
    eyebw.ClassDepth, "depth:left:tile-12-8", depthBytes)
meter.ReadUncached(eyebw.StageScanout, eyebw.EyeLeft,
    eyebw.ClassColor, "front:left", "display", frontBytes)

report := eyebw.AggregateAccesses(meter.Snapshot())
hotspots := eyebw.BuildHotspots(report.Stages)
budget := eyebw.CheckBudget(cfg.Budget, report.TransferTotal(), report.Stages)
```

把真实 RTV/DSV、history texture、patch upload、resolve、warp 和 scanout 事件映射到四个 `DataClass` 与阶段，即可复用预算、热点、缓存和质量检查。

## 模型边界

这里的字节系数和阶段顺序是可配置的保守参考模型，不替代 GPU 计数器或总线分析仪。它的价值是统一口径：逻辑流量、缓存命中、实际传输、预算和中心质量必须在同一份逐帧台账中对账。
