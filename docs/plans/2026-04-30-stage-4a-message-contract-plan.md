# Stage 4A Message Contract Direct Path Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 generated service 级 message contract 直连调用链，让 cgo message client 通过 protobuf bytes ABI 进入 dispatcher，并在 active server 是 cgo message server 时完成 message-to-message direct call。

**Architecture:** Stage 4A 只实现 contract 匹配的 message direct path：cgo message client 发起 protobuf bytes request，dispatcher 捕获 active server，直接调用 cgo message server callback adapter。Stage 4A 复用 Stage 1 `ServicePlan`、Stage 2 `rpcruntime` dispatcher/session primitive 和 Stage 3 generated service runtime glue，不实现 native/message converter，也不让 message client 调 native server。

**Follow-up:** 2026-04-30 dispatcher alignment follow-up 已把 Stage 3/4A runtime 收敛为每个 generated service 一个 `rpcruntime.Dispatcher[<Service>ActiveAdapter]`。native/message adapter interface 仍分离，注册 helpers 写入同一个 active server slot，contract mismatch 继续返回 converter-disabled error，留给 Stage 4B 替换为 generated codec conversion。

**Tech Stack:** Go 1.24、cgo、protobuf `protogen`、现有 `internal/generator`、`rpcruntime`、标准库 `testing`。

---

## 范围

Stage 4A 聚焦 message contract direct path：

- protobuf bytes ABI。
- cgo message client 到 dispatcher 的调用路径。
- cgo message server callback table、registration API 和 callback adapter。
- message-to-message unary、client streaming、server streaming、bidi streaming。
- active server snapshot、stream handle、terminal cleanup 与 Stage 2/3 语义保持一致。
- Stage 4A integration fixture 与迁移清单。

Stage 4A 不实现：

- generated `<service>.codec.rpccgo.go`。
- protobuf message 与 native field struct 双向转换。
- cgo message client 调 native server。
- cgo native client 调 message server。
- connect handler adapter。
- grpc server adapter。
- connect/grpc remote server adapter。
- 监听 server 启动模型。
- example 业务工程。
- 旧多 registry、多 provider bootstrap、framework selector。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| 旧 `message_client` renderer | 参考后重写 | 生成 cgo message client protobuf bytes ABI | protobuf bytes 输入输出、marshal/unmarshal error 关注点可参考；旧实现绕过新版 generated dispatcher，需要按单 active server slot 重写 |
| 旧 `message_server` renderer 中 cgo message callback 部分 | 参考后重写 | 生成 cgo message server callback table 与 adapter | callback table 与 streaming callback 形态可参考；旧代码混入 connect/grpc handler 与 framework selector，Stage 4A 只能保留 cgo message server direct path |
| 旧 `message_export_shim_cgo` | 参考后重写 | 承接 C callback ABI、error id 与 streaming shim | ABI 思路有价值；新版必须使用 `int32` error id、`rpcruntime.StreamHandle` 和 dispatcher snapshot |
| 旧 message mode integration fixture | 迁移测试思路 | 覆盖 protobuf bytes ABI、message streaming 生命周期和错误传播 | 测试场景可迁移，但生成物必须改写为新版 `<service>.*.rpccgo.go` 文件族 |
| 旧 binding/provider/bootstrap/framework selector | 不迁移 | 旧多 registry、多 provider 和 framework 选择模型 | 与新版单 dispatcher、单 active server、单插件 renderer pipeline 冲突，不能进入 Stage 4A |
| 旧 native/message codec | 不迁移 | contract mismatch 转换 | Stage 4A 完成标准只到 message-to-message direct call，converter 由 Stage 4B 承接 |

## 输出模型

Stage 4A 结束后，message direct path 至少生成以下文件或等价能力：

- `<service>.runtime.rpccgo.go`：补齐 message adapter contract、message stream session glue、single dispatcher active adapter union 和 dispatcher message entrypoints。
- `<service>.client.message.cgo.rpccgo.go`：新增 cgo message client exported ABI，所有调用进入 dispatcher。
- `<service>.server.message.cgo.rpccgo.go`：新增 cgo message server callback table、registration API 和 callback adapter。

Stage 4A 可以修改 Stage 3 已存在的 generated file family，但不能生成 `<service>.codec.rpccgo.go`。contract mismatch 时必须返回明确错误，不能隐式 fallback、不能临时转换。

## Task 1：建立 message renderer pipeline 与文件布局

**Files:**

- Create: `internal/generator/render_message_plan.go`
- Create: `internal/generator/render_message_plan_test.go`
- Modify: `internal/generator/render.go`
- Modify: `internal/generator/plan.go`
- Modify: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 不迁旧 message renderer 输出代码。
- 参考旧 renderer 的 message file family 命名经验，但按新版 `<service>.*.rpccgo.go` 布局和单 dispatcher 模型重建。

- [x] 定义 message direct path generated file family plan，复用 runtime、client.cgo、server.cgo 文件族。
- [x] cgo message client 生成策略不受 `@rpccgo` server adapter 注释控制。
- [x] 只有需要 cgo message server adapter 时才生成 message server callback section。
- [x] renderer plan 明确不生成 `<service>.codec.rpccgo.go`。
- [x] 添加测试：message direct path 输出预期文件名。
- [x] 添加测试：Stage 4A 不输出 connect/grpc/remote/codec 文件。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -run 'TestRenderMessage|TestGenerate' -count=1`。
- [x] 验收：生成文件布局与架构文档一致，且没有引入旧 framework selector。
- [x] 提交：`feat: add message direct renderer pipeline`

## Task 2：扩展 service-specific runtime glue 支持 message contract

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_runtime_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 不迁旧 message runtime glue。
- 新 runtime glue 只把 Stage 2 `rpcruntime.Dispatcher` 包装成 service-specific message adapter API。

- [x] 生成 service message adapter interface，覆盖 unary 与三类 streaming operation。
- [x] 生成 message active server registration helper 的内部接入点，供 cgo message server adapter 使用。
- [x] 生成 dispatcher message entrypoints，contract 匹配时直接调用 message adapter。
- [x] contract mismatch 时返回明确错误，错误文本说明 converter 尚未启用。
- [x] stream `Start` 捕获 active server snapshot，后续 message stream 操作固定路由到该 snapshot。
- [x] 添加 source assertion 测试：runtime glue import `rpcruntime`，不 import connect/grpc。
- [x] 添加测试：runtime glue 中 message stream handle 类型是 `rpcruntime.StreamHandle`。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderRuntime' -count=1`。
- [x] 验收：generated runtime 同时支持 native 和 message contract dispatch，但 message direct path 不依赖 converter。
- [x] 提交：`feat: render message service runtime glue`

## Task 3：生成 cgo message server callback ABI

**Files:**

- Create: `internal/generator/render_message_server_cgo.go`
- Create: `internal/generator/render_message_server_cgo_test.go`
- Create: `internal/integration/message_cgo_server_unary_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 参考旧 `message_server` 与 `message_export_shim_cgo` 的 callback table 形态。
- 重写 registration API，注册时只写入单 active server slot。

- [x] 生成 cgo message server callback table。
- [x] 生成 `Register<Service>CGOMessageServer(callbacks)`。
- [x] nil callback table 或缺失 unary callback 返回明确错误。
- [x] adapter 调用 C callback，并按 `int32` error id 传播错误。
- [x] callback request/response 使用 protobuf bytes ABI，不接触 native field wrapper。
- [x] 添加 generated source assertion 测试，覆盖 unary callback 签名。
- [x] 添加 integration fixture：cgo message client unary 调用 cgo message server。
- [x] 添加测试：后注册 cgo message server 覆盖后续调用，已启动 stream 保持旧 snapshot。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestMessageCGOServerUnary|TestMessageUnary' -count=1`。
- [x] 验收：cgo message server 可以注册成 active server，message unary direct path 可用。
- [x] 提交：`feat: render cgo message server callbacks`

## Task 4：实现 message unary client 到 cgo message server

**Files:**

- Create: `internal/generator/render_message_client_cgo.go`
- Create: `internal/generator/render_message_client_cgo_test.go`
- Create: `internal/integration/message_unary_test.go`
- Modify: `internal/generator/render.go`
- Modify: `internal/generator/render_message_server_cgo.go`

**迁移内容与理由：**

- 参考旧 `message_client` 的 protobuf bytes ABI 与 marshal/unmarshal error 传播关注点。
- 重写 cgo message client export，使调用总是进入 generated dispatcher。

- [x] 生成 cgo message unary client export。
- [x] request bytes 先执行 protobuf unmarshal 校验，再以 message contract 进入 dispatcher。
- [x] response message marshal 成 protobuf bytes 返回给 cgo message client。
- [x] Go error 存入 `rpcruntime` error store，并通过 `int32` error id 返回。
- [x] 缺少 active server 时返回明确 error id。
- [x] active server 是 native adapter 时返回明确 contract mismatch error，不执行转换。
- [x] 添加 integration fixture：cgo message client unary 调用 cgo message server。
- [x] 添加测试：invalid request bytes、marshal/unmarshal error、callback error propagation。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestMessageUnary' -count=1`。
- [x] 验收：cgo message client unary 通过 dispatcher 直连 cgo message server。
- [x] 提交：`feat: route message unary client to message server`

## Task 5：接入 message client streaming

**Files:**

- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_runtime.go`
- Create: `internal/integration/message_client_streaming_test.go`

**迁移内容与理由：**

- 参考旧 message streaming 的 Start、Send、Finish、Cancel 调用顺序。
- 不迁旧 stream registry；使用 Stage 2 `StreamHandle` 和 dispatcher stream helpers。

- [x] 生成 client streaming `Start`、`Send`、`Finish`、`Cancel` message ABI。
- [x] `Start` 捕获 active server snapshot 并返回 `rpcruntime.StreamHandle`。
- [x] `Send` 校验 request protobuf bytes，使用 handle 找回 session。
- [x] `Finish` terminal 操作使用 typed take，成功后 handle 不再可用。
- [x] `Cancel` 传播到底层 message adapter 并 finalize。
- [x] 添加 integration fixture：client streaming 到 cgo message server。
- [x] 添加测试：invalid send bytes、send closed、finalized、canceled 后返回明确错误。
- [x] 运行 `rtk go test ./internal/integration -run 'TestMessageClientStreaming' -count=1`。
- [x] 验收：client streaming message direct path 复用统一 stream lifecycle。
- [x] 提交：`feat: support message client streaming direct path`

## Task 6：接入 message server streaming

**Files:**

- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_runtime.go`
- Create: `internal/integration/message_server_streaming_test.go`

**迁移内容与理由：**

- 参考旧 server streaming onRead/onDone 测试关注点。
- 重写为 generated message session glue，不迁旧 generated runtime 文件结构。

- [x] 生成 server streaming `Start`、`Cancel`、`onRead`、`onDone` message ABI。
- [x] `Start` 校验 request protobuf bytes 并返回 `rpcruntime.StreamHandle`。
- [x] `onRead` 通过 handle 读取固定 snapshot session，并返回 response protobuf bytes。
- [x] `onDone` 使用 typed take，并执行 terminal cleanup。
- [x] `Cancel` 传播到底层 message adapter 并 finalize。
- [x] 添加 integration fixture：server streaming 到 cgo message server。
- [x] 添加测试：onDone 后 handle 不再可用，callback error 进入 error store。
- [x] 运行 `rtk go test ./internal/integration -run 'TestMessageServerStreaming' -count=1`。
- [x] 验收：server streaming message direct path 在 onDone 后释放 handle。
- [x] 提交：`feat: support message server streaming direct path`

## Task 7：接入 message bidi streaming

**Files:**

- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_runtime.go`
- Create: `internal/integration/message_bidi_streaming_test.go`

**迁移内容与理由：**

- 参考旧 bidi streaming 的 send、close send、onRead、onDone、cancel 覆盖。
- 使用 Stage 2 lifecycle helper 表达 terminal 和 close-send 规则。

- [x] 生成 bidi streaming `Start`、`Send`、`CloseSend`、`Cancel`、`onRead`、`onDone` message ABI。
- [x] `Send` 校验 request protobuf bytes，`onRead` 返回 response protobuf bytes。
- [x] `CloseSend` 后继续 `Send` 返回明确错误。
- [x] `onDone` 后所有 handle 操作返回稳定错误或 false。
- [x] `Cancel` 不重复执行 callback。
- [x] 添加 integration fixture：bidi streaming 到 cgo message server。
- [x] 添加测试：invalid send bytes、callback error、cancel 后 read/done 行为。
- [x] 运行 `rtk go test ./internal/integration -run 'TestMessageBidiStreaming' -count=1`。
- [x] 验收：bidi message direct path 覆盖 send/read/close/cancel/done 全生命周期。
- [x] 提交：`feat: support message bidi streaming direct path`

## Task 8：Stage 4A acceptance tests 与迁移清单

**Files:**

- Create: `internal/integration/message_stage4a_acceptance_test.go`
- Create: `docs/plans/2026-04-30-stage-4a-migration-inventory.md`
- Modify: `docs/plans/2026-04-30-stage-4a-message-contract-plan.md`

**迁移内容与理由：**

- 迁移旧 message mode integration 的测试关注点。
- 记录旧 message 代码为什么参考后重写，以及旧 bootstrap、framework selector 和 converter 为什么不进入 Stage 4A。

- [x] 添加 Stage 4A acceptance test，覆盖 cgo message client 到 cgo message server。
- [x] acceptance test 覆盖 unary、client streaming、server streaming、bidi streaming。
- [x] acceptance test 证明 cgo message client 调用都进入 dispatcher。
- [x] acceptance test 证明 active server 是 native adapter 时返回 contract mismatch error。
- [x] 写入已迁移、参考后重写、不迁移清单。
- [x] 明确旧多 registry、多 provider bootstrap、framework selector 不迁移。
- [x] 明确 Stage 4A 不生成 `<service>.codec.rpccgo.go`，不实现 native/message converter。
- [x] 记录验证命令：generator focused、integration focused、runtime focused、全仓测试、AGENTS.md 中的 forbidden unsigned scan。
- [x] 不记录机器环境处理。
- [x] 更新本计划 checkbox。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`。
- [x] 运行 `rtk go test ./internal/integration -count=1`。
- [x] 运行 `rtk go test ./rpcruntime -count=1`。
- [x] 运行 `rtk go test ./... -count=1`。
- [x] 运行 AGENTS.md 中的 forbidden unsigned scan。
- [x] 验收：阶段 4A “迁移了什么、为什么参考后重写、为什么不迁旧架构和 converter”有明确记录。
- [x] 提交：`docs: record stage 4a migration inventory`

## Stage 4A 完成标准

- cgo message client ABI 总是通过 dispatcher 调用。
- cgo message server 可以注册成 active server。
- cgo message client 能通过 dispatcher 调用 cgo message server。
- message unary、client streaming、server streaming、bidi streaming 都有端到端验证。
- stream handle 统一使用 `rpcruntime.StreamHandle`，底层类型保持 `int32`。
- error id 使用 `int32`。
- protobuf marshal/unmarshal 失败返回明确 error id。
- contract 匹配时 dispatcher 执行 message-to-message direct call。
- contract mismatch 时返回明确错误，不执行转换。
- Stage 4A 不引入 generated converter、connect/grpc adapter、remote adapter 或旧 bootstrap 模型。
- Stage 4A 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无命中。
