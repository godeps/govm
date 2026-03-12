package client

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

var ErrMountInvalidConfig = errors.New("invalid mount config")

type Mount struct {
	HostPath  string `json:"host_path,omitempty"`
	GuestPath string `json:"guest_path,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

func ValidateMounts(mounts []Mount) error {
	seen := make(map[string]struct{}, len(mounts))
	for i, mount := range mounts {
		if strings.TrimSpace(mount.HostPath) == "" {
			return fmt.Errorf("%w: mount[%d] host_path is required", ErrMountInvalidConfig, i)
		}
		if strings.TrimSpace(mount.GuestPath) == "" {
			return fmt.Errorf("%w: mount[%d] guest_path is required", ErrMountInvalidConfig, i)
		}
		if !strings.HasPrefix(mount.GuestPath, "/") {
			return fmt.Errorf("%w: mount[%d] guest_path must be absolute", ErrMountInvalidConfig, i)
		}
		clean := path.Clean(mount.GuestPath)
		if clean != mount.GuestPath {
			return fmt.Errorf("%w: mount[%d] guest_path must be clean", ErrMountInvalidConfig, i)
		}
		if _, ok := seen[mount.GuestPath]; ok {
			return fmt.Errorf("%w: duplicate guest_path %q", ErrMountInvalidConfig, mount.GuestPath)
		}
		seen[mount.GuestPath] = struct{}{}
	}
	return nil
}
