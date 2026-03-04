//go:build cgo && govm_native

package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNativeLifecycleE2E(t *testing.T) {
	if os.Getenv("GOVM_E2E") != "1" {
		t.Skip("set GOVM_E2E=1 to run native integration tests")
	}

	rt, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()
	name := fmt.Sprintf("govm-e2e-%d", time.Now().UnixNano())
	defer func() { _ = rt.RemoveBox(ctx, name, true) }()

	box, err := rt.CreateBox(ctx, name, BoxOptions{OfflineImage: "py312-alpine"})
	if err != nil {
		t.Fatalf("create box: %v", err)
	}
	defer box.Close()

	if err := box.Start(); err != nil {
		t.Fatalf("start box: %v", err)
	}

	res, err := box.Exec("/bin/sh", &ExecOptions{Args: []string{"-lc", "echo govm-e2e && python3 -V"}})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exec exit=%d stderr=%v", res.ExitCode, res.Stderr)
	}
	joined := strings.Join(res.Stdout, "\n")
	if !strings.Contains(joined, "govm-e2e") {
		t.Fatalf("stdout missing marker: %q", joined)
	}
	if !strings.Contains(joined, "Python 3.12") {
		t.Fatalf("stdout missing python version: %q", joined)
	}

	if err := box.Stop(); err != nil {
		t.Fatalf("stop box: %v", err)
	}

	if err := rt.RemoveBox(ctx, name, true); err != nil {
		t.Fatalf("remove box: %v", err)
	}
}
