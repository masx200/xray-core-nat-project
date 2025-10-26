package conf

import (
	"encoding/json"
	"testing"

	"github.com/xtls/xray-core/proxy/nat"
)

func TestNATOutboundConfig_Build(t *testing.T) {
	// Test basic NAT configuration
	config := &NATOutboundConfig{
		SiteID: "site-b",
		VirtualRanges: []*VirtualRange{
			{
				VirtualNetwork: "240.2.2.0/24",
				RealNetwork:    "192.168.1.0/24",
				IPv6Enabled:   true,
				IPv6Prefix:    "64:FF9B:2222::/96",
			},
		},
		Rules: []*NATRule{
			{
				RuleID:            "rule-1",
				VirtualDestination: "240.2.2.20",
				RealDestination:   "192.168.1.20",
				Protocol:          "tcp",
			},
		},
	}

	protoConfig, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to build NAT config: %v", err)
	}

	natConfig, ok := protoConfig.(*nat.Config)
	if !ok {
		t.Fatalf("Expected nat.Config, got %T", protoConfig)
	}

	if natConfig.SiteId != "site-b" {
		t.Errorf("Expected site ID 'site-b', got '%s'", natConfig.SiteId)
	}

	if len(natConfig.VirtualRanges) != 1 {
		t.Errorf("Expected 1 virtual range, got %d", len(natConfig.VirtualRanges))
	}

	vrange := natConfig.VirtualRanges[0]
	if vrange.VirtualNetwork != "240.2.2.0/24" {
		t.Errorf("Expected virtual network '240.2.2.0/24', got '%s'", vrange.VirtualNetwork)
	}
	if vrange.RealNetwork != "192.168.1.0/24" {
		t.Errorf("Expected real network '192.168.1.0/24', got '%s'", vrange.RealNetwork)
	}
	if !vrange.Ipv6Enabled {
		t.Error("Expected IPv6 enabled to be true")
	}
	if vrange.Ipv6VirtualPrefix != "64:FF9B:2222::/96" {
		t.Errorf("Expected IPv6 prefix '64:FF9B:2222::/96', got '%s'", vrange.Ipv6VirtualPrefix)
	}

	if len(natConfig.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(natConfig.Rules))
	}

	rule := natConfig.Rules[0]
	if rule.RuleId != "rule-1" {
		t.Errorf("Expected rule ID 'rule-1', got '%s'", rule.RuleId)
	}
	if rule.VirtualDestination != "240.2.2.20" {
		t.Errorf("Expected virtual destination '240.2.2.20', got '%s'", rule.VirtualDestination)
	}
	if rule.RealDestination != "192.168.1.20" {
		t.Errorf("Expected real destination '192.168.1.20', got '%s'", rule.RealDestination)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol 'tcp', got '%s'", rule.Protocol)
	}
}

func TestNATOutboundConfig_ValidationError(t *testing.T) {
	// Test validation error - missing site ID
	config := &NATOutboundConfig{
		SiteID: "", // Empty site ID should cause error
		VirtualRanges: []*VirtualRange{
			{
				VirtualNetwork: "240.2.2.0/24",
				RealNetwork:    "192.168.1.0/24",
			},
		},
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for missing site ID, got nil")
	}

	// Test validation error - invalid virtual range
	config.SiteID = "test-site"
	config.VirtualRanges[0].VirtualNetwork = "" // Empty virtual network should cause error

	_, err = config.Build()
	if err == nil {
		t.Error("Expected error for missing virtual network, got nil")
	}
}

func TestNATOutboundConfig_Defaults(t *testing.T) {
	// Test default values for timeouts and limits
	config := &NATOutboundConfig{
		SiteID: "test-site",
	}

	protoConfig, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to build NAT config: %v", err)
	}

	natConfig := protoConfig.(*nat.Config)

	// Check default session timeouts
	if natConfig.SessionTimeout.TcpTimeout != 300 {
		t.Errorf("Expected default TCP timeout 300, got %d", natConfig.SessionTimeout.TcpTimeout)
	}
	if natConfig.SessionTimeout.UdpTimeout != 60 {
		t.Errorf("Expected default UDP timeout 60, got %d", natConfig.SessionTimeout.UdpTimeout)
	}
	if natConfig.SessionTimeout.CleanupInterval != 30 {
		t.Errorf("Expected default cleanup interval 30, got %d", natConfig.SessionTimeout.CleanupInterval)
	}

	// Check default resource limits
	if natConfig.Limits.MaxSessions != 10000 {
		t.Errorf("Expected default max sessions 10000, got %d", natConfig.Limits.MaxSessions)
	}
	if natConfig.Limits.MaxMemoryMb != 100 {
		t.Errorf("Expected default max memory MB 100, got %d", natConfig.Limits.MaxMemoryMb)
	}
	if natConfig.Limits.CleanupThreshold != 0.8 {
		t.Errorf("Expected default cleanup threshold 0.8, got %f", natConfig.Limits.CleanupThreshold)
	}
}

func TestNATOutboundConfig_PortMapping(t *testing.T) {
	// Test port mapping configuration
	config := &NATOutboundConfig{
		SiteID: "test-site",
		Rules: []*NATRule{
			{
				RuleID:            "rule-1",
				VirtualDestination: "240.2.2.20",
				RealDestination:   "192.168.1.20",
				Protocol:          "tcp",
				PortMapping: &PortMapping{
					OriginalPort:    "8080",
					TranslatedPort:  "80",
				},
			},
		},
	}

	protoConfig, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to build NAT config: %v", err)
	}

	natConfig := protoConfig.(*nat.Config)

	if len(natConfig.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(natConfig.Rules))
	}

	rule := natConfig.Rules[0]
	if rule.PortMapping == nil {
		t.Fatal("Expected port mapping to be non-nil")
	}
	if rule.PortMapping.OriginalPort != "8080" {
		t.Errorf("Expected original port '8080', got '%s'", rule.PortMapping.OriginalPort)
	}
	if rule.PortMapping.TranslatedPort != "80" {
		t.Errorf("Expected translated port '80', got '%s'", rule.PortMapping.TranslatedPort)
	}
}

func TestNATOutboundConfig_JSONSerialization(t *testing.T) {
	// Test JSON serialization and deserialization
	config := &NATOutboundConfig{
		SiteID: "site-b",
		VirtualRanges: []*VirtualRange{
			{
				VirtualNetwork: "240.2.2.0/24",
				RealNetwork:    "192.168.1.0/24",
				IPv6Enabled:   true,
				IPv6Prefix:    "64:FF9B:2222::/96",
			},
		},
		Rules: []*NATRule{
			{
				RuleID:            "rule-1",
				VirtualDestination: "240.2.2.20",
				RealDestination:   "192.168.1.20",
				Protocol:          "tcp",
			},
		},
		ResourceLimits: &ResourceLimits{
			MaxSessions:      5000,
			MaxMemoryMB:     50,
			CleanupThreshold: 0.7,
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal NAT config to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var decodedConfig NATOutboundConfig
	err = json.Unmarshal(jsonData, &decodedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal NAT config from JSON: %v", err)
	}

	// Verify the decoded config
	if decodedConfig.SiteID != config.SiteID {
		t.Errorf("Expected site ID '%s', got '%s'", config.SiteID, decodedConfig.SiteID)
	}

	if len(decodedConfig.VirtualRanges) != len(config.VirtualRanges) {
		t.Errorf("Expected %d virtual ranges, got %d", len(config.VirtualRanges), len(decodedConfig.VirtualRanges))
	}

	if len(decodedConfig.Rules) != len(config.Rules) {
		t.Errorf("Expected %d rules, got %d", len(config.Rules), len(decodedConfig.Rules))
	}

	if decodedConfig.ResourceLimits == nil {
		t.Fatal("Expected resource limits to be non-nil")
	}
	if decodedConfig.ResourceLimits.MaxSessions != 5000 {
		t.Errorf("Expected max sessions 5000, got %d", decodedConfig.ResourceLimits.MaxSessions)
	}
	if decodedConfig.ResourceLimits.MaxMemoryMB != 50 {
		t.Errorf("Expected max memory MB 50, got %d", decodedConfig.ResourceLimits.MaxMemoryMB)
	}
	if decodedConfig.ResourceLimits.CleanupThreshold != 0.7 {
		t.Errorf("Expected cleanup threshold 0.7, got %f", decodedConfig.ResourceLimits.CleanupThreshold)
	}
}

