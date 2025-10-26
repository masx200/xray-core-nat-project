package nat

import (
	"strings"
	"sync"
	"testing"
	"time"

	xnet "github.com/xtls/xray-core/common/net"
)

func TestHandler_Init(t *testing.T) {
	config := &Config{
		SiteId:    "test-site",
		UserLevel: 0,
		EnableTcp: true,
		EnableUdp: true,
		VirtualRanges: []*VirtualIPRange{
			{
				VirtualNetwork: "240.2.2.0/24",
				RealNetwork:    "192.168.1.0/24",
				Ipv6Enabled:    false,
			},
		},
		Rules: []*NATRule{
			{
				RuleId:            "rule-1",
				VirtualDestination: "240.2.2.20",
				RealDestination:    "192.168.1.20",
				Protocol:          "tcp",
			},
		},
		SessionTimeout: &SessionTimeout{
			TcpTimeout:      300,
			UdpTimeout:      60,
			CleanupInterval: 30,
		},
		Limits: &ResourceLimits{
			MaxSessions:     10000,
			MaxMemoryMb:     100,
			CleanupThreshold: 0.8,
		},
	}

	handler := &Handler{}

	// Test initialization without policy manager for simplicity
	if err := handler.Init(config, nil); err != nil {
		t.Fatalf("Failed to initialize NAT handler: %v", err)
	}

	if handler.config == nil {
		t.Fatal("Handler config should not be nil after initialization")
	}

	if handler.config.SiteId != "test-site" {
		t.Errorf("Expected site ID 'test-site', got '%s'", handler.config.SiteId)
	}
}

func TestNATSession_Lifecycle(t *testing.T) {
	handler := New()
	if handler == nil {
		t.Fatal("Failed to create NAT handler")
	}

	// Create test destinations
	virtualDest := xnet.Destination{
		Address: xnet.ParseAddress("240.2.2.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	realDest := xnet.Destination{
		Address: xnet.ParseAddress("192.168.1.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	// Create NAT session
	session := handler.createNATSession(virtualDest, realDest, "outbound")
	if session == nil {
		t.Fatal("Failed to create NAT session")
	}

	// Verify session properties
	if session.SessionID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.VirtualDest.Address.String() != "240.2.2.20" {
		t.Errorf("Expected virtual destination '240.2.2.20', got '%s'", session.VirtualDest.Address.String())
	}

	if session.RealDest.Address.String() != "192.168.1.20" {
		t.Errorf("Expected real destination '192.168.1.20', got '%s'", session.RealDest.Address.String())
	}

	if session.Direction != "outbound" {
		t.Errorf("Expected direction 'outbound', got '%s'", session.Direction)
	}

	// Test session removal
	handler.removeSession(session.SessionID)

	// Verify session is removed
	if _, exists := handler.sessionTable.Load(session.SessionID); exists {
		t.Error("Session should be removed after calling removeSession")
	}

	// Close handler to stop cleanup routine
	handler.Close()
}

func TestShouldApplyNAT(t *testing.T) {
	config := &Config{
		Rules: []*NATRule{
			{
				RuleId:            "rule-1",
				VirtualDestination: "240.2.2.20",
				RealDestination:    "192.168.1.20",
				Protocol:          "tcp",
			},
		},
	}

	handler := &Handler{
		config: config,
	}

	// Test destination that should match NAT rule
	dest := xnet.Destination{
		Address: xnet.ParseAddress("240.2.2.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	rule, shouldTransform := handler.shouldApplyNAT(dest)
	if !shouldTransform {
		t.Error("Expected NAT transformation for virtual IP destination")
	}

	if rule == nil {
		t.Fatal("Expected NAT rule to be returned")
	}

	if rule.RuleId != "rule-1" {
		t.Errorf("Expected rule ID 'rule-1', got '%s'", rule.RuleId)
	}

	// Test destination that should not match NAT rule
	normalDest := xnet.Destination{
		Address: xnet.ParseAddress("8.8.8.8"),
		Network: xnet.Network_TCP,
		Port:    53,
	}

	_, shouldTransform = handler.shouldApplyNAT(normalDest)
	if shouldTransform {
		t.Error("Should not apply NAT transformation for non-virtual IP")
	}
}

func TestApplyDNAT(t *testing.T) {
	handler := &Handler{}

	rule := &NATRule{
		RuleId:            "rule-1",
		VirtualDestination: "240.2.2.20",
		RealDestination:    "192.168.1.20",
		Protocol:          "tcp",
	}

	virtualDest := xnet.Destination{
		Address: xnet.ParseAddress("240.2.2.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	transformed, err := handler.applyDNAT(virtualDest, rule)
	if err != nil {
		t.Fatalf("DNAT transformation failed: %v", err)
	}

	if transformed.Address.String() != "192.168.1.20" {
		t.Errorf("Expected transformed destination '192.168.1.20', got '%s'", transformed.Address.String())
	}

	if transformed.Network != xnet.Network_TCP {
		t.Errorf("Expected network TCP, got '%s'", transformed.Network.String())
	}

	if transformed.Port != 80 {
		t.Errorf("Expected port 80, got %d", transformed.Port)
	}
}

func TestSessionCleanup(t *testing.T) {
	config := &Config{
		SessionTimeout: &SessionTimeout{
			TcpTimeout:      1, // 1 second timeout for testing
			CleanupInterval: 1,
		},
	}

	handler := New()
	handler.config = config

	// Create a session
	virtualDest := xnet.Destination{
		Address: xnet.ParseAddress("240.2.2.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	realDest := xnet.Destination{
		Address: xnet.ParseAddress("192.168.1.20"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	session := handler.createNATSession(virtualDest, realDest, "outbound")

	// Wait for session to expire
	time.Sleep(2 * time.Second)

	// Run cleanup
	handler.cleanupExpiredSessions()

	// Verify session was cleaned up
	if _, exists := handler.sessionTable.Load(session.SessionID); exists {
		t.Error("Expired session should be removed during cleanup")
	}

	// Close handler to stop cleanup routine
	handler.Close()
}

func TestIPv6EmbeddedIPv4NAT(t *testing.T) {
	config := &Config{
		SiteId:    "test-ipv6-site",
		UserLevel: 0,
		EnableTcp: true,
		EnableUdp: true,
		VirtualRanges: []*VirtualIPRange{
			{
				VirtualNetwork:      "64:FF9B:1111::192.168.1.1/120",
				RealNetwork:         "192.168.1.0/24",
				Ipv6Enabled:         true,
				Ipv6VirtualPrefix:   "64:FF9B:1111::192.168.1.1/120",
			},
		},
		SessionTimeout: &SessionTimeout{
			TcpTimeout:      300,
			UdpTimeout:      60,
			CleanupInterval: 30,
		},
		Limits: &ResourceLimits{
			MaxSessions:     10000,
			MaxMemoryMb:     100,
			CleanupThreshold: 0.8,
		},
	}

	handler := &Handler{
		config:        config,
		sessionTable:  &sync.Map{},
		cleanupTicker: time.NewTicker(30 * time.Second),
		done:          make(chan struct{}),
	}

	// Test IPv6 embedded IPv4 addresses
	testCases := []struct {
		name     string
		ipv6Dest string
		expect   bool
	}{
		{
			name:     "IPv6 embedded IPv4 - first address",
			ipv6Dest: "64:FF9B:1111::192.168.1.1",
			expect:   true,
		},
		{
			name:     "IPv6 embedded IPv4 - middle address",
			ipv6Dest: "64:FF9B:1111::192.168.1.100",
			expect:   true,
		},
		{
			name:     "IPv6 embedded IPv4 - last address in /120",
			ipv6Dest: "64:FF9B:1111::192.168.1.255",
			expect:   true,
		},
		{
			name:     "IPv6 embedded IPv4 - out of range",
			ipv6Dest: "64:FF9B:1111::192.168.2.1",
			expect:   false,
		},
		{
			name:     "IPv6 embedded IPv4 - different prefix",
			ipv6Dest: "64:FF9B:2222::192.168.1.1",
			expect:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dest := xnet.Destination{
				Address: xnet.ParseAddress(tc.ipv6Dest),
				Network: xnet.Network_TCP,
				Port:    80,
			}

			rule, shouldTransform := handler.shouldApplyNAT(dest)
			if tc.expect && !shouldTransform {
				t.Errorf("Expected NAT transformation for %s, but none was applied", tc.ipv6Dest)
			}
			if !tc.expect && shouldTransform {
				t.Errorf("Expected no NAT transformation for %s, but one was applied", tc.ipv6Dest)
			}

			if shouldTransform {
				// Test DNAT transformation
				transformed, err := handler.applyDNAT(dest, rule)
				if err != nil {
					t.Fatalf("DNAT transformation failed for %s: %v", tc.ipv6Dest, err)
				}

				// Verify the transformation extracts IPv4 correctly
				expectedIPv4 := tc.ipv6Dest[strings.LastIndex(tc.ipv6Dest, ":")+1:]
				if transformed.Address.String() != expectedIPv4 {
					t.Errorf("Expected transformed address %s, got %s", expectedIPv4, transformed.Address.String())
				}
			}
		})
	}

	handler.Close()
}

func TestIPv6EmbeddedIPv4Extraction(t *testing.T) {
	handler := &Handler{}

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "64:FF9B:1111::192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			input:    "64:FF9B:1111::192.168.1.100",
			expected: "192.168.1.100",
		},
		{
			input:    "64:FF9B:1111::255.255.255.255",
			expected: "255.255.255.255",
		},
		{
			input:    "2001:db8::1", // Regular IPv6 without embedded IPv4
			expected: "",
		},
		{
			input:    "192.168.1.1", // Regular IPv4
			expected: "",
		},
		{
			input:    "[64:ff9b:1111::c0a8:164]", // Compressed IPv6 format for 192.168.1.100
			expected: "192.168.1.100",
		},
		{
			input:    "[64:ff9b:1111::c0a8:101]", // Compressed IPv6 format for 192.168.1.1
			expected: "192.168.1.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := handler.extractIPv4FromIPv6(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestIPv6NATSessionCreation(t *testing.T) {
	handler := New()

	// Create IPv6 embedded IPv4 destination
	ipv6Dest := xnet.Destination{
		Address: xnet.ParseAddress("64:FF9B:1111::192.168.1.100"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	// Create corresponding IPv4 destination
	ipv4Dest := xnet.Destination{
		Address: xnet.ParseAddress("192.168.1.100"),
		Network: xnet.Network_TCP,
		Port:    80,
	}

	// Create NAT session
	session := handler.createNATSession(ipv6Dest, ipv4Dest, "outbound")
	if session == nil {
		t.Fatal("Failed to create NAT session for IPv6->IPv4")
	}

	// Verify session properties (IPv6 addresses may be compressed)
	actualVirtualDest := session.VirtualDest.Address.String()
	if !strings.Contains(actualVirtualDest, "64:FF9B:1111") && !strings.Contains(actualVirtualDest, "64:ff9b:1111") {
		t.Errorf("Expected virtual destination containing '64:FF9B:1111', got '%s'", actualVirtualDest)
	}

	if session.RealDest.Address.String() != "192.168.1.100" {
		t.Errorf("Expected real destination '192.168.1.100', got '%s'", session.RealDest.Address.String())
	}

	if session.Direction != "outbound" {
		t.Errorf("Expected direction 'outbound', got '%s'", session.Direction)
	}

	// Clean up
	handler.removeSession(session.SessionID)
	handler.Close()
}