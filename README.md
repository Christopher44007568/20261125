# 降级控制器（单目 / 低采样 / 无重投影）

纯 Go 标准库实现的帧渲染降级状态机、审计报告、软件渲染状态画面与交互式 Web 控制台。
不依赖任何第三方模块（`go.mod` 无 require），离线可构建。

## 运行

```sh
go test ./...                 # 状态机 / 分类 / 安全不变量 / Web API 测试
go run .                      # 生成 artifacts/degrade_report.{json,md}、trace.jsonl、9 张 PNG
go run . -out /tmp/dr         # 输出到其他目录
go run . -serve :8080         # 交互状态画面：注入故障 / 步进 / 裁剪 / 作废 / 重置
```

打开 <http://localhost:8080/>：画面由标准库 `image/png` 实时渲染（`/screen.png`），
下方按钮与键盘等价。键盘：`B` 预算压力、`F` 视锥故障、`P` 姿态超限、`T` 裁剪、
`Space` 步进、`V` 核验作废、`R` 重置。

## 三类降级与能力边界

| 模式 | 触发 | 不可越过的边界 | 恢复 |
| --- | --- | --- | --- |
| `low_sampling` | 连续 2 帧 `frame_time_ms > 11.11`，或单帧 ≥ 1.45× 预算（严重超标立即确认） | 采样档 `100% / 75% / 50%`，最低 50%，不到 0 | `≤ 8.89ms` 连续 3 帧后逐档升回；升档会重启更严格的双眼计数 |
| `monocular` | 已到 50% 采样档，且在档连续 2 帧仍超预算 | 至少 1 只眼；`eyes_zero` 请求一律拒绝并记录 | 采样先回满，且 `≤ 7.78ms` 连续 4 帧才恢复双眼 |
| `no_reprojection` | 视锥计算失败（`frustum_failed` + 机器码）或姿态数 `>16` / 非法 | 只关重投影；历史捕获、交互层、状态提示永不自动关闭 | 视锥连续 2 帧合法，或裁剪到限内后姿态连续 2 帧合法 |

每次降级都记录：触发类型与机器码、实测值、起止帧/持续帧、起始/最差/结束与逐帧画质指标、
结构化恢复条件与实时进度。

## 误报 / 连续降级 / 部分恢复

- 预算告警先以 `over_budget_unconfirmed` 打开；下一超预算帧确认，否则在恢复帧判
  `false_positive`，不启动任何降级（场景 01）。
- 已启动的片段可经外部核验 `InvalidateEpisode` 改判误报，片段与链条分类为
  `false_positive`，并立即恢复能力（场景 08）。
- 同链内升级（低采样→单目）或全恢复后 `chain_gap_frames=6` 帧内再触发，归为
  `continuous_degradation` 并重开同一条 `C0001`（场景 03、07）。
- 一个能力先回来、另一个仍降级时记 `partial_recoveries`，片段结局为
  `recovered_partial`、链分类为 `ongoing`（场景 06；场景 03 先恢复满采样）。

## 安全底线

`RequestZeroEyes` 与非显式 `RequestSilentHistoryDisable` 都会被拒绝并写入
`safety_events` / 审计（`eyes_zero_rejected`、`history_silent_disable_rejected`）。
所有场景帧保证 `eye_count >= 1`、`history/interaction/overlay == true`，
报告 `summary.invariant_checks` 给出 5 项 PASS 证据。画面始终保留按键提示与状态行，
单目时右眼位置显示虚线 `R EYE SUSPENDED` 而非黑屏。

## 代码结构

- `config.go`：全部阈值（预算、确认帧、恢复 hold、姿态上限、链宽限）。
- `types.go` / `report_types.go`：信号、状态、指标、报告 DTO（JSON 字段与审计格式）。
- `metrics.go`：FPS、预算余量、`relative_load=(eyes/2)·scale²`、画质分。
- `controller.go` / `transition.go` / `recovery.go`：状态机、升级与恢复、链归类。
- `status.go` / `actions.go`：中文状态文案/恢复进度；安全拦截、裁剪、核验作废。
- `scenarios.go`：9 个脚本场景。
- `report.go` / `invariants.go` / `markdown.go` / `markdown_scenarios.go` / `trace.go`：产物。
- `font_generated.go` / `drawfont.go` / `render.go` / `render_scene.go`：纯标准库软件渲染。
- `server.go` / `web_page.go`：JSON API、`/screen.png` 与单页控制台。
- `*_test.go`：状态机、安全不变量、汇总分类、Web API，以及与 `artifacts/degrade_report.json`
  的逐帧一致性校验（缺 oracle 文件时自动 skip）。

## HTTP API

- `GET /api/state`：当前帧报告（信号/状态/指标/中文 status_detail）+ 安全事件。
- `POST /api/step`：空体用当前注入；也可传一帧 `Signals`，如 `{"frame_time_ms":13}`。
- `POST /api/inject`：`{"kind":"budget|severe|frustum|pose|clear"}` 或 `{"frame_time_ms":…}`。
- `POST /api/trim`、`/api/reset`、`/api/invalidate`。
- `POST /api/probe/eyes-zero`、`/api/probe/history-silent`：验证安全拒绝。
- `GET /screen.png`：当前状态的 960×572 软件渲染画面。
