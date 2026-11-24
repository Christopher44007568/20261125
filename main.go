package main

import (
	"fmt"
	"strings"
)

func main() {
	for _, s := range allSchemes() {
		report(s)
	}
}

func report(s Scheme) {
	g := s.Graph
	fmt.Printf("================ 方案：%s ================\n", g.Name)

	fmt.Println("-- 提交序列（版本调度固定） --")
	for _, t := range g.Tasks {
		mark := ""
		if t.Cancelled {
			mark = " [已取消]"
		}
		fmt.Printf("  %s %-13s queue=%-8s dur=%d%s\n", t.ID, t.Name, t.Queue, t.Duration, mark)
	}

	fmt.Println("-- 资源依赖边（提交者 -> 使用者） --")
	for _, e := range g.BuildEdges() {
		scope := "同队列"
		if e.CrossQ {
			scope = "跨队列"
		}
		fmt.Printf("  %-12s %s(%s) -> %s(%s)  %s\n",
			e.Resource, e.From, e.FromKind, e.To, e.ToKind, scope)
	}

	fmt.Println("-- 屏障表 --")
	if len(g.Barriers) == 0 {
		fmt.Println("  (空)")
	}
	for _, b := range g.Barriers {
		fmt.Printf("  %-12s %s -> %s  %s\n", b.Resource, b.From, b.To, b.Kind)
	}

	ivs := g.Simulate()
	fmt.Println("-- 完成顺序 != 提交顺序 --")
	fmt.Printf("  提交顺序: %s\n", strings.Join(taskIDs(g.Tasks), " "))
	fmt.Printf("  完成顺序: %s\n", strings.Join(g.CompletionOrder(ivs), " "))

	fmt.Println("-- 注入故障 --")
	for _, f := range s.Faults {
		fmt.Printf("  * %s\n", f)
	}

	fmt.Println("-- 屏障验证（按执行时间线，而非提交顺序） --")
	exposed := 0
	for _, h := range g.Validate() {
		fmt.Printf("  %s\n", h)
		if !h.Covered {
			exposed++
		}
	}
	fmt.Printf("  结论：%d 处冲突暴露\n\n", exposed)
}

func taskIDs(tasks []*Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}
