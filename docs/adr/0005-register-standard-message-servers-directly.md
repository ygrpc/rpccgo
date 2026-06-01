# 直接注册标准 message server

rpccgo 的注册 API 直接接收用户提供的 connect-go handler 和 grpc-go server，不生成本地 connect/grpc transport ingress 文件，也不生成独立 remote adapter 文件。Generated runtime 在注册阶段把这些标准 server 归一化为 service-local active server record，并预先组装 caller-facing method closure。调用阶段只读取 record snapshot 并调用 closure，不再按 server kind 或 contract 分支。Generated service 直接使用 typed atomic pointer 保存 record；`rpcruntime` 只保留 stream registry、lifecycle state，以及 connect-go 未公开 stream constructor 时所需的窄 unsafe shim。

## Consequences

- 本地 server ingress 不再生成 `*.server.connect.rpccgo.go` 和 `*.server.grpc.rpccgo.go`。
- `Register<Service>ConnectHandler` 和 `Register<Service>GRPCServer` 位于 generated service runtime，返回 `error`；内部 typed atomic active slot 保存 service-local active server record，而不是原始 concrete server。
- service-local active server record 是 generated runtime 内部的 package-private 调用表，不是旧 Provider bootstrap，不生成独立 adapter 文件，也不把 native/message conversion 下沉到 `rpcruntime`。
- 注册入口必须先校验完整 service 能力并构造不可变 active server record，最后才原子替换 active slot；不能发布 partial record。
- connect streaming 直接调用刻意依赖 connect-go 的未导出 stream 布局；unsafe 构造集中在 `rpcruntime`，并通过测试和版本兼容边界保护，而不是散落在 generated code 中。
- C-to-Go 调用拒绝 HTTP loopback，因为那会让 cgo bridge 支付 transport 开销，而不是直接调用 active server。
