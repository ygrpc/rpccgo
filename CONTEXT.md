# rpccgo Context

rpccgo 生成跨 Go/C 边界的 RPC glue code。本上下文记录项目特有的 contract 术语，避免把历史 native 语义与重构中的中间表示混淆。

## Language

**Native**:
旧项目定义的字段级函数边界 contract：proto request/response 顶层字段直接成为最终 Go/C 函数参数和返回值。
_Avoid_: struct native ABI, message-shaped native adapter

**Message contract**:
以完整 protobuf request/response message 或其序列化 bytes 作为调用边界的 contract。
_Avoid_: native

**Active server**:
新版架构中由 dispatcher 捕获并路由到的唯一服务实现。
_Avoid_: provider bootstrap

**Runtime core**:
手写的 `rpcruntime` 包，承载跨 service 复用的 active slot、dispatcher 和 stream registry 等通用机制。
_Avoid_: generated service runtime

**Generated service runtime**:
每个 service 生成的 `*.runtime.rpccgo.go`，只应承载 proto/service/method-specific 的 typed adapter、bridge 和 converter glue。
_Avoid_: runtime core

**Provider bootstrap**:
旧项目通过 provider/registry/bootstrap 组装服务能力的架构模型；新版不回迁该模型。
_Avoid_: active server

## Relationships

- **Native** 与 **Message contract** 是不同 contract；**Native** 不应退化成 request/response struct 或 message 指针边界。
- **Native** 的字段级函数边界必须覆盖 Go server interface、Go native client API、C callback ABI，以及 streaming 的 start/send/recv/finish/close/cancel 相关边界。
- C 侧 **Native** callback 必须使用字段级参数列表，例如 `field_ptr/field_len/ownership` 和输出字段指针参数；不能接收 generated `Request*` / `Response*` struct。
- Go **Native** server 输入字段类型沿用旧 wrapper：`string -> *rpcruntime.RpcString`、`bytes/message -> *rpcruntime.RpcBytes`、`repeated scalar -> *rpcruntime.RpcRepeat[T]`、`repeated bool -> *rpcruntime.RpcBoolRepeat`。
- Go **Native** server 返回值沿用旧 flat 返回：response 顶层字段按 Go 值/slice 顺序返回，最后一个返回值固定是 `error`。
- **Native** 只拍平 proto request/response 的顶层字段；nested message 作为整体 message bytes/wrapper 传递，不递归展开。
- `NativeContract` 这类字段计划可以作为参数转换的中间表示保留；它不是最终 **Native** 边界。
- **Active server** 是新版调度模型的一部分；它不能改变 **Native** 的字段级函数边界语义。
- **Runtime core** 负责通用调度和 stream 存储；**Generated service runtime** 负责 service-specific typed glue，不应重复生成可由 runtime core 泛型函数直接表达的薄包装。
- **Generated service runtime** 不应生成 per-method stream `load/take/delete` 薄包装；应在 generated bridge/client/server 代码中直接调用 **Runtime core** 的泛型 stream registry 函数。
- Register helper 可留在 **Generated service runtime** 中，因为它们封装 service-specific active adapter 包装并返回更窄的 typed snapshot，不是纯 runtime core 薄包装。
- Native/message client bridge 应留在 **Generated service runtime** 中，因为它表达 service-level active server contract 路由，并集中连接 native adapter、message adapter 与 converter glue。
- 每个 service 的 dispatcher 应保留为 generated package-level 变量；不要引入 runtime core 全局 service registry，以避免回到旧 **Provider bootstrap** 模型。
- 新版架构保留 dispatcher / active server；只恢复旧项目的 **Native** flat function boundary，不回迁旧 **Provider bootstrap**。
- `@rpccgo:native` 的新版 adapter selection 规则保留；它可以同时启用默认 message adapter，但 **Native** 侧仍必须是 flat function boundary。
- 旧 `go_role=go_client` / C provider 注册 Go client 能力不恢复；它属于旧 **Provider bootstrap** 架构，不是新版 **Native** 修复范围。

## Example dialogue

> **Dev:** “这个 `native` callback 能不能接收一个 generated input struct？”
> **Domain expert:** “不能。**Native** 的验收标准是 flat function boundary，request/response 顶层字段必须直接出现在最终函数边界。”

## Flagged ambiguities

- “struct native ABI” 曾被用来描述当前重构中的 generated input/output struct 边界；已决议：这不是 **Native**，应视为错误实现而非 native 的一种形态。
