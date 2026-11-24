# 临时几何缓冲生命周期模拟

纯 Go 标准库实现，无第三方依赖。目标是防止 arena 在 GPU 提交仍处于
`submitted` 状态时回收临时几何区间，导致同一段显存被新三角形覆盖，旧提交
随后画出随机三角形。

## 生命周期

提交状态只有三种：

- `submitted`：提交已在途，区间被该提交引用，不能按旧回调直接回收。
- `completed`：GPU 完成；该提交删除自己的引用，无其他持有者时才回收。
- `canceled`：取消或超时；同样只删除该提交自己的引用，并记录取消/超时原因。

每个 arena 对象都包含：

- `allocation_seq`：进程内单调递增的分配序号，区间复用后也不会重复。
- `lifecycle_state`：该对象最近关联提交的 `submitted` / `completed` /
  `canceled` 状态；`snapshot.leases` 保留每次“提交—资源”关联的状态。
- `last_used_submission_id` / `last_used_frame_id`：最后使用它的提交和帧。
- `claimed_by_submission_ids`：当前仍持有该区间的在途提交。
- `release_reason`：最终释放原因，例如 `submission_completed`、
  `submission_canceled`、`timeout_reclaimed`、`explicit_release`。
- `duplicate_release_attempts`、`stale_release_attempts`、
  `blocked_held_release_attempts`：重复释放、陈旧票据和越权释放审计计数。

## 防止旧提交释放新区间

实现同时使用两层保护：

1. **引用计数**：旧提交完成或取消时只移除自己的 claim。若新提交仍持有同一
   对象，区间继续保持 `held`，旧的显式释放返回 `ErrRangeHeld`。
2. **分配代际**：释放票据必须携带 `allocation_seq + offset + size`。即使旧
   区间已经被 arena 重新分配给新提交，旧票据也只会得到
   `ErrStaleTicket` / `ErrDuplicateRelease`，不会按 offset 释放当前区间。

`CheckInvariants` 会验证空闲区间和存活区间无缝拼满整个 capacity、不存在重叠、
所有存活 claim 都属于 `submitted` 提交，并且所有已回收对象都有释放原因和时间。

## 运行

```sh
go test -race ./...
go run ./cmd/geomlife-demo
```

CLI 默认执行：

- 确定性的双帧在途场景；
- 帧 1 完成但共享区间仍被帧 2 持有；
- 帧 3 复用同一 offset，但获得新的 `allocation_seq`；
- 旧提交的陈旧/重复释放攻击；
- 超时前不回收、超时后进入 `canceled` 并以 `timeout_reclaimed` 回收；
- 取消路径和重复释放；
- 两个真实 goroutine 在屏障后同时提交、再分别完成/取消的双帧并行场景。

默认写入 JSON 审计报告：

```sh
go run ./cmd/geomlife-demo -out artifacts/report.json
go run ./cmd/geomlife-demo -json
```

## 代码结构

- `types.go`：状态、原因、票据、对象、提交、事件和错误。
- `system.go`：线程安全的 `Arena`/`System`、快照和时钟接口。
- `alloc.go`：first-fit 区间分配、相邻空闲区间合并、代际票据和回收。
- `submission.go`：提交、绑定、完成、取消和超时扫描。
- `release.go`：显式释放、重复释放、陈旧票据和新持有者保护。
- `verify.go`：arena 区间与引用关系不变量校验。
- `canary.go`：旧顶点图案被新区间覆盖的故障对照。
- `scenario.go`：确定性完整演练。
- `parallel_scenario.go`：真实 goroutine 双帧并行演练。
- `cmd/geomlife-demo/main.go`：摘要输出和 JSON 报告。

## 最小 API 示例

```go
clock := geomlife.NewManualClock()
arena, _ := geomlife.NewArena(geomlife.Config{
    Capacity:          256,
    SubmissionTimeout: 50 * time.Millisecond,
    Clock:             clock,
})

first, tickets, _ := arena.Submit(geomlife.SubmitRequest{
    FrameID: 1,
    Sizes:   []int{64, 32},
})
second, _, _ := arena.Submit(geomlife.SubmitRequest{FrameID: 2, Sizes: []int{64}})
_, _ = arena.Bind(tickets[0], second.ID)

_, _ = arena.Complete(first.ID)  // seq=1 仍由 second 持有，不会回收
_, err := arena.Release(tickets[0], first.ID, geomlife.ReasonCompleted)
// errors.Is(err, geomlife.ErrRangeHeld) == true
```
