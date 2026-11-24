# 变半径圆角曲面诊断

仅使用 Go 标准库构造一个 60° 尖角两侧的变半径圆角，并输出四联诊断图。默认半径沿轴向由 1 mm 余弦缓动到 5 mm。

## 运行

```bash
go run . -out artifacts
go test ./...
```

输出：

- `artifacts/fillet_diagnostics.png`
- `artifacts/fillet_report.json`

常用参数：

```text
-r0 1 -r1 5       起止半径，mm
-length 30        轴向长度，mm
-half-angle 30    尖角半角，deg（默认夹角 60°）
-face-width 7     每张母面的宽度，mm
-mesh-u 80        圆角横向采样数
-mesh-v 220       轴向采样数
-supersample 2    渲染超采样倍数
```

## 曲面构造

取尖角棱线为 y 轴，母面方向为 `(cos β, 0, ±sin β)`。半径

```text
R(v) = R0 + (R1-R0) * (1-cos(pi*v))/2
```

圆角曲面为

```text
F(u,v) = ( R/sinβ + R*cos(phi),
           L*(v-1/2),
           R*sin(phi) )
phi = pi/2 + β + u*(pi-2β)
```

其中 `u=0` 接上母面，`u=1` 接下母面。两条接触线均落在母面内，并且位置、轴向切向和横截面切平面与母面一致。

## 验收口径

- G0：圆角与两张母面在接触线及两端点的最大位置误差。
- G1：两侧接触线整段的切向夹角和法线夹角。
- 圆角为圆弧截面、母面为平面，因此接缝存在 `1/R` 的曲率跃变。这是圆弧倒角的正常 G1 特征；报告单独记录但不作为 G1 失败判据。
- 当前默认模型的验收阈值为位置 `1e-9 mm`、切向 `1e-7 deg`、法线 `1e-7 deg`。

`fillet_report.json` 中的 `upper_contact` 和 `lower_contact` 分别对应两张母面，末尾的 `fillet_parent_seams_pass` 是总验收结果。

## 四联图

1. 轮廓：正交视图、平滑解析法线、深度缓冲消隐。
2. 法线符号：RGB 三通道分别编码 `Nx/Ny/Nz` 的正负。
3. 斑马纹：根据反射方向生成环境条纹，用于检查切平面连续性。
4. 曲率带：主曲率绝对值 `max(|k1|,|k2|)` 的 10 级色带。
