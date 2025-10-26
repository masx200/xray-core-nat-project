// Package nat implements bidirectional Network Address Translation functionality
package nat

//go:generate go run github.com/xtls/xray-core/common/proto -cproto=./config.proto -pnat -g

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/errors"
	xnet "github.com/xtls/xray-core/common/net"
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

	// LRU and memory management
	lruList       *list.List // Doubly-linked list for LRU tracking
	lruMap        map[string]*list.Element // Map for O(1) LRU access
	lruLock       sync.RWMutex
	maxSessions   int64
	maxMemoryMB   int64

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
	VirtualSource  xnet.Destination
	VirtualDest    xnet.Destination
	RealSource     xnet.Destination
	RealDest       xnet.Destination
	CreatedAt      time.Time
	LastActivity   time.Time
	Direction      string // "inbound" or "outbound"
}

// New creates a new NAT handler
func New() *Handler {
	return &Handler{
		sessionTable:   &sync.Map{},
		lruList:        list.New(),
		lruMap:         make(map[string]*list.Element),
		cleanupTicker:  time.NewTicker(30 * time.Second),
		done:          make(chan struct{}),
		maxSessions:   10000, // Default max sessions
		maxMemoryMB:   100,   // Default max memory in MB
	}
}

// Init initializes NAT handler with configuration
func (h *Handler) Init(config *Config, pm policy.Manager) error {
	if config == nil {
		return errors.New("NAT config cannot be nil")
	}

	h.config = config
	h.policyManager = pm

	// Configure limits from config
	if config.Limits != nil {
		if config.Limits.MaxSessions > 0 {
			h.maxSessions = int64(config.Limits.MaxSessions)
		}
		if config.Limits.MaxMemoryMb > 0 {
			h.maxMemoryMB = int64(config.Limits.MaxMemoryMb)
		}
	}

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
	natRule, shouldTransform := h.shouldApplyNAT(ctx, destination)
	if !shouldTransform {
		// Not a virtual IP, handle as normal outbound
		return h.handleNormalOutbound(ctx, link, destination, dialer)
	}

	// Apply NAT transformation
	return h.handleNATOutbound(ctx, link, destination, dialer, natRule)
}

// shouldApplyNAT determines if NAT transformation should be applied to destination
func (h *Handler) shouldApplyNAT(ctx context.Context, destination xnet.Destination) (*NATRule, bool) {
	// First check specific rules
	for _, rule := range h.config.Rules {
		if h.matchesVirtualDestination(destination, rule.VirtualDestination) &&
			h.matchesProtocol(destination, rule.Protocol) &&
			h.matchesPort(destination, rule) &&
			h.matchesSite(ctx, rule) {
			return rule, true
		}
	}

	// Then check virtual ranges
	for _, vrange := range h.config.VirtualRanges {
		if h.matchesVirtualRange(destination, vrange) {
			// Create a dynamic rule for this range
			return &NATRule{
				RuleId:            "dynamic-range-" + vrange.VirtualNetwork,
				VirtualDestination: destination.Address.String(),
				RealDestination:    vrange.RealNetwork,
				Protocol:          "tcp,udp", // Support both
			}, true
		}
	}

	return nil, false
}

// matchesVirtualDestination checks if destination matches virtual network
func (h *Handler) matchesVirtualDestination(destination xnet.Destination, virtualNetwork string) bool {
	destStr := destination.Address.String()

	// Handle IPv6 addresses with embedded IPv4 (like [prefix]::192.168.1.1)
	if strings.Contains(virtualNetwork, ":") && strings.Contains(virtualNetwork, ".") {
		return h.matchesIPv6EmbeddedIPv4(destination, virtualNetwork)
	}

	// Exact match for specific IP addresses
	return destStr == virtualNetwork
}

// matchesVirtualRange checks if destination matches any virtual IP range
func (h *Handler) matchesVirtualRange(destination xnet.Destination, vrange *VirtualIPRange) bool {
	destAddr := destination.Address.String()

	// Handle IPv6 with embedded IPv4
	if vrange.Ipv6Enabled && vrange.Ipv6VirtualPrefix != "" {
		if h.matchesIPv6EmbeddedIPv4Range(destination, vrange.Ipv6VirtualPrefix, vrange.RealNetwork) {
			return true
		}
	}

	// Handle regular IPv4 matching
	if strings.Contains(vrange.VirtualNetwork, "/") {
		return h.matchesCIDR(destAddr, vrange.VirtualNetwork)
	}

	return destAddr == vrange.VirtualNetwork
}

// matchesIPv6EmbeddedIPv4 matches IPv6 addresses with embedded IPv4
func (h *Handler) matchesIPv6EmbeddedIPv4(destination xnet.Destination, virtualNetwork string) bool {
	destStr := destination.Address.String()

	// Extract IPv4 from IPv6 if embedded
	if strings.Contains(destStr, ":") && strings.Contains(destStr, ".") {
		extractedIPv4 := h.extractIPv4FromIPv6(destStr)
		if extractedIPv4 != "" {
			// Check if this matches the pattern
			if strings.HasPrefix(virtualNetwork, "64:FF9B:1111::") {
				virtualIPv4 := strings.Replace(virtualNetwork, "64:FF9B:1111::", "", 1)
				if strings.Contains(virtualIPv4, "/") {
					// Handle CIDR notation
					return h.matchesCIDR(extractedIPv4, virtualIPv4)
				}
				return extractedIPv4 == virtualIPv4
			}
		}
	}

	return false
}

// matchesIPv6EmbeddedIPv4Range matches IPv6 embedded IPv4 addresses against range
func (h *Handler) matchesIPv6EmbeddedIPv4Range(destination xnet.Destination, ipv6Prefix, realNetwork string) bool {
	destStr := destination.Address.String()

	// First check if the IPv6 prefix matches (strip the CIDR part for comparison)
	prefixWithoutCIDR := ipv6Prefix
	if strings.Contains(ipv6Prefix, "/") {
		parts := strings.Split(ipv6Prefix, "/")
		prefixWithoutCIDR = parts[0]
	}

	// Check if the destination address starts with the expected IPv6 prefix
	// Handle both compressed and uncompressed formats
	if !strings.HasPrefix(strings.ToLower(destStr), strings.ToLower(prefixWithoutCIDR)) {
		// For compressed format, check if the address contains the prefix
		if !strings.Contains(strings.ToLower(destStr), strings.ToLower(prefixWithoutCIDR)) {
			return false
		}
	}

	// Handle both compressed and uncompressed IPv6 formats
	if strings.Contains(destStr, ":") {
		extractedIPv4 := h.extractIPv4FromIPv6(destStr)
		if extractedIPv4 != "" {
			// Check if extracted IPv4 is in the real network range
			return h.matchesCIDR(extractedIPv4, realNetwork)
		}
	}

	return false
}

// extractIPv4FromIPv6 extracts IPv4 address from IPv6 embedded notation
func (h *Handler) extractIPv4FromIPv6(ipv6Addr string) string {
	// Handle format like [prefix]::192.168.1.1
	if strings.Contains(ipv6Addr, ":") && strings.Contains(ipv6Addr, ".") {
		parts := strings.Split(ipv6Addr, ":")
		for _, part := range parts {
			if strings.Contains(part, ".") {
				return part
			}
		}
	}

	// Handle compressed IPv6 format like [prefix]::c0a8:164
	if strings.HasPrefix(ipv6Addr, "[") && strings.HasSuffix(ipv6Addr, "]") {
		ipv6Addr = strings.Trim(ipv6Addr, "[]")
	}

	// Try to parse the IPv6 address and convert embedded IPv4
	if strings.Contains(ipv6Addr, ":") {
		// Split by :: to find the IPv4 embedded part
		parts := strings.Split(ipv6Addr, "::")
		if len(parts) > 1 {
			lastPart := parts[len(parts)-1]
			// Check if last part contains IPv4-like hex pattern
			if strings.Contains(lastPart, ":") {
				hexParts := strings.Split(lastPart, ":")
				if len(hexParts) >= 2 {
					// Convert hex to decimal IPv4
					tryConvert := func(hex string) string {
						if val, err := strconv.ParseInt(hex, 16, 32); err == nil {
							return fmt.Sprintf("%d", val)
						}
						return hex
					}

					// Handle different hex patterns
					if len(hexParts) == 2 {
						// Pattern like c0a8:101 (192.168.1.1)
						// c0a8 = 192.168, 101 = 1.1 (need to split)
						hex1 := hexParts[0] // c0a8
						hex2 := hexParts[1] // 101

						if len(hex1) == 4 && len(hex2) >= 1 && len(hex2) <= 4 {
							// Convert c0a8 to 192.168
							part1 := tryConvert(hex1[:2]) // c0 -> 192
							part2 := tryConvert(hex1[2:]) // a8 -> 168

							// Convert hex2 to the last two octets
							var part3, part4 string
							if len(hex2) == 1 {
								// Pattern like c0a8:1 -> 192.168.0.1
								part3 = "0"
								part4 = tryConvert(hex2)
							} else if len(hex2) == 2 {
								// Pattern like c0a8:01 -> 192.168.0.1
								part3 = tryConvert(hex2[:1])
								part4 = tryConvert(hex2[1:])
							} else if len(hex2) == 3 {
								// Pattern like c0a8:101 -> 192.168.1.1
								part3 = tryConvert(hex2[:1])
								part4 = tryConvert(hex2[1:])
							} else if len(hex2) == 4 {
								// Pattern like c0a8:0101 -> 192.168.1.1
								part3 = tryConvert(hex2[:2])
								part4 = tryConvert(hex2[2:])
							} else {
								return ""
							}

							return part1 + "." + part2 + "." + part3 + "." + part4
						}
					} else if len(hexParts) >= 4 {
						ipv4 := tryConvert(hexParts[0]) + "." + tryConvert(hexParts[1]) + "." + tryConvert(hexParts[2]) + "." + tryConvert(hexParts[3])
						return ipv4
					}
				}
			}
		}
	}

	return ""
}

// matchesCIDR checks if an IP address matches a CIDR network
func (h *Handler) matchesCIDR(ip, cidr string) bool {
	// Parse CIDR
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	// Parse IP address
	addr := net.ParseIP(ip)
	if addr == nil {
		return false
	}

	return network.Contains(addr)
}

// matchesProtocol checks if destination protocol matches rule protocol specification
func (h *Handler) matchesProtocol(destination xnet.Destination, protocol string) bool {
	if protocol == "" {
		// Empty protocol means match all protocols
		return true
	}

	destProtocol := strings.ToLower(destination.Network.String())
	ruleProtocols := strings.Split(strings.ToLower(protocol), ",")

	for _, ruleProtocol := range ruleProtocols {
		ruleProtocol = strings.TrimSpace(ruleProtocol)
		if ruleProtocol == destProtocol || ruleProtocol == "tcp,udp" || ruleProtocol == "udp,tcp" {
			return true
		}
	}

	return false
}

// matchesPort checks if destination port matches rule port mapping
func (h *Handler) matchesPort(destination xnet.Destination, rule *NATRule) bool {
	if rule.PortMapping == nil {
		// No port mapping specified, match all ports
		return true
	}

	// For now, we match all ports when port mapping is specified
	// Port mapping logic will be applied during transformation
	return true
}

// mapPort maps the original port to the translated port based on port mapping configuration
func (h *Handler) mapPort(originalPort xnet.Port, portMapping *PortMapping) xnet.Port {
	if portMapping == nil {
		return originalPort
	}

	// If original port is specified, check if it matches
	if portMapping.OriginalPort != "" && portMapping.OriginalPort != "any" {
		// Parse the specified original port
		specifiedPorts := strings.Split(portMapping.OriginalPort, "-")
		if len(specifiedPorts) == 1 {
			// Single port
			if specifiedPort, err := xnet.PortFromString(specifiedPorts[0]); err == nil {
				if specifiedPort.Value() != originalPort.Value() {
					// Original port doesn't match, no mapping
					return originalPort
				}
			}
		}
	}

	// Map to translated port
	if portMapping.TranslatedPort != "" {
		if translatedPort, err := xnet.PortFromString(portMapping.TranslatedPort); err == nil {
			return translatedPort
		}
	}

	return originalPort
}

// matchesSite checks if the rule's source site matches the current site context
func (h *Handler) matchesSite(ctx context.Context, rule *NATRule) bool {
	if rule.SourceSite == "" {
		// Empty source site means match all sites
		return true
	}

	// Get the current site ID from configuration
	currentSite := h.config.SiteId
	if currentSite == "" {
		// No site ID configured, match all rules
		return true
	}

	// Check if the rule's source site matches the current site
	// Support for multiple sites separated by comma
	sites := strings.Split(strings.ToLower(rule.SourceSite), ",")
	for _, site := range sites {
		site = strings.TrimSpace(site)
		if strings.ToLower(site) == strings.ToLower(currentSite) {
			return true
		}
	}

	return false
}

// handleNormalOutbound handles non-NAT outbound traffic
func (h *Handler) handleNormalOutbound(ctx context.Context, link *transport.Link, destination xnet.Destination, dialer internet.Dialer) error {
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
func (h *Handler) handleNATOutbound(ctx context.Context, link *transport.Link, destination xnet.Destination, dialer internet.Dialer, rule *NATRule) error {
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
func (h *Handler) applyDNAT(destination xnet.Destination, rule *NATRule) (xnet.Destination, error) {
	var realAddr xnet.Address
	destStr := destination.Address.String()

	// Handle IPv6 embedded IPv4 addresses
	if strings.Contains(destStr, ":") && (strings.Contains(destStr, ".") || strings.Contains(destStr, "]")) {
		// Extract IPv4 from IPv6 embedded address
		extractedIPv4 := h.extractIPv4FromIPv6(destStr)
		if extractedIPv4 != "" {
			// Use the extracted IPv4 address
			realAddr = xnet.ParseAddress(extractedIPv4)
		} else {
			// Fallback to rule's real destination
			realAddr = xnet.ParseAddress(rule.RealDestination)
		}
	} else {
		// Regular IPv4 address or use rule's real destination
		if rule.RealDestination != "" {
			realAddr = xnet.ParseAddress(rule.RealDestination)
		} else {
			realAddr = destination.Address
		}
	}

	if realAddr == nil {
		return xnet.Destination{}, errors.New("invalid real destination address")
	}

	transformed := xnet.Destination{
		Address: realAddr,
		Network: destination.Network,
		Port:    destination.Port,
	}

	// Apply port mapping if specified
	if rule.PortMapping != nil {
		transformed.Port = h.mapPort(destination.Port, rule.PortMapping)
	}

	return transformed, nil
}

// createNATSession creates a new NAT session for tracking
func (h *Handler) createNATSession(virtualDest, realDest xnet.Destination, direction string) *NATSession {
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

	// Check memory limits and evict if necessary
	h.enforceMemoryLimits()

	// Check session limits and evict LRU if necessary
	h.enforceSessionLimits()

	h.sessionTable.Store(sessionID, session)

	// Add to LRU tracking
	h.lruLock.Lock()
	if elem, exists := h.lruMap[sessionID]; exists {
		h.lruList.MoveToFront(elem)
	} else {
		elem := h.lruList.PushFront(sessionID)
		h.lruMap[sessionID] = elem
	}
	h.lruLock.Unlock()

	h.totalSessions++
	h.activeSessions++

	return session
}

// removeSession removes a NAT session from tracking table
func (h *Handler) removeSession(sessionID string) {
	if _, loaded := h.sessionTable.LoadAndDelete(sessionID); loaded {
		h.activeSessions--

		// Remove from LRU tracking
		h.lruLock.Lock()
		if elem, exists := h.lruMap[sessionID]; exists {
			h.lruList.Remove(elem)
			delete(h.lruMap, sessionID)
		}
		h.lruLock.Unlock()
	}
}

// enforceSessionLimits enforces session count limits by evicting least recently used sessions
func (h *Handler) enforceSessionLimits() {
	h.lruLock.Lock()
	defer h.lruLock.Unlock()

	// Evict LRU sessions until we're under the limit
	for h.activeSessions >= h.maxSessions && h.lruList.Len() > 0 {
		// Get the least recently used session (back of the list)
		if elem := h.lruList.Back(); elem != nil {
			sessionID := elem.Value.(string)
			h.lruList.Remove(elem)
			delete(h.lruMap, sessionID)
			h.sessionTable.Delete(sessionID)
			h.activeSessions--
		}
	}
}

// enforceMemoryLimits enforces memory limits by estimating session memory usage
func (h *Handler) enforceMemoryLimits() {
	// Estimate memory usage per session (rough estimate in bytes)
	const sessionMemoryEstimate = 2048 // 2KB per session
	maxSessionsFromMemory := (h.maxMemoryMB * 1024 * 1024) / sessionMemoryEstimate

	// If session count would exceed memory limits, enforce it
	if maxSessionsFromMemory < h.maxSessions {
		h.maxSessions = maxSessionsFromMemory

		// Log the adjustment (in production, this would use the logging system)
		if h.activeSessions >= h.maxSessions {
			h.enforceSessionLimits()
		}
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

	var expiredSessions []string
	h.sessionTable.Range(func(key, value interface{}) bool {
		if session, ok := value.(*NATSession); ok {
			if now.Sub(session.LastActivity) > timeout {
				expiredSessions = append(expiredSessions, key.(string))
			}
		}
		return true
	})

	// Clean up expired sessions from both tables
	for _, sessionID := range expiredSessions {
		h.removeSession(sessionID)
	}
}


// generateSessionID generates a unique session identifier
func generateSessionID(virtualDest, realDest xnet.Destination) string {
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