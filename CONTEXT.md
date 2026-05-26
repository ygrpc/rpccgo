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

**Stream lifecycle**:
一次 streaming call 从 Start 到终态操作期间的 operation set、handle ownership、session lookup、half-close、finish/done/cancel 和 invalid-handle 语义；contract-level operation plan 与 runtime state machine 必须区分。
_Avoid_: stream registry access pattern

**Stream lifecycle projection**:
generator 侧的 contract-to-render 投影，把 **Stream lifecycle** 的 operation set、terminal policy 和 codec requirement 转换为 **Generated service runtime** 可消费的 typed facade / executor binding plan；不执行 runtime state machine，也不拥有 handle storage。
_Avoid_: runtime stream lifecycle executor, stream registry access pattern

**Generated service runtime**:
每个 service 生成的 `*.runtime.rpccgo.go`，只应承载 proto/service/method-specific 的 typed adapter、bridge 和 converter glue。
_Avoid_: runtime core

**Runtime bridge**:
Generated service runtime 内部的 package-private typed invocation layer，负责按 service/method contract 选择 Active server 并应用 native/message 转换；外部包只能通过 generated package-level invoke/start 函数进入。
_Avoid_: runtime core dispatcher, public client object, active router

**Call-scoped borrowed view**:
仅在一次 generated message→native bridge 调用同步执行期间有效的 borrowed wrapper 视图；不得把 wrapper 本身跨调用保存。
_Avoid_: owned wrapper, long-lived native input

**Native C ABI plan**:
由 **Native** 的 Go-level flat contract 派生出的跨 Go/C ABI shape；按 operation 表达 C signature、field lowering、ownership、cleanup、callback/export 和 error bridge 需求。
_Avoid_: NativeContract, Go native server/client API

**Method contract plan**:
单个 method 从 descriptor 派生出的完整 contract-level planning 结果；集中表达 **Native**、**Message contract**、**Stream lifecycle** operation plan、render planning inputs 和 **Native C ABI plan** 的来源关系，但不生成代码字符串。
_Avoid_: renderer, generated code, runtime state machine

**Native projection**:
从同一个 **Native** contract 派生出的具体语言/边界形态；Go native projection 表达 Go 函数参数和返回值，C native projection 表达跨 Go/C 的 ABI slot。
_Avoid_: separate native contract, incompatible Go/C native ABI

**Provider bootstrap**:
旧项目通过 provider/registry/bootstrap 组装服务能力的架构模型；新版不回迁该模型。
_Avoid_: active server

## Relationships

- **Native** 与 **Message contract** 是不同 contract；**Native** 不应退化成 request/response struct 或 message 指针边界。
- **Native** 的字段级函数边界必须覆盖 Go server interface、Go native client API、C callback ABI，以及 streaming 的 start/send/recv/finish/close/cancel 相关边界。
- Go native 与 C native 是同一个 **Native** contract 的不同 **Native projection**；它们不应被建模为两套独立 native contract。
- **Native C ABI plan** 必须从 **Native** / `NativeContractPlan` 派生，不能重新解释 proto descriptor 或形成独立 contract。
- C 侧 **Native** callback 必须使用字段级参数列表，例如 `field_ptr/field_len/ownership` 和输出字段指针参数；不能接收 generated `Request*` / `Response*` struct。
- 跨 runtime 的 C **Native** ABI 不能以 `struct` 或 `struct*` 作为调用边界参数；callback table 也必须拆成 flat callback 参数或逐项注册。
- C **Native** server callback 允许按 method 分开注册；首次注册任一 callback 即可把该 service 激活为 **Active server**，未注册的 method 在真正调用时返回 callback-missing 错误。
- C 导出符号命名以 `<contract> + <namespace> + <service> + <method> + <operation>` 组成；`namespace` 默认取 Go package name，冲突时由用户显式覆盖，不使用调用端/实现端语言前缀区分方向。
- Go **Native** server 输入字段类型沿用旧 wrapper：`string -> *rpcruntime.RpcString`、`bytes/message -> *rpcruntime.RpcBytes`、`repeated scalar -> *rpcruntime.RpcRepeat[T]`、`repeated bool -> *rpcruntime.RpcBoolRepeat`。
- 由 **Message contract** 适配到 **Native** 时，请求侧 wrapper 只应作为 **Call-scoped borrowed view** 存在；其底层数据只保证在该次 generated bridge 同步 native 调用期间有效。
- Go **Native** server 返回值沿用旧 flat 返回：response 顶层字段按 Go 值/slice 顺序返回，最后一个返回值固定是 `error`。
- **Native** 只拍平 proto request/response 的顶层字段；nested message 作为整体 message bytes/wrapper 传递，不递归展开。
- `NativeContract` 这类字段计划可以作为参数转换的中间表示保留；它不是最终 **Native** 边界。
- **Native C ABI plan** 可把 ownership / cleanup / transfer 作为生成计划表达；它不应新增现有 ABI 之外的 ownership 参数，但若现有 C boundary 已包含 ownership slot，plan 应把它作为 ABI slot 结构化表达。
- **Native C ABI plan** 位于 `NativeContract` 之后、renderer 之前；它按 C boundary operation 表达结构化 ABI shape，不生成代码字符串。
- **Method contract plan** 应由 `MethodPlan` 显式持有，集中保存单个 method 的 **Native**、**Message contract**、**Stream lifecycle**、render planning inputs 和 method-level **Native C ABI plan**，避免 renderer 或后续 builder 从 `RenderShape` 反读 contract facts。
- **Native C ABI plan** 应保留 slot role、source field metadata、最终 C type spelling、cleanup capability、export symbol naming 和 callback typedef naming，使 renderer 不再重复推断 ABI 语义。
- **Native C ABI plan** 不表达 callback missing policy、error bridge lifecycle 语义或 **Stream lifecycle** handle cleanup；这些分别属于 **Generated service runtime** / **Runtime bridge**、error bridge Module 和 **Runtime core**。
- protobuf schema 中的 unsigned 字段可进入 **Native C ABI plan** 的 field value slot；proto 无关的 length/count/handle/error id 等辅助 slot 不应使用 unsigned 32/64 类型。
- 修改 ABI / runtime type mapping 后，必须使用 `docs/release/verification-checklist.md` 验证测试命令和合同扫描。
- **Active server** 是新版调度模型的一部分；它不能改变 **Native** 的字段级函数边界语义。
- **Runtime core** 负责通用调度和 stream 存储；**Generated service runtime** 负责 service-specific typed glue，不应重复生成可由 runtime core 泛型函数直接表达的薄包装。
- **Stream lifecycle** 的 ownership、terminal-once 和 invalid-handle 通用语义属于 **Runtime core**；method-specific session 操作、native/message 转换和 flat ABI 编解码属于 **Generated service runtime**。
- **Generated service runtime** 不应生成 per-method stream `load/take/delete` 薄包装；应通过 **Stream lifecycle** Module 表达 Start 后的 lookup、half-close、finish/done/cancel 和终态释放规则。
- **Runtime core** 应提供 **Stream lifecycle** executor，集中执行通用 handle lookup/take/release、terminal-once、invalid-handle、send-closed/finalized/canceled 和 cancel/terminal finalization 语义；**Generated service runtime** 应只提供 method-specific typed facade，绑定 session callback、native/message conversion、active routing 和错误映射。
- Register helper 可留在 **Generated service runtime** 中，因为它们封装 service-specific active adapter 包装并返回更窄的 typed snapshot，不是纯 runtime core 薄包装。
- **Runtime bridge** 应留在 **Generated service runtime** 中，因为它表达 service-level active server contract 路由，并集中连接 native adapter、message adapter 与 converter glue。
- **Runtime bridge** 作为 service-local invocation layer 复用 **Runtime core** dispatcher 的 capture/start primitive；不应再额外生成只转发到 bridge 的 public client object。
- **Runtime bridge** 类型和方法不作为外部用户 API；外部包通过 generated package-level invoke/start 函数进入。它返回的 routing errors 应导出为 package-level sentinel vars，供用户通过 `errors.Is` 判断失败类型；无 active server 使用 `rpcruntime.ErrNoActiveServer`，service-specific 失败按 service + 分类命名，不按调用方向拆分。
- **Message contract** remote adapter 使用标准 transport client 作为外部能力；rpccgo generated code 不应构造 per-method client。
- **Message contract** remote adapter 只转发 protobuf message payload 和 error；metadata/header/trailer 不属于当前 contract。
- 一个 service 的 generated output 只能选择一个 message transport（connect 或 gRPC），避免标准 transport client API 在同包内重名。
- 每个 service 的 dispatcher 应保留为 generated package-level 变量；不要引入 runtime core 全局 service registry，以避免回到旧 **Provider bootstrap** 模型。
- 新版架构保留 dispatcher / active server；只恢复旧项目的 **Native** flat function boundary，不回迁旧 **Provider bootstrap**。
- `@rpccgo:native` 的新版 adapter selection 规则保留；它可以同时启用默认 message adapter，但 **Native** 侧仍必须是 flat function boundary。
- 旧 `go_role=go_client` / C provider 注册 Go client 能力不恢复；它属于旧 **Provider bootstrap** 架构，不是新版 **Native** 修复范围。

## Example dialogue

> **Dev:** “这个 `native` callback 能不能接收一个 generated input struct？”
> **Domain expert:** “不能。**Native** 的验收标准是 flat function boundary，request/response 顶层字段必须直接出现在最终函数边界。”

## Flagged ambiguities

- “struct native ABI” 曾被用来描述当前重构中的 generated input/output struct 边界；已决议：这不是 **Native**，应视为错误实现而非 native 的一种形态。
