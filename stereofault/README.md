# stereofault

纯 Go 标准库实现的双目（左眼/右眼）分层渲染故障注入模拟器，回答的核心问题是：
**一个图层的资源坏了，整个双目画面是否该黑？答案是否定的。**

坏层只能走两种被声明的出路——绘制其声明的占位图，或整体隐藏；其余层继续
独立合成，画面只有在「所有层同时隐藏」时才会全黑。

## 三层与三类故障

| 层 | 依赖资源 | 注入故障 | 失败策略 |
| --- | --- | --- | --- |
| world | 纹理 | 纹理丢失 `FaultMissingTexture` | 声明占位 `PH-WORLD` |
| ui | 深度缓冲 | 深度格式错误 `FaultBadDepthFormat` | 声明占位 `PH-UI` |
| hand | 投影矩阵 | 投影矩阵损坏（奇异矩阵）`FaultBadProjection` | 无占位，只能隐藏 |

## 隔离保证

- **命令缓冲不被污染**：图层在记录任何命令之前先做资源校验（`Layer.Validate`），
  校验失败即拒绝进入共享 `CommandBuffer`；占位绘制走 `BlitPlaceholder` 独立安全
  路径，直接写眼睛目标，绝不追加共享命令。
- **无残影**：每帧每眼先 `Clear`，隐藏层的区域保持清空；像素带有属主、内容类型
  和帧号，`countGhosts` 可审计「属主已宕机却仍有本帧 layer 内容」的残影像素。
- **占位必须声明**：没有声明占位的层（hand）不允许临时合成像素，只能隐藏。
- **可恢复**：`Repair` 还原已知良好资源快照，下一帧重新校验通过即恢复正常绘制，
  恢复后不留下残影。

## 记录内容

每帧生成 `FrameSnapshot`：各层状态（healthy/placeholder/hidden）、本帧共享命令数、
是否全黑、残影像素数、不变量违例；事件流记录 failure/fallback/recovery，
并累计每层 healthy/placeholder/hidden 帧计数（按左右两眼计数）。

## 运行

```sh
go test ./...            # 全部不变量测试
go test -race ./...      # 竞态检测
go run ./cmd/stereofault # 演示，报告同时写入 ../artifacts/stereo-fault-report.txt
```
