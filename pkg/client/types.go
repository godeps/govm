package client

import "time"

// BoxOptions configures a new box.
type BoxOptions struct {
	Image string `json:"image"`
	// LocalBundlePath points to a local OCI image layout directory.
	// When set, it takes precedence over Image.
	LocalBundlePath string `json:"local_bundle_path,omitempty"`
	// RootfsPath is kept for backward compatibility and aliases LocalBundlePath.
	// Deprecated: use LocalBundlePath.
	RootfsPath string `json:"rootfs_path,omitempty"`
	// OfflineImage references an embedded offline image bundle shipped in this repo.
	// Runtime.CreateBox will extract it and fill RootfsPath automatically.
	OfflineImage string            `json:"offline_image,omitempty"`
	CPUs         int               `json:"cpus,omitempty"`
	MemoryMB     int               `json:"memory_mb,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	Network      *NetworkConfig    `json:"network,omitempty"`
}

// BoxInfo contains metadata and state of a box.
type BoxInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Image     string    `json:"image"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// ExecOptions controls execution behavior for a command.
type ExecOptions struct {
	Args       []string
	Env        map[string]string
	TTY        bool
	User       string
	Timeout    time.Duration
	WorkingDir string
}

// ExecResult is the final result of command execution.
type ExecResult struct {
	ExitCode int
	Stdout   []string
	Stderr   []string
}

// RuntimeOptions configures Runtime creation.
type RuntimeOptions struct {
	HomeDir         string
	ImageRegistries []string
	NetworkDefaults *RuntimeNetworkDefaults
}
