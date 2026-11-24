# 20261125

Repository initialized for 20261125.

## meshfix · 非流形边安全修复工具（纯 Go 标准库）

针对“一处非流形边挂接三片薄板”的缺陷场景：自动删除任一面都会改变零件语义，
因此工具先输出缺陷覆盖图，再经用户确认后按薄片连通分量拆分脊边（只复制端点顶点，
不删除、不重绕任何面）。修复前后均渲染拓扑边与面法线，并给出每个面的来源对应。

### 运行

```sh
go build -o meshfix .
./meshfix            # 生成数据 -> 缺陷覆盖图 -> 交互确认 -> 拆分修复
./meshfix -y         # 跳过交互直接确认
./meshfix --no-repair# 只出覆盖图，不修复
./meshfix -mesh artifacts/three_sheet_fan.json
```

### 输出（artifacts/）

- `three_sheet_fan.json` / `three_sheet_fan.repaired.json` — 自生成多面体数据与修复后网格
- `defect_coverage.txt` / `defect_coverage.svg` — 缺陷覆盖图（终端版 + 图形版）
- `before.svg` / `after.svg` — 修复前后拓扑边 + 面法线视图
- `face_provenance.json` — 每个面的来源、顶点、法线、拓扑边前后对应
- `report.json` / `report.html` — 汇总报告
