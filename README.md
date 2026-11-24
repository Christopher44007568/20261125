# eyestage

逐眼、逐阶段分析采集场景、重投影、畸变、合成四个阶段的 Go 命令行工具。只用 Go 标准库，不需要安装额外组件。

工具解决的问题是：左右眼总帧时间可能相同，但其中一只眼在等待另一只眼。报告会分别计算每只眼、每个阶段的耗时分布、跨眼重叠和空等窗口，并识别：

- 单眼空等：对侧眼有阶段活动，本眼没有任何阶段活动。
- 时间戳回退：阶段结束早于开始，或跨帧起始时间倒退。
- 阶段缺失：某只眼缺少采集场景、重投影、畸变或合成阶段。
- 不可用时间戳：无法解析的字段、缺失单位、由时长反推的端点等。
- 长尾帧：按各眼、各阶段的分布判断，不使用单帧总时长代替归因。

## 快速开始

```bash
go run ./cmd/eyestage testdata/sample.json
go run ./cmd/eyestage -format json testdata/sample.json
go run ./cmd/eyestage -format html testdata/sample.json > report.html
go run ./cmd/eyestage -tail-percentile 95 -buckets 16 input.jsonl
```

也可以从标准输入读取：

```bash
cat input.jsonl | go run ./cmd/eyestage -
```

构建与测试：

```bash
go build -o bin/eyestage ./cmd/eyestage
go test ./...
```

`-format html` 生成单文件、无外部脚本或字体的离线报告，可直接用浏览器打开。

## 输入格式

推荐使用嵌套 JSON。每个帧包含 `left`、`right`，每只眼包含四个阶段；阶段使用 `start` / `end`：

```json
{
  "unit": "ms",
  "frames": [
    {
      "frame_id": "42",
      "left": {
        "capture_scene": {"start": 0, "end": 10},
        "reprojection": {"start": 10, "end": 20},
        "distortion": {"start": 20, "end": 30},
        "composition": {"start": 30, "end": 40}
      },
      "right": {
        "capture_scene": {"start": 2, "end": 12},
        "reprojection": {"start": 12, "end": 22},
        "distortion": {"start": 22, "end": 32},
        "composition": {"start": 32, "end": 42}
      }
    }
  ]
}
```

字段兼容：

- 眼睛：`left` / `right`、`l` / `r`、`left_eye` / `right_eye`、`左眼` / `右眼`。
- 阶段：`capture_scene`、`reprojection`、`distortion`、`composition`，以及 `scene_capture`、`reproject`、`undistort`、`compose`、`采集场景`、`重投影`、`畸变`、`合成` 等别名。
- 帧 ID：`frame_id`、`frame`、`id`。
- 时间字段：`start` / `end`、`begin` / `finish`、`start_ns` / `end_ns`、`start_us` / `end_us`、`start_ms` / `end_ms`、`start_s` / `end_s`、`started_at` / `ended_at`、`started_at` / `finished_at`。
- 阶段也可写成二元数组：`"capture_scene": [0, 10]`。

也可以使用扁平记录或 JSONL，每个记录表示一只眼的一个阶段：

```jsonl
{"frame_id":"42","eye":"left","stage":"capture_scene","start_ms":0,"end_ms":10}
{"frame_id":"42","eye":"right","stage":"capture_scene","start_ms":2,"end_ms":12}
```

扁平记录在同一个 `frame_id` 下合并；推荐每条记录显式给出 `frame_id`。

## 时间戳与回退

`unit` 支持 `ns`、`us`（也接受 `µs`）、`ms`、`s`；显式字段后缀优先于全局 `unit`。时间值也可以是 RFC3339 字符串：

```json
{"started_at": "2026-11-25T10:00:00.000Z", "ended_at": "2026-11-25T10:00:00.010Z"}
```

每个解析出的时间戳在 JSON 中保留 `source`，回退不会被静默隐藏。常见来源包括：

- `explicit:ns`、`explicit:us`、`explicit:ms`、`explicit:s`、`explicit:rfc3339`：显式时间戳。
- `fallback:default_ns`：字段没有单位，按纳秒处理。
- `fallback:unknown_unit_ns`：声明了不支持的 `unit`，回退到纳秒。
- `fallback:duration`：一个端点不可用或缺失，利用同一阶段的有效时长反推。
- `unsupported_timestamp` 诊断中的 `fallback_source: "none"`：没有可用回退，不伪造该阶段耗时。

帧级总时长字段（例如 `frame_duration`）不会被用于填充阶段耗时。若只有帧总时长而缺少阶段时间戳，结果会报告阶段缺失或时间戳不可用，而不是把总时长分摊给各阶段。

## 报告内容

文本报告包含：

- 逐眼帧概览：阶段耗时、活跃跨度、空等时长和缺失阶段。
- 逐眼逐阶段直方图：`min`、`p50`、`p95`、`p99`、`max`。
- 长尾帧：按逐眼逐阶段阈值归因后聚合，不按帧总时长分类。
- 跨眼对应阶段重叠：左右眼同一阶段的重叠、仅左眼、仅右眼和重叠率。
- 跨阶段错位重叠：例如左眼畸变与右眼重投影同时进行，用于定位流水线相位错位。
- 单眼空等汇总：有等待的帧数、总等待、中位数、P95 和最大值。
- 诊断：阶段缺失、时间戳回退、不可用时间戳及其回退来源、持续单眼空等。

JSON 报告保留全部结构化字段，包括每帧每眼的阶段起止时间戳与 `source`、空等窗口、每个阶段直方图桶和所有诊断。HTML 报告中的每个帧都可以展开查看逐阶段时间戳。

长尾阈值默认使用 P95。达到阈值且高于中位数的阶段实例会被列出；分类依据是阶段耗时，帧总跨度只作为上下文显示。

## 选项

```text
-format text|json|html
-tail-percentile 95
-buckets 12
-min-idle 0s
-fail-on-diagnostics
```

`-min-idle` 控制多少空等时长才写入单帧诊断，默认任何非零空等都记录。`-fail-on-diagnostics` 在已有报告正常输出后，只要有诊断就以状态码 2 退出，适合流水线门禁。
