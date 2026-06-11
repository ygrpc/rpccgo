# rpccgo Context

rpccgo 生成跨 Go/C 边界的 RPC glue code。本上下文记录项目特有的 contract 术语，避免把历史 native 语义与重构中的中间表示混淆。

## Language

**Native**:
旧项目定义的字段级函数边界 contract：proto request/response 顶层字段直接成为最终 Go/C 函数参数和返回值。
_Avoid_: struct native ABI, message-shaped native adapter

**Message contract**:
以完整 protobuf request/response message 作为 Go 侧调用边界的 contract；跨 C ABI 时投影为序列化 bytes。
_Avoid_: native

**Registered server**:
用户实现并注册到 rpccgo 运行时的 service server；它可以实现 Native、Message contract 或标准 transport server 形态，并作为 cgo 调用的最终执行目标。具体 server 形态由 `rpcruntime.ServerKind` 标记表达。
_Avoid_: active server, provider bootstrap

**Server registry**:
`rpcruntime` 中按 **Service ID** 保存当前 **Registered server** record 的统一运行时 registry；generated facade 从 registry 取得 server 后，按调用 contract 与 server kind 决定是否做 Native/Message 转换。
_Avoid_: active binding, active binding slot, adapter snapshot

**Server kind**:
`rpcruntime` 定义的通用 registered server 形态标记，固定包含 Go native、cgo native、cgo message、connect、gRPC、connect remote 和 gRPC remote；zero value 只表示未初始化，不能作为可注册业务值。它只描述来源形态，不承载 service-specific 方法调用或 protobuf 转换逻辑。
_Avoid_: service-local kind, dispatcher kind

**Service ID**:
generated code 用来标识 protobuf service 的稳定 registry key，默认取 protobuf full service name；`rpcruntime` 只把它当作 opaque key，不依赖 service-specific protobuf 类型。
_Avoid_: server kind, Go type name

**Registration source**:
generated registration function 接受的具体服务来源；使用 `Origin + Contract + Transport + Mode` 四个正交维度描述。source 被接受后注册为当前 **Registered server**。renderer 是由 source 推导出的生成策略，不是 source contract。
_Avoid_: Active record source, RecordRenderer as contract

**Runtime core**:
手写的 `rpcruntime` 包，承载跨 service 复用的 server registry、server kind、全局 stream session registry 和 connect stream unsafe shim 等通用机制。
_Avoid_: generated service runtime

**Stream session**:
一次 streaming call 在 `Start` 后保存到 **Runtime core** 的 `{ServerKind, session}` record；其中 `session` 是该 call 的 typed client endpoint。后续 stream operation 通过 handle 找回 record，并由 generated code 按 kind 转回对应 endpoint 直接调用。
_Avoid_: stream lifecycle state machine, operation closure session

**Stream endpoint**:
Streaming call 的单侧操作能力，必须同时按 streaming shape 与 client/server side 区分。Client streaming、server streaming 和 bidi streaming 各自拥有 client endpoint 与 server endpoint；两侧共享一次 call 的状态，但暴露不同方向的操作。
_Avoid_: direction-neutral stream, per-method facade

**Stream operation projection**:
generator 侧的 contract-to-render 投影，把 streaming method 的 operation capability、terminal policy 和 codec requirement 转换为 generated package-level stream operation 函数；不执行 runtime state machine，也不拥有 handle storage。
_Avoid_: runtime stream lifecycle executor, stream registry helper plan

**Generated service runtime**:
每个 service 生成的 `*.runtime.rpccgo.go`，只应承载 proto/service/method-specific 的 package-level invoke/start facade、registry lookup glue、transport registration glue、stream `Start` glue 和 converter glue。
_Avoid_: runtime core

**Remote registered server**:
以标准 connect/gRPC client 注册的 message-contract **Registered server**；真实执行目标位于远端进程。
_Avoid_: remote adapter, remote server adapter

**Call-scoped borrowed view**:
仅在一次 generated message→native 同步 native operation 调用期间有效的 borrowed wrapper 视图；generated request view 可持有底层 protobuf message 和 raw buffer 来维持 owner reachability，但不得把 wrapper 本身跨调用或跨 stream session 保存。
_Avoid_: owned wrapper, long-lived native input

**Native C ABI lowering**:
从 **Native** 的 Go-level flat contract 到跨 Go/C ABI shape 的共享投影规则；按 operation 表达 C signature、field lowering、ownership、cleanup、callback/export 和 error store 需求。它不形成独立 contract。
_Avoid_: persisted Native C ABI plan, separate native contract, renderer-specific ABI inference

**Method contract plan**:
单个 method 从 descriptor 派生出的完整 contract-level planning 结果；集中表达 **Native**、**Message contract**、stream operation plan 和 render planning inputs，但不生成代码字符串。
_Avoid_: renderer, generated code, runtime state machine

**Native projection**:
从同一个 **Native** contract 派生出的具体语言/边界形态；Go native projection 表达 Go 函数参数和返回值，C native projection 表达跨 Go/C 的 ABI slot。
_Avoid_: separate native contract, incompatible Go/C native ABI

**Provider bootstrap**:
旧项目通过 provider/registry/bootstrap 组装服务能力的架构模型；新版 registry 不回迁该模型中的 provider 分层、bootstrap 入口或 go_role 能力注册。
_Avoid_: active server

## Relationships

- **Native** 与 **Message contract** 是不同 contract；**Native** 不应退化成 request/response struct 或 message 指针边界。
- **Native** 的字段级函数边界必须覆盖 Go server interface、Go native client API、C callback ABI，以及 streaming 的 start/send/recv/finish/close/cancel 相关边界。
- Go native 与 C native 是同一个 **Native** contract 的不同 **Native projection**；它们不应被建模为两套独立 native contract。
- Go native server 与 C native server 都实现同一个 **Native** server contract；C message server 属于 **Message contract**，不应被混入 native server 命名。
- C message server 应有独立的 generated server contract，例如 `GreeterCGOMessageServer`；其方法名使用 service method Go name，不额外追加 `Message` 或 `Start` 前缀，message contract 由 server contract 名称表达。
- C message server 的 Go 侧 server contract 使用 typed protobuf request/response message；跨 C ABI 的 `ptr/len` bytes 只是 **Message contract** 的 C projection，不是 Go 侧 contract surface。
- Native 与 Message contract 共用按 shape 和 side 区分的 generic **Stream endpoint** surface：`ClientStreamingClient/Server`、`ServerStreamingClient/Server` 和 `BidiStreamingClient/Server`。Contract 差异由 endpoint 的 `Req` / `Resp` 类型表达，不使用 `Native`、`CGOMessage` 或 `Session` 前缀复制接口。
- 本地 Go server 调用使用成对的 generic client/server endpoint；两端通过私有 shared state 协调 queue、ack、cancel、close-send 和 finish。Generated code 不生成 per-method state、client facade 或重复 lifecycle methods。
- Native endpoint 的 `Req` / `Resp` 使用 generated method-local envelope；package-level native stream operation 和 Go native handler stream interface 仍暴露 flat field-level boundary。Go native 只生成 flat fields 与 server endpoint envelope 之间的 mapper。
- Go 侧 typed **Message contract** request/response 不接受 nil protobuf message；nil request、nil response 或 nil stream payload 必须返回显式错误。C ABI 的空 `ptr/len` bytes 表示空 protobuf message，由 generated C projection 转换为非 nil typed message。
- C **Message contract** projection 读取 `ptr/len` bytes 时，`len == 0` 一律转换为非 nil 空 protobuf message 且不读取 `ptr`；`len < 0` 或 `len > 0 && ptr == 0` 必须返回显式错误。
- C **Message contract** projection 写出 typed protobuf message 时，先拒绝 nil message，再序列化；序列化结果长度为 0 时输出 `ptr=0,len=0` 且不分配跨 C 边界 buffer。
- C message server streaming 方法属于 handler-style server contract：server endpoint 作为方法参数传入；`Start` 返回 client endpoint 只属于 generated runtime dispatch 与 C callback ABI 的内部投影。
- C **Message contract** server callback 与 C **Native** server callback 一样支持按 method 局部注册；未注册 method 调用时返回 generated unimplemented error，streaming method 的 operation callbacks 不允许半注册。
- **Native C ABI lowering** 必须从 **Native** / `NativeContractPlan` 派生，不能重新解释 proto descriptor 或形成独立 contract。
- C 侧 **Native** callback 必须使用字段级参数列表，例如 `field_ptr/field_len/ownership` 和输出字段指针参数；不能接收 generated `Request*` / `Response*` struct。
- 跨 runtime 的 C **Native** ABI 不能以 `struct` 或 `struct*` 作为调用边界参数；service-level callback 注册也必须使用 flat callback 参数。
- C **Native** server callback 支持按 method 局部注册；未注册的 method 仍属于同一个 **Registered server**，调用时返回 generated unimplemented error。每个 method 内部必须原子校验：unary callback nil 表示该 method 未实现，streaming method 的 operation callbacks 要么全 nil、要么全非 nil，不允许半注册。全部 method 都未注册时仍可注册为全 unimplemented server。
- C per-method register 在 current server 为同一 **Server kind** 时累积到现有 cgo adapter；current server 为空或不是同一 **Server kind** 时创建新的 cgo adapter 并替换当前 **Registered server**。C message per-method register 只累积到 cgo message adapter，C native per-method register 只累积到 cgo native adapter。
- C 导出符号命名以 `<contract> + <namespace> + <service> + <method> + <operation>` 组成；`namespace` 默认取 Go package name，冲突时由用户显式覆盖，不使用调用端/实现端语言前缀区分方向。
- Go **Native** server 输入字段类型沿用旧 wrapper：`string -> *rpcruntime.RpcString`、`bytes/message -> *rpcruntime.RpcBytes`、`repeated scalar -> *rpcruntime.RpcRepeat[T]`、`repeated bool -> *rpcruntime.RpcBoolRepeat`。
- 由 **Message contract** 适配到 **Native** 时，请求侧 wrapper 只应作为 **Call-scoped borrowed view** 存在；其底层数据只保证在该次 generated 同步 native operation 调用期间有效。
- typed **Message contract** surface 不改变 **Call-scoped borrowed view** 规则；message 到 native 的 wrapper 每个 unary 或 stream operation 单独创建，不得跨 stream session 保存。
- Go **Native** server 返回值沿用旧 flat 返回：response 顶层字段按 Go 值/slice 顺序返回，最后一个返回值固定是 `error`。
- Go **Native** server streaming / bidi streaming 的 response 顶层字段通过 native stream `Send` 的 flat 参数发送；method 本身只返回终态 `error`。
- **Native** 只拍平 proto request/response 的顶层字段；nested message 作为整体 message bytes/wrapper 传递，不递归展开。
- `NativeContract` 这类字段计划可以作为参数转换的中间表示保留；它不是最终 **Native** 边界。
- **Native C ABI lowering** 可表达 ownership / cleanup / transfer；它不应新增现有 ABI 之外的 ownership 参数，但若现有 C boundary 已包含 ownership slot，lowering 应把它作为 ABI slot 结构化表达。
- **Native C ABI lowering** 位于 `NativeContract` 之后、renderer 之前；client/server renderer 共享同一套按需 lowering，不持久化独立的 service-level 或 method-level C ABI plan。
- generator 不保留 `NativeCABIPlan`、`MethodNativeCABIPlan`、`MethodContractPlan.NativeCABI` 或 method C ABI attach/finalize 阶段；service-level callback 注册 ABI 由各 method 的按需 lowering 结果直接组装。Renderer 可以在渲染单个 artifact 时使用临时 service-level ABI 聚合值，但该值不得存入 `GenerationPlan`、`ServicePlan` 或 `MethodPlan`。
- **Native C ABI lowering** 应返回 slot role、最终 C type spelling、cleanup capability、export symbol naming 和 callback typedef naming，使 renderer 不再重复推断 ABI 语义。
- **Native C ABI lowering** 拥有 method-level C boundary operation inventory；renderer 不重复维护 unary、client streaming、server streaming 和 bidi streaming 的 operation 列表。
- lowered ABI slot 只保留 renderer 实际消费的最小字段：name、C type、cgo Go type、role 和可选 field Go name；不保留只用于解释旧 plan 的 source metadata 或未被 renderer 消费的 cleanup metadata。
- **Native C ABI lowering** 对未知 streaming kind、非法 operation 和无法组装的 service-level callback 注册 ABI 返回显式 `error`；renderer 逐层传递错误，不使用空 ABI slot 兜底，也不允许 `panic`。
- C native preamble、callback registration 和 C export renderer 遍历 **Native C ABI lowering** 返回的 operation inventory；renderer 仅在生成文本确实不同的地方按 streaming kind 分支。
- **Native C ABI lowering** 不表达 callback missing policy、error store lifecycle 语义或 stream handle cleanup；这些分别属于 **Generated service runtime**、error store module 和 **Runtime core** stream session registry。
- protobuf schema 中的 unsigned 字段可进入 **Native C ABI lowering** 的 field value slot；proto 无关的 length/count/handle/error id 等辅助 slot 不应使用 unsigned 32/64 类型。
- 修改 ABI / runtime type mapping 后，必须使用 `docs/release/verification-checklist.md` 验证测试命令和合同扫描。
- **Registered server** 是新版调用模型的一部分；它不能改变 **Native** 的字段级函数边界语义。
- **Runtime core** 负责通用 server registry、server kind、全局 stream session registry 和 connect stream unsafe shim；**Generated service runtime** 负责 service-specific typed glue、registry lookup、native/message 转换和 flat ABI 编解码。
- Stream 终态操作通过从 stream registry 移除 handle 来表达；移除后的 handle 再操作返回 invalid-handle 错误，不维护额外通用 lifecycle state machine。
- **Runtime core** 统一持有 stream session registry，并直接保存 `{ServerKind, session}` record。Generated `Start` 负责取得 typed client endpoint 后写入 **Runtime core**；generated stream operation 函数通过 handle 取回 record 后执行 service-specific typed dispatch 与 Native/Message 转换。
- Generated code 不应生成 service-local stream registry、method-specific final session record、只包一层 handle 的 stream handle facade，或把 `Send`、`Recv`、`Finish`、`CloseSend`、`Cancel` 实现为 handle wrapper 的成员方法。
- Native handler stream interface 仍生成在 native server contract artifact 中并保持 flat field-level boundary；Native active stream dispatch 使用 generic client endpoint 和 generated method-local envelope。`{ServerKind, session}` record type 属于 **Runtime core**，不在 generated artifact 中重复生成。
- Server contract registration helper 应生成在定义该 server contract 的 artifact 中；standard transport registration helper 可以留在 **Generated service runtime** 中。所有 helper 都必须把具体 server 注册到 **Server registry**；用户不直接手写 **Service ID** 或 **Server kind** 调用 runtime primitive。
- **Generated service runtime** 不应生成 native/message active closure 字段；native/message 差异应由 server contract、registry lookup 和转换逻辑表达。
- **Registration source** 使用 `Origin + Contract + Transport + Mode` 描述；renderer 选择由这四个维度派生，source plan 不存储 `RecordRenderer`。`Label` 只用于错误文本，不能控制生成逻辑。
- **Registration source** planner 只枚举 7 类合法 server source，不接受四个维度的任意组合再做宽泛校验；未列出的组合没有生成语义。
- 单个 service 的 **Registration source** 由 `ServiceGenerationSelection` 派生：未启用 `native` 时只包含 cgo message、本地 message transport 和 remote message transport 三类 source；启用 `native` 时再追加 Go native 与 cgo native source。
- cgo native source 复用 Go native server contract 注册路径是由 `Origin=cgo + Contract=native + Transport=none + Mode=local` 派生出的行为；source plan 不存储额外 alias flag。
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
- 完整 `GenerationPlan` 构建后、render 前必须通过 `ValidateGenerationPlan`；它向下校验 package、file、service、method、registration source 与 artifact invariant。renderer 只保留未知 kind/source 的防御性 `error`，不承担主 validation。
- `@rpccgo` token 表达 service generation selection，不是 adapter selection 或纯 server registration selection。generator 使用 `ServiceGenerationToken`、`ServiceGenerationSelection` 和 `ServicePlan.Generation` 命名，不保留 `AdapterToken`、`AdapterSelection` 或 `ServicePlan.Adapters`。
- `@rpccgo` token 只停留在 parser 层；planner 中的 `ServiceGenerationSelection` 收敛为结构化能力：一个 message transport 与 `NativeEnabled`。后续 planner 和 renderer 不重复扫描 token 列表。
- `ServiceGenerationSelection.MessageTransport` 必须是 `connect` 或 `grpc`；zero value 只表示未初始化并由 validation 拒绝，不引入具有业务含义的 `none`，因为当前没有 native-only generation 模式。
- **Server registry** 在注册阶段保存具体 server 与 **Server kind**；调用阶段从 registry 取得 server，并按调用 contract 与 server kind 选择直接调用或 Native/Message 转换。
- **Server kind** 由 `rpcruntime` 定义，但具体方法调用、type assertion、protobuf 编解码和 Native/Message 转换必须留在 **Generated service runtime**。
- 每个 **Service ID** 同一时刻只有一个 current **Registered server**；native、message、connect、gRPC 和 remote registration 都替换同一个 registry record。
- 非 C callback 的 registration helper 失败时会清空该 **Service ID** 当前 **Registered server** 并返回错误；后续调用应得到 `rpcruntime.ErrNoRegisteredServer`，而不是继续使用旧 server。
- C callback registration 以 method 为原子校验边界；service-level 和 per-method register 遇到半注册 streaming method 时清空该 method callbacks、保留其他有效 method callbacks、写入 cgo adapter，并返回错误报告被拒绝的 method。
- Generated service runtime 暴露 service-specific clear helper 来清空当前 **Registered server**，也暴露 service-specific load helper 供 generated cgo register 累积 current cgo adapter；用户不直接手写 **Service ID** 调用 runtime clear primitive。
- Unary 调用每次从 **Server registry** 读取 current **Registered server**；重新注册只影响后续 unary 调用和后续 stream `Start`。
- Streaming `Start` 捕获当前 **Registered server** record，取得 typed client endpoint 并创建 **Stream session**；后续 stream 操作只通过 stream handle 从 **Runtime core** 找回该 endpoint，不重新读取 **Server registry**，也不通过 operation closure 调用。
- 外部包只能通过 generated package-level entry 函数进入；不应再生成只转发到内部对象的 public client object，也不应保留 runtime forwarding struct。
- 无 registered server 使用 `rpcruntime.ErrNoRegisteredServer`。错误必须显式传递。
- **Remote registered server** 使用标准 transport client 作为注册输入；rpccgo generated code 不应构造 per-method client。
- **Remote registered server** 只转发 protobuf message payload 和 error；metadata/header/trailer 不属于当前 contract。
- `Register<Service>ConnectRemoteServer` 与 `Register<Service>GRPCRemoteServer` 命名可以保留，但它们应直接接收标准 transport client 并返回 `error`，不应构造 service-specific wrapper adapter。
- **Remote registered server** 的 direct invocation 与 final session glue 属于 **Generated service runtime**；不应再生成独立 `remote.connect.rpccgo.go` 或 `remote.grpc.rpccgo.go` adapter 文件。
- 一个 service 的 generated output 只能选择一个 message transport（connect 或 gRPC），避免标准 transport client API 在同包内重名。
- 每个 service 不应生成 native/message active binding slot；当前 registered server 应保存在 `rpcruntime` 的 **Server registry** 中。
- 新版架构保留 **Server registry** 调用模型；只恢复旧项目的 **Native** flat function boundary，不回迁旧 **Provider bootstrap**。
- `@rpccgo:native` 的新版 service generation selection 规则保留；它可以同时启用默认 message generation，但 **Native** 侧仍必须是 flat function boundary。
- 旧 `go_role=go_client` / C provider 注册 Go client 能力不恢复；它属于旧 **Provider bootstrap** 架构，不是新版 **Native** 修复范围。

## Example dialogue

> **Dev:** “这个 `native` callback 能不能接收一个 generated input struct？”
> **Domain expert:** “不能。**Native** 的验收标准是 flat function boundary，request/response 顶层字段必须直接出现在最终函数边界。”

## Flagged ambiguities

- “struct native ABI” 曾被用来描述当前重构中的 generated input/output struct 边界；已决议：这不是 **Native**，应视为错误实现而非 native 的一种形态。
