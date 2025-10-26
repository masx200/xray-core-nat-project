package nat

import (
	"sync"
	"testing"
	"time"

	"github.com/xtls/xray-core/common/net"
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
	handler := &Handler{
		sessionTable:   &sync.Map{},
		cleanupTicker:  time.NewTicker(30 * time.Second),
		done:          make(chan struct{}),
	}
	if handler == nil {
		t.Fatal("Failed to create NAT handler")
	}

	// Create test destinations
	virtualDest := net.Destination{
		Address: net.ParseAddress("240.2.2.20"),
		Network: net.Network_TCP,
		Port:    80,
	}

	realDest := net.Destination{
		Address: net.ParseAddress("192.168.1.20"),
		Network: net.Network_TCP,
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
	dest := net.Destination{
		Address: net.ParseAddress("240.2.2.20"),
		Network: net.Network_TCP,
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
	normalDest := net.Destination{
		Address: net.ParseAddress("8.8.8.8"),
		Network: net.Network_TCP,
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

	virtualDest := net.Destination{
		Address: net.ParseAddress("240.2.2.20"),
		Network: net.Network_TCP,
		Port:    80,
	}

	transformed, err := handler.applyDNAT(virtualDest, rule)
	if err != nil {
		t.Fatalf("DNAT transformation failed: %v", err)
	}

	if transformed.Address.String() != "192.168.1.20" {
		t.Errorf("Expected transformed destination '192.168.1.20', got '%s'", transformed.Address.String())
	}

	if transformed.Network != net.Network_TCP {
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

	handler := &Handler{
		config:        config,
		sessionTable:  &sync.Map{},
		cleanupTicker: time.NewTicker(30 * time.Second),
		done:          make(chan struct{}),
	}

	// Create a session
	virtualDest := net.Destination{
		Address: net.ParseAddress("240.2.2.20"),
		Network: net.Network_TCP,
		Port:    80,
	}

	realDest := net.Destination{
		Address: net.ParseAddress("192.168.1.20"),
		Network: net.Network_TCP,
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