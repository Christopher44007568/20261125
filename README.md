# 头皮发根 Poisson-disc 采样与渲染

仅使用 Go 标准库（`math`、`math/rand`、`image`、`image/png`）实现：在同一个椭球
头皮区域内生成发根，密度沿头顶 → 前额 → 耳后连续变化，并输出根点渲染、区域计数
热图与发丝方向向量。无任何第三方依赖。

## 运行

```bash
go run ./cmd/hairroots            # 默认输出到 artifacts/
go run ./cmd/hairroots -rmin 0.03 # 更密的全局发根
```

## 头皮参数化

`internal/scalp` 把头皮建模为椭球上半部分，使用极角 `theta`（从头顶量起）和方位角
`phi`（`phi=0` 朝脸、`±pi` 朝后脑）：

- `Point/Normal/Frame`：参数坐标到三维位置、法向和正交活动标架。
- `ThetaMax(phi)`：发际线边界，前额略高、耳后有平滑内凹。
- `AreaElement`：参数格对应的真实曲面面积元，热图按它归一化，避免参数畸变。
- `Weight(theta,phi)`：连续密度权重，只由高斯/余弦光滑波瓣叠加而成——头顶隆起、
  两侧耳后凹陷，没有硬阈值，因此疏密过渡连续且 `phi` 接缝处跳变为零。
- `Radius = rMin / sqrt(weight)`：密度越高，Poisson-disc 排斥半径越小，映射连续。

## Poisson-disc 采样

`internal/poisson` 实现变半径 Bridson 算法：候选点在当前根点切平面环带内各向同性
生成，再径向投影回椭球；空间哈希网格用“两点半径均值”做冲突检测。随机角度与位置
保证无晶格，环形各向异性指标 `R1/R2` 在报告中量化这一点。

## 输出文件（`artifacts/`）

- `roots_perspective.png`：透视视图，逐像素光线求交绘制头皮并做遮挡/背面剔除。
- `roots_side.png`：正交侧视图，可直接读出头顶密、两侧疏的密度分布。
- `density_heatmap.png`：参数域计数热图，按真实面积元归一化并做平滑、稳健配色。
- `flow_vectors_top.png`：顶视图，绘制采样根的发丝方向向量（含发旋旋转场）。
- `roots.csv`：每个根的参数、三维位置、权重、半径、切向与世界空间方向向量。
- `stats.txt`：区域密度、最近邻间距分布、晶格指标和连续性数值检查。
