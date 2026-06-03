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
新版架构中由 generated service runtime 的 typed atomic active slot 捕获并调用的唯一服务实现；slot 保存不可变 active server record，该 record 同时持有 native caller binding 与 message caller binding。
_Avoid_: provider bootstrap

**Binding**:
Generated service runtime 内部的 package-private service-local contract-specific 调用闭包集合；在注册阶段由完整且校验通过的具体 server 归一化而来。native caller binding 只持有 native caller-facing invoke/start closure，message caller binding 只持有 message caller-facing invoke/start closure。binding 发布后不可变。
_Avoid_: provider bootstrap, remote adapter file, active server record

**Current binding**:
Generated service runtime 内部保存当前 **Active server** record 的 package-private typed atomic slot；新调用和新 stream start 从这里读取 record，再选择 native caller binding 或 message caller binding。已开始的 stream session 固定使用 Start 时捕获的 binding closure。
_Avoid_: active record slot, adapter snapshot

**Registration source**:
generated registration function 接受的具体服务来源；使用 `Origin + Contract + Transport + Mode` 四个正交维度描述。source 被接受后归一化为 **Active server** record，record 内分别保存 native caller binding 与 message caller binding。renderer 是由 source 推导出的生成策略，不是 source contract。
_Avoid_: Active record source, RecordRenderer as contract

**Runtime core**:
手写的 `rpcruntime` 包，承载跨 service 复用的 stream registry、stream lifecycle state 和 connect stream unsafe shim 等通用机制。
_Avoid_: generated service runtime

**Stream lifecycle**:
一次 streaming call 从 Start 到终态操作期间的 operation set、handle ownership、session lookup、half-close、finish/close-send/cancel 和 invalid-handle 语义；contract-level operation plan 与 runtime state machine 必须区分。Finish 是统一的 graceful terminal；CloseSend 是 bidi 的 half-close；Cancel 是 abort terminal。
_Avoid_: stream registry access pattern

**Stream lifecycle projection**:
generator 侧的 contract-to-render 投影，把 **Stream lifecycle** 的 capability、terminal policy 和 codec requirement 转换为 **Generated service runtime** 可消费的 final session 计划；不执行 runtime state machine，也不拥有 handle storage。
_Avoid_: runtime stream lifecycle executor, stream registry helper plan

**Generated service runtime**:
每个 service 生成的 `*.runtime.rpccgo.go`，只应承载 proto/service/method-specific 的 active server record、package-level invoke/start facade、final session 和 converter glue。
_Avoid_: runtime core

**Remote client active server**:
以标准 connect/gRPC client 注册，并在 generated runtime 内归一化为 **Binding** 的 message-contract active server，真实执行目标位于远端进程。
_Avoid_: remote adapter, remote server adapter

**Call-scoped borrowed view**:
仅在一次 generated message→native 同步 native operation 调用期间有效的 borrowed wrapper 视图；generated request view 可持有底层 protobuf message 和 raw buffer 来维持 owner reachability，但不得把 wrapper 本身跨调用或跨 stream session 保存。
_Avoid_: owned wrapper, long-lived native input

**Native C ABI lowering**:
从 **Native** 的 Go-level flat contract 到跨 Go/C ABI shape 的共享投影规则；按 operation 表达 C signature、field lowering、ownership、cleanup、callback/export 和 error bridge 需求。它不形成独立 contract。
_Avoid_: persisted Native C ABI plan, separate native contract, renderer-specific ABI inference

**Method contract plan**:
单个 method 从 descriptor 派生出的完整 contract-level planning 结果；集中表达 **Native**、**Message contract**、**Stream lifecycle** operation plan 和 render planning inputs，但不生成代码字符串。
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
- Go native server 与 C native server 都实现同一个 **Native** server contract；C message server 属于 **Message contract**，不应被混入 native server 命名。
- C message server 应有独立的 generated server contract，例如 `GreeterCGOMessageServer`；其方法名使用 service method Go name，不额外追加 `Message` 或 `Start` 前缀，message contract 由 server contract 名称表达。
- C message server streaming 方法属于 handler-style server contract：stream 对象作为方法参数传入；`Start` 返回 final session 只属于 generated runtime 与 C callback ABI 的内部投影。
- **Native C ABI lowering** 必须从 **Native** / `NativeContractPlan` 派生，不能重新解释 proto descriptor 或形成独立 contract。
- C 侧 **Native** callback 必须使用字段级参数列表，例如 `field_ptr/field_len/ownership` 和输出字段指针参数；不能接收 generated `Request*` / `Response*` struct。
- 跨 runtime 的 C **Native** ABI 不能以 `struct` 或 `struct*` 作为调用边界参数；service-level callback 注册也必须使用 flat callback 参数。
- C **Native** server callback 必须作为完整 service callback set 注册；只有完整校验通过后才能激活为 **Active server**，不能按 method 增量激活。
- C 导出符号命名以 `<contract> + <namespace> + <service> + <method> + <operation>` 组成；`namespace` 默认取 Go package name，冲突时由用户显式覆盖，不使用调用端/实现端语言前缀区分方向。
- Go **Native** server 输入字段类型沿用旧 wrapper：`string -> *rpcruntime.RpcString`、`bytes/message -> *rpcruntime.RpcBytes`、`repeated scalar -> *rpcruntime.RpcRepeat[T]`、`repeated bool -> *rpcruntime.RpcBoolRepeat`。
- 由 **Message contract** 适配到 **Native** 时，请求侧 wrapper 只应作为 **Call-scoped borrowed view** 存在；其底层数据只保证在该次 generated 同步 native operation 调用期间有效。
- Go **Native** server 返回值沿用旧 flat 返回：response 顶层字段按 Go 值/slice 顺序返回，最后一个返回值固定是 `error`。
- Go **Native** server streaming / bidi streaming 的 response 顶层字段通过 native stream `Send` 的 flat 参数发送；method 本身只返回终态 `error`。
- **Native** 只拍平 proto request/response 的顶层字段；nested message 作为整体 message bytes/wrapper 传递，不递归展开。
- `NativeContract` 这类字段计划可以作为参数转换的中间表示保留；它不是最终 **Native** 边界。
- **Native C ABI lowering** 可表达 ownership / cleanup / transfer；它不应新增现有 ABI 之外的 ownership 参数，但若现有 C boundary 已包含 ownership slot，lowering 应把它作为 ABI slot 结构化表达。
- **Native C ABI lowering** 位于 `NativeContract` 之后、renderer 之前；client/server renderer 共享同一套按需 lowering，不持久化独立的 service-level 或 method-level C ABI plan。
- generator 不保留 `NativeCABIPlan`、`MethodNativeCABIPlan`、`MethodContractPlan.NativeCABI` 或 method C ABI attach/finalize 阶段；service-level callback 注册 ABI 由各 method 的按需 lowering 结果直接组装。
- **Native C ABI lowering** 应返回 slot role、最终 C type spelling、cleanup capability、export symbol naming 和 callback typedef naming，使 renderer 不再重复推断 ABI 语义。
- **Native C ABI lowering** 拥有 method-level C boundary operation inventory；renderer 不重复维护 unary、client streaming、server streaming 和 bidi streaming 的 operation 列表。
- lowered ABI slot 只保留 renderer 实际消费的最小字段：name、C type、cgo Go type、role 和可选 field Go name；不保留只用于解释旧 plan 的 source metadata 或未被 renderer 消费的 cleanup metadata。
- **Native C ABI lowering** 对未知 streaming kind、非法 operation 和无法组装的 service-level callback 注册 ABI 返回显式 `error`；renderer 逐层传递错误，不使用空 ABI slot 兜底，也不允许 `panic`。
- C native preamble、callback registration 和 C export renderer 遍历 **Native C ABI lowering** 返回的 operation inventory；renderer 仅在生成文本确实不同的地方按 streaming kind 分支。
- **Native C ABI lowering** 不表达 callback missing policy、error bridge lifecycle 语义或 **Stream lifecycle** handle cleanup；这些分别属于 **Generated service runtime**、error bridge Module 和 **Runtime core**。
- protobuf schema 中的 unsigned 字段可进入 **Native C ABI lowering** 的 field value slot；proto 无关的 length/count/handle/error id 等辅助 slot 不应使用 unsigned 32/64 类型。
- 修改 ABI / runtime type mapping 后，必须使用 `docs/release/verification-checklist.md` 验证测试命令和合同扫描。
- **Active server** 是新版调用模型的一部分；它不能改变 **Native** 的字段级函数边界语义。
- **Runtime core** 负责通用 stream registry、stream lifecycle state 和 connect stream unsafe shim；**Generated service runtime** 负责 typed atomic active slot、service-specific typed glue 和 active record closure。
- **Stream lifecycle** 的 ownership、terminal-once 和 invalid-handle 通用状态语义属于 **Runtime core**；method-specific session 操作、native/message 转换和 flat ABI 编解码属于 **Generated service runtime**。
- **Generated service runtime** 可以组合 **Runtime core** 的 stream registry 与 lifecycle primitive，但 registry 应直接保存 final session，不应生成无语义的 per-method `load/take/delete` 薄包装。
- Register helper 留在 **Generated service runtime** 中，因为它们校验并原子发布 service-specific **Binding**；成功时只需返回 `nil`，不返回 adapter snapshot。
- **Generated service runtime** 不应把 **Native** 与 **Message contract** 的 caller-facing closure 混放在同一个 **Binding**；单个 active server record 可以同时持有 native caller binding 与 message caller binding，以保持每个 service 一个 typed atomic active slot。
- **Registration source** 使用 `Origin + Contract + Transport + Mode` 描述；renderer 选择由这四个维度派生，source plan 不存储 `RecordRenderer`。`Label` 只用于错误文本，不能控制生成逻辑。
- **Registration source** planner 只枚举 7 类合法 server source，不接受四个维度的任意组合再做宽泛校验；未列出的组合没有生成语义。
- 单个 service 的 **Registration source** 由 `ServiceGenerationSelection` 派生：未启用 `native` 时只包含 cgo message、本地 message transport 和 remote message transport 三类 source；启用 `native` 时再追加 Go native 与 cgo native source。
- cgo native source 复用 Go native binding 构造路径是由 `Origin=cgo + Contract=native + Transport=none + Mode=local` 派生出的行为；source plan 不存储额外 alias flag。
- **Registration source** plan 只保留 `Origin + Contract + Transport + Mode` 身份字段；register name、input name/type、source expression、error label 和 nil error 等 renderer projection 数据统一从 service 与四轴身份派生。
- **Registration source** 的无 Connect/gRPC transport 来源使用显式 `Transport=none`；四轴字段的空字符串统一表示未初始化并由 validation 拒绝。
- **Registration source** 必须经过 validation：四轴字段非空、组合属于 7 类白名单、renderer projection 可完整派生。projection 与 renderer 对未知组合显式返回 `error`，不允许 `panic`。
- `Origin + Contract + Transport + Mode` 只描述 **Registration source**，不复用于 generated artifact planning。service-shared runtime、codec 和 cgo client artifact 不是 registration source，应使用独立 artifact plan 表达。
- generator plan 使用 `GenerationPlan -> PackagePlan -> FilePlan -> ServicePlan` 层级：package-level symbols、cgo import path 和 shared cgo exports 属于 `PackagePlan`，proto descriptor 与 service artifact 属于 `FilePlan` / `ServicePlan`。
- generator 入口直接返回 `GenerationPlan`；项目未发布，不保留返回 `[]FilePlan` 的兼容 API。
- generated artifact planner 使用 `PackagePlan.SharedArtifacts` 与 `ServicePlan.Artifacts` 两级白名单列表；两者共用同一个 `GeneratedArtifactPlan` item 类型，每项只保存 artifact kind 和 filename。不保留重复表达 runtime 的 native/message file family，也不保留 `Enabled` 字段。未启用 artifact 不进入列表。
- generator 只保留完整 artifact list renderer，不保留 native/message 分阶段生成 API 或 options。测试通过 artifact kind 定向筛选或验证完整生成结果。
- generated artifact enabled 规则固定：service runtime、codec 和 shared cgo exports 始终生成；`native` 启用 Go native server contract、cgo native server artifact 和 cgo native client artifact；`msg-connect` 或 `msg-grpc` 启用 Go message server contract、cgo message server artifact 和 cgo message client artifact。没有 `native` token 时不得生成 native artifact。
- native/message codec 是 **Generated service runtime** 的无条件能力；planner 不保留 `NeedsCodec` 或 `CodecEnabled` 这类总为真的选择字段。
- native/message converter 不可用不是调用期状态；生成器 validation 或 renderer projection 必须在生成阶段返回显式 `error`，generated runtime 不保留 `NativeMessageConverterUnavailableErr` 这类不可达 sentinel。
- generated artifact plan 必须经过 validation：artifact kind 属于白名单、filename 非空、同一 service kind 不重复、输出路径不重复。renderer 对未知 kind 显式返回 `error`。shared cgo exports 由 generation-level artifact planner 按 cgo Go package 生成一次，不参与 service-level 合并去重补丁。
- 完整 `GenerationPlan` 构建后、render 前必须通过 `ValidateGenerationPlan`；它向下校验 package、file、service、method、active source 与 artifact invariant。renderer 只保留未知 kind/source 的防御性 `error`，不承担主 validation。
- `@rpccgo` token 表达 service generation selection，不是 adapter selection 或纯 server registration selection。generator 使用 `ServiceGenerationToken`、`ServiceGenerationSelection` 和 `ServicePlan.Generation` 命名，不保留 `AdapterToken`、`AdapterSelection` 或 `ServicePlan.Adapters`。
- `@rpccgo` token 只停留在 parser 层；planner 中的 `ServiceGenerationSelection` 收敛为结构化能力：一个 message transport 与 `NativeEnabled`。后续 planner 和 renderer 不重复扫描 token 列表。
- `ServiceGenerationSelection.MessageTransport` 必须是 `connect` 或 `grpc`；zero value 只表示未初始化并由 validation 拒绝，不引入具有业务含义的 `none`，因为当前没有 native-only generation 模式。
- **Binding** 在注册阶段组装 caller-facing method closure；closure 内直接绑定具体 server 调用与必要的 native/message 转换。调用阶段不再按 server kind 或 contract 路由。
- 外部包只能通过 generated package-level invoke/start 函数进入；不应再生成只转发到内部对象的 public client object，也不应保留 runtime bridge struct。
- 无 active server 使用 `rpcruntime.ErrNoActiveServer`。错误必须显式传递，但不为注册阶段已经排除的不可能状态保留调用阶段 routing sentinel。
- **Remote client active server** 使用标准 transport client 作为注册输入，由 generated runtime 归一化为 **Binding**；rpccgo generated code 不应构造 per-method client。
- **Remote client active server** 只转发 protobuf message payload 和 error；metadata/header/trailer 不属于当前 contract。
- `Register<Service>ConnectRemoteServer` 与 `Register<Service>GRPCRemoteServer` 命名可以保留，但它们应直接接收标准 transport client 并返回 `error`，不应构造 service-specific wrapper adapter。
- **Remote client active server** 的 direct invocation 与 final session closure 属于 **Generated service runtime**；不应再生成独立 `remote.connect.rpccgo.go` 或 `remote.grpc.rpccgo.go` adapter 文件。
- 一个 service 的 generated output 只能选择一个 message transport（connect 或 gRPC），避免标准 transport client API 在同包内重名。
- 每个 service 的 active slot 应保留为 generated package-level typed atomic pointer；不要引入 `rpcruntime.ActiveServerSlot` 或 runtime core 全局 service registry，以避免无价值包装和旧 **Provider bootstrap** 模型。
- 新版架构保留 service-local active server；只恢复旧项目的 **Native** flat function boundary，不回迁旧 **Provider bootstrap**。
- `@rpccgo:native` 的新版 service generation selection 规则保留；它可以同时启用默认 message generation，但 **Native** 侧仍必须是 flat function boundary。
- 旧 `go_role=go_client` / C provider 注册 Go client 能力不恢复；它属于旧 **Provider bootstrap** 架构，不是新版 **Native** 修复范围。

## Example dialogue

> **Dev:** “这个 `native` callback 能不能接收一个 generated input struct？”
> **Domain expert:** “不能。**Native** 的验收标准是 flat function boundary，request/response 顶层字段必须直接出现在最终函数边界。”

## Flagged ambiguities

- “struct native ABI” 曾被用来描述当前重构中的 generated input/output struct 边界；已决议：这不是 **Native**，应视为错误实现而非 native 的一种形态。
