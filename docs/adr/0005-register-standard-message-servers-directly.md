# 直接注册标准 message server

rpccgo 将用户提供的 connect-go handler 和 grpc-go server 直接注册为 message-contract active server，不再生成本地 connect/grpc transport ingress 文件，也不再把它们包装进 service-specific active adapter struct。Generated runtime 负责从 C/native/message 调用到 active server slot 的 service-specific bridge；`rpcruntime` 只保留 active slot、stream registry、lifecycle state，以及 connect-go 未公开 stream constructor 时所需的窄 unsafe shim。

## Consequences

- 本地 server ingress 不再生成 `*.server.connect.rpccgo.go` 和 `*.server.grpc.rpccgo.go`。
- `Register<Service>ConnectHandler` 和 `Register<Service>GRPCServer` 位于 generated service runtime，返回 typed `AdapterSnapshot`，内部 active slot 可以用 `any` 保存 concrete server。
- connect streaming 直接调用刻意依赖 connect-go 的未导出 stream 布局；unsafe 构造集中在 `rpcruntime`，并通过测试和版本兼容边界保护，而不是散落在 generated code 中。
- C-to-Go 调用拒绝 HTTP loopback，因为那会让 cgo bridge 支付 transport 开销，而不是直接调用 active server。
