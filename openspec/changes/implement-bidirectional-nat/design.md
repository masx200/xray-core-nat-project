# Design Document: Bidirectional NAT Implementation

## Architecture Overview

This document provides the technical design details for implementing
bidirectional NAT functionality in Xray-core.

## System Architecture

### Core Components Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Site A                                    │
├─────────────────────────────────────────────────────────────────┤
│  Device C (192.168.1.20)                                        │
│          │                                                     │
│          │ Connection to 240.2.2.20 (B's virtual IP)          │
│          ▼                                                     │
│  Gateway (192.168.1.1)                                          │
│          │                                                     │
│          │ Route 240.2.2.2/24 → Zero Trust Tunnel             │
│          ▼                                                     │
│  Xray Router (Smart Router)                                    │
│          │                                                     │
│          │ Encapsulated traffic to Site B                      │
│          ▼                                                     │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │         Zero Trust Network Tunnel          ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Zero Trust Network
                                    │
┌─────────────────────────────────────────────────────────────────┐
│                      Site B                                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────┐│
│  │         Zero Trust Network Tunnel          ││
│  └─────────────────────────────────────────────────────────────┘│
│          │                                                     │
│          │ Encapsulated traffic from Site A                    │
│          ▼                                                     │
│  Xray Router (NAT Gateway)                                     │
│          │                                                     │
│  ┌───────┴────────┐    ┌────────────────────────────────────┐  │
│  │   DNAT Logic   │    │      Session Tracking Table        │  │
│  │ 240.2.2.20 →   │    │ ┌─────────┬─────────┬─────────────┐ │  │
│  │ 192.168.1.20   │    │ │SessionID│VIP→Real│  Timestamp  │ │  │
│  └───────┬────────┘    │ └─────────┴─────────┴─────────────┘ │  │
│          │             └────────────────────────────────────┘  │
│          ▼                                                     │
│  Device D (192.168.1.20)                                        │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │               Return Path Processing          ││
│  │ ┌─────────────────────────────────────────────────────────┐││
│  │ │ Response: 192.168.1.20 → 192.168.1.1                    │││
│  │ │                   │                                      │││
│  │ │                   ▼                                      │││
│  │ │               NAT Gateway                                │││
│  │ │                   │                                      │││
│  │ │             ┌─────┴─────┐                               │││
│  │ │             │   SNAT    │ 192.168.1.20 → 240.2.2.20    │││
│  │ │             └───────────┘                               │││
│  │ │                   │                                      │││
│  │ │                   ▼                                      │││
│  │ │          240.2.2.20 → 240.1.1.20 (via ZT Tunnel)         │││
│  │ └─────────────────────────────────────────────────────────┘││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow Analysis

### Outbound Flow (Site A → Site B)

1. **Connection Initiation**

   ```
   Device C: 192.168.1.20:12345 → 240.2.2.20:80
                │
                │ Request packet
                ▼
   Site A Gateway: Routes 240.2.2.2/24 to Zero Trust
                │
                │ Routed packet
                ▼
   Site A Xray: Forwards to Site B via Zero Trust tunnel
                │
                │ Encapsulated packet
                ▼
   Site B Xray: Receives encapsulated packet
                │
                │ Decapsulated packet
                ▼
   NAT Gateway: Applies DNAT transformation
                │
                │ Transformed packet
                ▼
   Device D: 192.168.1.20:80 (sees connection from Site A Gateway)
   ```

2. **NAT Transformation Details**

   ```
   Original:   Src: 192.168.1.20:12345 → Dst: 240.2.2.20:80
   DNAT Applied: Src: 192.168.1.20:12345 → Dst: 192.168.1.20:80

   Session Table Entry:
   {
     SessionID: "conn_001",
     VirtualSrc: "192.168.1.20:12345",
     VirtualDst: "240.2.2.20:80",
     RealDst: "192.168.1.20:80",
     Protocol: "TCP",
     Created: "2024-01-15T10:30:00Z",
     LastActive: "2024-01-15T10:30:00Z"
   }
   ```

### Return Flow (Site B → Site A)

1. **Response Processing**

   ```
   Device D: 192.168.1.20:80 → 192.168.1.1:12345
                │
                │ Response packet
                ▼
   Site B Gateway: Routes default to NAT gateway
                │
                │ Routed packet
                ▼
   NAT Gateway: Matches session and applies SNAT
                │
                │ Transformed packet
                ▼
   Site B Xray: Forwards to Site A via Zero Trust
                │
                │ Encapsulated response
                ▼
   Site A Xray: Receives and decapsulates
                │
                │ Response packet
                ▼
   Device C: 192.168.1.20:12345 (sees response from 240.2.2.20:80)
   ```

2. **SNAT Transformation Details**

   ```
   Original:   Src: 192.168.1.20:80 → Dst: 192.168.1.1:12345
   SNAT Applied: Src: 240.2.2.20:80 → Dst: 192.168.1.20:12345

   Session Table Update:
   {
     SessionID: "conn_001",
     VirtualSrc: "192.168.1.20:12345",
     VirtualDst: "240.2.2.20:80",
     RealDst: "192.168.1.20:80",
     LastActive: "2024-01-15T10:30:05Z",
     Bidirectional: true
   }
   ```

## Protocol Buffers Schema

### NAT Configuration Structure

```protobuf
syntax = "proto3";

package xray.proxy.nat;

import "common/net/address.proto";
import "common/net/port.proto";

message NATConfig {
  // Site identifier for this NAT gateway
  string site_id = 1;

  // Virtual IP ranges managed by this gateway
  repeated VirtualIPRange virtual_ranges = 2;

  // Translation rules for destination networks
  repeated NATRule rules = 3;

  // Session timeout configuration
  SessionTimeout session_timeout = 4;

  // Performance and memory limits
  ResourceLimits limits = 5;
}

message VirtualIPRange {
  // Virtual IP range (e.g., "240.2.2.0/24")
  string virtual_network = 1;

  // Corresponding real network (e.g., "192.168.1.0/24")
  string real_network = 2;

  // IPv6 support
  bool ipv6_enabled = 3;

  // IPv6 virtual prefix
  string ipv6_virtual_prefix = 4;
}

message NATRule {
  // Rule identifier
  string rule_id = 1;

  // Source site filter (optional)
  string source_site = 2;

  // Virtual IP destination to match
  string virtual_destination = 3;

  // Real destination to translate to
  string real_destination = 4;

  // Protocol filtering (tcp, udp, or both)
  string protocol = 5;

  // Port mapping (optional)
  PortMapping port_mapping = 6;
}

message PortMapping {
  // Original port or range
  string original_port = 1;

  // Translated port or range
  string translated_port = 2;
}

message SessionTimeout {
  // TCP connection timeout in seconds
  uint32 tcp_timeout = 1;

  // UDP session timeout in seconds
  uint32 udp_timeout = 2;

  // Idle session cleanup interval in seconds
  uint32 cleanup_interval = 3;
}

message ResourceLimits {
  // Maximum concurrent sessions
  uint32 max_sessions = 1;

  // Maximum memory usage in MB
  uint32 max_memory_mb = 2;

  // Session table cleanup threshold
  float cleanup_threshold = 3;
}
```

### Session Tracking Structure

```protobuf
message NATSession {
  // Unique session identifier
  string session_id = 1;

  // Protocol type
  string protocol = 2;

  // Original source address (before NAT)
  NetworkEndpoint original_source = 3;

  // Virtual destination address
  NetworkEndpoint virtual_destination = 4;

  // Translated destination address
  NetworkEndpoint translated_destination = 5;

  // Reverse mapping for return traffic
  NetworkEndpoint return_mapping = 6;

  // Session timestamps
  SessionMetadata metadata = 7;

  // Traffic statistics
  TrafficStats stats = 8;
}

message NetworkEndpoint {
  string address = 1;
  uint32 port = 2;
  bool is_ipv6 = 3;
}

message SessionMetadata {
  // Session creation time
  int64 created_at = 1;

  // Last activity timestamp
  int64 last_activity = 2;

  // Session state (active, closing, timeout)
  string state = 3;

  // Associated site IDs
  string source_site = 4;
  string destination_site = 5;
}

message TrafficStats {
  // Bytes sent through this session
  uint64 bytes_sent = 1;

  // Bytes received through this session
  uint64 bytes_received = 2;

  // Packet counts
  uint64 packets_sent = 3;
  uint64 packets_received = 4;
}
```

## Implementation Details

### Key Algorithms

#### 1. Session Lookup Algorithm

```go
func (n *NATHandler) FindSession(packet Packet) (*NATSession, error) {
    // Create lookup key based on packet characteristics
    key := n.createSessionKey(packet)

    // Fast lookup using concurrent map
    if session, exists := n.sessionTable[key]; exists {
        // Update last activity timestamp
        session.metadata.last_activity = time.Now().Unix()
        return session, nil
    }

    // If no existing session, check if this is return traffic
    if returnSession := n.findReturnSession(packet); returnSession != nil {
        return returnSession, nil
    }

    return nil, ErrSessionNotFound
}
```

#### 2. DNAT Transformation Algorithm

```go
func (n *NATHandler) ApplyDNAT(packet Packet, rule NATRule) (*Packet, error) {
    // Validate packet matches rule criteria
    if !n.matchesRule(packet, rule) {
        return nil, ErrRuleMismatch
    }

    // Create new session if none exists
    session := n.createNATSession(packet, rule)

    // Transform destination address
    transformed := packet.Copy()
    transformed.Destination.Address = rule.RealDestination

    // Handle port mapping if specified
    if rule.PortMapping != nil {
        transformed.Destination.Port = n.mapPort(
            transformed.Destination.Port,
            rule.PortMapping
        )
    }

    // Store session for return traffic
    n.sessionTable[session.SessionID] = session

    return transformed, nil
}
```

#### 3. SNAT Transformation Algorithm

```go
func (n *NATHandler) ApplySNAT(packet Packet, session *NATSession) (*Packet, error) {
    // Verify this packet belongs to an existing session
    if !n.belongsToSession(packet, session) {
        return nil, ErrSessionMismatch
    }

    // Transform source address
    transformed := packet.Copy()
    transformed.Source.Address = session.VirtualDestination.Address

    // Update session statistics
    session.Stats.BytesSent += uint64(len(transformed.Payload))
    session.Metadata.LastActivity = time.Now().Unix()

    return transformed, nil
}
```

### Performance Optimization Strategies

#### 1. Session Table Design

- **Concurrent Maps**: Use sync.Map for thread-safe session storage
- **LRU Cache**: Implement size-based eviction to prevent memory exhaustion
- **Index Structures**: Secondary indexes for source/destination lookups

#### 2. Connection Pooling

- **TCP Connection Reuse**: Maintain persistent connections to frequently
  accessed destinations
- **UDP Session Batching**: Group UDP packets with same destination to reduce
  overhead

#### 3. Memory Management

- **Object Pooling**: Reuse packet and session structures to reduce GC pressure
- **Compact Storage**: Use efficient data structures for session storage

## Error Handling and Recovery

### Common Error Scenarios

1. **Session Table Overflow**
   - Implement least-recently-used (LRU) eviction
   - Log warning and continue with new sessions
   - Consider implementing rate limiting

2. **Memory Pressure**
   - Monitor system memory usage
   - Implement adaptive session limits
   - Graceful degradation under memory pressure

3. **Network Connectivity Issues**
   - Implement retry logic with exponential backoff
   - Health checking for Zero Trust tunnel connectivity
   - Failover to alternative routing paths if available

### Monitoring and Observability

1. **Session Metrics**
   - Active session count
   - Session creation/termination rates
   - Average session duration

2. **Traffic Metrics**
   - Bytes processed through NAT
   - Packet transformation success rate
   - Latency impact of NAT processing

3. **Error Metrics**
   - Session lookup failures
   - Rule matching errors
   - Resource limit exceeded events

## Testing Strategy

### Unit Tests

- Session table operations
- NAT transformation logic
- Rule matching algorithms
- Error handling scenarios

### Integration Tests

- End-to-end connectivity
- Bidirectional traffic flow
- Session persistence
- Resource limit enforcement

### Performance Tests

- Concurrent connection handling
- Memory usage under load
- Latency impact measurement
- Session cleanup efficiency

### Security Tests

- Session isolation between sites
- Address spoofing prevention
- Resource exhaustion protection
- Configuration validation

## Future Extensibility

### 1. Multi-Site Support

- Hierarchical NAT topology
- Site-to-site routing policies
- Load balancing across multiple NAT gateways

### 2. Dynamic NAT

- DHCP integration for virtual IP allocation
- Dynamic port allocation
- NAT444 support

### 3. Advanced Features

- NAT logging and auditing
- Session persistence across failover
- Integration with SDN controllers
