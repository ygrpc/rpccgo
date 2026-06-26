# rpccgo Runtime Server Registry Architecture

新版 rpccgo 使用 `rpcruntime` 的统一 server registry 保存每个 service 的 current registered server，并使用 `rpcruntime` 的统一 stream session registry 保存 Go runtime-visible active streaming session。Generated server contract artifact 负责对应 contract 的注册 helper、handler stream interface 和 native stream envelope；generated service runtime 负责 registry lookup、typed dispatch、native/message 转换、cgo ABI glue、transport registration glue 和 stream `Start` glue；generated server contract artifact 或 runtime artifact 按 artifact selection 生成 package-level stream operation 函数。`rpcruntime` 只提供通用 server registry、`ServerKind`、Go runtime-visible stream session registry、callback receive ownership state 和 transport/runtime primitive；它不要求托管所有 foreign embedded server runtime 的 server-side stream 对象。

## 核心规则

- 每个 protobuf service 使用一个稳定 `ServiceID`，默认取 protobuf full service name。
- 每个 `ServiceID` 同一时刻只有一个 current registered server。
- Registered server record 使用 `{Kind, Server}`：`Kind` 是 `rpcruntime.ServerKind`，`Server` 是 opaque concrete server。
- `ServerKind` 固定为 Go native、cgo native、cgo message、connect、gRPC、connect remote 和 gRPC remote；zero value 表示未初始化，不可注册。
- `rpcruntime` 不依赖 service-specific protobuf 类型，不执行 native/message 转换，不调用 service method。
- Generated facade 根据调用 contract 与 `ServerKind` 执行 type assertion、direct call 或 Native/Message conversion。
- `rpcruntime` 统一管理 Go runtime-visible stream handle 分配、stream session record 存取和 terminal operation 后的 handle 移除。
- Generated Go runtime code 不生成 service-local stream registry、method-specific final session record，也不生成只封装 `rpcruntime.StreamHandle` 的 stream handle facade。若 Dart/JNI/Flutter 等 foreign embedded server runtime 的 C ABI contract 以本地 `int32 stream handle` 续接 server-side stream 操作，则该 foreign runtime 可以维护本地 `handle -> handler/session` registry。

## 注册

用户通过 generated registration helper 注册 server，不直接手写 `ServiceID` 或 `ServerKind` 调用 runtime primitive。具体 helper 命名统一记录在 `CONTEXT.md` 的 `Naming Rules`。

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

Go runtime-visible stream `Start` 读取 current registered server 一次，根据 `ServerKind` 取得具体 typed client endpoint，并把 `{ServerKind, session}` 存入 `rpcruntime` 的全局 stream session registry。后续 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 只通过 stream handle 找回该 endpoint，不重新读取 server registry。C per-method register 更新只影响后续 unary 调用和后续 stream `Start`，不影响已经开始的 Go runtime-visible stream session。

`rpcruntime` stream session record 不保存 operation closure。Generated package-level stream operation 按 session record 的 `ServerKind` 把 `session` 转回对应 typed client endpoint 并直接调用。终态操作从 `rpcruntime` stream session registry 移除 handle；移除后的 handle 再操作返回 invalid-handle 错误。callback receive stream 允许在 `rpcruntime` 内额外维护 callback ownership 与 terminal delivery state，因为这属于 Go runtime-visible callback contract。foreign embedded server runtime 自己持有的 server-side stream session 不属于这里的 runtime record。

`Finish` 是 graceful terminal，`CloseSend` 是 bidi/client-streaming half-close，`Cancel` 是 abort terminal。具体 method operation、native/message conversion 和 flat ABI 编解码留在 generated service runtime。

### Stream operation semantics

`Finish` 是 graceful terminal，`CloseSend` 是 bidi/client-streaming half-close，`Cancel` 是 abort terminal。具体 method operation、native/message conversion 和 flat ABI 编解码留在 generated service runtime。Generated stream naming rules live only in `CONTEXT.md` under `Naming Rules`.

## Server Types

支持注册的 server 类型：

- Go native server：实现 generated Go native server contract。
- cgo native server：由 C native callbacks 组装成 Go native server contract，支持 method-level partial implementation。
- cgo message server：实现 generated cgo message server contract。
- connect handler：标准 connect-go handler。
- gRPC server：标准 grpc-go server。
- connect remote server：标准 connect-go client。
- gRPC remote server：标准 grpc-go client。

标准 connect/gRPC server 和 remote client 直接注册为 standard transport type。rpccgo 不生成本地 connect/gRPC transport ingress 文件，不生成独立 remote adapter 文件，也不通过 HTTP loopback 处理 C-to-Go 调用。

## Generated Artifacts

Generated artifact naming rules live only in `CONTEXT.md` under `Naming Rules`.

Go native server contracts 位于 protobuf Go package 的 generated native server artifact，包含 native server contract、flat native handler stream interfaces、method-local native stream envelope、native package-level stream operation functions 和 unimplemented helper。Native 与 message active dispatch 共用 `rpcruntime.ClientStreamingClient[Req, Resp]`、`rpcruntime.ServerStreamingClient[Resp]` 和 `rpcruntime.BidiStreamingClient[Req, Resp]`；handler 侧对应使用 `ClientStreamingServer`、`ServerStreamingServer` 和 `BidiStreamingServer`。本地 Go server 调用由 reusable generic client/server endpoint structs 和 private shared state 管理 queue、acknowledgement、cancellation、close-send 和 completion；generated code 不生成 per-method state、session 或 thin facade。Go native 只生成 flat native fields 与 server endpoint envelope 之间的 mapper，C message server 则直接使用 typed protobuf message endpoint。

## 不生成的结构

Generated Go runtime code 不应生成 native/message active binding slot、service-wide binding closure table、per-contract current server、runtime forwarding struct、remote adapter file、operation closure session、generic dispatcher、service-local stream registry、method-specific stream state/session/facade 或只封装 handle 的 stream facade struct。foreign embedded server runtime 若其 server-side C ABI contract 通过本地 `int32 stream handle` 续接操作，则可以生成本地 stream registry 与对应 handle-owned facade。

`rpcruntime` 不应引入 `ActiveServerSlot`、`ServerContract`、`AdapterSnapshot`、stream executor、generic dispatcher 或 registry lifecycle helper layer。`rpcruntime` 可以拥有通用 stream session record，因为该 record 只保存 `ServerKind` 与 opaque session，不执行 service-specific dispatch。
