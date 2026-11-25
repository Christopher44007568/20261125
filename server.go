package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

// WebSession is the single local interactive simulation session.
type WebSession struct {
	mu         sync.Mutex
	cfg        Config
	c          *Controller
	frames     []*FrameReport
	report     *ScenarioReport
	inject     Signals
	injectKind string
}

// Server bundles the mux and session for httptest use.
type Server struct {
	mux  *http.ServeMux
	sess *WebSession
}

// NewServer builds the HTTP application (no listening yet).
func NewServer() *Server {
	s := &Server{}
	s.sess = newSession(DefaultConfig())
	mux := http.NewServeMux()
	s.mux = mux

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/step", s.handleStep)
	mux.HandleFunc("/api/inject", s.handleInject)
	mux.HandleFunc("/api/trim", s.handleTrim)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/invalidate", s.handleInvalidate)
	mux.HandleFunc("/api/probe/eyes-zero", s.handleEyesProbe)
	mux.HandleFunc("/api/probe/history-silent", s.handleHistoryProbe)
	mux.HandleFunc("/screen.png", s.handleScreen)
	mux.HandleFunc("/api/report", s.handleReport)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return s
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, BuildReport(s.sess.cfg))
}

func runServer(addr string) error {
	s := NewServer()
	return http.ListenAndServe(addr, s.mux)
}

func newSession(cfg Config) *WebSession {
	ws := &WebSession{cfg: cfg, inject: Signals{FrameTimeMS: 9}}
	ws.reset()
	return ws
}

func (ws *WebSession) reset() {
	ws.c = NewController(ws.cfg)
	ws.frames = []*FrameReport{}
	ws.injectKind = ""
	ws.inject = Signals{FrameTimeMS: 9}
	ws.stepLocked(ws.inject)
}

func (ws *WebSession) stepLocked(sig Signals) *FrameReport {
	fr := ws.c.Step(sig)
	fr.Scenario = "live_console"
	ws.frames = append(ws.frames, fr)
	def := ScenarioDef{ID: "live_console", Title: "Live Console"}
	ws.report = BuildScenarioReport(def, ws.cfg, ws.c, ws.frames)
	return fr
}

type stateResponse struct {
	*FrameReport
	SafetyEvents     []*SafetyEvent `json:"safety_events"`
	Injection        Signals        `json:"injection"`
	InjectKind       string         `json:"inject_kind"`
	ActiveEpisodeIDs []string       `json:"active_episode_ids"`
}

func (s *Server) currentState() stateResponse {
	ws := s.sess
	ws.mu.Lock()
	defer ws.mu.Unlock()
	fr := ws.frames[len(ws.frames)-1]
	ids := []string{}
	for _, e := range ws.report.Episodes {
		if e.Outcome == "active" {
			ids = append(ids, e.ID)
		}
	}
	return stateResponse{
		FrameReport: fr, SafetyEvents: ws.c.SafetyEvents,
		Injection: ws.inject, InjectKind: ws.injectKind, ActiveEpisodeIDs: ids,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.currentState())
}

// decodeSignals merges an optional JSON request body over the current injection.
func decodeSignals(r *http.Request, base Signals) Signals {
	sig := base
	if r.Body != nil && r.ContentLength != 0 {
		var in Signals
		if err := json.NewDecoder(r.Body).Decode(&in); err == nil {
			if in.FrameTimeMS != 0 {
				sig.FrameTimeMS = in.FrameTimeMS
			}
			if in.FrustumFailed {
				sig.FrustumFailed = true
				sig.FrustumCode = in.FrustumCode
			}
			if in.PoseCount != 0 {
				sig.PoseCount = in.PoseCount
			}
			sig.PoseInvalid = in.PoseInvalid
		}
	}
	return sig
}

func (s *Server) handleStep(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	sig := decodeSignals(r, ws.inject)
	fr := ws.stepLocked(sig)
	ws.mu.Unlock()
	writeJSON(w, stateResponseShim(s, fr))
}

func stateResponseShim(s *Server, _ *FrameReport) stateResponse { return s.currentState() }

func (s *Server) handleInject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind    string  `json:"kind"`
		FrameMS float64 `json:"frame_time_ms"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ws := s.sess
	ws.mu.Lock()
	if req.FrameMS > 0 {
		ws.inject = Signals{FrameTimeMS: req.FrameMS}
		ws.injectKind = "custom"
	} else {
		if ws.injectKind == req.Kind {
			ws.injectKind = ""
			ws.inject = Signals{FrameTimeMS: 9}
		} else {
			ws.injectKind = req.Kind
			switch req.Kind {
			case "budget":
				ws.inject = Signals{FrameTimeMS: 13}
			case "severe":
				ws.inject = Signals{FrameTimeMS: 20}
			case "frustum":
				ws.inject = Signals{FrameTimeMS: 9.4, FrustumFailed: true, FrustumCode: "aspect_zero"}
			case "pose":
				ws.inject = Signals{FrameTimeMS: 9.2, PoseCount: 22}
			default:
				ws.injectKind = ""
				ws.inject = Signals{FrameTimeMS: 9}
			}
		}
	}
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleTrim(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	ws.inject.PoseCount = ws.c.TrimPoseCount(ws.inject.PoseCount)
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	ws.reset()
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleInvalidate(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	ws.c.InvalidateEpisode(ws.c.Frame, "Web 核验：触发信号为上一帧残留读数，本帧实际合法")
	def := ScenarioDef{ID: "live_console"}
	ws.report = BuildScenarioReport(def, ws.cfg, ws.c, ws.frames)
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleEyesProbe(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	ws.c.RequestZeroEyes("web-user")
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleHistoryProbe(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	ws.c.RequestSilentHistoryDisable("degradation")
	ws.mu.Unlock()
	writeJSON(w, s.currentState())
}

func (s *Server) handleScreen(w http.ResponseWriter, r *http.Request) {
	ws := s.sess
	ws.mu.Lock()
	frameNo := ws.c.Frame
	pngRaw, err := RenderShotPNG(ws.report, frameNo)
	ws.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(pngRaw)
}
