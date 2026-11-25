package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ufServe(s *Server, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func ufStateFromRec(t *testing.T, rec *httptest.ResponseRecorder) stateResponse {
	t.Helper()
	var st stateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode state: %v body=%s", err, rec.Body.String())
	}
	return st
}

func TestWebIndexAndState(t *testing.T) {
	s := NewServer()
	rec := ufServe(s, http.MethodGet, "/", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "降级控制台") {
		t.Fatalf("index code=%d contains-ui=%v", rec.Code, strings.Contains(rec.Body.String(), "降级控制台"))
	}
	rec = ufServe(s, http.MethodGet, "/api/state", nil)
	st := ufStateFromRec(t, rec)
	if st.FrameReport.Frame != 1 || st.State.EyeCount != 2 || !st.State.History || !st.State.Interaction {
		t.Fatalf("bad initial state: %+v", st.FrameReport)
	}
}

func TestWebBudgetStepsDegradeAndReset(t *testing.T) {
	s := NewServer()
	ufServe(s, http.MethodPost, "/api/reset", nil)
	ufServe(s, http.MethodPost, "/api/step", map[string]any{"frame_time_ms": 13})
	rec := ufServe(s, http.MethodPost, "/api/step", map[string]any{"frame_time_ms": 13})
	st := ufStateFromRec(t, rec)
	if st.State.SampleLevel != 1 || len(st.ActiveEpisodeIDs) != 1 {
		t.Fatalf("expected low sampling active, level=%d eps=%v", st.State.SampleLevel, st.ActiveEpisodeIDs)
	}
	rec = ufServe(s, http.MethodPost, "/api/reset", nil)
	st = ufStateFromRec(t, rec)
	if st.FrameReport.Frame != 1 || st.State.SampleLevel != 0 || st.State.EyeCount != 2 {
		t.Fatalf("reset failed: %+v", st.FrameReport)
	}
}

func TestWebSafetyProbesRejected(t *testing.T) {
	s := NewServer()
	rec := ufServe(s, http.MethodPost, "/api/probe/eyes-zero", nil)
	st := ufStateFromRec(t, rec)
	if st.State.EyeCount != 2 {
		t.Fatalf("eye floor violated: %d", st.State.EyeCount)
	}
	found := false
	for _, ev := range st.SafetyEvents {
		if ev.Code == "eyes_zero_rejected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("eyes rejection not recorded: %+v", st.SafetyEvents)
	}
	rec = ufServe(s, http.MethodPost, "/api/probe/history-silent", nil)
	st = ufStateFromRec(t, rec)
	if !st.State.History {
		t.Fatal("history must stay enabled after silent-disable attempt")
	}
	if len(st.SafetyEvents) != 2 {
		t.Fatalf("expected 2 safety events, got %d", len(st.SafetyEvents))
	}
}

func TestWebPoseInjectTrimRecover(t *testing.T) {
	s := NewServer()
	ufServe(s, http.MethodPost, "/api/reset", nil)
	ufServe(s, http.MethodPost, "/api/inject", map[string]any{"kind": "pose"})
	st := ufStateFromRec(t, ufServe(s, http.MethodPost, "/api/step", map[string]any{}))
	if st.State.Reprojection {
		t.Fatal("pose overflow should disable reprojection")
	}
	ufServe(s, http.MethodPost, "/api/trim", nil)
	ufServe(s, http.MethodPost, "/api/step", map[string]any{})
	st = ufStateFromRec(t, ufServe(s, http.MethodPost, "/api/step", map[string]any{}))
	if !st.State.Reprojection {
		t.Fatal("reprojection should recover after two in-limit frames post-trim")
	}
}

func TestWebScreenPNG(t *testing.T) {
	s := NewServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/screen.png", nil)
	s.mux.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("screen code=%d type=%s", rec.Code, rec.Header().Get("Content-Type"))
	}
	body := rec.Body.Bytes()
	if len(body) < 8 || string(body[:4]) != "\x89PNG" {
		t.Fatalf("not a png, %d bytes", len(body))
	}
}
