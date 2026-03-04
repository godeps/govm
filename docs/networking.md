# Govm 网络方案说明

更新时间：2026-03-04

## 1. 目标与范围

本方案面向 `govm + boxlite` 运行栈，目标是：

- 提供统一、可跨平台编译的 Go 网络 API。
- 默认安全（可选严格模式），并保持与现有 `CreateBox` 兼容。
- 在不修改上游 `boxlite` 仓库代码的前提下完成兼容与扩展。

本文件描述的是“控制面 + 当前可落地的数据面能力”。

## 2. 架构概览

数据路径与控制路径分离：

1. 控制面（Go）
- `pkg/client` 暴露网络 API。
- 负责默认策略、参数校验、runtime 默认与 box 级覆盖合并。

2. 桥接层（Go binding + Rust bridge）
- `internal/binding` 将网络配置序列化为 JSON。
- `rust-bridge` 将 JSON 映射到 `boxlite::runtime::options::BoxOptions`。

3. 运行时（boxlite）
- `libkrun`：负责 microVM 执行。
- `gvproxy`：负责 VM NAT、端口转发等网络代理能力。

## 3. API 设计

### 3.1 Runtime 级默认网络

通过 `RuntimeOptions.NetworkDefaults` 配置：

- `Profile: strict|balanced|open`
- 或直接给 `Config *NetworkConfig`

### 3.2 Box 级网络覆盖

通过 `BoxOptions.Network *NetworkConfig` 覆盖 Runtime 默认配置。

### 3.3 核心类型

- `NetworkMode`: `disabled | nat | bridged`
- `PolicyMode`: `block_all | allow_all`
- `Protocol`: `tcp | udp | any`
- `PortForward`: host -> guest 端口映射
- `NetworkPolicy`: 规则意图（CIDR/Domain/DNS/Proxy/Limits）

说明：当前后端已稳定映射的是 `mode + port_forwards + macOS network_enabled`。

## 4. 默认策略与 Profile

- `strict`（建议默认）
  - `Enabled=true`
  - `Mode=nat`
  - `Policy=block_all`
- `balanced`
  - `Enabled=true`
  - `Mode=nat`
  - `Policy=allow_all`（后续可收敛为常见端口白名单）
- `open`
  - `Enabled=true`
  - `Mode=nat`
  - `Policy=allow_all`

## 5. 平台语义

### 5.1 Linux

当前可用：
- `nat`
- TCP/UDP 端口映射

说明：
- 细粒度 CIDR/Domain 等策略字段当前属于“策略意图层”，未在 runtime 内核态强制执行。

### 5.2 macOS

当前可用：
- `nat`
- 端口映射
- 通过 `advanced.security.network_enabled` 控制 sandbox 网络开关（由 govm 映射）

### 5.3 Windows

当前状态：
- API 可编译（stub/native 受限）
- 原生网络能力不完整，按 `unsupported` 语义返回

## 6. 当前实现状态（已完成）

已实现并验证：

- Go API：`pkg/client/network.go`
- 字段接入：
  - `RuntimeOptions.NetworkDefaults`
  - `BoxOptions.Network`
- 配置处理：
  - 默认 profile
  - 覆盖合并
  - 参数校验
- bridge 映射：
  - `network_mode`
  - `port_forwards`
  - `macos_network_enabled`
- 文档与示例：
  - `README.md`
  - `examples/all-api/main.go`
- 测试：
  - `pkg/client/network_test.go`
  - `go test ./...` 通过
  - `make test-native` 通过

## 7. 兼容与限制

1. 向后兼容
- 未设置 `Network` 时保持现有行为，不破坏旧调用。

2. `bridged` 模式
- 当前后端不支持，显式返回 unsupported 错误。

3. 策略字段分层
- `AllowCIDR/DenyCIDR/AllowDomain/DenyDomain/DNS/Proxy/Limits` 已在 API 层定义。
- 当前版本主要用于表达策略意图；强制执行能力待后续 Host Policy Engine。

## 8. 示例

```go
rt, err := client.NewRuntime(&client.RuntimeOptions{
    NetworkDefaults: &client.RuntimeNetworkDefaults{Profile: "strict"},
})
if err != nil { panic(err) }
defer rt.Close()

box, err := rt.CreateBox(context.Background(), "net-demo", client.BoxOptions{
    OfflineImage: "py312-alpine",
    Network: &client.NetworkConfig{
        Enabled: true,
        Mode:    client.NetworkNAT,
        Policy:  &client.NetworkPolicy{Mode: client.PolicyBlockAll},
        PortForwards: []client.PortForward{
            {HostIP: "127.0.0.1", HostPort: 18080, GuestPort: 8080, Protocol: client.ProtoTCP},
        },
    },
})
if err != nil { panic(err) }
_ = box
```

## 9. 后续优化路线

M2（建议）
- 实现 Linux host-side policy enforcement（如 nftables/ipset + DNS/proxy 协同）。
- 为“策略意图层”增加可观测状态与执行结果反馈。

M3
- 评估更强的跨平台策略一致性（Linux/macOS/Windows 降级矩阵）。
- 补充网络统计与审计 API（按 box 维度输出）。

## 10. 相关文档

- 设计稿：`docs/plans/2026-03-04-network-api-design.md`
- 实施计划：`docs/plans/2026-03-04-network-api-implementation-plan.md`
- 上游兼容修复记录：`docs/boxlite-ffi-box-stop-fix.md`
