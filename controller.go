package main

import "fmt"

const interactionHint = "交互保留：[B]帧预算压力 [F]视锥故障 [P]姿态超限 [空格]步进 [T]裁剪姿态 [R]重置"

// epRuntime tracks one live or finished degradation episode internally.
type epRuntime struct {
	ep                Episode
	mode              string
	active            bool
	causeFrustum      bool
	startPreMetrics   *Metrics
	samples           []Metrics
	levelChanges      []LevelChange
	recoveryCondition RecoveryCondition
}

// chainRuntime tracks one degradation chain internally.
type chainRuntime struct {
	ch        Chain
	runtimeID int
	active    bool
	endFrame  int
}

// Controller is the degradation state machine. A zero value is not usable;
// construct one with NewController(DefaultConfig()).
type Controller struct {
	Cfg   Config
	Frame int
	State State

	// counters
	overStreak      int
	floorOverStreak int
	floorWarnStreak int
	budgetGood      int
	stereoGood      int
	frustumGood     int
	poseGood        int

	// id sequences
	alertSeq int
	epSeq    int
	chainSeq int

	// records
	Alerts       []*Alert
	Audit        []*AuditEntry
	SafetyEvents []*SafetyEvent

	// runtime collections
	episodes []*epRuntime
	chains   []*chainRuntime

	// sticky / presentation state
	lastChain           *chainRuntime
	safetyStatusLine    string
	historyUserDisabled bool
}

// NewController builds a controller starting from full nominal capability.
func NewController(cfg Config) *Controller {
	if len(cfg.SampleScales) == 0 {
		cfg = DefaultConfig()
	}
	c := &Controller{Cfg: cfg}
	c.Reset()
	return c
}

// Reset restores the initial nominal state and clears all records.
func (c *Controller) Reset() {
	c.Frame = 0
	c.State = nominalState(c.Cfg.SampleScales)
	c.overStreak, c.floorOverStreak, c.floorWarnStreak = 0, 0, 0
	c.budgetGood, c.stereoGood, c.frustumGood, c.poseGood = 0, 0, 0, 0
	c.alertSeq, c.epSeq, c.chainSeq = 0, 0, 0
	c.Alerts, c.Audit, c.SafetyEvents = nil, nil, nil
	c.episodes, c.chains = nil, nil
	c.lastChain = nil
	c.safetyStatusLine = ""
	c.historyUserDisabled = false
}

func (c *Controller) newID(prefix string, seq *int) string {
	*seq++
	return fmt.Sprintf("%s%04d", prefix, *seq)
}

func (c *Controller) audit(frame int, level, code, msg, epID, chainID string) {
	c.Audit = append(c.Audit, &AuditEntry{
		Frame: frame, Level: level, Code: code, Message: msg,
		EpisodeID: epID, ChainID: chainID,
	})
}

func (c *Controller) activeEpisode(mode string) *epRuntime {
	for _, ep := range c.episodes {
		if ep.active && ep.mode == mode {
			return ep
		}
	}
	return nil
}

func (c *Controller) activeEpisodes() []*epRuntime {
	out := []*epRuntime{}
	for _, ep := range c.episodes {
		if ep.active {
			out = append(out, ep)
		}
	}
	return out
}

func (c *Controller) anyActiveEpisode() bool {
	return len(c.activeEpisodes()) > 0
}

// Step ingests one frame of signals, advances the state machine and returns
// the canonical frame report.
func (c *Controller) Step(sig Signals) *FrameReport {
	c.Frame++
	f := c.Frame

	pre := ComputeMetrics(f, sig, c.Cfg, c.State)

	activeAtStart := map[*epRuntime]bool{}
	for _, ep := range c.activeEpisodes() {
		activeAtStart[ep] = true
	}

	c.applyBudget(f, sig, pre)
	geared := c.applySamplingRecovery(f, sig)
	c.applyStereoRecovery(f, sig, geared)
	c.applyReprojection(f, sig)
	c.updateFloorWarning(f, sig)

	m := ComputeMetrics(f, sig, c.Cfg, c.State)

	// Record canonical samples for every episode touched by this frame.
	for _, ep := range c.episodes {
		joinedThisFrame := ep.ep.StartFrame == f
		if activeAtStart[ep] || joinedThisFrame || (ep.ep.EndFrame == f && !ep.active) {
			ep.samples = append(ep.samples, m)
		}
	}

	sd := c.buildStatus(f, sig)
	var frameActions []FrameActionRecord
	if sig.PoseCount > c.Cfg.PoseLimit {
		frameActions = append(frameActions, FrameActionRecord{
			Kind: "trim_pose_sequence", Keep: c.Cfg.PoseLimit,
			Message: fmt.Sprintf("将姿态序列裁剪到最近 %d 个姿态", c.Cfg.PoseLimit),
		})
	}
	fr := &FrameReport{
		Frame: f, Signals: sig, State: c.State, Metrics: m,
		Actions:     frameActions,
		StatusLevel: sd.Level, Headline: sd.Headline, StatusDetail: sd,
	}
	if c.lastChain != nil {
		fr.ChainID = c.lastChain.ch.ID
	}
	return fr
}
