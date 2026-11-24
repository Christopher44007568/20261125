package main

import "fmt"

// HazardKind 是注入/检测的冲突类型。
type HazardKind int

const (
	HazardRAW            HazardKind = iota // 写后读：读方未等到写方完成
	HazardWAR                              // 读后写：写方覆盖了仍在被读的数据
	HazardWAW                              // 写后写
	HazardUseAfterCancel                   // 取消后仍被引用
)

func (k HazardKind) String() string {
	switch k {
	case HazardRAW:
		return "RAW(写后读)"
	case HazardWAR:
		return "WAR(读后写)"
	case HazardWAW:
		return "WAW(写后写)"
	case HazardUseAfterCancel:
		return "UseAfterCancel(取消后仍被引用)"
	}
	return "?"
}

// Hazard 是一条冲突记录：Covered 表示已有屏障正确覆盖。
type Hazard struct {
	Kind     HazardKind
	Resource string
	From     string
	To       string
	CrossQ   bool
	Covered  bool
	Note     string
}

func (h Hazard) String() string {
	status := "暴露"
	if h.Covered {
		status = "已同步"
	}
	return fmt.Sprintf("[%s] %s: %s -> %s (%s)%s",
		h.Kind, h.Resource, h.From, h.To, status, h.Note)
}

// barrierCovers 判断屏障表是否覆盖 from->to 对 resource 的同步。
// 同队列需要 pipeline 屏障；跨队列需要 semaphore；别名切换需要 aliasing 屏障。
func barrierCovers(barriers []Barrier, resource, from, to string, crossQ bool) bool {
	for _, b := range barriers {
		if b.Resource != resource || b.From != from || b.To != to {
			continue
		}
		switch b.Kind {
		case BarrierSemaphore:
			if crossQ {
				return true
			}
		case BarrierPipeline:
			if !crossQ {
				return true
			}
		case BarrierAliasing:
			return true // 别名屏障对同/跨队列的别名切换均有效
		}
	}
	return false
}

// Validate 在执行时间线上验证依赖边，而不是按提交顺序臆测可见性。
// 完成顺序不等于提交顺序：跨队列的边若缺少信号量，
// 即使提交序正确也缺少"生产者先于使用者完成"的保证。
func (g *Graph) Validate() []Hazard {
	ivs := g.Simulate()
	span := map[string]Interval{}
	for _, iv := range ivs {
		span[iv.TaskID] = iv
	}
	taskOf := map[string]*Task{}
	for _, t := range g.Tasks {
		taskOf[t.ID] = t
	}
	var hazards []Hazard
	for _, e := range g.BuildEdges() {
		// 取消后仍被引用：生产者已取消，使用者仍访问其输出。
		if taskOf[e.From].Cancelled && !taskOf[e.To].Cancelled {
			hazards = append(hazards, Hazard{
				Kind:     HazardUseAfterCancel,
				Resource: e.Resource,
				From:     e.From,
				To:       e.To,
				CrossQ:   e.CrossQ,
				Covered:  false, // 屏障无法复活已取消的写入
				Note:     " 生产者已取消，引用悬空",
			})
			continue
		}
		if taskOf[e.To].Cancelled {
			continue // 使用者取消，边自然失效
		}
		kind, risky := classify(e)
		if !risky {
			continue // 读-读无冲突
		}
		covered := barrierCovers(g.Barriers, e.Resource, e.From, e.To, e.CrossQ)
		h := Hazard{
			Kind:     kind,
			Resource: e.Resource,
			From:     e.From,
			To:       e.To,
			CrossQ:   e.CrossQ,
			Covered:  covered,
		}
		if e.CrossQ && !covered {
			a, b := span[e.From], span[e.To]
			h.Note = fmt.Sprintf(" 执行区间 [%d,%d) 与 [%d,%d) 无跨队列同步",
				a.Start, a.End, b.Start, b.End)
		}
		hazards = append(hazards, h)
	}
	return hazards
}

// classify 把一条依赖边归类为冲突类型；读-读不冲突。
func classify(e Edge) (HazardKind, bool) {
	switch {
	case e.FromKind == AccessWrite && e.ToKind == AccessRead:
		return HazardRAW, true
	case e.FromKind == AccessRead && e.ToKind == AccessWrite:
		return HazardWAR, true
	case e.FromKind == AccessWrite && e.ToKind == AccessWrite:
		return HazardWAW, true
	}
	return 0, false
}
