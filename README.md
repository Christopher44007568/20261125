# 最终帧排序器

面向 `world`、`ui`、`hand`、`passthrough` 四个渲染队列的确定性最终排序器。实现只使用 Go 标准库。

排序依据是逻辑帧号 `frame`，不是队列提交或完成的先后。因此，最后到达的条目不会自动成为最新帧；比权威水位线更旧的帧会被记录为历史回填。

## 运行

```bash
go test ./...
go run ./cmd/final-sorter -in input.json -out report.json
```

省略 `-in`/`-out` 时使用标准输入/标准输出，`-pretty=false` 可关闭缩进。

## 输入

```json
{
  "config": {
    "reorder_window": 2,
    "timeout_ms": 50,
    "latest_frame": 120,
    "queue_order": ["world", "ui", "hand", "passthrough"],
    "eye_order": ["left", "right", "both"]
  },
  "submissions": [
    {
      "id": "world-120-left",
      "frame": 120,
      "queue": "world",
      "eye": "left",
      "submitted_at": "2026-11-25T10:00:00Z",
      "completed_at": "2026-11-25T10:00:00.020Z",
      "deadline": "2026-11-25T10:00:00.050Z"
    }
  ]
}
```

`config` 可省略。`reorder_window` 默认是 `0`，即只接纳最新帧；实际使用时通常应设置成流水线允许的乱序帧数。`latest_frame` 用于显式固定权威水位线；不提供时取所有合法提交中的最大 `frame`。

也可直接使用 Go API：

```go
report, err := sorter.Sort(submissions, sorter.Config{ReorderWindow: 2})
report, err = sorter.SortAt(submissions, 120, sorter.Config{ReorderWindow: 2})
```

## 判定规则

- 逻辑键是 `(frame, queue, eye)`。同一键只保留一个规范代表，其余记录为 `duplicate`。
- 重复项按 `submitted_at` 最早、`completed_at` 最早、`id` 字典序依次选择规范代表。时间缺失排在有时间的条目之后。输入切片顺序不参与选择。
- 帧号大于权威水位线时记录为 `future_frame`。`Sort` 自动推断水位线，`SortAt` 使用调用方给出的水位线，迟到异常帧不能改写它。
- 帧号落后水位线超过 `reorder_window` 时记录为 `historical_backfill`。恰好落后 `reorder_window` 帧仍可接纳。
- 完成时间晚于 `deadline`，或 `completed_at - submitted_at > timeout_ms` 时记录为 `timeout`。边界值（等于截止时间或超时上限）可接纳。
- 乱序但仍在窗口内的条目可接纳，并以 `out_of_order=true` 标记。判定依据是完成顺序中出现了更大的逻辑帧。
- 非法 `queue`/`eye` 分别记录为 `invalid_queue`、`invalid_eye`，不会静默退化成默认值。

当多条规则同时命中时，拒绝原因的优先级为：`invalid_*`、`duplicate`、`timeout`、`future_frame`、`historical_backfill`。

## 最终顺序

接纳项按以下键升序排列，并写入从 1 开始的 `adopted_order`：

1. `frame`
2. `queue_order` 中的位置，默认 `world`、`ui`、`hand`、`passthrough`
3. `eye_order` 中的位置，默认 `left`、`right`、`both`
4. `id`

报告同时给出每条接纳项的 `submission_order`、`completion_order` 和 `out_of_order`。两个序号分别由 `submitted_at` 和 `completed_at` 排序得到；完全相同的时间线值使用密集排名，因此同一输入集合换序后报告保持一致。

拒绝项记录 `reason`、`duplicate_of`（若适用）及原条目的 `eye`。拒绝列表也按逻辑键稳定排序。

## 报告结构

```json
{
  "latest_frame": 120,
  "adopted": [
    {
      "id": "world-120-left",
      "frame": 120,
      "queue": "world",
      "eye": "left",
      "adopted_order": 1,
      "submission_order": 1,
      "completion_order": 1,
      "out_of_order": false
    }
  ],
  "rejected": [
    {
      "id": "old-frame",
      "frame": 80,
      "queue": "hand",
      "eye": "right",
      "reason": "historical_backfill",
      "submission_order": 2,
      "completion_order": 2
    }
  ],
  "counts": {
    "submitted": 2,
    "adopted": 1,
    "out_of_order_adopted": 0,
    "duplicates": 0,
    "timeouts": 0,
    "historical_backfills": 1,
    "future_frames": 0,
    "invalid": 0
  }
}
```
