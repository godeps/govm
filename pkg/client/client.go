package client

import (
	"context"
	"os"
	"runtime"
	"strings"

	"github.com/godeps/govm/internal/binding"
	"github.com/godeps/govm/internal/offline"
	"github.com/godeps/govm/internal/runtimeassets"
)

type boxProvider interface {
	Start() error
	Stop() error
	Info() (binding.BoxInfo, error)
	Exec(command string, opts binding.ExecOptions) (binding.ExecResult, error)
	Free()
}

type runtimeProvider interface {
	CreateBox(name string, opts binding.BoxOptions) (string, error)
	GetBox(idOrName string) (boxProvider, string, error)
	ListBoxes() ([]binding.BoxInfo, error)
	RemoveBox(idOrName string, force bool) error
	Free()
}

type defaultRuntimeProvider struct {
	rt *binding.Runtime
}

func (p *defaultRuntimeProvider) CreateBox(name string, opts binding.BoxOptions) (string, error) {
	return p.rt.CreateBox(name, opts)
}

func (p *defaultRuntimeProvider) GetBox(idOrName string) (boxProvider, string, error) {
	box, id, err := p.rt.GetBox(idOrName)
	if box == nil {
		return nil, id, err
	}
	return box, id, err
}

func (p *defaultRuntimeProvider) ListBoxes() ([]binding.BoxInfo, error) { return p.rt.ListBoxes() }
func (p *defaultRuntimeProvider) RemoveBox(idOrName string, force bool) error {
	return p.rt.RemoveBox(idOrName, force)
}
func (p *defaultRuntimeProvider) Free() { p.rt.Free() }

// Runtime is the high-level client entry point.
type Runtime struct {
	runtime        runtimeProvider
	cacheHome      string
	defaultNetwork *NetworkConfig
}

// NewRuntime creates a new runtime.
func NewRuntime(opts *RuntimeOptions) (*Runtime, error) {
	cacheHome := resolveCacheHome(opts)
	runtimeDir, err := runtimeassets.Ensure(cacheHome)
	if err != nil {
		return nil, err
	}
	if err := setRuntimeDirEnv(runtimeDir); err != nil {
		return nil, err
	}

	bindingOpts := &binding.RuntimeOptions{}
	if opts != nil {
		bindingOpts.HomeDir = opts.HomeDir
		bindingOpts.ImageRegistries = opts.ImageRegistries
	}
	rt, err := binding.NewRuntime(bindingOpts)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		runtime:        &defaultRuntimeProvider{rt: rt},
		cacheHome:      cacheHome,
		defaultNetwork: resolveRuntimeDefaultNetwork(opts),
	}, nil
}

func newRuntimeWith(p runtimeProvider) *Runtime { return &Runtime{runtime: p, cacheHome: ".govm"} }

// Close releases runtime resources.
func (r *Runtime) Close() {
	if r.runtime != nil {
		r.runtime.Free()
		r.runtime = nil
	}
}

// CreateBox creates and returns a box handle.
func (r *Runtime) CreateBox(ctx context.Context, name string, opts BoxOptions) (*Box, error) {
	_ = ctx
	bundlePath := opts.LocalBundlePath
	if bundlePath == "" {
		bundlePath = opts.RootfsPath
	}
	if opts.OfflineImage != "" && bundlePath == "" {
		rootfsPath, err := offline.EnsureRootfs(r.cacheHome, opts.OfflineImage)
		if err != nil {
			return nil, err
		}
		bundlePath = rootfsPath
	}
	effectiveNetwork := effectiveNetworkConfig(r.defaultNetwork, opts.Network)
	if err := ValidateNetworkConfig(effectiveNetwork); err != nil {
		return nil, err
	}

	networkMode := ""
	networkPolicyMode := ""
	macosNetworkEnabled := true
	var ports []binding.PortForward
	if effectiveNetwork != nil {
		networkMode = string(effectiveNetwork.Mode)
		if effectiveNetwork.Policy != nil {
			networkPolicyMode = string(effectiveNetwork.Policy.Mode)
		}
		if !effectiveNetwork.Enabled {
			macosNetworkEnabled = false
		}
		for _, pf := range effectiveNetwork.PortForwards {
			proto := string(pf.Protocol)
			if proto == "" {
				proto = string(ProtoTCP)
			}
			ports = append(ports, binding.PortForward{
				HostIP: pf.HostIP, HostPort: int(pf.HostPort), GuestPort: int(pf.GuestPort), Protocol: proto,
			})
		}
	}

	id, err := r.runtime.CreateBox(name, binding.BoxOptions{
		Image:               opts.Image,
		RootfsPath:          bundlePath,
		CPUs:                opts.CPUs,
		MemoryMB:            opts.MemoryMB,
		Env:                 opts.Env,
		WorkingDir:          opts.WorkingDir,
		NetworkMode:         networkMode,
		NetworkPolicyMode:   networkPolicyMode,
		PortForwards:        ports,
		MacOSNetworkEnabled: macosNetworkEnabled,
	})
	if err != nil {
		return nil, err
	}
	handle, _, err := r.runtime.GetBox(id)
	if err != nil {
		return nil, err
	}
	return &Box{handle: handle, id: id, name: name, runtime: r}, nil
}

// GetBox returns nil,nil when the box does not exist.
func (r *Runtime) GetBox(ctx context.Context, idOrName string) (*Box, error) {
	_ = ctx
	handle, id, err := r.runtime.GetBox(idOrName)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		return nil, nil
	}
	return &Box{handle: handle, id: id, runtime: r}, nil
}

// ListBoxes returns all boxes in this runtime.
func (r *Runtime) ListBoxes(ctx context.Context) ([]BoxInfo, error) {
	_ = ctx
	infos, err := r.runtime.ListBoxes()
	if err != nil {
		return nil, err
	}
	out := make([]BoxInfo, len(infos))
	for i, info := range infos {
		out[i] = BoxInfo(info)
	}
	return out, nil
}

// RemoveBox removes a box by id or name.
func (r *Runtime) RemoveBox(ctx context.Context, idOrName string, force bool) error {
	_ = ctx
	return r.runtime.RemoveBox(idOrName, force)
}

func setRuntimeDirEnv(runtimeDir string) error {
	current := os.Getenv("BOXLITE_RUNTIME_DIR")
	paths := []string{runtimeDir}
	if current != "" {
		paths = append(paths, current)
	}
	if err := os.Setenv("BOXLITE_RUNTIME_DIR", strings.Join(paths, string(os.PathListSeparator))); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return prependPathEnv("LD_LIBRARY_PATH", runtimeDir)
	case "darwin":
		return prependPathEnv("DYLD_LIBRARY_PATH", runtimeDir)
	default:
		return nil
	}
}

func prependPathEnv(key, entry string) error {
	current := os.Getenv(key)
	parts := []string{entry}
	if current != "" {
		parts = append(parts, current)
	}
	return os.Setenv(key, strings.Join(parts, string(os.PathListSeparator)))
}
