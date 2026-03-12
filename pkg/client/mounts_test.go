package client

import (
	"context"
	"errors"
	"testing"
)

func TestValidateMountsRejectsRelativeGuestPath(t *testing.T) {
	err := ValidateMounts([]Mount{
		{HostPath: "/tmp/in", GuestPath: "workspace"},
	})
	if !errors.Is(err, ErrMountInvalidConfig) {
		t.Fatalf("expected ErrMountInvalidConfig, got %v", err)
	}
}

func TestCreateBoxMapsMounts(t *testing.T) {
	m := newMockRuntimeProvider()
	r := newRuntimeWith(m)

	_, err := r.CreateBox(context.Background(), "demo-mounts", BoxOptions{
		Image: "alpine:latest",
		Mounts: []Mount{
			{HostPath: "/tmp/input", GuestPath: "/workspace/in", ReadOnly: true},
			{HostPath: "/tmp/output", GuestPath: "/workspace/out", ReadOnly: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(m.lastCreate.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(m.lastCreate.Mounts))
	}
	if got := m.lastCreate.Mounts[0]; got.HostPath != "/tmp/input" || got.GuestPath != "/workspace/in" || !got.ReadOnly {
		t.Fatalf("unexpected first mount: %#v", got)
	}
	if got := m.lastCreate.Mounts[1]; got.HostPath != "/tmp/output" || got.GuestPath != "/workspace/out" || got.ReadOnly {
		t.Fatalf("unexpected second mount: %#v", got)
	}
}
