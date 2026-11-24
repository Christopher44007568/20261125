# 双眼会话断开清理与重连验证

纯 Go 标准库实现，无第三方依赖，也不安装任何额外组件。模拟头显双眼管线的
纹理、深度、历史、队列任务和缓存键生命周期，并验证左眼、右眼、整机三种断开
路径。

## 行为约束

- 任一眼或整机断开都会终止当前双眼会话中的全部任务：运行中的任务收到上下文
  取消，队列中的任务被取消，`Close` 等待所有 worker 退出后才释放资源。
- 所有缓冲写入都必须经过 `Lease`。租约绑定 `session_id + generation`，旧会话
  的租约对新会话返回 `ErrStaleLease`，无法写入新缓冲或新缓存。
- 重连总是创建新的会话 ID、代际、资源集、随机私有令牌和空缓存；不复制上一
  会话的历史、缓存、epoch 或父会话引用。
- 首帧与无状态基准帧逐字节比较。释放报告按对象列出纹理、深度、历史、任务和
  缓存键的释放结果，并单独列出未释放对象。

## 运行

```bash
go test -race ./...
go run ./cmd/faultcheck -out artifacts/report.json
```

默认对 `left_eye`、`right_eye`、`device` 各执行两轮“启动双眼负载 -> 断开 ->
重连 -> 旧租约攻击 -> 首帧校验”。`-cycles` 可调整每类断开的连续重连次数。

## 报告字段

- `release.categories`：`textures`、`depth`、`history`、`queue_tasks`、
  `cache_keys` 的发现/释放/失败数量、`ok` 和逐对象名称。
- `release.tasks`：断开时运行/排队任务数，以及停止、取消、失败、残留数量。
- `release.unreleased_objects` 与 `unreleased_objects`：释放后及重连后的
  未释放对象。
- `stale_writes`：旧会话租约分别写旧会话和新会话的结果，`accepted` 必须全为
  `false`。
- `private_state`：新会话令牌、epoch、父会话、历史、缓存继承检查。
- `first_frame`：左右眼首帧相对基准的哈希、差异字节数和差异像素数。
- `reconnect`：拆解、建会话和总耗时（纳秒与毫秒）。

最近一次本机运行结果：6 次重连，每轮 8 个运行任务和 2 个排队任务全部停止；
纹理/深度/历史/任务/缓存键释放失败均为 0，未释放对象为 0，首帧差异为 0，
重连总耗时为亚毫秒级；耗时取决于机器负载，以生成的 JSON 为准。
