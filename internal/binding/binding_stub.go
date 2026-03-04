//go:build !govm_native

package binding

import (
	"errors"
	"time"
)

var errNativeUnavailable = errors.New("govm native bridge unavailable: build with -tags govm_native after vendoring native libs")

type BoxOptions struct {
	Image               string            `json:"image"`
	RootfsPath          string            `json:"rootfs_path,omitempty"`
	CPUs                int               `json:"cpus,omitempty"`
	MemoryMB            int               `json:"memory_mb,omitempty"`
	Env                 map[string]string `json:"env,omitempty"`
	WorkingDir          string            `json:"working_dir,omitempty"`
	NetworkMode         string            `json:"network_mode,omitempty"`
	NetworkPolicyMode   string            `json:"network_policy_mode,omitempty"`
	PortForwards        []PortForward     `json:"port_forwards,omitempty"`
	MacOSNetworkEnabled bool              `json:"macos_network_enabled,omitempty"`
}

type PortForward struct {
	HostIP    string `json:"host_ip,omitempty"`
	HostPort  int    `json:"host_port,omitempty"`
	GuestPort int    `json:"guest_port,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type BoxInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Image     string    `json:"image"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type RuntimeOptions struct {
	HomeDir         string   `json:"home_dir,omitempty"`
	ImageRegistries []string `json:"image_registries,omitempty"`
}

type ExecOptions struct {
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TTY        bool              `json:"tty,omitempty"`
	User       string            `json:"user,omitempty"`
	TimeoutSec float64           `json:"timeout_secs,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
}

type ExecResult struct {
	ExitCode int
	Stdout   []string
	Stderr   []string
}

type Box struct{}

func (b *Box) Start() error           { return errNativeUnavailable }
func (b *Box) Stop() error            { return errNativeUnavailable }
func (b *Box) Info() (BoxInfo, error) { return BoxInfo{}, errNativeUnavailable }
func (b *Box) Exec(command string, opts ExecOptions) (ExecResult, error) {
	return ExecResult{}, errNativeUnavailable
}
func (b *Box) Free() {}

type Runtime struct{}

func NewRuntime(opts *RuntimeOptions) (*Runtime, error) { return nil, errNativeUnavailable }
func (r *Runtime) CreateBox(name string, opts BoxOptions) (string, error) {
	return "", errNativeUnavailable
}
func (r *Runtime) GetBox(idOrName string) (*Box, string, error) { return nil, "", errNativeUnavailable }
func (r *Runtime) ListBoxes() ([]BoxInfo, error)                { return nil, errNativeUnavailable }
func (r *Runtime) RemoveBox(idOrName string, force bool) error  { return errNativeUnavailable }
func (r *Runtime) Free()                                        {}
