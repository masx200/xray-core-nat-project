# OpenSpec Proposal: Implement Bidirectional NAT

## Overview

This proposal outlines the implementation of bidirectional Network Address
Translation (NAT) functionality for Xray-core to enable connectivity between
sites with conflicting IP addresses through a Zero Trust network.

## Problem Statement

Two sites (A and B) need to establish network connectivity but have overlapping
IP address ranges that cannot be modified:

- **Site A**: Gateway IP `192.168.1.1/24`, Zero Trust IP `10.10.10.10/24`
- **Site B**: Gateway IP `192.168.1.1/24`, Zero Trust IP `10.20.20.20/24`

The goal is to enable communication between `192.168.1.20` in Site A and
`192.168.1.20` in Site B without modifying device configurations.

## Solution Approach

### Virtual IP Address Assignment

- **Site A Virtual IP**: `240.1.1.1/24` and `64:FF9B:1111::/96`
- **Site B Virtual IP**: `240.2.2.2/24` and `64:FF9B:2222::/96`

### Core Architecture

1. **Site A Router**: Smart router that directs traffic for Site B's virtual IPs
   through Zero Trust tunnel
2. **Site B Router**: Bidirectional NAT gateway performing DNAT/SNAT
   translations
3. **Virtual IP Layer**: Neutral address space decoupling overlapping physical
   networks

## Key Components

### 1. NAT Proxy Handler

- **Location**: `proxy/nat/`
- **Purpose**: Implement DNAT/SNAT translation with session tracking
- **Key Features**:
  - Bidirectional address translation
  - Session state management with timeout cleanup
  - Support for both IPv4 and IPv6 virtual IP ranges

### 2. Enhanced Router Configuration

- **Location**: `app/router/`
- **Purpose**: Add NAT-specific routing rules and virtual IP matching
- **Key Features**:
  - Virtual IP destination matching
  - NAT rule precedence over standard routing
  - Site-based NAT policy management

### 3. Session Context Extensions

- **Location**: `common/session/`
- **Purpose**: Add NAT-specific metadata to connection sessions
- **Key Features**:
  - Original/source address preservation
  - Virtual/real address mapping
  - NAT mode detection and tracking

### 4. Configuration Schema

- **Location**: `infra/conf/`
- **Purpose**: Define NAT configuration structure
- **Key Features**:
  - Virtual IP range definitions
  - NAT routing rule specification
  - Site identifier and association

## Dependencies

- Xray-core routing infrastructure
- Zero Trust network connectivity (assumed existing)
- Current session management system
- Existing proxy handler framework

## Change Impact

This is a **breaking change** that requires:

1. New proxy handler type
2. Extended router configuration schema
3. Enhanced session context structure
4. Additional routing condition matchers

## Validation Approach

1. Unit tests for NAT translation logic
2. Integration tests with existing routing system
3. End-to-end connectivity validation
4. Performance testing with concurrent sessions

## Success Criteria

- ✅ Site A device `192.168.1.20` can communicate with Site B device
  `192.168.1.20` using virtual IPs
- ✅ Bidirectional NAT maintains session state correctly
- ✅ No modification required on endpoint devices
- ✅ Performance impact within acceptable limits
- ✅ IPv4 and IPv6 virtual IP support
- ✅ Graceful error handling and fallback

## Risks and Mitigations

### Risk 1: Session Table Exhaustion

- **Mitigation**: Implement aggressive timeout cleanup and memory limits

### Risk 2: Performance Degradation

- **Mitigation**: Optimize lookup algorithms and consider connection pooling

### Risk 3: Complex State Management

- **Mitigation**: Design clear state lifecycle with proper cleanup

### Risk 4: Routing Complexity

- **Mitigation**: Maintain clear separation between NAT and standard routing

## Future Considerations

- Support for dynamic NAT (DHCP-based virtual IP allocation)
- Multiple site interoperability (beyond two sites)
- NAT traversal for additional protocols
- Integration with existing enterprise NAT solutions

---

**Approvals:**

- [ ] Architecture Review
- [ ] Security Review
- [ ] Performance Review
- [ ] Product Owner Approval

**Change ID**: `implement-bidirectional-nat`
