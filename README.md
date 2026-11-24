# 20261125 — 单眼故障回滚与共享资源保护（stereorollback）

纯 Go 标准库实现的双目渲染故障注入/回滚模拟器。核心保证：**只有一只眼失败时，
回滚只撤销该眼本次尝试产生的私有对象与租约，保住另一只眼的真实画面，并且绝不删除
仍有其他使用者的共享资源。**

无第三方依赖（`go.mod` 中没有任何 require），不需要安装额外组件。

## 运行

```sh
go test -race ./...          # 全部不变量测试
go run ./cmd/stereorollback --check
# 终端打印文本报告，并写出：
#   artifacts/rollback-report.json
#   artifacts/stereo-rollback-report.txt
#   artifacts/frames/*-f*-fallback.png   （真实 PNG 回退画面）
#   reports/...                          （同样内容的工作区副本）
```

`--check` 在任一不变量失败时以非零码退出。

## 需求对照

| 用户反馈要求 | 实现位置 | 记录字段 / 证据 |
| --- | --- | --- |
| 一只眼失败时保住另一眼，不删除整套共享资源 | `render.go` 的 `RunFrame`/`renderEye`，`transaction.go` 的 `rollback` | 每一失败帧的对侧眼 `status=rendered`；`rollback.retained_resources`；测试 `TestLeft/RightEyeFailure...` |
| 注入附件创建失败 | `FaultAttachmentCreate` + `StageAttachment`，失败时登记半成品 attachment | `failure_stage="attachment_creation"`、`fault="attachment_create_failed"`、`rollback.destroyed_private_resources` |
| 注入材质参数错误 | `material.go` 的 `MaterialParams.Validate`（如 roughness 越界） | `failure_stage="material_parameters"`、`field="roughness"` |
| 注入重投影超时 | `FaultReprojectionTimeout`，逻辑耗时超过预算；错误链包装 `context.DeadlineExceeded` | `failure_stage="reprojection"`、`elapsed_ms`、`budget_ms` |
| 记录失败阶段 | `FailureRecord.FailureStage` | JSON 键 `failure_stage` |
| 记录保留资源 | `RollbackRecord.RetainedResources` | JSON 键 `retained_resources`，含 `ref_count`/`users`/`other_users` |
| 记录回退画面 | `FallbackFrame` + `compose.go` 合成并排 PNG | JSON 键 `fallback_frame`，文件 `artifacts/frames/*.png` |
| 记录成功帧数 | `metrics.go` 的 `Counters` | `successful_frames`（仅真实渲染的眼帧），另有 `full_success_stereo_frames` 等 |
| 左眼失败单独处理 | `RunLeftEyeFailure`、`ScopeLeftEye` | 场景 `left_eye_failure`，`left_eye_failures=3 right_eye_failures=0` |
| 右眼失败单独处理 | `RunRightEyeFailure`、`ScopeRightEye` | 场景 `right_eye_failure` |
| 共享纹理损坏单独处理 | `corruption.go` 的 `handleSharedCorruption` | 场景 `shared_texture_corruption`，`scope="shared_texture_corruption"`，无 `eye`/`failure` |
| 失败眼占位图不计入成功 | `Counters.RecordSurface` 只认 `status=rendered` | 占位图 `success=false, placeholder=true`；测试 `TestPlaceholderNeverCountedAsSuccess` |
| 不清掉共享资源的其他使用者 | `resource.go` 引用计数 + 持有者名册，`Destroy` 在有租约时返回 `ErrResourceBusy` | `other_users` 含对侧眼/reprojector/recorder；测试 `TestDestroyRefusesWhileOtherUsersHoldRefs` |

## 三个确定性场景

- `left_eye_failure`（5 帧）：第 2/3/4 帧分别给左眼注入附件创建、材质参数、重投影
  超时；右眼 5 帧全部真实成功。成功眼帧 `successful_frames=7`（左 2 + 右 5），
  左眼占位 3 张且不计成功，完整双目成功帧 2。
- `right_eye_failure`：镜像，左眼 5 帧全部成功，`successful_frames=7`（左 5 + 右 2）。
- `shared_texture_corruption`（4 帧）：第 2 帧共享 atlas 校验和损坏，走独立路径隔离并
  用已知良好快照原地重建（generation 1→2），4 个使用者（双眼、reprojector、recorder）
  全部保留，depth/material/history 等无关共享资源不变；该帧双眼使用共享应急图
  （2 个 shared fallback，不计成功），第 3、4 帧恢复。

汇总：14 个双目帧、20 个真实成功眼帧、6 张失败眼占位、2 个共享应急表面，
左/右失败各 3 次、共享损坏 1 次，全部不变量 `all_invariants_hold=true`。

## 关键不变量

- 回滚只 `Destroy` 本眼 `created` 且 `shared=false` 的对象；共享 ID 有显式保护分支，
  资源池在 `ref_count>0` 时也会以 `ErrResourceBusy` 拒绝删除。
- 租约按唯一 `LeaseID` 释放，重复释放返回 `ErrUnknownLease`，不会误减他人引用。
- 成功计数只有唯一入口 `RecordSurface`：`status != rendered` 一律不计成功。
- `ResourcePool` 带互斥锁，`go test -race` 与并发租约抖动测试通过。
