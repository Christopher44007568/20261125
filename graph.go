package main

// BuildEdges 按固定提交顺序，对每个资源把相邻访问连成依赖边。
// 边的语义：同一资源上，前一次访问（提交者）必须先对后一次访问（使用者）可见。
// 显存别名：对资源 R 的写会传播到与 R 共享同一分配的所有资源。
func (g *Graph) BuildEdges() []Edge {
	last := map[string]struct {
		task string
		kind AccessKind
	}{}
	queueOf := map[string]Queue{}
	for _, t := range g.Tasks {
		queueOf[t.ID] = t.Queue
	}
	var edges []Edge
	for _, t := range g.Tasks {
		for _, a := range g.effectiveAccesses(t) {
			prev, ok := last[a.Resource]
			if ok && prev.task != t.ID { // 跳过同一任务内的自边
				edges = append(edges, Edge{
					Resource: a.Resource,
					From:     prev.task,
					To:       t.ID,
					FromKind: prev.kind,
					ToKind:   a.Kind,
					CrossQ:   queueOf[prev.task] != t.Queue,
				})
			}
			last[a.Resource] = struct {
				task string
				kind AccessKind
			}{t.ID, a.Kind}
		}
	}
	return edges
}

// effectiveAccesses 返回任务的声明访问加上别名传播出的隐式写。
func (g *Graph) effectiveAccesses(t *Task) []Access {
	out := make([]Access, 0, len(t.Accesses))
	for _, a := range t.Accesses {
		out = append(out, a)
		if a.Kind != AccessWrite {
			continue
		}
		for _, r := range g.Resources {
			if r.ID == a.Resource {
				continue
			}
			aliasOf := g.Resources[a.Resource]
			if r.Alias == a.Resource || (aliasOf != nil && aliasOf.Alias == r.ID) {
				out = append(out, Access{Resource: r.ID, Kind: AccessWrite, Stage: "alias"})
			}
		}
	}
	return out
}

// Interval 是一次任务在模拟时间线上的执行区间。
type Interval struct {
	TaskID string
	Queue  Queue
	Start  int
	End    int
}

// Simulate 按队列分别推进时间线：同一队列内保持提交顺序，
// 不同队列并发推进。完成顺序由 End 排序得出，通常不等于提交顺序。
// 取消的任务不占执行时间，但其资源引用关系仍保留在图中。
func (g *Graph) Simulate() []Interval {
	clock := map[Queue]int{}
	var ivs []Interval
	for _, t := range g.Tasks {
		if t.Cancelled {
			continue
		}
		start := clock[t.Queue]
		end := start + t.Duration
		clock[t.Queue] = end
		ivs = append(ivs, Interval{TaskID: t.ID, Queue: t.Queue, Start: start, End: end})
	}
	return ivs
}

// CompletionOrder 返回按完成时间排序的任务 ID（稳定排序，并列时按提交序）。
func (g *Graph) CompletionOrder(ivs []Interval) []string {
	idx := map[string]int{}
	for i, t := range g.Tasks {
		idx[t.ID] = i
	}
	sorted := make([]Interval, len(ivs))
	copy(sorted, ivs)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0; j-- {
			a, b := sorted[j-1], sorted[j]
			if a.End > b.End || (a.End == b.End && idx[a.TaskID] > idx[b.TaskID]) {
				sorted[j-1], sorted[j] = b, a
			} else {
				break
			}
		}
	}
	out := make([]string, len(sorted))
	for i, iv := range sorted {
		out[i] = iv.TaskID
	}
	return out
}
