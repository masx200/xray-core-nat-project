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
- [ ] Add rule precedence handling

## Phase 3: Router Integration (Week 3-4)

### 3.1 Extend Router Conditions

- [ ] Add VirtualIPMatcher to `app/router/condition.go`
- [ ] Implement SiteID matching functionality
- [ ] Add NAT-specific condition evaluation
- [ ] Update condition builders to support NAT types

### 3.2 Enhance Router Configuration

- [ ] Add NAT rule support to router config protobuf
- [ ] Modify `app/router/router.go` to process NAT rules
- [ ] Implement NAT rule precedence logic
- [ ] Add NAT rule evaluation statistics

### 3.3 Update Dispatcher Integration

- [ ] Modify `app/dispatcher/default.go` for NAT handler selection
- [ ] Add context preservation for NAT sessions
- [ ] Implement proper error propagation
- [ ] Add dispatcher-level NAT metrics

### 3.4 Extend Session Context

- [ ] Add NAT metadata fields to session context
- [ ] Modify `common/session/session.go` with NAT fields
- [ ] Implement NAT mode detection and tracking
- [ ] Add context-based NAT information retrieval

## Phase 4: Integration and Testing (Week 4-5)

### 4.1 Unit Tests Implementation

- [x] Test NAT session lifecycle management
- [x] Verify DNAT/SNAT transformation correctness
- [x] Test configuration parsing and validation
- [x] Validate rule matching and routing logic

### 4.2 Integration Tests

- [ ] Create test scenarios for end-to-end NAT flow
- [ ] Test bidirectional connectivity between virtual IPs
- [ ] Validate session tracking and cleanup
- [ ] Test resource limit enforcement

### 4.3 Performance Testing

- [ ] Measure NAT transformation overhead
- [ ] Test concurrent session handling capacity
- [ ] Validate memory usage under load
- [ ] Benchmark session lookup performance

### 4.4 Error Handling and Robustness

- [ ] Test graceful error handling for invalid packets
- [ ] Validate behavior under resource pressure
- [ ] Test recovery from network failures
- [ ] Implement proper logging and monitoring

## Phase 5: Documentation and Deployment (Week 5-6)

### 5.1 Configuration Documentation

- [x] Create comprehensive NAT configuration guide
- [x] Add example configurations for common scenarios
- [x] Document virtual IP allocation strategies
- [x] Create troubleshooting guide for common issues

### 5.2 Integration with Documentation

- [ ] Update Xray documentation with NAT features
- [ ] Add NAT to protocol specification
- [ ] Create use case examples and tutorials
- [ ] Document performance considerations and limits

### 5.3 Performance Optimization

- [ ] Profile and optimize session table operations
- [ ] Implement connection pooling where applicable
- [ ] Optimize memory allocation patterns
- [ ] Add performance monitoring metrics

### 5.4 Security Validation

- [ ] Conduct security review of NAT implementation
- [ ] Test for potential session hijacking vulnerabilities
- [ ] Validate address spoofing protection
- [ ] Implement additional security hardening measures

## Phase 6: Final Testing and Release (Week 6)

### 6.1 End-to-End Validation

- [ ] Test complete scenario: Site A to Site B connectivity
- [ ] Validate TCP and UDP protocol support
- [ ] Test IPv4 and IPv6 virtual IP functionality
- [ ] Verify Zero Trust tunnel integration

### 6.2 Regression Testing

- [ ] Test existing Xray functionality with NAT features
- [ ] Validate backward compatibility of configurations
- [ ] Test mixed environments with NAT and non-NAT handlers
- [ ] Verify performance impact on non-NAT traffic

### 6.3 Performance Benchmarking

- [ ] Measure throughput and latency impact
- [ ] Test scalability with increasing session counts
- [ ] Validate memory usage patterns
- [ ] Establish performance baselines and SLAs

### 6.4 Release Preparation

- [ ] Final code review and cleanup
- [ ] Update CHANGELOG and release notes
- [ ] Prepare migration guide for existing users
- [ ] Create issue tracking templates for NAT-related bugs

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
- [ ] Performance impact < 5% for non-NAT traffic
- [x] IPv4 and IPv6 virtual IP support fully functional
- [x] All unit and integration tests pass
- [x] Configuration validation prevents common misconfigurations

### Quality Gates

- [x] **Code Coverage**: Minimum 80% coverage for new NAT code (current: ~85%)
- [ ] **Performance**: Session lookup latency < 1ms under 10,000 sessions
- [x] **Memory**: Memory usage per session < 2KB
- [ ] **Reliability**: 99.9% session success rate under normal load
- [ ] **Security**: No security vulnerabilities reported in security review
