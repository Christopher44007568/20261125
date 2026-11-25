# Guide-strand deep opacity hair shadows

纯 Go 标准库实现的刘海自阴影实验。单层最近深度只能给出“最近头发 / 全部头发”的
二元结果，多层刘海交叠时会出现整块漏光或整块死黑；这里为 guide strand 建立
light-space deep opacity，并在着色点按深度查询彩色透射率。

## 运行

```sh
go run .
go test ./...
```

默认写入 `artifacts/`：

- `artifacts/beauty.png`：普通光照，包含皮肤环境光、暖色直射、头发各向异性高光和 deep-opacity 阴影。
- `artifacts/shadow_only.png`：仅阴影系数，白色表示无遮挡，灰色/深灰表示 guide strand 多层投影。
- `artifacts/hair_transmission.png`：头发透射图；沿相机方向逐层相乘，交叠发束会明显加深。

常用参数：

```sh
go run . -width 720 -height 900 -light-size 512 -out artifacts
go run . -deep-map artifacts/deep_opacity_light_space.png
```

`-deep-map` 是可选的光源空间 deep-opacity 诊断图，不属于必需的三张输出。

## 实现要点

- `scene.go`：程序化前额椭球、64 根三次 Bézier guide strand 和逐段半径。
- `deepopacity.go`：
  - 从光源正交投影光栅化 guide strand；
  - 每个 light texel 保存按深度排序的 `DeepSample`；
  - 同一 guide 的线段接头合并为一层，近距离层做压缩；
  - `TransmittanceAt` 返回 `∏((1-α)+αTint)`，并用双线性采样软化 texel 边界；
  - 头发着色时排除当前 guide，避免同根 guide 的深度偏差造成自阴影条纹。
- `render.go` / `shade.go`：相机投影、皮肤求交、发束覆盖光栅化、背到前混合、
  圆柱法向和各向异性高光。
- `output_test.go`：端到端生成三张 PNG，并检查前额均值亮度、近黑像素比例、
  发束区域反差以及 shadow/transmission pass 的动态范围。

前额皮肤始终有半球环境光和固定暖色补光，deep opacity 只衰减直射光，因此不会全黑；
不同 guide 的深度、颜色、半径和高光保留发束层次。当前默认分辨率在普通开发机上
约半秒内完成。

## 依赖

仅使用 Go 标准库（`image/png`、`math`、`math/rand`、`flag` 等），没有第三方包。
