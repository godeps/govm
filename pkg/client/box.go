package client

import "github.com/godeps/govm/internal/binding"

// Box is a handle to one sandbox VM.
type Box struct {
	handle  boxProvider
	id      string
	name    string
	runtime *Runtime
}

func (b *Box) ID() string   { return b.id }
func (b *Box) Name() string { return b.name }

func (b *Box) Start() error { return b.handle.Start() }
func (b *Box) Stop() error  { return b.handle.Stop() }

func (b *Box) Info() (BoxInfo, error) {
	info, err := b.handle.Info()
	if err != nil {
		return BoxInfo{}, err
	}
	return BoxInfo(info), nil
}

func (b *Box) Exec(command string, opts *ExecOptions) (*ExecResult, error) {
	bindOpts := binding.ExecOptions{}
	if opts != nil {
		bindOpts.Args = opts.Args
		bindOpts.Env = opts.Env
		bindOpts.TTY = opts.TTY
		bindOpts.User = opts.User
		bindOpts.WorkingDir = opts.WorkingDir
		if opts.Timeout > 0 {
			bindOpts.TimeoutSec = opts.Timeout.Seconds()
		}
	}

	result, err := b.handle.Exec(command, bindOpts)
	if err != nil {
		return nil, err
	}
	out := ExecResult(result)
	return &out, nil
}

// Close only frees the handle; it does not remove the underlying box.
func (b *Box) Close() {
	if b.handle != nil {
		b.handle.Free()
		b.handle = nil
	}
}
