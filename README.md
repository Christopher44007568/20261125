# 发丝高光模型对比（Go 标准库）

用同一程序化头皮、同一灯光与正交相机，对三种发丝着色模型做公平对比。
仅依赖 Go 标准库（`image`、`image/png`、`math` 等），无第三方依赖。

## 构建与运行

```bash
go run .
# 或
go build -o hairrender . && ./hairrender
```

输出全部写入 `artifacts/`，随机种子固定（`20261125`），结果可复现。

## 三种模型

- `single`：紧凑 Marschner 风格单次散射，含 R / TT / TRT 三个方位瓣与
  shifted-cone 纵向项（`shaders.go` 的 `marschnerSpec`）。
- `kajiya_kay`：经典 Kajiya-Kay（1989）切锥高光 + 切向漫反射
  （`kajiyaSpec` / `hairDiffuse`）。
- `double`：Zinke 风格双散射近似 = 同标定的局部单次散射 + 方位平均的
  全局透射项，按发丝壳层厚度做吸收衰减（`doubleGlobal`）。

三者共用同一个漫反射项；各模型的高光项用解析球面积分（`integrateScalar` /
`integrateColor`）标定到相同的目标积分能量，使能量差异反映瓣形与全局
传输，而不是任意权重。

## 共享几何与公平性

- `scene.go`：半球头皮 + Fibonacci 抖动分布的约 1.16 万根发丝折线。
- `gbuffer.go`：几何只光栅化一次，保存位置、头皮法线、纤维切线、壳层厚度
  和深度；三个模型从同一个 G-buffer 着色，保证头皮、灯光、相机完全一致。
- 灯光与相机参数集中在 `buildScene`，三种模型无任何场景差异。

## 输出文件

- `render_single_scatter.png` / `render_kajiya_kay.png` /
  `render_double_scatter.png`：三张主渲染图。
- `highlight_overlay.png`：三色高光位置叠加（红=单次，绿=Kajiya-Kay，
  蓝=双散射；重叠趋近白色）。
- `grazing_diagnostic.png`：掠射角诊断，水平平行纤维 + 各向同性
  Blinn-Phong 塑料对照组。
- `energy_report.txt`：线性总能量、纯高光能量、相对能量差、成对 L1 差、
  高光质心位移与掠射角各向异性指标。

## 掠射角判据

真实发丝高光是各向异性的：随纤维切线沿纤维方向移动。诊断图里三种发丝
模型的 `highlightTravelX` 约 70 px（亮段沿相邻纤维横向游走），而塑料
对照组锁定在固定屏幕带，`highlightTravelX = 0`、`bandSigmaY` 很窄，
证明没有退化成普通塑料球反射。
