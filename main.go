// Command eyediag classifies binocular transparent-material rendering
// inconsistencies from known-depth/color/coverage marker pixels. It uses
// only the Go standard library and needs no extra components.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"eyediag/internal/model"
	"eyediag/internal/report"
	"eyediag/internal/synth"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "generate":
		err = runGenerate(os.Args[2:])
	case "analyze":
		err = runAnalyze(os.Args[2:])
	case "list":
		fmt.Println("内置场景：")
		for _, s := range synth.Scenarios() {
			fmt.Println("  -", s)
		}
	case "-h", "--help", "help":
		usage()
	default:
		err = fmt.Errorf("未知子命令 %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误：", err)
		os.Exit(1)
	}
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	scenario := fs.String("scenario", synth.ScenMulti, "要生成的场景名（见 list）")
	seed := fs.Int64("seed", 20261125, "位置抖动随机种子")
	out := fs.String("out", "", "输出 JSON 文件路径，缺省输出到标准输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := synth.Generate(*scenario, *seed)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if *out == "" {
		_, err = os.Stdout.Write(append(data, '\n'))
		return err
	}
	return os.WriteFile(*out, append(data, '\n'), 0o644)
}

func runAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	in := fs.String("in", "", "标记像素输入 JSON（含 markers 的 case）；缺省读标准输入")
	format := fs.String("format", "text", "输出格式：text 或 json")
	fire := fs.Float64("fire", 0, "原因触发阈值（0..1），缺省用内置 0.55")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var data []byte
	var err error
	if *in == "" || *in == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*in)
	}
	if err != nil {
		return err
	}

	var c model.Case
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("解析输入 JSON 失败：%w", err)
	}
	th := model.Thresholds{FireScore: *fire}
	r := report.Build(c, th.Resolve())

	switch *format {
	case "json":
		return r.WriteJSON(os.Stdout)
	case "text", "":
		r.WriteText(os.Stdout)
		return nil
	default:
		return fmt.Errorf("未知输出格式 %q", *format)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `eyediag — 双眼透明材质不一致因果分类器

用法：
  eyediag list
  eyediag generate -scenario <name> [-seed n] [-out file.json]
  eyediag analyze [-in file.json] [-format text|json] [-fire 0.55]

流程：标记像素特征提取 → 四类原因独立打分（可多标签）→
定位到顶点/图层/采样阶段 → 聚合报告。totalPixelDiff 仅作上下文，
不参与任何因果判定。`)
}
