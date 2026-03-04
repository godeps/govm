package client

import (
	"errors"
	"time"

	"github.com/godeps/govm/internal/binding"
)

type mockBoxProvider struct {
	id         string
	name       string
	info       binding.BoxInfo
	execResult binding.ExecResult
	execErr    error
}

func (m *mockBoxProvider) Start() error { return nil }
func (m *mockBoxProvider) Stop() error  { return nil }
func (m *mockBoxProvider) Info() (binding.BoxInfo, error) {
	if m.info.ID == "" {
		m.info = binding.BoxInfo{ID: m.id, Name: m.name, Image: "alpine:latest", State: "running", CreatedAt: time.Now()}
	}
	return m.info, nil
}
func (m *mockBoxProvider) Exec(command string, opts binding.ExecOptions) (binding.ExecResult, error) {
	if m.execErr != nil {
		return binding.ExecResult{}, m.execErr
	}
	if m.execResult.ExitCode == 0 && len(m.execResult.Stdout) == 0 && len(m.execResult.Stderr) == 0 {
		return binding.ExecResult{ExitCode: 0, Stdout: []string{"ok"}}, nil
	}
	return m.execResult, nil
}
func (m *mockBoxProvider) Free() {}

type mockRuntimeProvider struct {
	boxes      map[string]*mockBoxProvider
	createErr  error
	getErr     error
	listErr    error
	removeErr  error
	freed      bool
	lastCreate binding.BoxOptions
}

func newMockRuntimeProvider() *mockRuntimeProvider {
	return &mockRuntimeProvider{boxes: map[string]*mockBoxProvider{}}
}

func (m *mockRuntimeProvider) AddBox(id, image string) *mockBoxProvider {
	b := &mockBoxProvider{id: id, info: binding.BoxInfo{ID: id, Image: image, State: "configured", CreatedAt: time.Now()}}
	m.boxes[id] = b
	return b
}

func (m *mockRuntimeProvider) CreateBox(name string, opts binding.BoxOptions) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	m.lastCreate = opts
	id := name
	if id == "" {
		id = "box-auto"
	}
	if _, ok := m.boxes[id]; !ok {
		m.AddBox(id, opts.Image)
	}
	return id, nil
}

func (m *mockRuntimeProvider) GetBox(idOrName string) (boxProvider, string, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	b, ok := m.boxes[idOrName]
	if !ok {
		return nil, "", nil
	}
	return b, b.id, nil
}

func (m *mockRuntimeProvider) ListBoxes() ([]binding.BoxInfo, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]binding.BoxInfo, 0, len(m.boxes))
	for _, b := range m.boxes {
		info, _ := b.Info()
		out = append(out, info)
	}
	return out, nil
}

func (m *mockRuntimeProvider) RemoveBox(idOrName string, force bool) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	if _, ok := m.boxes[idOrName]; !ok {
		return errors.New("not found")
	}
	delete(m.boxes, idOrName)
	return nil
}

func (m *mockRuntimeProvider) Free() { m.freed = true }
