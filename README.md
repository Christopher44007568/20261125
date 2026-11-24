# 双眼纹理生命周期模拟

纯 Go 标准库实现的眼缓冲生命周期追踪器，用于验证视口重建、单眼销毁和设备重建失败时，颜色、深度、历史纹理不会泄漏、悬空或提前释放。

## 运行

```bash
go test ./...
go run ./cmd/texturelife
```

无第三方依赖，也不需要安装图形驱动。

## 资源记录

每个纹理都保留完整历史，物理释放后元数据仍可查询：

- `Kind`：`color`、`depth`、`history`
- `Eye`：`left`、`right`
- `Size`、`Bytes`：尺寸和模拟占用；颜色/历史按 RGBA8、深度按 32 位计算
- `RefCount`、`PeakRefCount`：当前及峰值引用数
- `CreatedFrame`：创建帧
- `LastUsedFrame`：最后一次被帧引用的帧
- `RetireFrame`：首次请求释放的帧
- `ReleasedAt`：真正物理释放的帧；`0` 表示仍存活

`Frame` 是逻辑时间，从 1 开始。`ReleaseAt(owner, at, id)` 区分引用所属帧和释放事件帧，例如旧帧 `owner=2` 的引用可以在窗口切换帧 `at=6` 完成。

## 生命周期规则

1. `Create` 只分配元数据，不隐含引用。
2. `Acquire` 增加引用，并更新 `LastUsedFrame`。
3. `Retire` 只标记退休；引用数不为零时绝不释放。
4. 只有最后一个引用被 `Release`/`ReleaseAt` 移除后，退休纹理才记录 `ReleasedAt` 并物理释放。
5. 新帧可以先 `Acquire` 旧纹理；此时旧帧随后释放也不会让纹理提前消失。
6. `Snapshot` 审计引用图并报告孤儿分配；`FinalAudit` 把所有仍存活的纹理视为泄漏。

## 设备模拟

`Device` 为每只眼维护一组颜色、深度、历史纹理，并拥有当前帧引用：

- `Initialize(frame, size)`：建立双眼缓冲。
- `Resize(frame, size)` / `Rebuild(frame, size)`：先分配并引用新缓冲，再退休旧缓冲并释放旧帧引用。
- `DestroyEye(frame, eye)`：仅退休目标眼；另一只眼在新帧重新引用后继续存活。销毁最后一只眼会关闭设备。
- `SimulateRebuildFailure(frame, size)`：分配候选缓冲后故意失败，回滚候选资源，保留原视口和旧缓冲。
- `Shutdown(frame)`：退休并释放当前设备仍持有的缓冲。

设备的每次状态变化都要求帧号递增，避免把新帧引用与旧帧释放混在同一时间点。

## 审计输出

`Report` 提供：

- `Textures`：所有创建过的纹理及其眼别、尺寸、引用、使用/退休/释放帧
- `Live`、`Leaks`、`RetiringCount`
- `References`：帧到纹理的当前引用边
- `Dangling`：指向已释放资源的帧引用
- `Peak`、`PeakByEye`、`PeakByKind`：资源数、字节数和引用数峰值
- `Issues`：泄漏、悬空引用、引用计数不一致、带引用释放、退休状态异常

命令行场景依次执行：

1. 1280x720 双眼初始化
2. 窗口变化到 1920x1080
3. 销毁左眼
4. 2560x1440 设备重建失败并回滚
5. 新帧持有右眼旧历史纹理，再重建视口
6. 直到旧历史纹理的最后一个引用释放
7. 关闭设备并执行最终审计

预期最终结果：`live=0 leaks=0 dangling=0 issues=0`，同时峰值仍保留在报告中。

## 主要文件

- `tracker.go`：引用计数、推迟释放、峰值统计
- `audit.go`：审计报告和异常分类
- `device.go`：双眼视口、单眼销毁、重建失败回滚
- `cmd/texturelife/main.go`：完整交错场景
- `*_test.go`：生命周期、设备状态和命令输出测试
