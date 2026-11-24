# pomtrace — 高度场视差遮挡搜索

纯 Go 标准库实现的视差遮挡映射（POM）射线搜索与诊断工具，用来避免掠射角下
固定层数提前停止、纹理看起来浮在几何表面上的问题。

## 坐标约定

- `UV` 是切线空间纹理坐标，内置 `HeightMap` 使用 `[0,1]²` 严格边界并支持双线性采样。
- 高度场值 `HeightAt(uv)` 是归一化高度 `[0,1]`；对应表面深度为 `1-height`。
- 射线是深度的线性函数：
  `UV(d) = GeometryUV + UVVelocity * d`。
- `UVVelocity.X < 0` 即反向视线；掠射角对应较大的 `UVVelocity`。
- `d=0` 对应顶点插值得到的几何表面坐标。搜索失败时结果固定为
  `UV=GeometryUV`、`Depth=0`、`HitLayer=-1`。
- `Ray.PreviousUV` 只用于固化“上一帧坐标不能参与兜底”的回归测试；搜索器
  从不读取该字段，`Trace` 也没有跨帧可变状态。

## 三种搜索

1. `fixed` / `FixedStepSearch`：把深度区间分成 `Layers` 个固定层，返回第一个
   射线深度进入高度场的层。
2. `binary` / `BinaryRefineSearch`：固定步长找到跨越区间 `[outside, inside]`
   后，做固定次数二分细化；命中层仍报告粗搜首次穿越层。
3. `max-iteration` / `MaxIterationSearch`：继续二分直到深度区间小于
   `Tolerance`，或总高度采样数达到 `MaxIterations`。达到硬上限仍未收敛即视为
   失败并回到几何表面。

所有模式都遵守 `Config.MaxIterations`；每次高度场采样计为一次迭代。UV 越界、
高度为 `NaN/Inf` 或超出 `[MinHeight,MaxHeight]`、射线深度内无交点、迭代预算
耗尽，都会返回 `StatusFallbackGeometry`。

## 运行

```bash
go run . -format table
go run . -format json -out artifacts/result.json
go run . -format csv  -out artifacts/result.csv
go run . -format table > artifacts/report.txt
go test ./...
go vet ./...
```

输出列包含：

- `hit_layer` / `LAYER`：首次穿越的固定深度层，兜底为 `-1`。
- `iterations` / `ITERS`：高度采样总次数。
- `binary_steps` / `BINARY`：二分采样次数。
- `depth`、`expected_depth`、`depth_error`：命中深度、期望深度和绝对误差。
- `status`、`fallback`、`reason`：命中或几何兜底及失败原因。

## 内置场景

- `plane_forward`：常量平面，精确交点深度 `0.6`、命中第 `5` 层。
- `steep_slope`：高度梯度为 `4` 的陡坡，精确交点深度 `0.1`、命中第 `1` 层。
- `reverse_view`：负向 UV 速度，验证反向视线朝较小 U 搜索。
- `height_out_of_bounds`：第二次步进后返回高度 `1.25`，验证中途采样失败也
  必须丢弃内部 `previous/current` 括号并回到当前几何 UV。
- `grazing_uv_exit`：掠射速度离开严格 UV 域，验证越界兜底。

测试还覆盖负高度、深度范围耗尽、硬迭代上限、任意预算不超限、连续两帧结果
独立，以及 JSON/CSV 输出。

## 示例结果

在 `Layers=8`、二分 `10` 次或最大迭代容差 `1e-9` 下：

- 固定步长的平面/陡坡深度误差为 `0.025`。
- 固定 10 次二分后误差约 `9.77e-5`。
- 最大迭代搜索误差约 `5.6e-10`（返回可靠的内侧括号端点）。
- 高度或 UV 越界时三种搜索均输出 `hit_layer=-1`，且 UV 精确等于当前
  `GeometryUV`，不会使用 `PreviousUV`。
