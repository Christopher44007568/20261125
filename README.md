# XR Fixed-Path Replay Lab

纯 Go 标准库实现，不依赖第三方组件。它用同一条确定性 XR 路径连续覆盖：
静止、快转、前移、遮挡揭示、追踪丢失/重定位，并在路径中加入暂停恢复和参数热重载。

运行后会完整录制第一趟、重新运行第二趟、落盘 JSONL 与双眼 PNG，再从磁盘读回逐项比对。

## 运行

```sh
go test ./...
go test -race ./...
go run ./cmd/xrpath
```

默认输出到 `artifacts/xrpath/`：

- `baseline/frames.jsonl`：第一趟逐帧记录。
- `baseline/images/left-XXXX.png`、`right-XXXX.png`：每帧双眼画面。
- `replay/`：全新重跑后的第二趟记录和画面。
- `comparison.json` / `comparison.md`：逐项比较与首个分叉。
- `fault-injections.json`：八类故障注入的定位结果和“哈希盲区”反证。

手动注入单个故障：

```sh
go run ./cmd/xrpath fault -kind resource_missing -at 22 -target tex-crate-blue
go run ./cmd/xrpath fault -kind pose_drift -at 14
```

也可以分两次进程录制，再从磁盘跨进程比对：

```sh
go run ./cmd/xrpath record -out artifacts/run-a
go run ./cmd/xrpath record -out artifacts/run-b
go run ./cmd/xrpath compare -a artifacts/run-a -b artifacts/run-b \
  -report artifacts/run-a-vs-b.json
```

## 固定路径

共 72 帧、60 Hz，所有几何和哈希均由固定帧号推导，不使用墙钟时间或随机数。

| 帧范围 | 阶段 | 验证点 |
| --- | --- | --- |
| 0–11 | `static` | 姿态和角速度完全静止 |
| 12–19 | `fast_rotation` | 峰值偏航角速度超过 20 rad/s |
| 20–37 | `forward` | 沿路径前移超过 3 m |
| 38–51 | `occlusion_reveal` | `reveal-relic` 从不可见变为有实际像素和屏幕包围盒 |
| 52–71 | `relocalization` | `lost -> relocalizing -> relocalized -> tracking` |

第 25–28 帧暂停，姿态和速度冻结；第 27 帧暂存 revision 2 参数，第 29 帧恢复时生效。

## 每帧记录

`FrameRecord` 包含：

- `pose`：位置、四元数、Euler 角、线速度和角速度。
- `parameters`：FOV、近远裁剪面、IPD、曝光、重投影策略和 revision。
- `tracking`：追踪状态、质量、匹配键帧、上一个好帧。
- `resources[]`：网格、纹理、warp mesh、重定位键帧的 revision、内容哈希、状态、驻留和引用。
- `visibility.left/right[]`：由 CPU 光线投射真实命中的对象、AABB、深度、像素数和屏幕矩形。
- `reprojection[2]`：每只眼的来源帧/键帧、来源姿态、来源画面哈希、warp mesh 哈希、置信度和新暴露像素数。
- `images.left/right`：分辨率、原始 RGBA 像素 SHA-256、PNG 字节 SHA-256 和文件路径。
- `frame_sha256`：上述规范化记录的哈希。

## 比对和首个分叉

比较器从磁盘读回两趟 JSONL，按帧号升序，并在每一帧内按因果顺序比较：

1. `frame_sequence`
2. `lifecycle`
3. `parameters`
4. `tracking_state`
5. `pose`
6. `resource_state`
7. `visible_set`
8. `reprojection_source`
9. `left_image`
10. `right_image`
11. `frame_hash`

任何分叉都会得到具体帧、子系统、条目和字段，例如：

```json
{
  "frame": 22,
  "subsystem": "resource_state",
  "item": "resource:tex-crate-blue",
  "field": "status",
  "expected": "ready",
  "actual": "missing"
}
```

图片哈希只作为其中一个比较项，绝不会短路姿态、资源或可见几何检查。测试
`TestIdenticalImageHashesDoNotMaskGeometryOrResources` 会故意保持左右眼哈希不变，再篡改姿态和资源，
确认比较器仍分别报告 `pose` 与 `resource_state` 分叉。

## 内置故障定位矩阵

| 帧 | 故障 | 首个子系统 |
| ---: | --- | --- |
| 14 | `pose_drift` | `pose` |
| 22 | `resource_missing` | `resource_state` |
| 38 | `visibility_drop` | `visible_set` |
| 40 | `reprojection_stale` | `reprojection_source` |
| 33 | `left_image_pixel_corruption` | `left_image` |
| 26 | `lifecycle_skip_pause_frame` | `lifecycle` |
| 29 | `parameters_stale_revision` | `parameters` |
| 56 | `tracking_wrong_keyframe` | `tracking_state` |

默认演示会逐项验证这些预期；如果注入后的第一个分叉不是指定帧和子系统，命令以非零状态退出。
