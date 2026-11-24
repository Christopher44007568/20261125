package main

// 固定的版本调度：提交顺序在三个方案中保持一致，
// 变化只集中在资源依赖与屏障上。
//
//	T1 main-opaque   graphics  W:SceneColor W:Depth
//	T2 async-reproj  按方案定  R:Depth R:HistoryColor W:ReprojOut
//	T3 composite     graphics  R:SceneColor R:ReprojOut W:Backbuffer
//	T4 history-clear graphics  W:HistoryColor
func baseResources() map[string]*Resource {
	return map[string]*Resource{
		"SceneColor":   {ID: "SceneColor"},
		"Depth":        {ID: "Depth"},
		"HistoryColor": {ID: "HistoryColor"},
		"ReprojOut":    {ID: "ReprojOut"},
		"Backbuffer":   {ID: "Backbuffer"},
	}
}

func baseTasks(reprojQueue Queue) []*Task {
	return []*Task{
		{ID: "T1", Name: "main-opaque", Queue: QueueGraphics, Duration: 3,
			Accesses: []Access{
				{Resource: "SceneColor", Kind: AccessWrite, Stage: "color"},
				{Resource: "Depth", Kind: AccessWrite, Stage: "depth"},
			}},
		{ID: "T2", Name: "async-reproj", Queue: reprojQueue, Duration: 2,
			Accesses: []Access{
				{Resource: "Depth", Kind: AccessRead, Stage: "sample"},
				{Resource: "HistoryColor", Kind: AccessRead, Stage: "sample"},
				{Resource: "ReprojOut", Kind: AccessWrite, Stage: "cs-write"},
			}},
		{ID: "T3", Name: "composite", Queue: QueueGraphics, Duration: 2,
			Accesses: []Access{
				{Resource: "SceneColor", Kind: AccessRead, Stage: "sample"},
				{Resource: "ReprojOut", Kind: AccessRead, Stage: "sample"},
				{Resource: "Backbuffer", Kind: AccessWrite, Stage: "color"},
			}},
		{ID: "T4", Name: "history-clear", Queue: QueueGraphics, Duration: 1,
			Accesses: []Access{
				{Resource: "HistoryColor", Kind: AccessWrite, Stage: "clear"},
			}},
	}
}

// Scheme 是一种资源/队列组织方案，附带注入的故障说明。
type Scheme struct {
	Graph  *Graph
	Faults []string
}

// 方案一：单队列。全部提交到 graphics，靠管线屏障排序。
// 注入故障：T2 读 HistoryColor 之后，T4 清写它的管线屏障被遗漏（WAR）。
func schemeSingleQueue() Scheme {
	g := &Graph{Name: "单队列", Resources: baseResources(), Tasks: baseTasks(QueueGraphics)}
	g.Barriers = []Barrier{
		{Resource: "SceneColor", From: "T1", To: "T3", Kind: BarrierPipeline},
		{Resource: "Depth", From: "T1", To: "T2", Kind: BarrierPipeline},
		{Resource: "ReprojOut", From: "T2", To: "T3", Kind: BarrierPipeline},
		// 注入：缺少 HistoryColor T2->T4 的 pipeline 屏障
	}
	return Scheme{Graph: g, Faults: []string{
		"注入 WAR：HistoryColor 在 T2(读) 与 T4(写) 之间缺少 pipeline 屏障",
	}}
}

// 方案二：双队列。重投影提交到 async compute 队列，跨队列依赖用信号量。
// 注入故障：Depth 的 T1->T2 跨队列信号量被遗漏（RAW，写后读）。
// 完成顺序不等于提交顺序：T2 在 compute 队列上先于 T1 完成。
func schemeDualQueue() Scheme {
	g := &Graph{Name: "双队列", Resources: baseResources(), Tasks: baseTasks(QueueCompute)}
	g.Barriers = []Barrier{
		{Resource: "SceneColor", From: "T1", To: "T3", Kind: BarrierPipeline},
		// 注入：缺少 Depth T1->T2 的 semaphore
		{Resource: "HistoryColor", From: "T2", To: "T4", Kind: BarrierSemaphore},
		{Resource: "ReprojOut", From: "T2", To: "T3", Kind: BarrierSemaphore},
	}
	return Scheme{Graph: g, Faults: []string{
		"注入 RAW：Depth 在 T1(写,graphics) 与 T2(读,compute) 之间缺少 semaphore",
	}}
}

// 方案三：显存别名。ReprojOut 与 HistoryColor 共享同一显存分配，
// 别名切换需要 aliasing 屏障。
// 注入故障一：T3 读 ReprojOut 之后，T4 经别名写同一分配，aliasing 屏障遗漏。
// 注入故障二：T2 被取消（预算回收），但 T3 仍采样 ReprojOut。
func schemeAliasing() Scheme {
	res := baseResources()
	res["ReprojOut"].Alias = "HistoryColor"
	tasks := baseTasks(QueueGraphics)
	tasks[1].Cancelled = true // 注入：重投影任务取消
	g := &Graph{Name: "显存别名", Resources: res, Tasks: tasks}
	g.Barriers = []Barrier{
		{Resource: "SceneColor", From: "T1", To: "T3", Kind: BarrierPipeline},
		{Resource: "Depth", From: "T1", To: "T2", Kind: BarrierPipeline},
		// 注入：缺少 ReprojOut/HistoryColor 别名分配上 T3->T4 的 aliasing 屏障
	}
	return Scheme{Graph: g, Faults: []string{
		"注入取消后仍被引用：T2 取消，T3 仍读 ReprojOut",
		"注入别名 WAR：T4 写 HistoryColor 经别名覆盖 T3 正在读的 ReprojOut",
	}}
}

func allSchemes() []Scheme {
	return []Scheme{schemeSingleQueue(), schemeDualQueue(), schemeAliasing()}
}
