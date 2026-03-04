package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/godeps/govm/internal/runtimeassets"
	"github.com/godeps/govm/pkg/client"
)

type doctorResult struct {
	Platform                 string                        `json:"platform"`
	BridgeFound              bool                          `json:"bridge_found"`
	BridgePath               string                        `json:"bridge_path"`
	RuntimeValid             bool                          `json:"runtime_valid"`
	RuntimeDir               string                        `json:"runtime_dir,omitempty"`
	RuntimeError             string                        `json:"runtime_error,omitempty"`
	EmbeddedRuntimePlatforms []string                      `json:"embedded_runtime_platforms"`
	ExpectedPlatforms        []string                      `json:"expected_platforms"`
	MissingNativePlatforms   []string                      `json:"missing_native_platforms,omitempty"`
	MissingRuntimePlatforms  []string                      `json:"missing_runtime_platforms,omitempty"`
	OfflineBundles           []client.OfflineImageMetadata `json:"offline_bundles,omitempty"`
	OfflineError             string                        `json:"offline_error,omitempty"`
	NativeReady              bool                          `json:"native_ready"`
}

func main() {
	var (
		jsonOut bool
		strict  bool
	)
	flag.BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	flag.BoolVar(&strict, "strict", false, "fail when expected cross-platform assets are missing")
	flag.Parse()

	platform := runtime.GOOS + "_" + runtime.GOARCH
	expectedPlatforms := []string{"linux_amd64", "linux_arm64", "darwin_arm64"}
	if !strict {
		strict = strings.EqualFold(os.Getenv("GOVM_DOCTOR_STRICT"), "1")
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get cwd: %v\n", err)
		os.Exit(1)
	}

	nativeDir := filepath.Join(root, "internal", "native", platform)
	bridge := filepath.Join(nativeDir, "libgovm_boxlite_bridge.a")
	bridgeFound := true
	if _, err := os.Stat(bridge); err != nil {
		bridgeFound = false
	}

	cacheHome := resolveCacheHome()
	runtimeDir, runtimeErr := runtimeassets.Ensure(cacheHome)
	if runtimeErr == nil {
		runtimeErr = runtimeassets.ValidateDir(runtimeDir, platform)
	}
	supportedPlatforms, _ := runtimeassets.SupportedPlatforms()
	sort.Strings(supportedPlatforms)

	missingNative := make([]string, 0)
	missingRuntime := make([]string, 0)
	for _, p := range expectedPlatforms {
		nativePath := filepath.Join(root, "internal", "native", p, "libgovm_boxlite_bridge.a")
		if _, err := os.Stat(nativePath); err != nil {
			missingNative = append(missingNative, p)
		}
		runtimePath := filepath.Join(root, "internal", "runtimeassets", "runtime", p)
		if _, err := os.Stat(runtimePath); err != nil {
			missingRuntime = append(missingRuntime, p)
		}
	}

	offlineMeta, offlineErr := client.ListOfflineImageMetadata()

	res := doctorResult{
		Platform:                 platform,
		BridgeFound:              bridgeFound,
		BridgePath:               bridge,
		RuntimeValid:             runtimeErr == nil,
		RuntimeDir:               runtimeDir,
		EmbeddedRuntimePlatforms: supportedPlatforms,
		ExpectedPlatforms:        expectedPlatforms,
		MissingNativePlatforms:   missingNative,
		MissingRuntimePlatforms:  missingRuntime,
		OfflineBundles:           offlineMeta,
	}
	if runtimeErr != nil {
		res.RuntimeError = runtimeErr.Error()
	}
	if offlineErr != nil {
		res.OfflineError = offlineErr.Error()
	}
	res.NativeReady = bridgeFound && runtimeErr == nil && (!strict || (len(missingNative) == 0 && len(missingRuntime) == 0))

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Printf("platform: %s\n", res.Platform)
		if res.BridgeFound {
			fmt.Printf("bridge: ok (%s)\n", res.BridgePath)
		} else {
			fmt.Printf("bridge: missing (%s)\n", res.BridgePath)
		}
		if res.RuntimeValid {
			fmt.Printf("runtime: ok (%s)\n", res.RuntimeDir)
		} else {
			fmt.Printf("runtime: invalid (%v)\n", res.RuntimeError)
		}
		fmt.Printf("embedded runtime platforms: %v\n", res.EmbeddedRuntimePlatforms)
		if len(res.MissingNativePlatforms) > 0 {
			fmt.Printf("missing native platforms (expected %v): %v\n", res.ExpectedPlatforms, res.MissingNativePlatforms)
		}
		if len(res.MissingRuntimePlatforms) > 0 {
			fmt.Printf("missing runtime platforms (expected %v): %v\n", res.ExpectedPlatforms, res.MissingRuntimePlatforms)
		}
		if res.OfflineError != "" {
			fmt.Printf("offline bundles: failed to inspect (%v)\n", res.OfflineError)
		} else {
			fmt.Printf("offline bundles: %d\n", len(res.OfflineBundles))
			for _, m := range res.OfflineBundles {
				fmt.Printf("  - %s size=%d sha256=%s\n", m.Name, m.SizeBytes, shortSHA(m.SHA256))
			}
		}
	}

	if !res.NativeReady {
		os.Exit(2)
	}
	if !jsonOut {
		fmt.Println("native setup looks ready")
	}
}

func resolveCacheHome() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".govm")
	}
	return ".govm"
}

func shortSHA(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}
