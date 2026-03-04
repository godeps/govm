package runtimeassets

import "testing"

func TestRequiredRuntimeFiles(t *testing.T) {
	got := requiredRuntimeFiles("linux_amd64")
	if len(got) < 3 {
		t.Fatalf("expected required files for linux_amd64, got %v", got)
	}
	if got[0] != "boxlite-shim" || got[1] != "boxlite-guest" {
		t.Fatalf("unexpected required files order: %v", got)
	}
}

func TestShouldBeExecutable(t *testing.T) {
	for _, n := range []string{"boxlite-shim", "boxlite-guest", "bwrap", "mke2fs", "debugfs"} {
		if !shouldBeExecutable(n) {
			t.Fatalf("expected executable: %s", n)
		}
	}
	if shouldBeExecutable("libkrunfw.so.5") {
		t.Fatalf("library should not be marked as required executable")
	}
}
