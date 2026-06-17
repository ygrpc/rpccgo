# rpccgo Runtime Server Registry Architecture

新版 rpccgo 使用 `rpcruntime` 的统一 server registry 保存每个 service 的 current registered server，并使用 `rpcruntime` 的统一 stream session registry 保存 active streaming session。Generated server contract artifact 负责对应 contract 的注册 helper、handler stream interface 和 native stream envelope；generated service runtime 负责 registry lookup、typed dispatch、native/message 转换、cgo ABI glue、transport registration glue 和 stream `Start` glue；generated server contract artifact 或 runtime artifact 按 artifact selection 生成 package-level stream operation 函数。`rpcruntime` 只提供通用 server registry、`ServerKind`、stream session registry 和 transport/runtime primitive。

## 核心规则

- 每个 protobuf service 使用一个稳定 `ServiceID`，默认取 protobuf full service name。
- 每个 `ServiceID` 同一时刻只有一个 current registered server。
- Registered server record 使用 `{Kind, Server}`：`Kind` 是 `rpcruntime.ServerKind`，`Server` 是 opaque concrete server。
- `ServerKind` 固定为 Go native、cgo native、cgo message、connect、gRPC、connect remote 和 gRPC remote；zero value 表示未初始化，不可注册。
- `rpcruntime` 不依赖 service-specific protobuf 类型，不执行 native/message 转换，不调用 service method。
- Generated facade 根据调用 contract 与 `ServerKind` 执行 type assertion、direct call 或 Native/Message conversion。
- `rpcruntime` 统一管理 stream handle 分配、stream session record 存取和 terminal operation 后的 handle 移除。
- Generated code 不生成 service-local stream registry、method-specific final session record，也不生成只封装 `rpcruntime.StreamHandle` 的 stream handle facade。

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

非 C callback generated helper 完成 nil check 和 service contract validation 后，把 `{Kind, Server}` 写入 `rpcruntime` server registry；这类 service-level registration helper 失败时清空该 `ServiceID` 当前 registered server 并返回错误，后续调用应返回 `rpcruntime.ErrNoRegisteredServer`，而不是继续使用旧 server。

C native/message callback registration 支持 service-level 和 per-method 两种入口。未注册 method 调用时返回 generated unimplemented error，全部 method 都未注册时仍可注册为全 unimplemented server。每个 method 内部原子校验：unary callback nil 表示该 method 未实现，streaming method 的 operation callbacks 必须全 nil 或全非 nil。C service-level register 和 per-method register 都在 current server 为同一 `ServerKind` 时累积到现有 cgo adapter；current server 为空或不是同一 `ServerKind` 时创建新的 cgo adapter 并替换 current server。任一 method 校验失败时只清空该 method callbacks，使该 method 回到 unimplemented 状态，不清空 current server 或其他 method callbacks；如果 current server 不是同一 `ServerKind`，仍创建并注册新的同 kind cgo adapter，再把失败 method 保持为 unimplemented。C callback registration 返回错误报告被拒绝的 method，但保留其他已校验通过的 callback 更新。

Generated service runtime 也应暴露 service-specific clear helper 和 load helper。clear helper 内部调用 runtime clear primitive；load helper 内部调用 runtime load primitive，用于 generated cgo register 读取并累积当前同 kind cgo adapter。clear 只影响后续 unary 调用和后续 stream `Start`，不影响已经开始的 stream session。

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

注册新 server 会影响后续 unary 调用。没有 current registered server 时返回 `rpcruntime.ErrNoRegisteredServer`。

## Streaming

Stream `Start` 读取 current registered server 一次，根据 `ServerKind` 取得具体 typed client endpoint，并把 `{ServerKind, session}` 存入 `rpcruntime` 的全局 stream session registry。后续 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 只通过 stream handle 找回该 endpoint，不重新读取 server registry。C per-method register 更新只影响后续 unary 调用和后续 stream `Start`，不影响已经开始的 stream session。

Stream session record 不保存 operation closure，也不维护 registry-level lifecycle state machine。Generated package-level stream operation 按 session record 的 `ServerKind` 把 `session` 转回对应 typed client endpoint 并直接调用。终态操作从 `rpcruntime` stream session registry 移除 handle；移除后的 handle 再操作返回 invalid-handle 错误。

`Finish` 是 graceful terminal，`CloseSend` 是 bidi/client-streaming half-close，`Cancel` 是 abort terminal。具体 method operation、native/message conversion 和 flat ABI 编解码留在 generated service runtime。

### Stream operation semantics and naming

Generated stream code uses one canonical operation vocabulary across runtime, Go, C ABI, Dart, Kotlin, JNI C++ and examples:

- `Start` creates a stream session, captures the current registered server, and returns a stream handle or host-language stream object.
- `Send` sends one request payload on a client-streaming or bidi stream.
- `Recv` receives one response payload on a server-streaming or bidi stream.
- `Finish` performs graceful terminal completion and releases the stream handle where that layer owns a handle.
- `CloseSend` half-closes the send side on client-streaming or bidi streams.
- `Cancel` aborts the stream and releases the stream handle where that layer owns a handle.

Generated stream operation names must preserve these exact operation tokens when the operation is user-visible, ABI-visible, or used as a cross-language bridge symbol. `Recv` is the canonical receive operation because gRPC-Go exposes receive as `RecvMsg` and generated gRPC stream clients commonly expose `Recv`; Connect-Go exposes the same direction as `Receive`. Do not use synonyms such as `Read` for `Recv`, and do not hide `Start` behind a method-only name such as `CollectRuntimeState` when the call creates a stream session.

Every generated layer uses one global operation position rule: when a generated stream symbol name contains both the protobuf method name and the stream operation, the operation is always the suffix after the method name. The symbol may include contract, namespace, service, language-runtime, or binding qualifiers, but those qualifiers must not move the operation before the method.

- C exports: `rpccgo<Contract><Namespace><Service><Method><Operation>`, for example `rpccgoMsgDemoGreeterChatRecv`.
- Go package-level stream operation functions: `<Service><Contract><Method><Operation>`, for example `GreeterMessageChatRecv`.
- Go server contract stream objects and host-language stream objects: operation methods are exactly `Send`, `Recv`, `Finish`, `CloseSend` and `Cancel`.
- Dart and Kotlin public stream entry methods: `<Method>Start`, for example `CollectRuntimeStateStart`; the returned stream object exposes `Send`, `Recv`, `Finish`, `CloseSend` and `Cancel`.
- JNI C++ bridge helpers follow the Java/Kotlin native method shape but keep the canonical operation as the final method segment, for example `sharedSoDemoCollectRuntimeStateRecv`.
- Private/raw bindings may use the host language's casing convention, but they must keep the same operation token and the same operation position as their layer, for example `_collectRuntimeStateRecvRaw` instead of `_collectRuntimeStateReadRaw`.

Generated code must not mix operation-first and method-first naming across layers or within a layer. This avoids inconsistent pairs such as `collectRuntimeStateStart` and `startCollectRuntimeState`; only the method-first, operation-suffix form is valid for combined method-operation symbols.

## Server Types

支持注册的 server 类型：

- Go native server：实现 generated `<Service>NativeServer`。
- cgo native server：由 C native callbacks 组装成 `<Service>NativeServer`，支持 method-level partial implementation。
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

Go native server contracts 位于 protobuf Go package 的 generated native server file，包含 `<Service>NativeServer`、flat native handler stream interfaces、method-local native stream envelope、native package-level stream operation functions 和 `Unimplemented<Service>NativeServer` helper。Native 与 message active dispatch 共用 `rpcruntime.ClientStreamingClient[Req, Resp]`、`rpcruntime.ServerStreamingClient[Resp]` 和 `rpcruntime.BidiStreamingClient[Req, Resp]`；handler 侧对应使用 `ClientStreamingServer`、`ServerStreamingServer` 和 `BidiStreamingServer`。本地 Go server 调用由 reusable generic client/server endpoint structs 和 private shared state 管理 queue、acknowledgement、cancellation、close-send 和 completion；generated code 不生成 per-method state、session 或 thin facade。Go native 只生成 flat native fields 与 server endpoint envelope 之间的 mapper，C message server 则直接使用 typed protobuf message endpoint。

Package-level stream operation functions use the operation, contract, service and method in the function name, and accept `rpcruntime.StreamHandle` directly, for example:

```go
SendGreeterMessageCollect(ctx, handle, req)
FinishGreeterMessageCollect(ctx, handle)
RecvGreeterNativeBroadcast(ctx, handle)
CloseSendGreeterNativeChat(ctx, handle)
```

## 不生成的结构

Generated code 不应生成 native/message active binding slot、service-wide binding closure table、per-contract current server、runtime forwarding struct、remote adapter file、operation closure session、generic dispatcher、service-local stream registry、method-specific stream state/session/facade 或只封装 handle 的 stream facade struct。

`rpcruntime` 不应引入 `ActiveServerSlot`、`ServerContract`、`AdapterSnapshot`、stream executor、generic dispatcher 或 registry lifecycle helper layer。`rpcruntime` 可以拥有通用 stream session record，因为该 record 只保存 `ServerKind` 与 opaque session，不执行 service-specific dispatch。
