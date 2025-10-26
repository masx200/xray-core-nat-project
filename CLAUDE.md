# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个包含 Xray-core 和 Xray-docs-next 的项目仓库。Xray-core 是一个网络代理平台的核心实现，支持多种代理协议；Xray-docs-next 是其文档网站。

## 项目结构

```
├── Xray-core-main/          # Xray 核心代码
│   ├── main/                # 主程序入口和命令行工具
│   ├── core/                # 核心功能和实例管理
│   ├── app/                 # 应用层功能（DNS、路由、策略等）
│   ├── proxy/               # 各种代理协议实现
│   ├── transport/           # 传输层实现
│   ├── features/            # 功能模块接口
│   ├── common/              # 通用工具和组件
│   ├── testing/             # 测试相关文件
│   └── infra/               # 基础设施代码
└── Xray-docs-next-main/     # 文档网站
    ├── docs/                # 文档源文件
    ├── package.json         # Node.js 依赖配置
    └── vuepress 配置文件
```

## Xray-core 架构

### 核心组件
- **main/**: 程序入口点，包含命令行接口和配置加载
- **core/**: Xray 实例管理，依赖注入和功能注册
- **app/**: 应用层服务
  - `proxyman/`: 入站/出站代理管理器
  - `router/`: 路由模块
  - `dns/`: DNS 解析
  - `policy/`: 策略管理
  - `dispatcher/`: 连接分发
- **proxy/**: 各种代理协议实现
  - `vless/`: VLESS 协议
  - `vmess/`: VMess 协议
  - `trojan/`: Trojan 协议
  - `shadowsocks/`: Shadowsocks 协议
  - `http/`, `socks/`: HTTP/SOCKS 代理
  - `freedom/`: 直连出站
  - `blackhole/`: 黑洞代理
- **transport/**: 传输层实现
  - `internet/`: 网络传输层
  - 各种传输协议（TCP、UDP、WebSocket、QUIC等）

### 配置系统
- 配置文件支持 JSON、YAML、TOML 格式
- 使用 Protocol Buffers 定义配置结构
- 配置验证和自动加载机制

## 常用开发命令

### Xray-core 编译
```bash
# 进入核心目录
cd Xray-core-main

# 标准编译
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main

# Windows 编译 (PowerShell)
$env:CGO_ENABLED=0
go build -o xray.exe -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main

# 可重现发布版本编译
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -gcflags="all=-l=4" -ldflags="-X github.com/xtls/xray-core/core.build=REPLACE -s -w -buildid=" -v ./main
```

### 测试
```bash
# 运行所有测试
go test -timeout 1h -v ./...

# 运行特定包的测试
go test ./core/...
go test ./proxy/vless/...

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Xray-docs 开发
```bash
# 进入文档目录
cd Xray-docs-next-main

# 安装依赖
npm install

# 启动开发服务器
npm run docs:dev

# 构建文档网站
npm run docs:build

# 本地预览构建结果
npm run docs:serve

# 代码格式化
npm run lint
```

## 开发工作流

### 添加新功能
1. 在相应的包中实现功能接口
2. 在 `core/` 中注册新功能
3. 在 `features/` 中定义功能接口
4. 添加相应的配置定义
5. 编写单元测试

### 添加新代理协议
1. 在 `proxy/` 下创建新目录
2. 实现 `proxy.Inbound` 和 `proxy.Outbound` 接口
3. 添加配置结构定义
4. 在 `main/distro/` 中注册新协议
5. 编写集成测试

### 调试技巧
- 使用 `go test -v` 查看详细测试输出
- 使用 `delve` 进行断点调试
- 查看日志输出来诊断问题
- 使用 `testing/scenarios` 中的测试场景

## 依赖管理

- Xray-core 使用 Go modules 管理依赖
- 主要依赖包括：
  - `github.com/quic-go/quic-go`: QUIC 协议实现
  - `github.com/gorilla/websocket`: WebSocket 支持
  - `github.com/miekg/dns`: DNS 库
  - `github.com/refraction-networking/utls`: uTLS 支持
  - `github.com/xtls/reality`: REALITY 协议

## 代码规范

- 遵循 Go 语言标准代码风格
- 使用 `gofmt` 格式化代码
- 函数和类型需要添加适当的文档注释
- 错误处理要完整和一致
- 使用接口抽象功能模块

## 注意事项

- Xray-core 是网络代理工具，仅用于防御性安全目的
- 编译时需要设置适当的 Go 版本（参考 go.mod）
- 测试需要访问网络资源（geoip.dat, geosite.dat）
- Windows 和 Linux/macOS 的编译命令略有不同
- 文档网站使用 VuePress 2.x 构建