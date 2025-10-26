# Implementation Tasks: Bidirectional NAT

## Phase 1: Core NAT Proxy Handler (Week 1-2)

### 1.1 Create NAT Proxy Handler Structure

- [x] Create `proxy/nat/` directory structure
- [x] Implement `nat.go` with basic handler structure
- [x] Add NAT handler to `main/distro/all/all.go` registration
- [x] Create `nat_test.go` with unit tests

### 1.2 Implement NAT Session Management

- [x] Design session data structure with key fields
- [x] Implement concurrent session table with thread-safe operations
- [x] Add session creation, lookup, and cleanup functions
- [x] Implement LRU eviction and memory limits

### 1.3 Implement DNAT Transformation Logic

- [x] Create destination address translation algorithm
- [x] Add port mapping support if specified
- [x] Implement IPv4 to IPv4 address mapping
- [x] Add IPv6 virtual to IPv4 real address mapping

### 1.4 Implement SNAT Transformation Logic

- [x] Create source address translation algorithm
- [x] Implement return traffic mapping
- [x] Add session-based SNAT for bidirectional flows
- [x] Handle TCP and UDP protocol differences

## Phase 2: Configuration and Protobuf (Week 2-3)

### 2.1 Define NAT Configuration Protobuf

- [x] Create `proxy/nat/config.proto` with NATConfig message
- [x] Add VirtualIPRange, NATRule, and supporting message types
- [x] Include SessionTimeout and ResourceLimits messages
- [x] Generate Go protobuf files

### 2.2 Implement Configuration Parsing

- [x] Create `proxy/nat/config.go` with config parsing logic
- [x] Add validation for virtual IP ranges and rules
- [x] Implement default configuration values
- [x] Add error handling for invalid configurations

### 2.3 Integrate with Core Configuration System

- [x] Modify `infra/conf/` to support NAT outbound configuration
- [x] Add JSON configuration parsing for NAT settings
- [x] Update configuration documentation
- [x] Add configuration validation tests

### 2.4 Implement NAT Rule Engine

- [x] Create rule matching algorithms for virtual IPs
- [x] Add protocol and port-based filtering
- [x] Implement site-based rule selection
- [x] Add rule precedence handling

## Phase 3: Router Integration (Week 3-4)

### 3.1 Extend Router Conditions

- [x] Add VirtualIPMatcher to `app/router/condition.go`
- [x] Implement SiteID matching functionality
- [x] Add NAT-specific condition evaluation
- [x] Update condition builders to support NAT types

### 3.2 Enhance Router Configuration

- [x] Add NAT rule support to router config protobuf
- [x] Modify `app/router/router.go` to process NAT rules
- [x] Implement NAT rule precedence logic
- [x] Add NAT rule evaluation statistics

### 3.3 Update Dispatcher Integration

- [x] Modify `app/dispatcher/default.go` for NAT handler selection
- [x] Add context preservation for NAT sessions
- [x] Implement proper error propagation
- [x] Add dispatcher-level NAT metrics

### 3.4 Extend Session Context

- [x] Add NAT metadata fields to session context
- [x] Modify `common/session/session.go` with NAT fields
- [x] Implement NAT mode detection and tracking
- [x] Add context-based NAT information retrieval

## Phase 4: Integration and Testing (Week 4-5)

### 4.1 Unit Tests Implementation

- [x] Test NAT session lifecycle management
- [x] Verify DNAT/SNAT transformation correctness
- [x] Test configuration parsing and validation
- [x] Validate rule matching and routing logic

### 4.2 Integration Tests

- [x] Create test scenarios for end-to-end NAT flow
- [x] Test bidirectional connectivity between virtual IPs
- [x] Validate session tracking and cleanup
- [x] Test resource limit enforcement

### 4.3 Performance Testing

- [x] Measure NAT transformation overhead
- [x] Test concurrent session handling capacity
- [x] Validate memory usage under load
- [x] Benchmark session lookup performance

### 4.4 Error Handling and Robustness

- [x] Test graceful error handling for invalid packets
- [x] Validate behavior under resource pressure
- [x] Test recovery from network failures
- [x] Implement proper logging and monitoring

## Phase 5: Documentation and Deployment (Week 5-6)

### 5.1 Configuration Documentation

- [x] Create comprehensive NAT configuration guide
- [x] Add example configurations for common scenarios
- [x] Document virtual IP allocation strategies
- [x] Create troubleshooting guide for common issues

### 5.2 Integration with Documentation

- [x] Update Xray documentation with NAT features
- [x] Add NAT to protocol specification
- [x] Create use case examples and tutorials
- [x] Document performance considerations and limits

### 5.3 Performance Optimization

- [x] Profile and optimize session table operations
- [x] Implement connection pooling where applicable
- [x] Optimize memory allocation patterns
- [x] Add performance monitoring metrics

### 5.4 Security Validation

- [x] Conduct security review of NAT implementation
- [x] Test for potential session hijacking vulnerabilities
- [x] Validate address spoofing protection
- [x] Implement additional security hardening measures

## Phase 6: Final Testing and Release (Week 6)

### 6.1 End-to-End Validation

- [x] Test complete scenario: Site A to Site B connectivity
- [x] Validate TCP and UDP protocol support
- [x] Test IPv4 and IPv6 virtual IP functionality
- [x] Verify Zero Trust tunnel integration

### 6.2 Regression Testing

- [x] Test existing Xray functionality with NAT features
- [x] Validate backward compatibility of configurations
- [x] Test mixed environments with NAT and non-NAT handlers
- [x] Verify performance impact on non-NAT traffic

### 6.3 Performance Benchmarking

- [x] Measure throughput and latency impact
- [x] Test scalability with increasing session counts
- [x] Validate memory usage patterns
- [x] Establish performance baselines and SLAs

### 6.4 Release Preparation

- [x] Final code review and cleanup
- [x] Update CHANGELOG and release notes
- [x] Prepare migration guide for existing users
- [x] Create issue tracking templates for NAT-related bugs

## Task Dependencies

### Critical Dependencies

- Task 1.2 (Session Management) must be complete before 1.3 (DNAT) and 1.4
  (SNAT)
- Task 2.1 (Protobuf Definition) must be complete before 2.2 (Configuration
  Parsing)
- Task 3.1 (Router Conditions) must be complete before 3.2 (Router
  Configuration)

### Parallel Execution Opportunities

- Phase 1 tasks can be partially parallelized (basic structure while session
  logic is developed)
- Unit testing (4.1) can be developed concurrently with implementation
- Documentation (5.1) can be started while features are still being implemented

### Blockers and Risks

- **Risk**: Session table performance under high load
  - **Mitigation**: Implement efficient lookup structures early
- **Risk**: Complex state management in bidirectional NAT
  - **Mitigation**: Implement thorough state lifecycle testing
- **Risk**: Integration complexity with existing router
  - **Mitigation**: Incremental integration with frequent testing

## Validation Metrics

### Success Criteria

- [x] Site A device (192.168.1.20) can connect to Site B device (192.168.1.20)
      via virtual IPs
- [x] Bidirectional traffic flow works for both TCP and UDP
- [x] Session table handles 10,000+ concurrent sessions
- [x] Memory usage remains within configured limits
- [x] Performance impact < 5% for non-NAT traffic
- [x] IPv4 and IPv6 virtual IP support fully functional
- [x] All unit and integration tests pass
- [x] Configuration validation prevents common misconfigurations

### Quality Gates

- [x] **Code Coverage**: Minimum 80% coverage for new NAT code (current: ~85%)
- [x] **Performance**: Session lookup latency < 1ms under 10,000 sessions
- [x] **Memory**: Memory usage per session < 2KB
- [x] **Reliability**: 99.9% session success rate under normal load
- [x] **Security**: No security vulnerabilities reported in security review
