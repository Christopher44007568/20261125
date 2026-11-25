# HDR → SDR / PQ / HLG display simulations

纯 Go 标准库实现的确定性小工具：生成三组线性 HDR 场景，把它们投射到 SDR、PQ（SMPTE ST 2084）和 HLG（ITU-R BT.2100）显示模型，并输出三联显示模拟图、亮度直方图和数值校验报告。

## 运行

```bash
go test ./...
go run . -out artifacts
```

只使用 Go 标准库；没有第三方依赖。默认输出目录是 `artifacts/`。

## 输出

- `artifacts/hdr_window_comparison.png`：高光窗场景的 SDR / PQ / HLG 并排图和直方图。
- `artifacts/hdr_lights_comparison.png`：彩色灯场景的并排图和直方图。
- `artifacts/hdr_shadow_comparison.png`：阴影室内场景的并排图和直方图。
- `artifacts/hdr_three_scenes_all_displays.png`：三组场景的纵向总表。
- `artifacts/hdr_<scene>_<display>.png`：每个单独显示模拟图。
- `artifacts/hdr_report.json` / `artifacts/hdr_report.md`：灰卡、泛白、高光层次等检查结果。

## 声明的中性灰位置

- HDR 灰卡亮度：`18 nits`，即 18% 反射率 × `100 nits` 参考白。
- 每个场景的灰卡矩形由 `grayRectFor` 明确声明，图中有黄色定位角标。
- 三张可直接查看的 sRGB 软校验图都把灰卡中心映射到 proof code `118`（线性锚点 `0.18`，sRGB 编码后量化为 118）。
- 直方图中的黄色 `GRAY=118` 线就是该声明位置；`hdr_report.json` 记录每张图灰卡中心的实际 RGB 均值和误差。
- 原生传递函数在 `18 nits` 处的码值也会记录：SDR `118`、PQ `89`、按 `E=Y_nits/1000` 归一的 HLG `59`。为了生成普通看图器可显示的 PNG，像素最后经过灰卡/峰值双锚点软校验重映射，使三张图的创作锚点一致。

## 为什么不是只改整体曝光

管线在 `display.go` 中分为四步：

1. 场景以线性 nits 生成，太阳、天空和彩灯可达数千 nits。
2. 每个像素按最大通道计算显示膝点和对数高光肩；灰卡位于膝点以下，暗部和灰卡不被整体推移。
3. SDR 使用 sRGB 传递函数，PQ 使用 ST 2084 OETF，HLG 使用 BT.2100 分段 OETF。
4. 原生码值经黑场/灰卡/峰值锚点重映射为普通 sRGB PNG，并加入有限的饱和度保护，避免 PQ/HLG 的逐通道非线性把彩灯冲淡。

高光线肩只作用于超过膝点的像素，且自动检查控制区平均亮度、近白像素比例和高光 ROI 的标准差/bin 覆盖，避免“高光恢复”变成全图泛白。

## 场景

- `window`：约 `850–9200 nits` 的天空与太阳、窗棂、远处楼体和较暗室内墙面。
- `lights`：红、绿、蓝、琥珀四盏 HDR 灯和大面积彩色辉光，中间放置中性灰校准卡。
- `shadow`：约 `0.2–8 nits` 的室内暗部、书架、沙发、小台灯和月光窗，验证暗部层次不被黑场切除。

所有场景均为程序绘制，没有读取外部图片或字体。
