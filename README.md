# 20261125

Repository initialized for 20261125.

## Notification scanout frame

这个模块用 Go 标准库模拟系统通知在帧截止、提交和 scanout 之间的版本一致性。

- `State` 使用单调 `Revision`；旧快照、重复版本和旧世界帧中的通知副本会被拒绝。
- `Schedule.Deadline` 前到达的输入进入当前帧；晚到输入只进入目标后续帧。
- 连续更新按通知 ID 合并为最新版本，同时保留每次注入及 `superseded` 记录。
- `Commit` 生成快照式 `Frame`；之后的晚到输入不会修改已提交帧或 scanout 结果。
- `Closed` 是关闭墓碑，会持续进入后续帧，防止旧的 open 快照让通知重新出现。
- `Diff` 区分 `position`、`opacity` 和 `closed` 变化；报告同时记录旧版、提交版、最终版以及两两差异。

运行测试：

```sh
go test ./...
```

生成 JSON 和 Markdown 记录：

```sh
go run ./cmd/notifyframe
```

默认输出：

- `artifacts/notifyframe/report.json`
- `artifacts/notifyframe/report.md`

也可以用 `go run ./cmd/notifyframe -out -` 输出 JSON 到标准输出。
