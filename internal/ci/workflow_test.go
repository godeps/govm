package ci

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildNativeWorkflowDownloadsArtifactsIntoInternalDir(t *testing.T) {
	t.Helper()

	workflowPath := filepath.Join("..", "..", ".github", "workflows", "build-native.yml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}

	workflow := string(content)
	if !strings.Contains(workflow, "uses: actions/download-artifact@v4") {
		t.Fatalf("workflow missing download-artifact step")
	}

	if !strings.Contains(workflow, "merge-multiple: true") {
		t.Fatalf("workflow missing merge-multiple setting")
	}

	if !strings.Contains(workflow, "path: internal") {
		t.Fatalf("workflow should download merged artifacts into internal/ so platform-check finds internal/native and internal/runtimeassets")
	}

	if !strings.Contains(workflow, "scripts/find-boxlite-runtime-dir.sh") {
		t.Fatalf("workflow should resolve runtime dir via scripts/find-boxlite-runtime-dir.sh")
	}
}

func TestFindBoxliteRuntimeDirPrefersRuntimeWithRequiredFiles(t *testing.T) {
	t.Helper()

	targetDir := t.TempDir()

	incomplete := filepath.Join(targetDir, "x86_64-unknown-linux-gnu", "release", "build", "boxlite-old", "out", "runtime")
	complete := filepath.Join(targetDir, "x86_64-unknown-linux-gnu", "release", "build", "boxlite-new", "out", "runtime")

	for _, dir := range []string{incomplete, complete} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	for _, name := range []string{"bwrap", "debugfs", "libkrunfw.so.5", "mke2fs"} {
		if err := os.WriteFile(filepath.Join(incomplete, name), []byte(name), 0o755); err != nil {
			t.Fatalf("write incomplete runtime file %s: %v", name, err)
		}
	}

	for _, name := range []string{"boxlite-guest", "boxlite-shim", "bwrap", "debugfs", "libkrunfw.so.5", "mke2fs"} {
		if err := os.WriteFile(filepath.Join(complete, name), []byte(name), 0o755); err != nil {
			t.Fatalf("write complete runtime file %s: %v", name, err)
		}
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "find-boxlite-runtime-dir.sh")
	cmd := exec.Command(scriptPath, targetDir)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("find runtime dir: %v\n%s", err, output)
	}

	got := strings.TrimSpace(string(output))
	if got != complete {
		t.Fatalf("expected runtime dir %q, got %q", complete, got)
	}
}

func TestFindBoxliteRuntimeDirAcceptsWorkspaceRootLayout(t *testing.T) {
	t.Helper()

	workspaceRoot := t.TempDir()
	targetDir := filepath.Join(workspaceRoot, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}

	for _, name := range []string{"boxlite-guest", "boxlite-shim", "bwrap", "debugfs", "libkrunfw.so.5", "mke2fs"} {
		if err := os.WriteFile(filepath.Join(workspaceRoot, name), []byte(name), 0o755); err != nil {
			t.Fatalf("write runtime file %s: %v", name, err)
		}
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "find-boxlite-runtime-dir.sh")
	cmd := exec.Command(scriptPath, targetDir)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("find runtime dir: %v\n%s", err, output)
	}

	got := strings.TrimSpace(string(output))
	if got != workspaceRoot {
		t.Fatalf("expected workspace root runtime dir %q, got %q", workspaceRoot, got)
	}
}
