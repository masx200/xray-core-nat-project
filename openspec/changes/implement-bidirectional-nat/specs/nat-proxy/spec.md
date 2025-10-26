# NAT Proxy Handler Specification

## ADDED Requirements

### 1. Core NAT Handler Implementation

The system shall provide a new proxy handler that implements bidirectional Network Address Translation (NAT) functionality.

#### Scenario: Basic DNAT Transformation
**Given** a NAT gateway is configured with virtual IP range `240.2.2.0/24`
**When** a TCP packet arrives with destination `240.2.2.20:80`
**Then** the destination shall be translated to `192.168.1.20:80` based on the NAT mapping rules

#### Scenario: Bidirectional Session Tracking
**Given** a DNAT session is established between `192.168.1.20:12345` and `240.2.2.20:80`
**When** a response packet returns from `192.168.1.20:80` to `192.168.1.1:12345`
**Then** the source address shall be SNAT-translated to `240.2.2.20:80` before return routing

### 2. Virtual IP Range Management

The NAT handler shall support configurable virtual IP ranges with automatic address translation.

#### Scenario: IPv4 Virtual IP Range Configuration
**Given** a NAT configuration with virtual network `240.2.2.0/24` and real network `192.168.1.0/24`
**When** traffic is destined for any address in `240.2.2.0/24`
**Then** the destination shall be mapped to the corresponding address in `192.168.1.0/24`

#### Scenario: IPv6 Virtual IP Support
**Given** a NAT configuration with IPv6 virtual prefix `64:FF9B:2222::/96`
**When** IPv6 traffic is destined for `64:FF9B:2222::20`
**Then** the destination shall be translated to the corresponding IPv4 address `192.168.1.20`

### 3. Session Lifecycle Management

The NAT handler shall maintain session state with automatic cleanup and timeout handling.

#### Scenario: TCP Session Timeout
**Given** a TCP NAT session with 300-second timeout configured
**When** no traffic has occurred for 300 seconds
**Then** the session shall be automatically cleaned up from the session table

#### Scenario: UDP Session Cleanup
**Given** a UDP NAT session with 60-second timeout configured
**When** a UDP packet is received matching an existing session
**Then** the session timeout shall be reset to 60 seconds

### 4. Resource Limitation and Protection

The NAT handler shall implement configurable resource limits to prevent resource exhaustion.

#### Scenario: Maximum Session Limit
**Given** a maximum session limit of 10,000 configured
**When** attempting to create session 10,001
**Then** the connection shall be rejected with appropriate error logging

#### Scenario: Memory Pressure Protection
**Given** memory usage exceeding configured threshold
**When** a new session request arrives
**Then** the oldest idle sessions shall be evicted to free memory

## MODIFIED Requirements

### 5. Proxy Handler Interface Extension

The existing proxy handler interface shall be extended to support NAT-specific functionality.

#### Scenario: NAT Handler Registration
**Given** a NAT proxy handler implementation
**When** registering handlers in the Xray core
**Then** the NAT handler shall be available as `nat` proxy type in configuration files

#### Scenario: Handler Selection Integration
**Given** a routing rule targeting a NAT outbound handler
**When** dispatching a connection
**Then** the NAT handler shall be selected and invoked appropriately

### 6. Session Context Enhancement

The existing session context shall be enhanced to include NAT-specific metadata.

#### Scenario: NAT Metadata Preservation
**Given** a connection passing through NAT transformation
**When** accessing session context in subsequent processing stages
**Then** original source, virtual destination, and NAT mode shall be available

#### Scenario: Address Override Support
**Given** a NAT-transformed connection
**When** processing outbound traffic
**Then** the endpoint override mechanism shall handle NAT address translation

### 7. Configuration Schema Extension

The existing Xray configuration schema shall be extended to support NAT-specific settings.

#### Scenario: NAT Outbound Configuration
**Given** an outbound configuration with type `nat`
**When** parsing the configuration
**Then** the following fields shall be supported:
```json
{
  "protocol": "nat",
  "settings": {
    "siteId": "site-b",
    "virtualRanges": [
      {
        "virtualNetwork": "240.2.2.0/24",
        "realNetwork": "192.168.1.0/24",
        "ipv6VirtualPrefix": "64:FF9B:2222::/96"
      }
    ],
    "rules": [
      {
        "virtualDestination": "240.2.2.0/24",
        "realDestination": "192.168.1.0/24",
        "protocol": "tcp"
      }
    ],
    "sessionTimeout": {
      "tcpTimeout": 300,
      "udpTimeout": 60,
      "cleanupInterval": 30
    },
    "limits": {
      "maxSessions": 10000,
      "maxMemoryMB": 512
    }
  }
}
```

### 8. Performance Monitoring Integration

The existing metrics collection system shall be extended to include NAT-specific metrics.

#### Scenario: NAT Metrics Collection
**Given** a NAT handler processing traffic
**When** accessing system metrics
**Then** the following NAT-specific metrics shall be available:
- `nat_active_sessions`: Current number of active NAT sessions
- `nat_bytes_processed`: Total bytes processed through NAT
- `nat_sessions_created_total`: Total NAT sessions created
- `nat_sessions_closed_total`: Total NAT sessions closed
- `nat_transform_errors_total`: Total NAT transformation errors

## REMOVED Requirements

### 9. None

No existing requirements are being removed in this implementation.

## Implementation Constraints

### 10. Compatibility Requirements

The NAT implementation shall maintain backward compatibility with existing Xray core functionality.

#### Scenario: Mixed Handler Environment
**Given** a configuration with both NAT and non-NAT outbound handlers
**When** processing connections
**Then** non-NAT connections shall continue to work without any modification

#### Scenario: Configuration Migration
**Given** an existing Xray configuration without NAT settings
**When** upgrading to a version with NAT support
**Then** existing configurations shall continue to work without modification

### 11. Security Requirements

The NAT implementation shall enforce proper security boundaries between different sites and sessions.

#### Scenario: Session Isolation
**Given** multiple concurrent NAT sessions from different source sites
**When** processing traffic
**Then** sessions from different sites shall be completely isolated

#### Scenario: Address Validation
**Given** a packet with suspicious address characteristics
**When** applying NAT transformation
**Then** the packet shall be rejected if validation fails