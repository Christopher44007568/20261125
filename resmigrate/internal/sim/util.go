package sim

import "strconv"

func itoa(n int) string { return strconv.Itoa(n) }

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// rng is a tiny deterministic LCG so every scenario is reproducible without
// any third-party dependency.
type rng struct {
	state uint64
}

func newRNG(seed uint64) *rng {
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return &rng{state: seed}
}

func (r *rng) next() uint64 {
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state
}

func (r *rng) float64() float64 {
	return float64(r.next()>>11) / float64(uint64(1)<<53)
}

// meter tracks simulated RGBA history and linear frame-buffer allocations.
// Every allocation models a real buffer so peak memory is deterministic.
type meter struct {
	historyLive int
	imageLive   int

	historyPeak int
	imagePeak   int
	peak        int
	baseline    int

	allocCount int
	freeCount  int
}

// allocBytes reserves a generic buffer and classifies it as history or image
// storage via forHistory.
func (m *meter) allocBytes(bytes int, forHistory bool) {
	if forHistory {
		m.historyLive += bytes
	} else {
		m.imageLive += bytes
	}
	m.allocCount++
	m.observe()
}

func (m *meter) freeBytes(bytes int, forHistory bool) {
	if forHistory {
		m.historyLive -= bytes
		if m.historyLive < 0 {
			m.historyLive = 0
		}
	} else {
		m.imageLive -= bytes
		if m.imageLive < 0 {
			m.imageLive = 0
		}
	}
	m.freeCount++
}

func (m *meter) allocHistory(pixels int) int {
	bytes := pixels * historyBytesPerPx
	m.historyLive += bytes
	m.allocCount++
	m.observe()
	return bytes
}

func (m *meter) freeHistory(pixels int) {
	m.historyLive -= pixels * historyBytesPerPx
	if m.historyLive < 0 {
		m.historyLive = 0
	}
	m.freeCount++
}

func (m *meter) allocImage(pixels int) int {
	bytes := pixels * imageBytesPerPx
	m.imageLive += bytes
	m.allocCount++
	m.observe()
	return bytes
}

func (m *meter) freeImage(pixels int) {
	m.imageLive -= pixels * imageBytesPerPx
	if m.imageLive < 0 {
		m.imageLive = 0
	}
	m.freeCount++
}

func (m *meter) observe() {
	if m.historyLive > m.historyPeak {
		m.historyPeak = m.historyLive
	}
	if m.imageLive > m.imagePeak {
		m.imagePeak = m.imageLive
	}
	if total := m.historyLive + m.imageLive; total > m.peak {
		m.peak = total
	}
}

func (m *meter) snapshot() int {
	m.observe()
	return m.historyLive + m.imageLive
}

func (m *meter) markBaseline() {
	m.baseline = m.historyLive + m.imageLive
}
