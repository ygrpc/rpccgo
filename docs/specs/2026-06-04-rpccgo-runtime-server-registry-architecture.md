# rpccgo Runtime Server Registry Architecture

新版 rpccgo 使用 `rpcruntime` 的统一 server registry 保存每个 service 的 current registered server。Generated server contract artifact 负责对应 contract 的注册 helper、source session interface、stream handle facade 和 final session record；generated service runtime 负责 registry lookup、typed dispatch、native/message 转换、cgo ABI glue、transport registration glue 和 stream `Start` glue；`rpcruntime` 只提供通用 registry、`ServerKind`、stream registry 和 transport/runtime primitive。

## 核心规则

- 每个 protobuf service 使用一个稳定 `ServiceID`，默认取 protobuf full service name。
- 每个 `ServiceID` 同一时刻只有一个 current registered server。
- Registered server record 使用 `{Kind, Server}`：`Kind` 是 `rpcruntime.ServerKind`，`Server` 是 opaque concrete server。
- `ServerKind` 固定为 Go native、cgo native、cgo message、connect、gRPC、connect remote 和 gRPC remote；zero value 表示未初始化，不可注册。
- `rpcruntime` 不依赖 service-specific protobuf 类型，不执行 native/message 转换，不调用 service method。
- Generated facade 根据调用 contract 与 `ServerKind` 执行 type assertion、direct call 或 Native/Message conversion。

## 注册

用户通过 generated registration helper 注册 server，不直接手写 `ServiceID` 或 `ServerKind` 调用 runtime primitive。

```go
greeterv1.RegisterGreeterGoNativeServer(server)
greeterv1.RegisterGreeterCGOMessageServer(server)
greeterv1.RegisterGreeterConnectHandler(handler)
greeterv1.RegisterGreeterGRPCServer(server)
greeterv1.RegisterGreeterConnectRemoteServer(client)
greeterv1.RegisterGreeterGRPCRemoteServer(client)
```

Generated helper 完成 nil check、service contract validation 和 callback set validation 后，把 `{Kind, Server}` 写入 `rpcruntime` server registry。注册失败时 helper 清空该 `ServiceID` 当前 registered server 并返回错误；后续调用应返回 no-active-server 错误，而不是继续使用旧 server。

Generated service runtime 也应暴露 service-specific clear helper，内部调用 runtime clear primitive。clear 只影响后续 unary 调用和后续 stream `Start`，不影响已经开始的 stream session。

## 调用

Unary 调用每次从 `rpcruntime` server registry 读取 current registered server：

```text
cgo native unary call
  -> generated native facade
  -> load {Kind, Server} by ServiceID
  -> switch Kind
  -> direct native call or native->message conversion
```

```text
cgo message unary call
  -> generated message facade
  -> load {Kind, Server} by ServiceID
  -> switch Kind
  -> direct message call or message->native conversion
```

注册新 server 会影响后续 unary 调用。没有 current registered server 时返回 `rpcruntime.ErrNoActiveServer`。

## Streaming

Stream `Start` 读取 current registered server 一次，根据 `ServerKind` 创建具体 typed stream session，并把 `{ServerKind, session}` 存入 service-local stream registry。后续 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 只通过 stream handle 找回该 session，不重新读取 server registry。

Stream session 不保存 operation closure，也不维护通用 lifecycle state machine。Generated stream operation 按 session record 的 `ServerKind` 把 `session` 转回对应 typed session 并直接调用。终态操作从 service-local stream registry 移除 handle；移除后的 handle 再操作返回 invalid-handle 错误。

`Finish` 是 graceful terminal，`CloseSend` 是 bidi/client-streaming half-close，`Cancel` 是 abort terminal。具体 method operation、native/message conversion 和 flat ABI 编解码留在 generated service runtime。

## Server Types

支持注册的 server 类型：

- Go native server：实现 generated `<Service>NativeServer`。
- cgo native server：由完整 C native callback set 组装成 `<Service>NativeServer`。
- cgo message server：实现 generated `<Service>CGOMessageServer`。
- connect handler：标准 connect-go handler。
- gRPC server：标准 grpc-go server。
- connect remote server：标准 connect-go client。
- gRPC remote server：标准 grpc-go client。

标准 connect/gRPC server 和 remote client 直接注册为 standard transport type。rpccgo 不生成本地 connect/gRPC transport ingress 文件，不生成独立 remote adapter 文件，也不通过 HTTP loopback 处理 C-to-Go 调用。

## Generated Artifacts

生成文件按 service 拆分为 `<proto-prefix>.<service>.<role>[.<contract|transport>].rpccgo.go`。cgo 文件输出到 `cgo_dir`，使用 `package main`，并在文件名中显式写出 native/message contract token：

- `.server.native.cgo.rpccgo.go`
- `.client.native.cgo.rpccgo.go`
- `.server.message.cgo.rpccgo.go`
- `.client.message.cgo.rpccgo.go`

Shared cgo exports 按 cgo Go package 只生成一次，文件名固定为 `rpccgo.exports.cgo.rpccgo.go`。

Go native server contracts 位于 protobuf Go package 的 generated native server file，包含 `<Service>NativeServer`、native stream interfaces 和 `Unimplemented<Service>NativeServer` helper。C message server contracts 位于 generated message server file，包含 `<Service>CGOMessageServer`、message stream interfaces 和 `Unimplemented<Service>CGOMessageServer` helper。

## 不生成的结构

Generated service runtime 不应生成 native/message active binding slot、service-wide binding closure table、per-contract current server、runtime forwarding struct、remote adapter file、operation closure session 或 generic dispatcher。

`rpcruntime` 不应引入 `ActiveServerSlot`、`ServerContract`、`AdapterSnapshot`、`StreamEntry`、stream executor 或 registry lifecycle helper。
