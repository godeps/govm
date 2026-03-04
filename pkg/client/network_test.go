package client

import (
	"context"
	"errors"
	"testing"
)

func TestStrictProfileDefaults(t *testing.T) {
	cfg := StrictNetworkProfile()
	if cfg == nil {
		t.Fatal("expected strict profile config")
	}
	if !cfg.Enabled {
		t.Fatal("strict profile should enable network stack with deny policy intent")
	}
	if cfg.Mode != NetworkNAT {
		t.Fatalf("expected NAT mode, got %q", cfg.Mode)
	}
	if cfg.Policy == nil || cfg.Policy.Mode != PolicyBlockAll {
		t.Fatalf("expected block_all policy, got %#v", cfg.Policy)
	}
}

func TestValidateNetworkConfigRejectsInvalidPortRange(t *testing.T) {
	cfg := &NetworkConfig{
		Enabled: true,
		Mode:    NetworkNAT,
		PortForwards: []PortForward{
			{HostPort: 7000, GuestPort: 80, Protocol: ProtoTCP},
			{HostPort: 7001, GuestPort: 80, Protocol: Protocol("icmp")},
		},
	}
	if err := ValidateNetworkConfig(cfg); !errors.Is(err, ErrNetworkInvalidConfig) {
		t.Fatalf("expected ErrNetworkInvalidConfig, got %v", err)
	}
}

func TestValidateNetworkConfigRejectsBridged(t *testing.T) {
	cfg := &NetworkConfig{Enabled: true, Mode: NetworkBridged}
	if err := ValidateNetworkConfig(cfg); !errors.Is(err, ErrNetworkUnsupportedPlatform) {
		t.Fatalf("expected ErrNetworkUnsupportedPlatform, got %v", err)
	}
}

func TestCreateBoxMergesRuntimeNetworkDefaults(t *testing.T) {
	m := newMockRuntimeProvider()
	r := newRuntimeWith(m)
	r.defaultNetwork = StrictNetworkProfile()

	_, err := r.CreateBox(context.Background(), "demo-net", BoxOptions{Image: "alpine:latest"})
	if err != nil {
		t.Fatal(err)
	}

	if m.lastCreate.NetworkMode != string(NetworkNAT) {
		t.Fatalf("expected NAT mode mapped, got %q", m.lastCreate.NetworkMode)
	}
	if m.lastCreate.NetworkPolicyMode != string(PolicyBlockAll) {
		t.Fatalf("expected block_all policy mapped, got %q", m.lastCreate.NetworkPolicyMode)
	}
}

func TestCreateBoxOverrideDisablesNetwork(t *testing.T) {
	m := newMockRuntimeProvider()
	r := newRuntimeWith(m)
	r.defaultNetwork = StrictNetworkProfile()

	_, err := r.CreateBox(context.Background(), "demo-off", BoxOptions{
		Image:   "alpine:latest",
		Network: &NetworkConfig{Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}

	if m.lastCreate.NetworkMode != string(NetworkDisabled) {
		t.Fatalf("expected disabled mode, got %q", m.lastCreate.NetworkMode)
	}
}
