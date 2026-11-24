package main

// AccessKind 描述一次资源访问的读写类型。
type AccessKind int

const (
	AccessRead AccessKind = iota
	AccessWrite
)

func (k AccessKind) String() string {
	if k == AccessRead {
		return "R"
	}
	return "W"
}

// Queue 是提交队列。单队列方案只有 graphics。
type Queue string

const (
	QueueGraphics Queue = "graphics"
	QueueCompute  Queue = "compute"
)

// Access 是某任务对某资源的一次访问。
type Access struct {
	Resource string
	Kind     AccessKind
	Stage    string // 管线阶段，仅用于报告
}

// Task 是一次提交（pass）。提交顺序固定，由版本调度决定。
type Task struct {
	ID        string
	Name      string
	Queue     Queue
	Accesses  []Access
	Duration  int  // 模拟执行耗时（时间片）
	Cancelled bool // 取消后仍可能被引用
}

// Edge 是依赖图中的一条边：生产者 -> 消费者。
type Edge struct {
	Resource string
	From     string // 提交者 task ID
	To       string // 使用者 task ID
	FromKind AccessKind
	ToKind   AccessKind
	CrossQ   bool // 是否跨队列
}

// BarrierKind 是同步原语类型。
type BarrierKind int

const (
	BarrierNone      BarrierKind = iota
	BarrierPipeline              // 同队列管线屏障
	BarrierSemaphore             // 跨队列信号量
	BarrierAliasing              // 显存别名屏障
)

func (k BarrierKind) String() string {
	switch k {
	case BarrierPipeline:
		return "pipeline"
	case BarrierSemaphore:
		return "semaphore"
	case BarrierAliasing:
		return "aliasing"
	}
	return "none"
}

// Barrier 覆盖 From 任务到 To 任务之间对 Resource 的同步。
type Barrier struct {
	Resource string
	From     string
	To       string
	Kind     BarrierKind
}

// Resource 是被共享的纹理；Alias 指向同一显存分配的另一资源。
type Resource struct {
	ID    string
	Alias string // 非空表示与本资源别名共享同一分配
}

// Graph 是资源依赖图：固定的提交序列 + 资源表 + 屏障表。
type Graph struct {
	Name      string
	Resources map[string]*Resource
	Tasks     []*Task // 提交顺序（固定）
	Barriers  []Barrier
}
