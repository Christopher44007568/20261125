package main

import "testing"

func exposed(g *Graph) []Hazard {
	var out []Hazard
	for _, h := range g.Validate() {
		if !h.Covered {
			out = append(out, h)
		}
	}
	return out
}

func TestSingleQueueExposesWAR(t *testing.T) {
	hs := exposed(schemeSingleQueue().Graph)
	if len(hs) != 1 || hs[0].Kind != HazardWAR || hs[0].Resource != "HistoryColor" {
		t.Fatalf("期望 1 处 HistoryColor WAR 暴露，得到 %v", hs)
	}
}

func TestDualQueueExposesCrossQueueRAW(t *testing.T) {
	s := schemeDualQueue()
	hs := exposed(s.Graph)
	if len(hs) != 1 || hs[0].Kind != HazardRAW || !hs[0].CrossQ {
		t.Fatalf("期望 1 处跨队列 RAW 暴露，得到 %v", hs)
	}
	// 完成顺序不等于提交顺序：compute 上的 T2 先于 T1 完成。
	got := s.Graph.CompletionOrder(s.Graph.Simulate())
	if got[0] != "T2" {
		t.Fatalf("期望 T2 最先完成，得到 %v", got)
	}
}

func TestAliasingExposesUseAfterCancelAndAliasWAR(t *testing.T) {
	hs := exposed(schemeAliasing().Graph)
	var cancel, war int
	for _, h := range hs {
		switch h.Kind {
		case HazardUseAfterCancel:
			cancel++
		case HazardWAR:
			war++
		}
	}
	if cancel == 0 || war == 0 {
		t.Fatalf("期望取消后引用与别名 WAR 均暴露，得到 %v", hs)
	}
}

func TestFullBarriersCoverAll(t *testing.T) {
	// 对照组：双队列方案补齐所有屏障后不应有暴露。
	s := schemeDualQueue()
	g := s.Graph
	g.Barriers = append(g.Barriers,
		Barrier{Resource: "Depth", From: "T1", To: "T2", Kind: BarrierSemaphore})
	if hs := exposed(g); len(hs) != 0 {
		t.Fatalf("补齐屏障后仍有暴露：%v", hs)
	}
	// 错误屏障类型不能覆盖：跨队列边用 pipeline 屏障无效。
	g2 := schemeDualQueue().Graph
	g2.Barriers = append(g2.Barriers,
		Barrier{Resource: "Depth", From: "T1", To: "T2", Kind: BarrierPipeline})
	if hs := exposed(g2); len(hs) != 1 {
		t.Fatalf("pipeline 屏障不应覆盖跨队列边，得到 %v", hs)
	}
}
