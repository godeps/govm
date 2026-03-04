package client

import (
	"os"
	"path/filepath"
)

func resolveCacheHome(opts *RuntimeOptions) string {
	if opts != nil && opts.HomeDir != "" {
		return opts.HomeDir
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".govm"
	}
	return filepath.Join(home, ".govm")
}
