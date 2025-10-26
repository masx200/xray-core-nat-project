package conf

import (
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/proxy/nat"
	"google.golang.org/protobuf/proto"
)

// NATOutboundConfig represents the JSON configuration for NAT outbound proxy
type NATOutboundConfig struct {
	SiteID        string           `json:"siteId"`
	VirtualRanges []*VirtualRange `json:"virtualRanges"`
	Rules         []*NATRule      `json:"rules"`
	SessionTimeout *SessionTimeout `json:"sessionTimeout"`
	ResourceLimits *ResourceLimits `json:"resourceLimits"`
}

// VirtualRange defines a virtual IP range configuration
type VirtualRange struct {
	VirtualNetwork string `json:"virtualNetwork"`
	RealNetwork    string `json:"realNetwork"`
	IPv6Enabled   bool   `json:"ipv6Enabled"`
	IPv6Prefix    string `json:"ipv6Prefix"`
}

// NATRule defines a NAT translation rule
type NATRule struct {
	RuleID            string      `json:"ruleId"`
	SourceSite        string      `json:"sourceSite"`
	VirtualDestination string      `json:"virtualDestination"`
	RealDestination   string      `json:"realDestination"`
	Protocol          string      `json:"protocol"`
	PortMapping       *PortMapping `json:"portMapping"`
}

// PortMapping defines port mapping configuration
type PortMapping struct {
	OriginalPort    string `json:"originalPort"`
	TranslatedPort  string `json:"translatedPort"`
}

// SessionTimeout defines session timeout configuration
type SessionTimeout struct {
	TCPTimeout      uint32 `json:"tcpTimeout"`
	UDPTimeout      uint32 `json:"udpTimeout"`
	CleanupInterval uint32 `json:"cleanupInterval"`
}

// ResourceLimits defines resource limits configuration
type ResourceLimits struct {
	MaxSessions      uint32  `json:"maxSessions"`
	MaxMemoryMB     uint32  `json:"maxMemoryMB"`
	CleanupThreshold float32 `json:"cleanupThreshold"`
}

// Build implements Buildable interface for NAT outbound configuration
func (c *NATOutboundConfig) Build() (proto.Message, error) {
	config := &nat.Config{
		SiteId: c.SiteID,
	}

	// Validate basic configuration
	if c.SiteID == "" {
		return nil, errors.New("NAT configuration: siteId is required")
	}

	// Process virtual IP ranges
	if len(c.VirtualRanges) > 0 {
		config.VirtualRanges = make([]*nat.VirtualIPRange, len(c.VirtualRanges))
		for i, vr := range c.VirtualRanges {
			if vr.VirtualNetwork == "" || vr.RealNetwork == "" {
				return nil, errors.New("NAT virtual range: both virtualNetwork and realNetwork are required")
			}

			config.VirtualRanges[i] = &nat.VirtualIPRange{
				VirtualNetwork: vr.VirtualNetwork,
				RealNetwork:    vr.RealNetwork,
				Ipv6Enabled:   vr.IPv6Enabled,
				Ipv6VirtualPrefix: vr.IPv6Prefix,
			}
		}
	}

	// Process NAT rules
	if len(c.Rules) > 0 {
		config.Rules = make([]*nat.NATRule, len(c.Rules))
		for i, rule := range c.Rules {
			if rule.VirtualDestination == "" {
				return nil, errors.New("NAT rule: virtualDestination is required")
			}

			natRule := &nat.NATRule{
				RuleId:            rule.RuleID,
				VirtualDestination: rule.VirtualDestination,
				RealDestination:   rule.RealDestination,
				Protocol:          rule.Protocol,
				SourceSite:        rule.SourceSite,
			}

			// Add port mapping if specified
			if rule.PortMapping != nil {
				natRule.PortMapping = &nat.PortMapping{
					OriginalPort:    rule.PortMapping.OriginalPort,
					TranslatedPort:  rule.PortMapping.TranslatedPort,
				}
			}

			config.Rules[i] = natRule
		}
	}

	// Process session timeout configuration
	if c.SessionTimeout != nil {
		config.SessionTimeout = &nat.SessionTimeout{
			TcpTimeout:       c.SessionTimeout.TCPTimeout,
			UdpTimeout:       c.SessionTimeout.UDPTimeout,
			CleanupInterval:  c.SessionTimeout.CleanupInterval,
		}
	} else {
		// Set default timeouts
		config.SessionTimeout = &nat.SessionTimeout{
			TcpTimeout:      300,  // 5 minutes
			UdpTimeout:      60,   // 1 minute
			CleanupInterval: 30,   // 30 seconds
		}
	}

	// Process resource limits
	if c.ResourceLimits != nil {
		config.Limits = &nat.ResourceLimits{
			MaxSessions:      c.ResourceLimits.MaxSessions,
			MaxMemoryMb:     c.ResourceLimits.MaxMemoryMB,
			CleanupThreshold: c.ResourceLimits.CleanupThreshold,
		}
	} else {
		// Set default limits
		config.Limits = &nat.ResourceLimits{
			MaxSessions:      10000,
			MaxMemoryMb:     100,
			CleanupThreshold: 0.8,
		}
	}

	return config, nil
}