//go:build mage

package main

import "testing"

func TestMergedEnvOverridesExistingValues(t *testing.T) {
	got := mergedEnv([]string{
		"PATH=/usr/bin",
		"GOBIN=/old/bin",
		"KEEP=value",
	}, map[string]string{
		"PATH":  "/tmp/rpccgo-bin:/usr/bin",
		"GOBIN": "/tmp/rpccgo-bin",
	})

	want := []string{
		"PATH=/tmp/rpccgo-bin:/usr/bin",
		"GOBIN=/tmp/rpccgo-bin",
		"KEEP=value",
	}
	if len(got) != len(want) {
		t.Fatalf("mergedEnv length = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mergedEnv[%d] = %q, want %q; all: %v", i, got[i], want[i], got)
		}
	}
}
