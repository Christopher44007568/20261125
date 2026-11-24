package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSimulationCompletesWithoutLeaks(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out); err != nil {
		t.Fatalf("run: %v", err)
	}
	output := out.String()
	for _, want := range []string{
		"new frame references old right-history texture",
		"remains alive",
		"released at frame 7",
		"created=18 live=0 retiring=0 refs=0 leaks=0 dangling=0 issues=0",
		"PASS: no leaks, dangling references, or premature releases",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}
