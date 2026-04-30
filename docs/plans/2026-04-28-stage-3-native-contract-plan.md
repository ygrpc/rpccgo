# Stage 3 Native Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 generated service 级 native contract 调用链，让 cgo native client 可以通过 dispatcher 调用 go native server 或 cgo native server。

**Architecture:** 阶段 3 采用 native contract first：先生成 service-specific runtime glue，再生成 Go native server adapter、cgo native server callback ABI 和 cgo native client ABI。阶段 3 复用 Stage 1 `ServicePlan` 与 Stage 2 `rpcruntime` dispatcher/session primitive，不实现 message contract、native/message converter、connect/grpc adapter 或 remote adapter。

**Tech Stack:** Go 1.24、cgo、protobuf `protogen`、现有 `internal/generator`、`rpcruntime`、标准库 `testing`。

---

## 范围

阶段 3 聚焦 native contract：

- native renderer 框架与 generated file layout。
- service-specific runtime glue。
- Go native server interface、adapter 和 registration API。
- cgo native server callback ABI。
- cgo native client ABI。
- native unary、client streaming、server streaming、bidi streaming 的 dispatcher 调用链。
- Stage 3 integration fixture 与迁移清单。

阶段 3 不实现：

- cgo message client。
- cgo message server。
- protobuf bytes ABI。
- native/message converter。
- connect handler adapter。
- grpc server adapter。
- connect/grpc remote server adapter。
- 监听 server 启动模型。
- example 业务工程。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| 旧 `native_server` renderer | 参考后重写 | 生成 Go native server interface、adapter 和 registration API | Go native server 的接口语义有参考价值，但旧实现绑定旧 registry/provider，需要按新版 generated service dispatcher 重写 |
| 旧 `native_bridge` | 参考后重写 | 在 native ABI 与 Go 调用之间做字段映射和错误传播 | 字段映射、owned memory、error store 的调用顺序可参考；旧桥接不使用 Stage 2 dispatcher snapshot，不能直接迁移 |
| 旧 `native_bridge_cgo` | 参考后重写 | 生成 C callback/export ABI | callback table 与 export 形态可参考；新版必须使用 `int32` stream handle、`int32` error id 和单 active server slot |
| 旧 `native_runtime_cgo` | 参考后重写 | 支撑 cgo native 调用的 runtime glue | callback 生命周期有参考价值；旧 bootstrap、多 provider 入口与新版单 dispatcher 冲突 |
| 旧 native codec 测试 | 迁移测试思路 | 覆盖 native 字段 ABI、owned wrapper、repeated wrapper、error propagation | 测试关注点仍是阶段 3 质量门槛，适合改写为新版 generated output 与 integration fixture |
| 旧 binding/provider/bootstrap/framework selector | 不迁移 | 旧多 registry、多 provider 和 framework 选择模型 | 与新版单 dispatcher、单 active server slot、单插件 renderer pipeline 冲突，不能进入阶段 3 |

## 输出模型

阶段 3 结束后，native-enabled service 至少生成以下文件或等价能力：

- `<service>.runtime.rpccgo.go`：service-specific dispatcher wrapper、native adapter union、stream session glue。
- `<service>.server.native.rpccgo.go`：Go native server interface、adapter、registration API。
- `<service>.server.cgo.rpccgo.go`：cgo native server callback table、registration API、callback adapter。
- `<service>.client.cgo.rpccgo.go`：cgo native client exported ABI，所有调用进入 dispatcher。

本阶段只要求 native contract direct call。message-only service 可以继续不生成 native server adapter，但 cgo native client 的最终全局策略留给 Stage 4 与 converter 一起收口。

## Task 1：建立 native renderer pipeline 与文件布局

**Files:**

- Create: `internal/generator/render.go`
- Create: `internal/generator/render_native_plan.go`
- Create: `internal/generator/render_native_plan_test.go`
- Modify: `internal/generator/generator.go`
- Modify: `internal/generator/plan.go`
- Modify: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 不迁旧 renderer 输出代码。
- 参考旧 renderer 的文件族命名经验，但按新版 `<service>.*.rpccgo.go` 布局重建。

- [x] 定义 generated file family plan，包含 runtime、native server、cgo server、cgo client。
- [x] 只有 `AdapterTokenNative` 启用时生成 native server 与 cgo native server 文件。
- [x] cgo native client 文件是否生成先由 Stage 3 plan 显式记录，不能被 `@rpccgo` server adapter 注释误控。
- [x] `Generate` 可以在 renderer enabled 时输出 native stage files；Stage 1 plan-only tests 继续可验证。
- [x] 添加测试：native-enabled service 输出预期文件名，message-only service 不输出 native server 文件。
- [x] 添加测试：renderer 不输出 connect/grpc/message/remote 文件。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -run 'TestRenderNative|TestGenerate' -count=1`。
- [x] 验收：生成文件布局与架构文档一致，且没有引入旧 framework selector。
- [x] 提交：`feat: add native renderer pipeline`

## Task 2：生成 service-specific runtime glue

**Files:**

- Create: `internal/generator/render_runtime.go`
- Create: `internal/generator/render_runtime_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 不迁旧 binding runtime。
- 新 runtime glue 只把 Stage 2 `rpcruntime.Dispatcher` 包装成 service-specific API。

- [x] 生成 service package 内的 dispatcher 变量或 holder，类型参数为 generated native adapter interface。
- [x] 生成 service native adapter interface，包含 unary 与 streaming operation 方法。
- [x] 生成 active server registration helper，内部调用 `rpcruntime.Dispatcher.Register`。
- [x] 生成 stream helper，使用 `rpcruntime.StreamHandle` 和 public dispatcher stream helpers。
- [x] 添加 golden 或 source assertion 测试，证明 runtime glue import `rpcruntime`，不 import connect/grpc。
- [x] 添加测试：runtime glue 中 stream handle 类型是 `rpcruntime.StreamHandle`，不使用旧 handle 类型。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderRuntime' -count=1`。
- [x] 验收：generated runtime 是 service-specific，但通用状态仍在 `rpcruntime`。
- [x] 提交：`feat: render native service runtime glue`

## Task 3：生成 Go native server interface 与 registration API

**Files:**

- Create: `internal/generator/render_native_server.go`
- Create: `internal/generator/render_native_server_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 参考旧 `native_server` 的 interface 与 adapter 语义。
- 重写 registration API，注册时只写入 Stage 2 active server slot。

- [x] 生成 `<Service>NativeServer` interface，覆盖 unary 与三类 streaming 方法。
- [x] 生成 Go native adapter，将 interface 方法暴露为 dispatcher native adapter。
- [x] 生成 `Register<Service>GoNativeServer(server)`。
- [x] nil server 必须返回明确错误。
- [x] 后注册 server 只影响后续调用，已启动 stream 保持旧 snapshot。
- [x] 添加 generated source assertion 测试，覆盖 unary 和 streaming 方法签名。
- [x] 添加 compile fixture，证明 generated Go native server 文件可编译。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderNativeServer' -count=1`。
- [x] 验收：Go native server 可以注册成 active server，且不依赖 cgo。
- [x] 提交：`feat: render go native server adapter`

## Task 4：实现 native unary client 到 Go native server

**Files:**

- Create: `internal/generator/render_native_client_cgo.go`
- Create: `internal/generator/render_native_client_cgo_test.go`
- Create: `internal/integration/native_unary_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 参考旧 `native_bridge` 的 unary ABI、error store 和 wrapper 使用顺序。
- 重写 cgo native client export，使调用总是进入 generated dispatcher。

- [x] 生成 cgo native unary client export。
- [x] request native fields 转为 Go request 值。
- [x] Go response 值转为 native response output。
- [x] Go error 存入 `rpcruntime` error store，并通过 `int32` error id 返回。
- [x] 缺少 active server 时返回明确 error id。
- [x] 添加 integration fixture：cgo native client unary 调用 Go native server。
- [x] 添加测试：error propagation、string/bytes wrapper release、repeated wrapper 不泄漏。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestNativeUnary' -count=1`。
- [x] 验收：cgo native client unary 通过 dispatcher 调用 Go native server。
- [x] 提交：`feat: route native unary client to go server`

## Task 5：生成 cgo native server callback ABI

**Files:**

- Create: `internal/generator/render_native_server_cgo.go`
- Create: `internal/generator/render_native_server_cgo_test.go`
- Create: `internal/integration/native_cgo_server_unary_test.go`
- Modify: `internal/generator/render.go`

**迁移内容与理由：**

- 参考旧 `native_bridge_cgo` callback table 形态。
- 重写为 generated cgo native server adapter，注册时写入单 active server slot。

- [x] 生成 cgo native server callback table。
- [x] 生成 `Register<Service>CGONativeServer(callbacks)`。
- [x] nil callback table 或缺失 unary callback 返回明确错误。
- [x] adapter 调用 C callback，并按 error id 传播错误。
- [x] cgo callback request/response 使用 Stage 0 native wrappers。
- [x] 添加 integration fixture：cgo native client unary 调用 cgo native server。
- [x] 添加测试：Go native server 与 cgo native server 后注册互相覆盖后续调用。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestNativeCGOServerUnary|TestNativeUnary' -count=1`。
- [x] 验收：cgo native server 可以注册成 active server，cgo native client 通过 dispatcher 调用它。
- [x] 提交：`feat: render cgo native server callbacks`

## Task 6：接入 native client streaming

**Files:**

- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Create: `internal/integration/native_client_streaming_test.go`

**迁移内容与理由：**

- 参考旧 native streaming 的 Start、Send、Finish、Cancel 调用顺序。
- 不迁旧 stream registry；使用 Stage 2 `StreamHandle` 和 dispatcher stream helpers。

- [x] 生成 client streaming `Start`、`Send`、`Finish`、`Cancel` native ABI。
- [x] `Start` 捕获 active server snapshot 并返回 `rpcruntime.StreamHandle`。
- [x] `Send` 使用 handle 找回 session，并在 send closed/finalized/canceled 后返回明确错误。
- [x] `Finish` terminal 操作使用 typed take，成功后 handle 不再可用。
- [x] `Cancel` 传播到底层 adapter 并 finalize。
- [x] 添加 integration fixture：client streaming 到 Go native server。
- [x] 添加 integration fixture：client streaming 到 cgo native server。
- [x] 运行 `rtk go test ./internal/integration -run 'TestNativeClientStreaming' -count=1`。
- [x] 验收：client streaming native 调用链复用统一 stream lifecycle。
- [x] 提交：`feat: support native client streaming`

## Task 7：接入 native server streaming

**Files:**

- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Create: `internal/integration/native_server_streaming_test.go`

**迁移内容与理由：**

- 参考旧 server streaming onRead/onDone 测试关注点。
- 重写为 generated native session glue，不迁旧 generated runtime 文件结构。

- [x] 生成 server streaming `Start`、`Cancel`、`onRead`、`onDone` native ABI。
- [x] `Start` 返回 `rpcruntime.StreamHandle`。
- [x] `onRead` 通过 handle 读取固定 snapshot session。
- [x] `onDone` 使用 typed take，并执行 terminal cleanup。
- [x] `Cancel` 传播到底层 adapter 并 finalize。
- [x] 添加 integration fixture：server streaming 到 Go native server。
- [x] 添加 integration fixture：server streaming 到 cgo native server。
- [x] 运行 `rtk go test ./internal/integration -run 'TestNativeServerStreaming' -count=1`。
- [x] 验收：server streaming native 调用链在 onDone 后 handle 不再可用。
- [x] 提交：`feat: support native server streaming`

## Task 8：接入 native bidi streaming

**Files:**

- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Create: `internal/integration/native_bidi_streaming_test.go`

**迁移内容与理由：**

- 参考旧 bidi streaming 的 send、close send、onRead、onDone、cancel 覆盖。
- 使用 Stage 2 lifecycle helper 表达 terminal 和 close-send 规则。

- [x] 生成 bidi streaming `Start`、`Send`、`CloseSend`、`Cancel`、`onRead`、`onDone` native ABI。
- [x] `CloseSend` 后继续 `Send` 返回明确错误。
- [x] `onDone` 后所有 handle 操作返回稳定错误或 false。
- [x] `Cancel` 不重复执行 callback。
- [x] 添加 integration fixture：bidi streaming 到 Go native server。
- [x] 添加 integration fixture：bidi streaming 到 cgo native server。
- [x] 运行 `rtk go test ./internal/integration -run 'TestNativeBidiStreaming' -count=1`。
- [x] 验收：bidi native 调用链覆盖 send/read/close/cancel/done 全生命周期。
- [x] 提交：`feat: support native bidi streaming`

## Task 9：Stage 3 acceptance tests 与迁移清单

**Files:**

- Create: `internal/integration/native_stage3_acceptance_test.go`
- Create: `docs/plans/2026-04-28-stage-3-migration-inventory.md`
- Modify: `docs/plans/2026-04-28-stage-3-native-contract-plan.md`

**迁移内容与理由：**

- 迁移旧 native integration 的测试关注点。
- 记录旧 native 代码为什么参考后重写，以及旧 bootstrap 为什么不迁移。

- [x] 添加 Stage 3 acceptance test，覆盖 Go native server 与 cgo native server。
- [x] acceptance test 覆盖 unary、client streaming、server streaming、bidi streaming。
- [x] acceptance test 证明 cgo native client 调用都进入 dispatcher。
- [x] 写入已迁移、参考后重写、不迁移清单。
- [x] 明确旧 native renderer、bridge、cgo runtime 只能参考，不能照搬旧 registry/provider/bootstrap。
- [x] 记录验证命令：generator focused、integration focused、runtime focused、全仓测试、AGENTS.md 中的 forbidden unsigned scan。
- [x] 不记录机器环境处理。
- [x] 更新本计划 checkbox。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`。
- [x] 运行 `rtk go test ./internal/integration -count=1`。
- [x] 运行 `rtk go test ./rpcruntime -count=1`。
- [x] 运行 `rtk go test ./... -count=1`。
- [x] 运行 AGENTS.md 中的 forbidden unsigned scan。
- [x] 验收：阶段 3 “迁移了什么、为什么参考后重写、为什么不迁旧架构”有明确记录。
- [x] 提交：`docs: record stage 3 migration inventory`

## 阶段 3 完成标准

- native-enabled service 生成 Go native server adapter。
- native-enabled service 生成 cgo native server callback adapter。
- cgo native client ABI 总是通过 dispatcher 调用。
- cgo native client 能调用 Go native server。
- cgo native client 能调用 cgo native server。
- native unary、client streaming、server streaming、bidi streaming 都有端到端验证。
- stream handle 统一使用 `rpcruntime.StreamHandle`，底层类型保持 `int32`。
- error id 使用 `int32`。
- 阶段 3 不引入 message converter、connect/grpc adapter、remote adapter 或旧 bootstrap 模型。
- 阶段 3 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无命中。
