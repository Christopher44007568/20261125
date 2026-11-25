# Dither Lab — 8-bit 暗部渐变抖动实验

纯 Go 标准库实现。比较无抖动、Bayer 8×8 有序抖动和 Floyd–Steinberg
误差扩散在三组低亮度渐变上的表现，并提供可交互的像素放大检视。

## 运行

```sh
go test ./...
go run .                      # 生成 artifacts/index.html 与 9 张 PNG
go run . -serve 127.0.0.1:8080
```

浏览器打开 <http://127.0.0.1:8080/>。也可以直接打开
`artifacts/index.html`：PNG 同时以 data URL 写入 `assets.js`，
因此本地文件模式下的 Canvas 像素检视同样可用。

## 测试图

| 图 | 输入 | 观察点 |
| --- | --- | --- |
| 灰阶 | 中性暗部 `L 2—38` | 最直接的阶梯与噪声对照 |
| 肤色 | 暗调肤色，三通道斜率不同 | 各通道色带边缘错位后的彩色轮廓 |
| 低亮度彩色 | 青绿 → 紫罗兰 → 暗玫瑰 | 低色阶下的色相分离与颗粒 |

每张图高 `196 px`，顶部 `40 px` 是固定整数 sRGB 值的平坦参照条。三种
量化器在该区域都必须输出与原值完全相同的像素，误差为 `0`。

## 算法

- **无抖动**：每通道直接四舍五入到 8-bit，保留硬边色带。
- **Bayer 8×8**：用 0—63 阈值矩阵做确定性阈值量化。矩阵对整数输入不产生
  噪声，对小数输入则按比例分配上、下两个码值。
- **Floyd–Steinberg**：蛇形扫描；误差按 `7/16、3/16、5/16、1/16` 扩散。
  平坦整数区不会积累误差，因此不会出现纹理。

浮点渐变始终保留了 8-bit 以下的精度，PNG 输出则是真正的每通道 8-bit。
因此抖动并非伪造噪声，而是在相邻码值间分配亚码误差。

## 检视界面

- 3×3 矩阵同时显示三组测试图与三种量化方式。
- 点击任意画面取样，三个放大镜同步显示相同坐标。
- `原始` 显示真实 8-bit 像素；`噪声 ×8` 只放大相对局部均值的偏差，
  用于辨认 Bayer 网格与误差扩散颗粒，不修改源 PNG。
- `渐变区 / 平坦区` 可验证平坦区是否真正保持零纹理。
- `5× / 8× / 10×` 使用最近邻放大，像素边界不会被二次平滑。

## 输出

```text
artifacts/
├── index.html
├── style.css
├── app.js
├── assets.js
├── report.json
└── images/
    ├── gray_none.png
    ├── gray_ordered.png
    ├── gray_diffusion.png
    ├── skin_none.png
    ├── skin_ordered.png
    ├── skin_diffusion.png
    ├── chroma_none.png
    ├── chroma_ordered.png
    └── chroma_diffusion.png
```

`report.json` 记录每个变体的 MAE、RMSE、最大误差、平坦区最大误差和
唯一 RGB 数。
