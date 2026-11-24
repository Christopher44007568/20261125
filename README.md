# 20261125

Repository initialized for 20261125.

## 线性光 UI 合成

本示例只使用 Go 标准库，演示不能在 sRGB 编码域直接混合透明 UI。程序把每层输入先转换为“线性光、预乘 Alpha”，再按 Porter-Duff `over` 逐层合成，直到最终显示前才做 Reinhard tonemap 和 sRGB 编码。

### 覆盖场景

- HDR 线性背景：线性非预乘输入，允许线性 RGB 大于 1，合成阶段不钳制。
- 半透明黑边：sRGB 非预乘输入，展示灰色边缘的 Gamma 风险。
- 彩色文字：sRGB 预乘输入，先反推 straight color，再按线性光重新预乘。
- 高亮提示：sRGB 非预乘输入，和文字、黑边、HDR 背景重叠。
- 逐眼 Alpha：左右眼分别使用不同透明度并独立输出。
- 错误参考：同时生成 sRGB 域 `over` 和线性 HDR 域直接加法两种错误结果。

### 运行

```bash
go test ./...
go run . artifacts/output
```

### 输出

- `eye*_layer*.csv`：每层输入、解码后的线性预乘值、累计合成值。
- `eye*_correct_final_linear_premul.csv`：正确管线的最终线性预乘像素。
- `eye*_correct.png`：正确 tonemap 与 sRGB 显示结果。
- `eye*_wrong_gamma.png`、`eye*_gamma_heatmap.png`：sRGB 域 `over` 错误及热图。
- `eye*_wrong_addition.png`、`eye*_addition_heatmap.png`：线性 HDR 直接加法错误及热图。
- `report.json`：逐眼、逐层探针数值与聚合误差。

热图中的红色表示错误结果相对正确结果整体变暗，蓝色表示变亮；灰色表示过暗、过亮或饱和，Gamma 比值无效，对应线性 RMSE 仍写入 CSV。
