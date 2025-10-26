// Package nat implements bidirectional Network Address Translation functionality
package nat

//go:generate go run github.com/xtls/xray-core/common/proto -cproto=./config.proto -pnat -g

import (
	"context"
	"sync"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/stat"
	"github.com/xtls/xray-core/common/retry"
	"github.com/xtls/xray-core/common/task"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		h := &Handler{}
		if err := core.RequireFeatures(ctx, func(pm policy.Manager) error {
			return h.Init(config.(*Config), pm)
		}); err != nil {
			return nil, err
		}
		return h, nil
	}))
}

// Handler implements bidirectional NAT functionality
type Handler struct {
	config        *Config
	policyManager policy.Manager

	// Session management
	sessionTable   *sync.Map // Concurrent map for session storage
	sessionLock    sync.RWMutex
	cleanupTicker  *time.Ticker
	done          chan struct{}

	// Metrics and statistics
	activeSessions int64
	totalSessions  int64
	totalBytes    int64
	totalErrors   int64
}

// NATSession represents a NAT translation session
type NATSession struct {
	SessionID      string
	Protocol       string
	VirtualSource  net.Destination
	VirtualDest    net.Destination
	RealSource     net.Destination
	RealDest       net.Destination
	CreatedAt      time.Time
	LastActivity   time.Time
	Direction      string // "inbound" or "outbound"
}

// New creates a new NAT handler
func New() *Handler {
	return &Handler{
		sessionTable:   &sync.Map{},
		cleanupTicker:  time.NewTicker(30 * time.Second),
		done:          make(chan struct{}),
	}
}

// Init initializes NAT handler with configuration
func (h *Handler) Init(config *Config, pm policy.Manager) error {
	if config == nil {
		return errors.New("NAT config cannot be nil")
	}

	h.config = config
	h.policyManager = pm

	// Only start cleanup routine if not already running
	if h.cleanupTicker != nil {
		go h.sessionCleanupRoutine()
	}

	return nil
}

// Type implements proxy.Outbound
func (h *Handler) Type() interface{} {
	return h.config
}

// Process implements outbound proxy processing
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbounds := session.OutboundsFromContext(ctx)
	if len(outbounds) == 0 {
		return errors.New("no outbound destination specified")
	}

	destination := outbounds[len(outbounds)-1].Target
	if !destination.Address.Family().IsIP() {
		return errors.New("NAT only supports IP destinations")
	}

	// Determine if this is virtual IP traffic that needs NAT transformation
	natRule, shouldTransform := h.shouldApplyNAT(destination)
	if !shouldTransform {
		// Not a virtual IP, handle as normal outbound
		return h.handleNormalOutbound(ctx, link, destination, dialer)
	}

	// Apply NAT transformation
	return h.handleNATOutbound(ctx, link, destination, dialer, natRule)
}

// shouldApplyNAT determines if NAT transformation should be applied to destination
func (h *Handler) shouldApplyNAT(destination net.Destination) (*NATRule, bool) {
	for _, rule := range h.config.Rules {
		if h.matchesVirtualDestination(destination, rule.VirtualDestination) {
			return rule, true
		}
	}
	return nil, false
}

// matchesVirtualDestination checks if destination matches virtual network
func (h *Handler) matchesVirtualDestination(destination net.Destination, virtualNetwork string) bool {
	// TODO: Implement proper IP network matching
	// For now, simple string matching for demonstration
	destStr := destination.Address.String()
	return destStr == virtualNetwork
}

// handleNormalOutbound handles non-NAT outbound traffic
func (h *Handler) handleNormalOutbound(ctx context.Context, link *transport.Link, destination net.Destination, dialer internet.Dialer) error {
	// Implement standard outbound connection
	// This will be similar to freedom proxy implementation

	var conn stat.Connection
	var err error

	err = retry.ExponentialBackoff(5, 100).On(func() error {
		rawConn, dialErr := dialer.Dial(ctx, destination)
		if dialErr != nil {
			return dialErr
		}
		conn = rawConn
		return nil
	})

	if err != nil {
		return errors.New("failed to establish connection").Base(err)
	}

	// Handle bidirectional traffic
	requestDone := func() error {
		defer conn.Close()
		return buf.Copy(buf.NewReader(conn), link.Writer)
	}

	responseDone := func() error {
		defer conn.Close()
		return buf.Copy(link.Reader, buf.NewWriter(conn))
	}

	return task.Run(ctx, requestDone, task.OnSuccess(responseDone, task.Close(link.Writer)))
}

// handleNATOutbound handles NAT-transformed outbound traffic
func (h *Handler) handleNATOutbound(ctx context.Context, link *transport.Link, destination net.Destination, dialer internet.Dialer, rule *NATRule) error {
	// Apply DNAT transformation
	transformedDest, err := h.applyDNAT(destination, rule)
	if err != nil {
		return errors.New("DNAT transformation failed").Base(err)
	}

	// Create NAT session for tracking
	session := h.createNATSession(destination, transformedDest, "outbound")

	// Establish connection with transformed destination
	var conn stat.Connection
	err = retry.ExponentialBackoff(5, 100).On(func() error {
		rawConn, dialErr := dialer.Dial(ctx, transformedDest)
		if dialErr != nil {
			return dialErr
		}
		conn = rawConn
		return nil
	})

	if err != nil {
		h.removeSession(session.SessionID)
		return errors.New("failed to establish NAT connection").Base(err)
	}

	// Handle bidirectional traffic with NAT transformation
	requestDone := func() error {
		defer func() {
			h.removeSession(session.SessionID)
			conn.Close()
		}()
		return buf.Copy(buf.NewReader(conn), link.Writer)
	}

	responseDone := func() error {
		defer func() {
			h.removeSession(session.SessionID)
			conn.Close()
		}()
		return buf.Copy(link.Reader, buf.NewWriter(conn))
	}

	return task.Run(ctx, requestDone, task.OnSuccess(responseDone, task.Close(link.Writer)))
}

// applyDNAT applies Destination Network Address Translation
func (h *Handler) applyDNAT(destination net.Destination, rule *NATRule) (net.Destination, error) {
	// Transform virtual destination to real destination
	realAddr := net.ParseAddress(rule.RealDestination)
	if realAddr == nil {
		return net.Destination{}, errors.New("invalid real destination address")
	}

	transformed := net.Destination{
		Address: realAddr,
		Network: destination.Network,
		Port:    destination.Port,
	}

	// Apply port mapping if specified
	if rule.PortMapping != nil {
		// TODO: Implement port mapping logic
	}

	return transformed, nil
}

// createNATSession creates a new NAT session for tracking
func (h *Handler) createNATSession(virtualDest, realDest net.Destination, direction string) *NATSession {
	sessionID := generateSessionID(virtualDest, realDest)

	session := &NATSession{
		SessionID:     sessionID,
		Protocol:      virtualDest.Network.String(),
		VirtualDest:   virtualDest,
		RealDest:      realDest,
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
		Direction:     direction,
	}

	h.sessionTable.Store(sessionID, session)
	h.totalSessions++
	h.activeSessions++

	return session
}

// removeSession removes a NAT session from tracking table
func (h *Handler) removeSession(sessionID string) {
	if _, loaded := h.sessionTable.LoadAndDelete(sessionID); loaded {
		h.activeSessions--
	}
}

// sessionCleanupRoutine periodically cleans up expired sessions
func (h *Handler) sessionCleanupRoutine() {
	for {
		select {
		case <-h.cleanupTicker.C:
			h.cleanupExpiredSessions()
		case <-h.done:
			return
		}
	}
}

// cleanupExpiredSessions removes sessions that have exceeded their timeout
func (h *Handler) cleanupExpiredSessions() {
	now := time.Now()
	var timeout time.Duration

	// Use default timeout if config is not available
	if h.config != nil && h.config.SessionTimeout != nil {
		timeout = time.Duration(h.config.SessionTimeout.TcpTimeout) * time.Second
	} else {
		timeout = 300 * time.Second // Default 5 minutes
	}

	h.sessionTable.Range(func(key, value interface{}) bool {
		if session, ok := value.(*NATSession); ok {
			if now.Sub(session.LastActivity) > timeout {
				h.sessionTable.Delete(key)
				h.activeSessions--
			}
		}
		return true
	})
}


// generateSessionID generates a unique session identifier
func generateSessionID(virtualDest, realDest net.Destination) string {
	return virtualDest.Address.String() + ":" + virtualDest.Port.String() + "->" +
		realDest.Address.String() + ":" + realDest.Port.String() + "_" +
		time.Now().Format("20060102150405")
}

// Close implements common.Closable
func (h *Handler) Close() error {
	close(h.done)
	h.cleanupTicker.Stop()
	return nil
}