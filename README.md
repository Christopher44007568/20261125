# 裁剪 NURBS 片：环提取、带孔耳切与 OBJ 往返

纯 Go 标准库实现，无第三方依赖。演示“只耳切外轮廓会填平内孔”的问题，并给出裁剪环提取、
环方向/包含关系分类、桥接、耳切、越界三角形校验、PNG 叠加、OBJ 导出和回读重渲染。

## 运行

```sh
go test ./...
go run . -out artifacts
```

也可以单独解析并渲染已有 OBJ：

```sh
go run . -render-obj artifacts/narrow_slot.obj -png /tmp/narrow_reload.png
```

默认会在 `artifacts/` 生成每个用例的：

- `<case>.obj`：导出的三角网格，Z 来自二次 NURBS 曲面；
- `<case>_direct_outer.png`：只耳切外轮廓的错误对照，叠加三角形边、孔洞掩码和红色越界三角形；
- `<case>_trimmed.png`：提取裁剪环并桥接内孔后的耳切结果；
- `<case>_obj_reload.png`：重新解析导出的 OBJ，再叠加 OBJ 边、边界轮廓、孔洞掩码和重新校验结果；
- `trim_report.json`：环点数、交叉点、三角形数、越界原因、OBJ 往返计数和图片路径。

## 测试轮廓

- `narrow_slot`：长条形窄槽，使用密集有理二次 NURBS 圆角弧，测试细长孔和右侧桥接。
- `multiple_holes`：一个 NURBS 圆、一个竖向窄槽和一个三角形内孔，测试多孔排序与连续桥接。
- `self_intersecting`：Bow-tie 自相交裁剪线，先报告非相邻边的真正交点，再在交点处拆成两个
  简单 even-odd 瓣；两个瓣共享交点时会先合并为一条接触孔边界走查，再进行耳切。

直切外轮廓的对照组在三个用例中都会产生越界三角形；裁剪感知网格和 OBJ 回读网格的越界数均为 0。

## 实现要点

- `nurbs.go`：非均匀 B 样条基函数、有理曲线权重、曲线采样和 NURBS 曲面求值。
- `loops.go`：采样 NURBS 裁剪段、按端点链接成闭合环、方向/面积计算、自相交检测和交点拆分。
- `triangulate.go`：右向射线求桥接点、孔方向归一化、接触孔合并、弱简单环耳切和越界校验。
- `obj.go`：OBJ 写出与解析，支持 `v`、`f`、`v/vt/vn` 形式、负索引和面扇出；从边使用次数提取边界。
- `paint.go`、`font.go`、`render_scene.go`：标准库 `image` 超采样 PNG、半透明掩码、斜线填充、
  三角形边、红色越界高亮、交点标记和内置位图字体。

## 几何校验

`ValidateTriangles` 同时检查：

- 三角形质心或顶点落到外轮廓外/内孔中；
- 孔顶点或外轮廓顶点被三角形严格包含；
- 三角形边与裁剪环发生非端点真正相交。

因此红色高亮表示整个三角形与禁止区域相交，而不只是质心错误。
