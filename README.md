# Stereo UI Drift Lab

使用 Go 标准库实现的近平面 / 中距离 / 系统层双眼 UI 投影与校验工具。它会为每个场景、每组 UI、每只眼睛生成独立画面，并输出视差、边界裁切、可读性和系统层边缘清晰度结果。

## 运行

```bash
go test ./...
go run . -out artifacts/stereoui
go run . -out artifacts/stereoui -strict
```

常用参数：

- `-out`：结果目录，默认 `artifacts/stereoui`
- `-width` / `-height`：单眼画面尺寸，默认 `1280x960`
- `-ipd`：瞳距（米），默认 `0.064`
- `-strict`：存在失败项时以状态码 `2` 退出

## 场景与 UI 组

场景定义在 `eye.go`：

- `baseline`：常规镜像非对称 FOV
- `fast_head_turn`：约 `310°/s` 快速转头和运动残差
- `asymmetric_fov`：强非对称、非镜像 FOV
- `mono_tracking_lost`：右眼追踪失效，使用最后已知 IPD、头部姿态和扩大安全区
- `combined_stress`：快速转头、强非对称 FOV、单眼失效叠加

UI 组定义在 `ui.go`：

- `near_plane`：`0.72m`，强调大视差和近平面文字稳定性
- `mid_distance`：`2.45m`，中等视差
- `system_layer`：`8m`，低视差，并执行边缘锐度检测

## 产物

- `frames/<scenario>_<layer>_left.png`
- `frames/<scenario>_<layer>_right.png`
- `report.json`：完整机器可读报告
- `report.md`：中文总报告并引用每眼 PNG
- `parallax.csv`：逐字形左右坐标、原始像素差、角度视差、朴素复制误差
- `boundary.csv`：每眼边界、安全区、裁切/剔除状态和边距
- `readability.csv`：角字高、笔画像素、对比度、运动残差
- `sharpness.csv`：系统层文字 10%–90% 边缘过渡宽度

## 坐标原则

每只眼睛都通过自己的眼部位姿、IPD、离轴 FOV 主点和切平面参数独立投影：

```text
screen_x = width * ((X / Z) + fov.leftTan) / (fov.leftTan + fov.rightTan)
screen_y = height * (fov.upTan - (Y / Z)) / (fov.upTan + fov.downTan)
```

单眼追踪失效时不会把另一只眼的屏幕坐标复制过来，而是使用最后已知 IPD、当前/预测头部姿态、该眼自己的 FOV 和更保守的安全区。`parallax.csv` 中的 `coordinates_copied` 恒为 `false`，`naive_same_coordinate_copy_error_px` 给出了图省事直接复制坐标会产生的像素误差。

非对称 FOV 会移动每眼主点，因此原始像素坐标差包含标定偏移；判断立体深度和垂直对齐时应同时查看角度视差列。

## 判定口径

- 边界：`visible`、`edge_risk`、`clipped`、`culled`
- 可读性：角字高至少 `24 arcmin`、笔画至少 `1.2 device px`、对比度至少 `4.5:1`
- 视差：禁止坐标复制，垂直角差超过 `3 arcmin` 判失败
- 系统锐度：边缘样本不少于 20，中位 10%–90% 过渡不超过 `2px`，P90 不超过 `3px`

渲染使用内置 5×7 位图字体和 `image/png`，没有第三方 Go 依赖。
