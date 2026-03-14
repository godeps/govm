//go:build cgo && govm_native

package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

func TestNativeMountE2E(t *testing.T) {
	if os.Getenv("GOVM_E2E") != "1" {
		t.Skip("set GOVM_E2E=1 to run native integration tests")
	}

	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "in.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	rt, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()
	name := fmt.Sprintf("govm-mount-%d", time.Now().UnixNano())
	defer func() { _ = rt.RemoveBox(ctx, name, true) }()

	box, err := rt.CreateBox(ctx, name, BoxOptions{
		OfflineImage: "py312-alpine",
		Mounts: []Mount{
			{HostPath: workDir, GuestPath: "/workspace", ReadOnly: false},
		},
	})
	if err != nil {
		t.Fatalf("create box: %v", err)
	}
	defer box.Close()

	if err := box.Start(); err != nil {
		t.Fatalf("start box: %v", err)
	}

	res, err := box.Exec("/bin/sh", &ExecOptions{
		Args: []string{"-lc", "cat /workspace/in.txt && printf 'done' > /workspace/out.txt"},
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exec exit=%d stderr=%v", res.ExitCode, res.Stderr)
	}
	if got := strings.Join(res.Stdout, "\n"); !strings.Contains(got, "seed") {
		t.Fatalf("stdout missing seed marker: %q", got)
	}

	data, err := os.ReadFile(filepath.Join(workDir, "out.txt"))
	if err != nil {
		t.Fatalf("read host output: %v", err)
	}
	if strings.TrimSpace(string(data)) != "done" {
		t.Fatalf("unexpected host output: %q", string(data))
	}

	if err := box.Stop(); err != nil {
		t.Fatalf("stop box: %v", err)
	}
}

func TestNativeExecStreamE2E(t *testing.T) {
	if os.Getenv("GOVM_E2E") != "1" {
		t.Skip("set GOVM_E2E=1 to run native integration tests")
	}

	rt, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()
	name := fmt.Sprintf("govm-stream-%d", time.Now().UnixNano())
	defer func() { _ = rt.RemoveBox(ctx, name, true) }()

	box, err := rt.CreateBox(ctx, name, BoxOptions{OfflineImage: "py312-alpine"})
	if err != nil {
		t.Fatalf("create box: %v", err)
	}
	defer box.Close()

	if err := box.Start(); err != nil {
		t.Fatalf("start box: %v", err)
	}

	var stdout []string
	var stderr []string
	res, err := box.ExecStream("/bin/sh", &ExecOptions{
		Args: []string{"-lc", "echo stream-out && echo stream-err 1>&2"},
	}, ExecStreamCallbacks{
		OnStdout: func(line string) { stdout = append(stdout, line) },
		OnStderr: func(line string) { stderr = append(stderr, line) },
	})
	if err != nil {
		t.Fatalf("exec stream: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exec stream exit=%d stderr=%v", res.ExitCode, res.Stderr)
	}
	if got := strings.Join(stdout, "\n"); !strings.Contains(got, "stream-out") {
		t.Fatalf("stdout callbacks missing marker: %q", got)
	}
	if got := strings.Join(stderr, "\n"); !strings.Contains(got, "stream-err") {
		t.Fatalf("stderr callbacks missing marker: %q", got)
	}
	if got := strings.Join(res.Stdout, "\n"); !strings.Contains(got, "stream-out") {
		t.Fatalf("result stdout missing marker: %q", got)
	}
	if got := strings.Join(res.Stderr, "\n"); !strings.Contains(got, "stream-err") {
		t.Fatalf("result stderr missing marker: %q", got)
	}

	if err := box.Stop(); err != nil {
		t.Fatalf("stop box: %v", err)
	}
}
