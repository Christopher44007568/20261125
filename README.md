# Rec.2020 → sRGB 色域映射对照

零外部依赖的 Go 实现：把高饱和 Rec.2020 颜色映射到 sRGB，并生成色轮、渐变和肤色三组对照图。直接裁切与感知色域映射共用同一份源色越界掩码。

## 运行

```bash
go run ./cmd/gamut-viz -out artifacts
go test ./...
```

生成的 PNG：

- `artifacts/rec2020-color-wheel.png`：高色度色轮，左右为两种方案。
- `artifacts/rec2020-gradients.png`：红系色度、红系色相漂移、全色相和中性轴渐变。
- `artifacts/skin-tone-comparison.png`：五档肤色 × 原色/2 倍色度/3 倍色度矩阵，并附中性轴。

斜纹叠加表示源色超出 sRGB；掩码在两种方案的同一位置出现。

## 两种方案

- **直接裁切**：Rec.2020 线性 RGB → XYZ(D65) → 线性 sRGB，然后把三个线性通道夹到 `[0,1]`。高饱和红色常被压成同一纯红，失去源色层次。
- **感知映射**：在 Oklab/OkLCh 中固定明度 `L` 和色相 `h`，用二分法把 chroma `C` 降到 sRGB 色域边界。色域内颜色原样通过；只减少 chroma，不增加。

## 中性色约束

中性色显式走无彩色路径：线性 `R=G=B` 或 Oklab chroma 接近零时只执行裁切，不进入色度搜索。因此映射不会给黑、白、灰添加色度，也不会为了“提高饱和度”改变中性轴。`TestNeutralAxisIsNeverSaturated` 对此有数值回归测试。

## 包接口

输入、输出都使用编码后的归一化 RGB（`[0,1]`）：

```go
source := gamut.RGB{R: 1, G: 0, B: 0} // 编码 Rec.2020
clip := gamut.MapRec2020ToSRGB(source, gamut.Clip)
smooth := gamut.MapRec2020ToSRGB(source, gamut.Perceptual)

fmt.Println(clip.OutOfGamut) // true：源色超出 sRGB
fmt.Println(smooth.SRGB)     // 可显示的 sRGB 编码值
```

颜色转换、Oklab 运算、PNG 编码和绘图字体全部使用 Go 标准库。
