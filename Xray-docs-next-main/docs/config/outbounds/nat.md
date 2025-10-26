# NAT Outbound

NAT (Network Address Translation) 出站协议提供双向网络地址转换功能，专为零信任网络中IP地址冲突的站点间通信设计。

## 概述

NAT出站协议允许配置虚拟IP地址空间，将虚拟IP地址映射到真实的IP地址网络。当多个站点具有重叠的私有IP地址空间时，此功能特别有用。

**主要特性：**
- 双向NAT转换（DNAT和SNAT）
- 虚拟IP范围管理
- 基于站点的规则选择
- 协议和端口过滤
- IPv6嵌入式IPv4支持
- 会话管理和超时清理
- LRU资源管理

## 基础配置

```json
{
  "outbounds": [
    {
      "protocol": "nat",
      "tag": "nat-out",
      "settings": {
        "siteId": "site-b",
        "virtualRanges": [
          {
            "virtualNetwork": "240.2.2.0/24",
            "realNetwork": "192.168.1.0/24",
            "ipv6Enabled": true,
            "ipv6Prefix": "64:FF9B:2222::/96"
          }
        ],
        "rules": [
          {
            "ruleId": "rule-web-server",
            "virtualDestination": "240.2.2.20",
            "realDestination": "192.168.1.20",
            "protocol": "tcp",
            "sourceSite": "site-a"
          }
        ]
      }
    }
  ]
}
```

## 配置结构

### `settings` 对象

```json
{
  "siteId": "string",
  "virtualRanges": [VirtualRange],
  "rules": [NATRule],
  "sessionTimeout": SessionTimeout,
  "resourceLimits": ResourceLimits
}
```

#### `siteId` (string)

必需字段。标识当前站点的唯一标识符。用于站点间规则匹配。

#### `virtualRanges` (array of VirtualRange)

定义虚拟IP地址范围和对应的真实网络映射。

#### `rules` (array of NATRule)

定义特定的NAT转换规则。

#### `sessionTimeout` (SessionTimeout)

会话超时配置。

#### `resourceLimits` (ResourceLimits)

资源限制配置。

### VirtualRange

```json
{
  "virtualNetwork": "240.2.2.0/24",
  "realNetwork": "192.168.1.0/24",
  "ipv6Enabled": true,
  "ipv6Prefix": "64:FF9B:2222::/96"
}
```

#### `virtualNetwork` (string)

虚拟IP地址范围，支持IPv4 CIDR格式（如 `240.2.2.0/24`）。

#### `realNetwork` (string)

对应的真实IP地址范围，支持IPv4 CIDR格式。

#### `ipv6Enabled` (boolean)

是否启用IPv6嵌入式IPv4支持。默认为 `false`。

#### `ipv6Prefix` (string)

IPv6虚拟前缀，用于IPv6嵌入式IPv4地址转换。

### NATRule

```json
{
  "ruleId": "rule-web-server",
  "sourceSite": "site-a,site-b",
  "virtualDestination": "240.2.2.20",
  "realDestination": "192.168.1.20",
  "protocol": "tcp",
  "portMapping": {
    "originalPort": "8080",
    "translatedPort": "80"
  }
}
```

#### `ruleId` (string)

规则的唯一标识符。

#### `sourceSite` (string, 可选)

源站点过滤器。支持多个站点用逗号分隔。空值表示匹配所有站点。

#### `virtualDestination` (string)

虚拟目标地址，可以是单个IP或CIDR范围。

#### `realDestination` (string)

对应的真实目标地址。

#### `protocol` (string, 可选)

协议过滤器。支持：
- `"tcp"` - 仅TCP
- `"udp"` - 仅UDP
- `"tcp,udp"` 或 `"udp,tcp"` - TCP和UDP

默认为空字符串（匹配所有协议）。

#### `portMapping` (PortMapping, 可选)

端口映射配置。

### PortMapping

```json
{
  "originalPort": "8080",
  "translatedPort": "80"
}
```

#### `originalPort` (string)

原始端口号，支持单个端口或端口范围（如 `"8080"` 或 `"8000-9000"`）。

#### `translatedPort` (string)

转换后的端口号。

### SessionTimeout

```json
{
  "tcpTimeout": 300,
  "udpTimeout": 60,
  "cleanupInterval": 30
}
```

#### `tcpTimeout` (uint32, 单位：秒)

TCP会话超时时间。默认为 300秒（5分钟）。

#### `udpTimeout` (uint32, 单位：秒)

UDP会话超时时间。默认为 60秒（1分钟）。

#### `cleanupInterval` (uint32, 单位：秒)

清理过期会话的间隔时间。默认为 30秒。

### ResourceLimits

```json
{
  "maxSessions": 10000,
  "maxMemoryMB": 100,
  "cleanupThreshold": 0.8
}
```

#### `maxSessions` (uint32)

最大会话数量限制。默认为 10000。

#### `maxMemoryMB` (uint32)

最大内存使用限制（MB）。默认为 100MB。

#### `cleanupThreshold` (float32)

清理阈值（0.0-1.0）。当会话数量达到此比例时触发清理。默认为 0.8。

## 高级配置

### 多站点配置

```json
{
  "outbounds": [
    {
      "protocol": "nat",
      "tag": "nat-site-b",
      "settings": {
        "siteId": "site-b",
        "virtualRanges": [
          {
            "virtualNetwork": "240.2.2.0/24",
            "realNetwork": "192.168.1.0/24"
          }
        ],
        "rules": [
          {
            "ruleId": "from-site-a",
            "sourceSite": "site-a",
            "virtualDestination": "240.2.2.0/24",
            "realDestination": "192.168.1.0/24",
            "protocol": "tcp,udp"
          },
          {
            "ruleId": "web-server-specific",
            "sourceSite": "site-a",
            "virtualDestination": "240.2.2.20",
            "realDestination": "192.168.1.20",
            "protocol": "tcp"
          }
        ]
      }
    }
  ]
}
```

### IPv6嵌入式IPv4支持

```json
{
  "outbounds": [
    {
      "protocol": "nat",
      "tag": "nat-ipv6",
      "settings": {
        "siteId": "site-c",
        "virtualRanges": [
          {
            "virtualNetwork": "64:FF9B:1111::192.168.1.1/120",
            "realNetwork": "192.168.1.0/24",
            "ipv6Enabled": true,
            "ipv6Prefix": "64:FF9B:1111::/96"
          }
        ]
      }
    }
  ]
}
```

### 端口映射配置

```json
{
  "outbounds": [
    {
      "protocol": "nat",
      "settings": {
        "siteId": "site-b",
        "rules": [
          {
            "ruleId": "http-redirect",
            "virtualDestination": "240.2.2.20",
            "realDestination": "192.168.1.20",
            "protocol": "tcp",
            "portMapping": {
              "originalPort": "8080",
              "translatedPort": "80"
            }
          },
          {
            "ruleId": "port-range",
            "virtualDestination": "240.2.2.21",
            "realDestination": "192.168.1.21",
            "protocol": "tcp",
            "portMapping": {
              "originalPort": "8000-9000",
              "translatedPort": "80"
            }
          }
        ]
      }
    }
  ]
}
```

## 工作原理

### 地址转换流程

1. **出站连接**：
   - 检查目标地址是否匹配虚拟IP规则
   - 应用DNAT将虚拟IP转换为真实IP
   - 建立连接并创建会话记录
   - 返回流量应用SNAT将真实IP转换为虚拟IP

2. **会话管理**：
   - 使用LRU算法管理会话
   - 定期清理过期会话
   - 监控资源使用情况

3. **规则匹配**：
   - 优先匹配特定规则
   - 然后匹配虚拟范围
   - 支持协议、端口、站点过滤

### IPv6嵌入式IPv4

NAT支持IPv6地址中嵌入IPv4地址，遵循RFC 6052标准：
- 格式：`64:FF9B:XXXX::IPv4地址`
- 自动提取IPv4部分并应用转换
- 支持压缩和扩展IPv6格式

## 使用场景

### 场景1：企业网络互联

两个办公室都使用 `192.168.1.0/24` 网络段，需要互相访问：

**Site A配置：**
```json
{
  "siteId": "site-a",
  "virtualRanges": [
    {
      "virtualNetwork": "240.1.1.0/24",
      "realNetwork": "192.168.1.0/24"
    }
  ]
}
```

**Site B配置：**
```json
{
  "siteId": "site-b",
  "virtualRanges": [
    {
      "virtualNetwork": "240.2.2.0/24",
      "realNetwork": "192.168.1.0/24"
    }
  ],
  "rules": [
    {
      "sourceSite": "site-a",
      "virtualDestination": "240.2.2.0/24",
      "realDestination": "192.168.1.0/24"
    }
  ]
}
```

### 场景2：数据中心整合

整合多个使用相同IP空间的数据中心，通过NAT实现隔离：

```json
{
  "siteId": "dc-east",
  "virtualRanges": [
    {
      "virtualNetwork": "240.10.0.0/16",
      "realNetwork": "10.0.0.0/8"
    }
  ],
  "rules": [
    {
      "sourceSite": "dc-west",
      "virtualDestination": "240.10.0.0/16",
      "realDestination": "10.0.0.0/8",
      "protocol": "tcp,udp"
    }
  ]
}
```

## 性能优化建议

1. **会话管理**：
   - 根据业务需求调整 `maxSessions` 和 `maxMemoryMB`
   - 适当设置 `cleanupThreshold` 避免资源耗尽

2. **规则配置**：
   - 将高频访问的规则放在前面
   - 使用具体的IP地址而不是大范围CIDR

3. **网络设计**：
   - 选择不冲突的虚拟IP范围
   - 合理规划IPv6前缀以支持嵌入式IPv4

## 故障排除

### 常见问题

1. **连接超时**：
   - 检查虚拟IP范围配置是否正确
   - 验证真实网络路由可达性

2. **会话泄露**：
   - 调整 `sessionTimeout` 设置
   - 监控 `resourceLimits` 使用情况

3. **规则不匹配**：
   - 验证 `sourceSite` 配置
   - 检查协议和端口过滤条件

### 日志监控

启用详细日志以调试NAT转换：

```json
{
  "log": {
    "loglevel": "info"
  }
}
```

## 安全考虑

1. **访问控制**：
   - 使用 `sourceSite` 限制哪些站点可以访问
   - 结合路由规则实现细粒度控制

2. **资源保护**：
   - 设置合理的 `resourceLimits`
   - 监控内存和会话使用情况

3. **网络隔离**：
   - 确保虚拟IP范围不与现有网络冲突
   - 在边界设备上实施适当过滤

## 高级特性

### 动态规则

通过结合API功能，可以实现动态NAT规则管理：

```json
{
  "api": {
    "tag": "api",
    "services": ["StatsService", "RoutingService"]
  }
}
```

### 监控统计

```json
{
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "stats": {
          "userUplink": true,
          "userDownlink": true
        }
      }
    }
  }
}
```

这将启用NAT连接的详细统计信息收集。