# 双眼动态参数隔离验证器

共享着色程序、uniform 布局和静态场景布局，不等于共享相机、材质覆盖或
眼级常量。这个 Go 程序用同一份着色程序渲染左右眼，同时让四组同名材质实例
在每个眼睛中拥有不同的粗糙度、alpha 覆盖和常量，并通过乱序批次与 LRU
淘汰验证动态状态不会串眼。

## 必须保持的边界

- 共享：`*ShaderProgram`、uniform 静态布局、`*StaticLayout`（面板位置、
  视口尺寸、base material 名称）。
- 隔离：每眼 `Camera`、`MaterialState`、`AlphaOverride`、材质常量、
  眼级常量、jitter、viewport seed 和快照 ID。
- 缓存键：`program_id + layout_hash + eye + instance + dynamic_hash`。
  `dynamic_hash` 覆盖相机、材质、眼级常量和其余动态字段。
- 批次：每轮都打乱左右眼和四个材质实例的绘制顺序；缓存跨轮保留，默认容量
  3、共 32 个批次，因此会持续发生淘汰。
- 输出：每轮使用独立目标缓冲，只有最后一轮进入画面哈希，避免 alpha
  被重复叠加，也保证结果与无缓存参考逐字节一致。

## 运行

```sh
go run .
```

只使用 Go 标准库，不需要安装额外组件。默认输出到
`artifacts/eye-isolation/`：

- `report.json`：每眼参数快照、快照哈希、参考/隔离画面哈希、缓存统计、
  批次记录，以及故障探针的首个污染像素和 RGBA。
- `report.md`：便于快速检查的摘要。
- `left.png`、`right.png`：隔离路径的实际画面。
- `fault-left.png`、`fault-right.png`：故障注入画面。
- `fault-left-diff.png`、`fault-right-diff.png`：品红标出所有污染像素，
  白色十字标出按光栅顺序发现的第一个污染像素。

可用参数：

```sh
go run . -seed 20261125 -rounds 4 -cache-capacity 3 -out artifacts/eye-isolation
```

## 故障注入

正确路径不会故意制造污染。为了记录“首个污染像素”并验证检测器，程序额外执行
一次显式故障探针：缓存键被故意降级为
`program_id + layout_hash + base_material`，遗漏 `eye`、`instance` 和
`dynamic_hash`。同一 base material 的所有眼和面板因此命中同一条动态
uniform，画面相对无缓存参考出现污染。

故障探针只用于回归演示，不参与正确渲染。默认种子下左右眼都会在
`(59, 27)` 报出首个污染像素；具体像素坐标记录在 JSON 的
`fault_injection.eyes[].first_polluted_pixel` 和每眼的
`fault_probe.first_polluted_pixel` 中。

## 验证

```sh
go test ./...
go test -race ./...
go vet ./...
```

测试覆盖：

- 深拷贝快照，修改克隆不会影响原眼状态。
- 同 base material 的左右眼材质参数、alpha 覆盖和常量确实不同。
- 正确缓存键共享程序/布局但不会跨眼命中。
- 乱序、跨轮缓存淘汰后仍与无缓存参考完全一致。
- 故障注入会产生污染，并按从上到下、从左到右的顺序定位首像素。
- 报告包含快照、画面哈希、首污染像素及全部 PNG/JSON/Markdown 产物。

## 文件结构

- `state.go`：眼快照、相机和材质动态状态，深拷贝与哈希。
- `scene.go`：共享程序、uniform 布局、静态面板和双眼默认数据。
- `uniforms.go`：按共享布局打包每眼动态值，定义正确/故障缓存键。
- `render.go`：CPU 光线-平面着色和 alpha 合成。
- `cache.go`：LRU uniform 缓存及统计。
- `engine.go`：确定性的轮内乱序、跨轮执行与快照稳定性检查。
- `imageutil.go`：RGBA 哈希、像素比较、污染差异图。
- `report.go`、`output.go`：报告构建及 JSON/Markdown/PNG 输出。
