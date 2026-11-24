package binocular

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type sessionState uint8

const (
	sessionActive sessionState = iota
	sessionClosing
	sessionClosed
)

// TaskFunc is the unit of work scheduled by a Session. Implementations must
// watch ctx and return promptly when it is canceled.
type TaskFunc func(context.Context, *Task) error

type Task struct {
	id      uint64
	eye     Eye
	kind    ResourceKind
	fn      TaskFunc
	session *Session

	mu        sync.Mutex
	state     TaskState
	started   bool
	startedAt time.Time
	err       error
}

func (t *Task) ID() uint64         { return t.id }
func (t *Task) Eye() Eye           { return t.eye }
func (t *Task) Kind() ResourceKind { return t.kind }

func (t *Task) State() TaskState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

func (t *Task) setRunning() {
	t.mu.Lock()
	t.state = TaskRunning
	t.started = true
	t.startedAt = time.Now()
	t.mu.Unlock()
}

func (t *Task) finish(state TaskState, err error) {
	t.mu.Lock()
	t.state = state
	t.err = err
	t.mu.Unlock()
}

func (t *Task) queuedCancel() {
	t.mu.Lock()
	t.state = TaskCanceled
	t.mu.Unlock()
}

func (t *Task) snapshot() (TaskState, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state, t.started
}

type privateState struct {
	mu              sync.Mutex
	token           string
	epoch           uint64
	parentSessionID uint64
	historyFrames   int
}

func newPrivateState() *privateState {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		copy(raw[:], []byte(fmt.Sprintf("%016x", time.Now().UnixNano())))
	}
	return &privateState{token: hex.EncodeToString(raw[:])}
}

func (p *privateState) bumpEpoch() {
	p.mu.Lock()
	p.epoch++
	p.mu.Unlock()
}

func (p *privateState) noteHistory() {
	p.mu.Lock()
	p.historyFrames++
	p.mu.Unlock()
}

// Lease is a capability to write one resource of one session generation.
// It intentionally carries no pointer that can be reused by a later session.
type Lease struct {
	id         uint64
	sessionID  uint64
	generation uint64
	kind       ResourceKind
	eye        Eye
	bufferID   string
	buffer     *frameBuffer
	revoked    bool
}

func (l *Lease) ID() uint64         { return l.id }
func (l *Lease) Kind() ResourceKind { return l.kind }
func (l *Lease) Eye() Eye           { return l.eye }

type Session struct {
	id         uint64
	generation uint64
	baseline   Frame

	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.Mutex
	state     sessionState
	queue     chan *Task
	tasks     map[uint64]*Task
	nextTask  uint64
	leases    map[uint64]*Lease
	nextLease uint64
	private   *privateState
	res       *resourceSet
	errors    []string

	sem chan struct{}
	wg  sync.WaitGroup

	closeOnce sync.Once
	report    ReleaseReport
}

func newSession(id, generation uint64, baseline Frame) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		id:         id,
		generation: generation,
		baseline:   baseline,
		ctx:        ctx,
		cancel:     cancel,
		state:      sessionActive,
		queue:      make(chan *Task, 256),
		tasks:      make(map[uint64]*Task),
		leases:     make(map[uint64]*Lease),
		private:    newPrivateState(),
		res:        newResourceSet(id),
		sem:        make(chan struct{}, 8),
	}
	s.wg.Add(1)
	go s.dispatch()
	return s
}

func (s *Session) ID() uint64         { return s.id }
func (s *Session) Generation() uint64 { return s.generation }

// RenderFrame snapshots the currently active session buffers.
func (s *Session) RenderFrame() (Frame, error) {
	s.mu.Lock()
	if s.state != sessionActive {
		s.mu.Unlock()
		return Frame{}, ErrSessionClosed
	}
	s.mu.Unlock()
	left, right := s.res.colorSnapshots()
	return Frame{Left: left, Right: right}, nil
}

// AcquireLease creates a generation-bound write capability. A lease from an
// old session is rejected by every later session even if its ID is reused.
func (s *Session) AcquireLease(kind ResourceKind, eye Eye) (*Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != sessionActive {
		return nil, ErrSessionClosed
	}
	if !eye.Valid() {
		return nil, fmt.Errorf("invalid eye %q", eye)
	}
	lease := &Lease{
		sessionID:  s.id,
		generation: s.generation,
		kind:       kind,
		eye:        eye,
	}
	if kind != ResourceCache {
		buf, err := s.res.bufferFor(kind, eye)
		if err != nil {
			return nil, err
		}
		lease.buffer = buf
		lease.bufferID = buf.id
	}
	s.nextLease++
	lease.id = s.nextLease
	s.leases[lease.id] = lease
	return lease, nil
}

// Write applies data only while the lease's session generation is active.
func (s *Session) Write(lease *Lease, off int, data []byte) error {
	if lease == nil {
		return ErrStaleLease
	}
	s.mu.Lock()
	if s.state != sessionActive {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	if lease.sessionID != s.id || lease.generation != s.generation {
		s.mu.Unlock()
		return ErrStaleLease
	}
	if lease.revoked {
		s.mu.Unlock()
		return ErrLeaseRevoked
	}
	buf := lease.buffer
	kind := lease.kind
	s.mu.Unlock()
	if kind == ResourceCache || buf == nil {
		return fmt.Errorf("%w: cache leases use PutCache", ErrUnknownBuffer)
	}
	return buf.write(off, data)
}

func (s *Session) PutCache(lease *Lease, key, value string) error {
	if lease == nil {
		return ErrStaleLease
	}
	s.mu.Lock()
	if s.state != sessionActive {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	if lease.sessionID != s.id || lease.generation != s.generation {
		s.mu.Unlock()
		return ErrStaleLease
	}
	if lease.revoked {
		s.mu.Unlock()
		return ErrLeaseRevoked
	}
	if lease.kind != ResourceCache {
		s.mu.Unlock()
		return fmt.Errorf("%w: lease is %s", ErrUnknownBuffer, lease.kind)
	}
	s.mu.Unlock()
	s.res.putCache(key, value)
	return nil
}

// Submit queues a task. The dispatcher bounds concurrency, so surplus work
// remains queued and is canceled as part of session teardown.
func (s *Session) Submit(eye Eye, kind ResourceKind, fn TaskFunc) (*Task, error) {
	if fn == nil {
		return nil, errors.New("nil task function")
	}
	if !eye.Valid() {
		return nil, fmt.Errorf("invalid eye %q", eye)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != sessionActive {
		return nil, ErrSessionClosed
	}
	s.nextTask++
	t := &Task{
		id:      s.nextTask,
		eye:     eye,
		kind:    kind,
		fn:      fn,
		session: s,
		state:   TaskQueued,
	}
	s.tasks[t.id] = t
	select {
	case s.queue <- t:
	default:
		t.state = TaskCanceled
		return nil, fmt.Errorf("task queue full")
	}
	return t, nil
}

func (s *Session) dispatch() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			s.drainQueue()
			return
		case t := <-s.queue:
			if !s.isActive() {
				t.queuedCancel()
				s.drainQueue()
				return
			}
			select {
			case s.sem <- struct{}{}:
			case <-s.ctx.Done():
				t.queuedCancel()
				s.drainQueue()
				return
			}
			s.wg.Add(1)
			go s.runTask(t)
		}
	}
}

func (s *Session) runTask(t *Task) {
	defer s.wg.Done()
	defer func() { <-s.sem }()
	t.setRunning()
	err := t.fn(s.ctx, t)
	switch {
	case s.ctx.Err() != nil:
		t.finish(TaskStopped, nil)
	case err != nil:
		t.finish(TaskFailed, err)
		s.mu.Lock()
		s.errors = append(s.errors, fmt.Sprintf("task %d (%s/%s): %v", t.id, t.eye, t.kind, err))
		s.mu.Unlock()
	default:
		t.finish(TaskStopped, nil)
	}
}

func (s *Session) drainQueue() {
	for {
		select {
		case t := <-s.queue:
			t.queuedCancel()
		default:
			return
		}
	}
}

func (s *Session) isActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == sessionActive
}

// Close cancels every queued and running task, waits for all workers to exit,
// then releases resources. It is idempotent and safe for concurrent callers.
func (s *Session) Close(scope DisconnectScope) ReleaseReport {
	s.closeOnce.Do(func() { s.performClose(scope) })
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneReleaseReport(s.report)
}

func (s *Session) performClose(scope DisconnectScope) {
	start := time.Now()
	s.mu.Lock()
	s.state = sessionClosing
	running := 0
	queued := 0
	for _, t := range s.tasks {
		state, started := t.snapshot()
		switch {
		case state == TaskRunning:
			running++
		case state == TaskQueued && !started:
			queued++
		}
	}
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()

	s.mu.Lock()
	s.state = sessionClosed
	revoked := 0
	for _, l := range s.leases {
		if !l.revoked {
			l.revoked = true
			revoked++
		}
	}
	taskRelease, taskUnreleased := s.taskReleaseLocked(running, queued)
	taskIDs := make([]string, 0, len(s.tasks))
	for id := range s.tasks {
		taskIDs = append(taskIDs, fmt.Sprintf("%d", id))
	}
	taskErrors := append([]string(nil), s.errors...)
	s.mu.Unlock()
	sort.Strings(taskIDs)

	categories, resourceUnreleased := s.res.release()
	queueCat := ReleaseCategory{
		Name:       "queue_tasks",
		Discovered: taskRelease.Submitted,
		Released:   taskRelease.Stopped + taskRelease.QueuedCancel,
		Failed:     taskRelease.Failed,
		OK:         taskRelease.StillRunning == 0 && taskRelease.Failed == 0,
		Objects:    taskIDs,
	}
	categories = append(categories, queueCat)
	sort.Slice(categories, func(i, j int) bool {
		order := map[string]int{
			"textures": 0, "depth": 1, "history": 2, "queue_tasks": 3, "cache_keys": 4,
		}
		return order[categories[i].Name] < order[categories[j].Name]
	})

	unreleased := append([]ObjectRef(nil), taskUnreleased...)
	unreleased = append(unreleased, resourceUnreleased...)
	if unreleased == nil {
		unreleased = []ObjectRef{}
	}
	for i := range unreleased {
		unreleased[i].Session = s.id
	}
	complete := true
	for _, c := range categories {
		if !c.OK {
			complete = false
		}
	}
	if taskRelease.StillRunning != 0 {
		complete = false
	}
	for _, e := range taskErrors {
		_ = e
		complete = false
	}

	report := ReleaseReport{
		Scope:         scope,
		SessionID:     s.id,
		Generation:    s.generation,
		Categories:    categories,
		Tasks:         taskRelease,
		LeasesRevoked: revoked,
		Unreleased:    unreleased,
		Complete:      complete,
		TeardownNS:    time.Since(start).Nanoseconds(),
	}
	s.mu.Lock()
	s.report = report
	s.mu.Unlock()
}

func (s *Session) taskReleaseLocked(runningAtStop, queuedAtStop int) (TaskRelease, []ObjectRef) {
	result := TaskRelease{
		Submitted: len(s.tasks),
		Running:   runningAtStop,
		Queued:    queuedAtStop,
	}
	var unreleased []ObjectRef
	for _, t := range s.tasks {
		state, started := t.snapshot()
		switch state {
		case TaskStopped, TaskCanceled:
			if started {
				result.Stopped++
			} else {
				result.QueuedCancel++
			}
		case TaskFailed:
			result.Failed++
		default:
			result.StillRunning++
			unreleased = append(unreleased, ObjectRef{
				Kind:   "task",
				ID:     fmt.Sprintf("%d", t.id),
				Detail: string(state),
			})
		}
	}
	return result, unreleased
}

func cloneReleaseReport(in ReleaseReport) ReleaseReport {
	out := in
	out.Categories = append([]ReleaseCategory(nil), in.Categories...)
	if in.Unreleased != nil {
		out.Unreleased = append([]ObjectRef{}, in.Unreleased...)
	}
	return out
}

type privateSnapshot struct {
	Token           string
	Epoch           uint64
	ParentSessionID uint64
	HistoryFrames   int
	CacheEntries    int
}

func (s *Session) snapshotPrivate() privateSnapshot {
	s.private.mu.Lock()
	snap := privateSnapshot{
		Token:           s.private.token,
		Epoch:           s.private.epoch,
		ParentSessionID: s.private.parentSessionID,
		HistoryFrames:   s.private.historyFrames,
	}
	s.private.mu.Unlock()
	snap.CacheEntries = s.res.cacheSize()
	return snap
}

// UnreleasedObjects inspects a closed session. A correctly closed session
// returns an empty slice.
func (s *Session) UnreleasedObjects() []ObjectRef {
	s.mu.Lock()
	var out []ObjectRef
	for _, t := range s.tasks {
		state, _ := t.snapshot()
		if !state.terminal() {
			out = append(out, ObjectRef{
				Kind:    "task",
				ID:      fmt.Sprintf("%d", t.id),
				Session: s.id,
				Detail:  string(state),
			})
		}
	}
	for _, l := range s.leases {
		if !l.revoked {
			out = append(out, ObjectRef{
				Kind:    "lease",
				ID:      fmt.Sprintf("%d", l.id),
				Session: s.id,
				Detail:  l.bufferID,
			})
		}
	}
	s.mu.Unlock()
	out = append(out, s.res.unreleased()...)
	for i := range out {
		out[i].Session = s.id
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].ID < out[j].ID
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func (s *Session) TaskCounts() (running, queued, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		state, _ := t.snapshot()
		total++
		switch state {
		case TaskRunning:
			running++
		case TaskQueued:
			queued++
		}
	}
	return running, queued, total
}
