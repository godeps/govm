package runtimeassets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

//go:embed runtime/**
var runtimeFS embed.FS

var ErrRuntimeNotFound = errors.New("embedded runtime assets not found")
var ErrRuntimeInvalid = errors.New("embedded runtime assets invalid")

// Ensure extracts embedded runtime binaries for current platform and returns runtime dir.
func Ensure(cacheBaseDir string) (string, error) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	sub := filepath.ToSlash(filepath.Join("runtime", platform))
	entries, err := fs.ReadDir(runtimeFS, sub)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrRuntimeNotFound, platform)
	}

	h := sha256.New()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(runtimeFS, filepath.ToSlash(filepath.Join(sub, e.Name())))
		if err != nil {
			return "", err
		}
		_, _ = h.Write(b)
	}
	digest := hex.EncodeToString(h.Sum(nil)[:8])
	targetDir := filepath.Join(cacheBaseDir, "runtime", platform+"-"+digest)
	marker := filepath.Join(targetDir, ".ready")
	if _, err := os.Stat(marker); err == nil {
		if err := ValidateDir(targetDir, platform); err == nil {
			return targetDir, nil
		}
		_ = os.RemoveAll(targetDir)
	}
	if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		b, err := fs.ReadFile(runtimeFS, filepath.ToSlash(filepath.Join(sub, name)))
		if err != nil {
			return "", err
		}
		mode := os.FileMode(0o644)
		if shouldBeExecutable(name) {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(targetDir, name), b, mode); err != nil {
			return "", err
		}
	}
	if err := ValidateDir(targetDir, platform); err != nil {
		return "", err
	}
	if err := os.WriteFile(marker, []byte("ok\n"), 0o644); err != nil {
		return "", err
	}
	return targetDir, nil
}

func Platform() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}

// SupportedPlatforms lists embedded runtime platform directories.
func SupportedPlatforms() ([]string, error) {
	entries, err := fs.ReadDir(runtimeFS, "runtime")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// ValidateDir validates required runtime files and basic executable permissions.
func ValidateDir(dir, platform string) error {
	required := requiredRuntimeFiles(platform)
	for _, name := range required {
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("%w: missing required file %s", ErrRuntimeInvalid, path)
		}
		if st.IsDir() {
			return fmt.Errorf("%w: expected file but got dir %s", ErrRuntimeInvalid, path)
		}
		if shouldBeExecutable(name) && st.Mode()&0o111 == 0 {
			return fmt.Errorf("%w: file is not executable %s", ErrRuntimeInvalid, path)
		}
	}
	for _, name := range optionalRuntimeExecutables() {
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !st.IsDir() && st.Mode()&0o111 == 0 {
			return fmt.Errorf("%w: optional runtime executable is not executable %s", ErrRuntimeInvalid, path)
		}
	}
	return nil
}

func requiredRuntimeFiles(platform string) []string {
	out := []string{"boxlite-shim", "boxlite-guest"}
	switch platform {
	case "linux_amd64", "linux_arm64":
		out = append(out, "libkrunfw.so.5")
	case "darwin_arm64":
		out = append(out, "libkrunfw.5.dylib")
	}
	return out
}

func optionalRuntimeExecutables() []string {
	return []string{"bwrap", "mke2fs", "debugfs"}
}

func shouldBeExecutable(name string) bool {
	switch {
	case name == "boxlite-shim", name == "boxlite-guest":
		return true
	case name == "bwrap", name == "mke2fs", name == "debugfs":
		return true
	default:
		return false
	}
}
