# 玻璃层预算审计器

用 Go 标准库实现透明玻璃层的逐眼预算审计。层数超过预算时，输出会明确记录保留层、未处理层、稳定位掩码、深度写入状态和预乘合成结果。

## 规则

- 深度值越大表示越远，左右眼分别读取自己的深度。
- 每眼独立排序。排序键依次为量化深度桶、层 ID；提交顺序不参与排序。
- 量化深度桶由 `DepthEpsilon` 控制，默认 `1e-6`。近共面层落在同一桶时按 ID 稳定排序。
- 每眼保留最近的 `N` 层，其余层标记为 `unprocessed-budget`。
- `mask_bit_order` 是全部层 ID 的字面排序；掩码最低位对应 `mask_bit_order[0]`。
- `processed_mask` 标记保留层，`unprocessed_mask` 标记未处理层。
- `depth_order` 和 `kept_ids` 为近到远，`render_order` 为实际合成用的远到近顺序。
- 材质颜色是直通颜色；合成前按层不透明度预乘，再使用 source-over。
- 玻璃透明通道始终关闭深度写入。近不透明层也不会改变该通道语义。
- 左右眼可以选择不同的层序和预算，但同一只眼对任意提交顺序必须得到相同证据。

## 内置场景

- `intersection`：`cross-left` 与 `cross-right` 在双眼中的前后关系相反。
- `near-coplanar`：五个层位于同一深度桶，验证近共面情况下的 ID 决胜。
- `shared-material`：五个层共享 `frosted` 材质，仍保持逐层独立排序。
- 默认预算为左眼 3 层、右眼 2 层，五层场景因此覆盖预算不足。
- 场景包含 `1-1e-9` 的近不透明层和 `1e-12` 的极低不透明度层。

## 运行

```sh
go test ./...
go run ./cmd/glassaudit
go run ./cmd/glassaudit -out artifacts/glass-audit.json
```

命令会穷举每条场景的全部 `5! = 120` 种提交顺序，逐眼比对审计证据；任一顺序导致结果变化时以非零状态退出。可直接复查的完整记录位于 `artifacts/glass-audit.json`。

可用参数：

```text
-left 3
-right 2
-epsilon 0.000001
-out -
```

项目不依赖第三方模块。
