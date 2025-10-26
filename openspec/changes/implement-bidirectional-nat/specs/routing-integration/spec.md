# Routing Integration Specification

## ADDED Requirements

### 1. Virtual IP Destination Matching

The routing system shall support matching packets based on virtual IP destinations.

#### Scenario: Virtual IP Route Matching
**Given** a routing rule with `virtualDestination` set to `240.2.2.0/24`
**When** a packet arrives with destination `240.2.2.20:80`
**Then** the rule shall match and the corresponding outbound handler shall be selected

#### Scenario: IPv6 Virtual IP Matching
**Given** a routing rule for IPv6 virtual prefix `64:FF9B:2222::/96`
**When** an IPv6 packet arrives with destination `64:FF9B:2222::20`
**Then** the rule shall match and the packet shall be routed appropriately

### 2. NAT Routing Rules

The routing configuration shall support NAT-specific rules with site-based filtering.

#### Scenario: Site-Based NAT Routing
**Given** a NAT routing rule configured with `siteId: "site-b"`
**When** traffic originates from a connection associated with site B
**Then** the NAT-specific routing logic shall be applied

#### Scenario: NAT Rule Precedence
**Given** both standard and NAT routing rules in configuration
**When** evaluating routing decisions
**Then** NAT rules shall be evaluated before standard rules

### 3. Enhanced Routing Conditions

The existing routing condition system shall be enhanced with NAT-specific matchers.

#### Scenario: Virtual IP Condition Matching
**Given** a routing condition with `virtualIP` set to `240.2.2.0/24`
**When** a connection request contains destination `240.2.2.20:80`
**Then** the condition shall evaluate to true

#### Scenario: Site Identifier Matching
**Given** a routing condition with `sourceSite` set to `site-a`
**When** traffic originates from Site A's gateway
**Then** the condition shall match and appropriate routing applied

## MODIFIED Requirements

### 4. Router Configuration Schema

The existing router configuration shall be extended to support NAT rules and virtual IP matching.

#### Scenario: Extended Router Configuration
**Given** a router configuration with NAT support
**When** parsing the configuration
**Then** the following additional fields shall be supported:
```json
{
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "nat",
        "virtualDestination": "240.2.2.0/24",
        "outboundTag": "nat-outbound-site-b"
      },
      {
        "type": "field",
        "outboundTag": "proxy-outbound",
        "domain": ["geosite:cn"]
      }
    ],
    "natRules": [
      {
        "ruleId": "site-b-nat",
        "siteId": "site-b",
        "virtualDestination": "240.2.2.0/24",
        "realDestination": "192.168.1.0/24",
        "protocol": "tcp,udp"
      }
    ]
  }
}
```

### 5. Rule Evaluation Engine

The existing rule evaluation engine shall be enhanced to process NAT rules alongside standard routing rules.

#### Scenario: Mixed Rule Evaluation
**Given** a configuration containing both NAT and standard routing rules
**When** processing a connection request
**Then** NAT rules shall be evaluated first, followed by standard rules

#### Scenario: Rule Caching Optimization
**Given** frequent connections to the same virtual destinations
**When** processing repeated requests
**Then** NAT routing decisions shall be cached for improved performance

### 6. Dispatcher Integration

The existing dispatcher shall be enhanced to handle NAT-specific handler selection.

#### Scenario: NAT Handler Selection
**Given** a routing decision selecting a NAT outbound handler
**When** dispatching the connection
**Then** the dispatcher shall route the connection to the appropriate NAT proxy handler

#### Scenario: Context Preservation
**Given** a connection being dispatched to a NAT handler
**When** the NAT handler processes the connection
**Then** all relevant routing context and metadata shall be preserved

### 7. Router Statistics Enhancement

The existing router statistics shall be enhanced to include NAT-specific metrics.

#### Scenario: NAT Route Statistics
**Given** a router with NAT rules configured
**When** accessing router statistics
**Then** the following metrics shall be available:
- `nat_rules_evaluated_total`: Number of NAT rule evaluations
- `nat_routes_matched_total`: Number of successful NAT route matches
- `virtual_destination_hits_total`: Hits per virtual destination
- `nat_routing_errors_total`: NAT-specific routing errors

## REMOVED Requirements

### 8. None

No existing requirements are being removed in this implementation.

## Implementation Constraints

### 9. Routing Compatibility

The enhanced routing system shall maintain backward compatibility with existing configurations.

#### Scenario: Backward Compatibility
**Given** an existing Xray configuration without NAT rules
**When** upgrading to a version with NAT support
**Then** all existing routing behavior shall remain unchanged

#### Scenario: Graceful Degradation
**Given** a configuration issue with NAT rules
**When** the router encounters errors
**Then** the router shall continue processing standard rules with appropriate logging

### 10. Performance Requirements

The routing enhancements shall not significantly impact performance of existing routing operations.

#### Scenario: Performance Impact Measurement
**Given** a system processing standard routing operations
**When** NAT rules are added to configuration
**Then** the performance impact shall be less than 5% for non-NAT traffic

#### Scenario: NAT Rule Optimization
**Given** a configuration with many NAT rules
**When** processing virtual IP destinations
**Then** NAT rule matching shall be optimized using efficient data structures

### 11. Configuration Validation

The system shall validate NAT routing configurations to prevent common misconfigurations.

#### Scenario: Overlapping Virtual IP Ranges
**Given** a configuration with overlapping virtual IP ranges
**When** validating the configuration
**Then** the system shall detect and report the conflict

#### Scenario: Invalid Virtual IP Networks
**Given** a NAT rule with invalid virtual IP network specification
**When** validating the configuration
**Then** the system shall reject the configuration with descriptive error

## Integration Scenarios

### 12. End-to-End NAT Routing

The complete flow from connection initiation to NAT transformation shall work seamlessly.

#### Scenario: Complete NAT Routing Flow
**Given** a complete configuration with NAT routing rules and handlers
**When** Device A (`192.168.1.20`) connects to `240.2.2.20:80`
**Then** the following sequence shall occur:
1. Router matches the virtual destination to NAT rule
2. Dispatcher selects NAT outbound handler
3. NAT handler applies DNAT transformation
4. Connection reaches Device B (`192.168.1.20:80`)
5. Return traffic follows reverse SNAT path

#### Scenario: Multi-Protocol Support
**Given** NAT rules configured for both TCP and UDP
**When** traffic of either protocol is routed
**Then** both protocols shall work through the NAT system with appropriate session handling