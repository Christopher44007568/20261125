package colorlab

import "testing"

func TestAllChainMetrics(t *testing.T) {
	metrics := AllChainMetrics(StandardChart())
	if len(metrics) != 12 {
		t.Fatalf("got %d chains, want 12", len(metrics))
	}
	for _, metric := range metrics {
		if metric.Samples != 33 {
			t.Errorf("%s -> %s has %d samples, want 33",
				metric.From, metric.To, metric.Samples)
		}
		if metric.PeakError > 1e-9 {
			t.Errorf("%s -> %s peak error = %.3g, want <= 1e-9",
				metric.From, metric.To, metric.PeakError)
		}
		if metric.Peak8Bit > 1 {
			t.Errorf("%s -> %s 8-bit delta = %d, want <= 1",
				metric.From, metric.To, metric.Peak8Bit)
		}
	}
}

func TestMonotonicChecks(t *testing.T) {
	checks := CheckMonotonicity()
	for _, check := range checks {
		if !check.Monotonic || !check.ZeroOK || !check.OneOK {
			t.Errorf("%s failed: %+v", check.Name, check)
		}
		if check.Samples < 65536 {
			t.Errorf("%s sampled only %d points", check.Name, check.Samples)
		}
	}
}
