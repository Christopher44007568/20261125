package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"eyestage"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("eyestage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "text", "output format: text, json, or html")
	tailPercentile := flags.Float64("tail-percentile", 95, "per-stage long-tail percentile")
	buckets := flags.Int("buckets", 12, "histogram bucket count")
	minIdle := flags.Duration("min-idle", 0, "minimum idle wait included in diagnostics")
	failOnDiagnostics := flags.Bool("fail-on-diagnostics", false, "exit with status 2 when diagnostics exist")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "用法: eyestage [选项] [input.json|input.jsonl|-]")
		fmt.Fprintln(stderr, "逐眼读取采集场景、重投影、畸变、合成阶段时间戳并生成归因报告。")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return 2
	}
	if *format != "text" && *format != "json" && *format != "html" {
		fmt.Fprintf(stderr, "不支持的输出格式 %q，可选 text、json 或 html\n", *format)
		return 2
	}

	input := stdin
	name := "stdin"
	if flags.NArg() == 1 && flags.Arg(0) != "-" {
		file, err := os.Open(flags.Arg(0))
		if err != nil {
			fmt.Fprintf(stderr, "打开输入失败: %v\n", err)
			return 1
		}
		defer file.Close()
		input = file
		name = flags.Arg(0)
	}

	dataset, err := eyestage.Parse(input)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", name, err)
		return 1
	}
	report := eyestage.Analyze(dataset, eyestage.Config{
		TailPercentile:   *tailPercentile,
		HistogramBuckets: *buckets,
		MinIdleWait:      *minIdle,
	})
	switch *format {
	case "json":
		err = eyestage.WriteJSON(stdout, report)
	case "html":
		err = eyestage.WriteHTML(stdout, report)
	default:
		err = eyestage.WriteText(stdout, report)
	}
	if err != nil {
		fmt.Fprintf(stderr, "写入输出失败: %v\n", err)
		return 1
	}
	if *failOnDiagnostics && len(report.Diagnostics) > 0 {
		return 2
	}
	return 0
}
