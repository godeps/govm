package client

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNetworkUnsupportedPlatform = errors.New("network unsupported on this platform")
	ErrNetworkInvalidConfig       = errors.New("invalid network config")
)

type NetworkMode string

const (
	NetworkDisabled NetworkMode = "disabled"
	NetworkNAT      NetworkMode = "nat"
	NetworkBridged  NetworkMode = "bridged"
)

type PolicyMode string

const (
	PolicyBlockAll PolicyMode = "block_all"
	PolicyAllowAll PolicyMode = "allow_all"
)

type Protocol string

const (
	ProtoTCP Protocol = "tcp"
	ProtoUDP Protocol = "udp"
	ProtoAny Protocol = "any"
)

type CIDRRule struct {
	CIDR      string   `json:"cidr"`
	Protocol  Protocol `json:"protocol,omitempty"`
	PortStart uint16   `json:"port_start,omitempty"`
	PortEnd   uint16   `json:"port_end,omitempty"`
}

type DomainRule struct {
	Domain   string   `json:"domain"`
	Port     uint16   `json:"port,omitempty"`
	Protocol Protocol `json:"protocol,omitempty"`
}

type PortForward struct {
	HostIP    string   `json:"host_ip,omitempty"`
	HostPort  uint16   `json:"host_port"`
	GuestPort uint16   `json:"guest_port"`
	Protocol  Protocol `json:"protocol,omitempty"`
}

type DNSConfig struct {
	Servers       []string `json:"servers,omitempty"`
	SearchDomains []string `json:"search_domains,omitempty"`
	BlockPrivate  bool     `json:"block_private,omitempty"`
}

type ProxyConfig struct {
	HTTPProxy  string   `json:"http_proxy,omitempty"`
	HTTPSProxy string   `json:"https_proxy,omitempty"`
	NoProxy    []string `json:"no_proxy,omitempty"`
	Enforce    bool     `json:"enforce,omitempty"`
}

type TrafficLimits struct {
	MaxEgressBytesPerSec  int64 `json:"max_egress_bps,omitempty"`
	MaxIngressBytesPerSec int64 `json:"max_ingress_bps,omitempty"`
	MaxConnections        int   `json:"max_connections,omitempty"`
}

type NetworkPolicy struct {
	Mode        PolicyMode     `json:"mode,omitempty"`
	AllowCIDR   []CIDRRule     `json:"allow_cidr,omitempty"`
	DenyCIDR    []CIDRRule     `json:"deny_cidr,omitempty"`
	AllowDomain []DomainRule   `json:"allow_domain,omitempty"`
	DenyDomain  []DomainRule   `json:"deny_domain,omitempty"`
	DNS         *DNSConfig     `json:"dns,omitempty"`
	Proxy       *ProxyConfig   `json:"proxy,omitempty"`
	Limits      *TrafficLimits `json:"limits,omitempty"`
}

type NetworkConfig struct {
	Enabled            bool           `json:"enabled"`
	Mode               NetworkMode    `json:"mode,omitempty"`
	Policy             *NetworkPolicy `json:"policy,omitempty"`
	PortForwards       []PortForward  `json:"port_forwards,omitempty"`
	IsolateFromHostLAN bool           `json:"isolate_from_host_lan,omitempty"`
}

type RuntimeNetworkDefaults struct {
	Profile string         `json:"profile,omitempty"`
	Config  *NetworkConfig `json:"config,omitempty"`
}

func StrictNetworkProfile() *NetworkConfig {
	return &NetworkConfig{
		Enabled: true,
		Mode:    NetworkNAT,
		Policy: &NetworkPolicy{
			Mode: PolicyBlockAll,
		},
	}
}

func BalancedNetworkProfile() *NetworkConfig {
	return &NetworkConfig{
		Enabled: true,
		Mode:    NetworkNAT,
		Policy: &NetworkPolicy{
			Mode: PolicyAllowAll,
		},
	}
}

func OpenNetworkProfile() *NetworkConfig {
	return &NetworkConfig{
		Enabled: true,
		Mode:    NetworkNAT,
		Policy: &NetworkPolicy{
			Mode: PolicyAllowAll,
		},
	}
}

func ValidateNetworkConfig(cfg *NetworkConfig) error {
	if cfg == nil {
		return nil
	}

	mode := cfg.Mode
	if !cfg.Enabled {
		if mode == "" {
			mode = NetworkDisabled
		}
		if mode != NetworkDisabled {
			return fmt.Errorf("%w: disabled config must use mode=%q", ErrNetworkInvalidConfig, NetworkDisabled)
		}
		if len(cfg.PortForwards) > 0 {
			return fmt.Errorf("%w: disabled network cannot publish ports", ErrNetworkInvalidConfig)
		}
		return nil
	}

	if mode == "" {
		mode = NetworkNAT
	}
	if mode != NetworkNAT && mode != NetworkDisabled {
		if mode == NetworkBridged {
			return fmt.Errorf("%w: bridged mode is not supported by current backend", ErrNetworkUnsupportedPlatform)
		}
		return fmt.Errorf("%w: unsupported mode %q", ErrNetworkInvalidConfig, mode)
	}

	if cfg.Policy != nil {
		if cfg.Policy.Mode != "" && cfg.Policy.Mode != PolicyBlockAll && cfg.Policy.Mode != PolicyAllowAll {
			return fmt.Errorf("%w: unsupported policy mode %q", ErrNetworkInvalidConfig, cfg.Policy.Mode)
		}
	}

	for i, pf := range cfg.PortForwards {
		if pf.HostPort == 0 || pf.GuestPort == 0 {
			return fmt.Errorf("%w: port forward[%d] host/guest ports must be > 0", ErrNetworkInvalidConfig, i)
		}
		proto := pf.Protocol
		if proto == "" {
			proto = ProtoTCP
		}
		if proto != ProtoTCP && proto != ProtoUDP {
			return fmt.Errorf("%w: port forward[%d] protocol must be tcp/udp", ErrNetworkInvalidConfig, i)
		}
	}
	return nil
}

func resolveRuntimeDefaultNetwork(opts *RuntimeOptions) *NetworkConfig {
	if opts == nil || opts.NetworkDefaults == nil {
		return nil
	}
	if opts.NetworkDefaults.Config != nil {
		return cloneNetworkConfig(opts.NetworkDefaults.Config)
	}
	switch strings.ToLower(opts.NetworkDefaults.Profile) {
	case "", "strict":
		return StrictNetworkProfile()
	case "balanced":
		return BalancedNetworkProfile()
	case "open":
		return OpenNetworkProfile()
	default:
		return StrictNetworkProfile()
	}
}

func effectiveNetworkConfig(defaultCfg, override *NetworkConfig) *NetworkConfig {
	if override != nil {
		cfg := cloneNetworkConfig(override)
		if cfg.Mode == "" {
			if cfg.Enabled {
				cfg.Mode = NetworkNAT
			} else {
				cfg.Mode = NetworkDisabled
			}
		}
		if cfg.Policy != nil && cfg.Policy.Mode == "" {
			cfg.Policy.Mode = PolicyBlockAll
		}
		return cfg
	}
	if defaultCfg == nil {
		return nil
	}
	cfg := cloneNetworkConfig(defaultCfg)
	if cfg.Mode == "" {
		if cfg.Enabled {
			cfg.Mode = NetworkNAT
		} else {
			cfg.Mode = NetworkDisabled
		}
	}
	if cfg.Policy != nil && cfg.Policy.Mode == "" {
		cfg.Policy.Mode = PolicyBlockAll
	}
	return cfg
}

func cloneNetworkConfig(in *NetworkConfig) *NetworkConfig {
	if in == nil {
		return nil
	}
	out := *in
	if in.Policy != nil {
		p := *in.Policy
		if len(in.Policy.AllowCIDR) > 0 {
			p.AllowCIDR = append([]CIDRRule(nil), in.Policy.AllowCIDR...)
		}
		if len(in.Policy.DenyCIDR) > 0 {
			p.DenyCIDR = append([]CIDRRule(nil), in.Policy.DenyCIDR...)
		}
		if len(in.Policy.AllowDomain) > 0 {
			p.AllowDomain = append([]DomainRule(nil), in.Policy.AllowDomain...)
		}
		if len(in.Policy.DenyDomain) > 0 {
			p.DenyDomain = append([]DomainRule(nil), in.Policy.DenyDomain...)
		}
		if in.Policy.DNS != nil {
			d := *in.Policy.DNS
			d.Servers = append([]string(nil), in.Policy.DNS.Servers...)
			d.SearchDomains = append([]string(nil), in.Policy.DNS.SearchDomains...)
			p.DNS = &d
		}
		if in.Policy.Proxy != nil {
			pr := *in.Policy.Proxy
			pr.NoProxy = append([]string(nil), in.Policy.Proxy.NoProxy...)
			p.Proxy = &pr
		}
		if in.Policy.Limits != nil {
			l := *in.Policy.Limits
			p.Limits = &l
		}
		out.Policy = &p
	}
	if len(in.PortForwards) > 0 {
		out.PortForwards = append([]PortForward(nil), in.PortForwards...)
	}
	return &out
}
