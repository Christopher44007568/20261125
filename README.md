# 直纹曲面求交与连续性拼接

纯 Go 实现的两张有限直纹曲面求交、重复段识别、交线连续性拼接，以及三视角 PPM
诊断图。运行时不依赖第三方 Go 模块或外部几何库。

## 快速运行

```bash
go test ./...
go vet ./...
go run ./cmd/ruled-intersect -out artifacts
```

只生成某一个场景：

```bash
go run ./cmd/ruled-intersect -case tangent -out artifacts
go run ./cmd/ruled-intersect -case near_parallel -out artifacts
go run ./cmd/ruled-intersect -case crossing -out artifacts
```

输出包括：

- `artifacts/<case>_3views.ppm`：单张三视角拼图，含图例。
- `artifacts/<case>_front.ppm`、`_top.ppm`、`_side.ppm`：裁切后的单视图。
- `artifacts/intersection_report.json`：原始段、有效段、重复段、拼接曲线数和曲面残差。

## 曲面定义

每张曲面是有限参数带：

```text
P(u,v) = Base(u) + v * Dir(u),  u,v in [0,1]
```

`Base` 与 `Dir` 都可以弯曲，因此不要求曲面是平面。默认通过三角网格离散，网格密度由
`RuledSurface.Nu/Nv` 控制。

```go
s1 := geom.NewRuledSurface("s1", base1, dir1)
s2 := geom.NewRuledSurface("s2", base2, dir2)
result := intersect.Surfaces(s1, s2, intersect.DefaultOptions())
```

## 算法

1. 两张曲面各自按 `(u,v)` 网格三角化，并建立三角形 AABB BVH。
2. 对候选三角形对，分别求每个三角形与另一三角形所在平面的截线，再把两条截线投影
   到两平面的交线上，取一维区间交集。该做法同时覆盖普通穿越、极小夹角和相切接触。
3. 对每个交线端点，从三角形参数反算初值，使用四参数、三方程的阻尼最小范数 Newton
   迭代求解 `S1(u1,v1)=S2(u2,v2)`。随后分别计算点到两张参数曲面的距离并拒绝超差结果。
4. 对共线且区间重叠的三角片输出做重复段识别；保留最长代表作为有效段，其余标记为
   重复段，而不是静默删除。
5. 用并查集吸附容差内的端点，将端点位置取簇平均，再按方向连续性追踪图链，输出无
   局部断裂的折线。相切场景的重复支撑和有效交线会分别保留。
6. 三个正交视图统一比例，先绘制带明暗的未裁剪曲面，再叠加橙色重复段和绿色有效
   交线，使交线在图中始终可见。

## 测试场景

| 场景 | 几何构造 | 验证重点 |
| --- | --- | --- |
| `tangent` | 平面 `z=0.35` 与抛物柱面带在 `u=0.5` 的整条母线上相切 | 切触线可检出；同一支撑被多个三角片重复输出时，重复段与有效段同时可辨 |
| `near_parallel` | 两张夹角极小的斜平面带 | 病态平面交线仍能稳定精化，并拼成一条长曲线 |
| `crossing` | 波浪直纹带被横向曲线带穿过 | 交线连续；各端点同时落在两张参数曲面上 |

当前输出数据：

```text
tangent        raw=112 valid= 28 duplicate= 84 curves=1 maxgap=1.55e-11
near_parallel  raw=190 valid=190 duplicate=  0 curves=1 maxgap=3.42e-11
crossing       raw=198 valid=198 duplicate=  0 curves=1 maxgap=4.04e-05
```

`go test ./...` 会检查：

- 三种场景都产生有效段和恰好一条拼接曲线。
- 每个原始段端点、拼接曲线顶点到两张曲面的距离均不超过 `OnSurfaceTolerance`。
- 相切场景确实暴露重复段，且曲线落在接触平面内。
- 近似平行与交叉场景的拼接曲线长度符合预期。
- PPM 文件头、像素载荷长度三种颜色类别的实际落图。

## 颜色约定

- 绿色 `#149A4E`：有效交线段/拼接曲线。
- 橙色 `#E25C16`：重复三角片支撑，作为宽底衬绘制，避免被绿色有效线完全遮住。
- 蓝色 `#82B5D6`：未裁剪曲面区域；深蓝网格线用于显示曲面形状和两个视图中的深度关系。

顶点焊接容差、曲面吸附容差、重复判定容差和最短段长度均集中在
`intersect.DefaultOptions()`。对曲率更大或更薄的特征，可提高 `Nu/Nv` 后重新运行。
