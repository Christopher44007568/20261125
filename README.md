# Stereo Projection Truth Harness

纯 Go 标准库实现的双目投影校验工程，不安装任何第三方组件。

## 真值原则

测试中的期望值全部由独立解析公式生成，不调用被测的 `ProjectToScreen`、`UnprojectEye` 或 `EyeToWorld`：

- 先由已知相机位姿、眼偏移和世界点解析出眼空间 `(x, y, z)`。
- 用非对称视锥的近裁面边界、远裁面边界和透视分式计算 NDC、屏幕坐标、深度与裁剪状态。
- 逆投影使用独立的 NDC 反解公式，再从眼空间重建世界坐标。
- 水平视差由左右眼独立屏幕 x 坐标相减得到。

## 覆盖范围

- 已知相机、双眼偏移和左右非对称视锥。
- 屏幕 x/y 坐标、反向深度、近裁面与远裁面。
- 四角样本、中心线样本和多深度样本。
- 眼空间逆投影与世界坐标往返。
- 轻微相机姿态扰动。
- 单眼可见、另一只眼裁剪，以及近/远平面外裁剪。
- 多点理论水平视差。
- 每个指标输出数量、最小值、均值、P95、最大值和最坏样本。

## 运行

```bash
go test ./... -count=1
```

生成 CSV 误差分布报告：

```bash
STEREO_TRUTH_REPORT=stereo_truth_report.csv go test ./... -count=1 -v
```

报告包含每个指标的误差分布和最坏样本，字段包括 `metric`、`count`、`min`、`mean`、`p95`、`max`、`worst_case`、`worst_eye`、`got`、`want`、`error`。
