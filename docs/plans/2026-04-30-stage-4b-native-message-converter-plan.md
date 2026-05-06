# Stage 4B Native Message Converter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 generated service 级 native/message converter，让 dispatcher 可以在 client contract 与 active server contract 不匹配时完成 protobuf message 与 native field struct 的双向转换。

**Architecture:** Stage 4B 在 Stage 3 native direct path 和 Stage 4A message direct path 之上增加 generated `<service>.codec.rpccgo.go`。converter 只存在于 generated service runtime，负责 native request/response 与 protobuf request/response 的双向转换，并覆盖 unary、client streaming、server streaming、bidi streaming 的 send/read payload。Stage 4B 不重新设计 connect/grpc client，不引入旧多 registry、多 provider bootstrap 或 framework selector。

**Tech Stack:** Go 1.24、cgo、protobuf `protogen`、现有 `internal/generator`、`rpcruntime`、标准库 `testing`。

**Prerequisite:** Stage 4B 必须基于 2026-04-30 Stage 4A dispatcher alignment follow-up 后的 runtime 形态实现：每个 generated service 只有一个 `rpcruntime.Dispatcher[<Service>ActiveAdapter]`，`<Service>ActiveAdapter` 同时持有 Native 与 Message adapter，direct path 由 `snapshot.Contract` 分支决定。Stage 4B 的工作是在当前 converter-disabled mismatch 分支中接入 generated codec，不能恢复双 dispatcher、旧 bootstrap 或 framework selector。

---

## 范围

Stage 4B 聚焦 native/message contract mismatch conversion：

- generated `<service>.codec.rpccgo.go`。
- protobuf message 到 native field struct。
- native field struct 到 protobuf message。
- cgo message client 调 go native server。
- cgo message client 调 cgo native server。
- cgo native client 调 cgo message server。
- native/message mismatch 的 unary、client streaming、server streaming、bidi streaming。
- converter error propagation、owned wrapper release、repeated wrapper 生命周期。
- Stage 4B integration fixture 与迁移清单。

Stage 4B 不实现：

- cgo message client 到 cgo message server direct path 的新语义；该路径由 Stage 4A 完成。
- cgo native client 到 native server direct path 的新语义；该路径由 Stage 3 完成。
- connect handler adapter。
- grpc server adapter。
- connect/grpc remote server adapter。
- 监听 server 启动模型。
- example 业务工程。
- 旧多 registry、多 provider bootstrap、framework selector。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| 旧 `native_codec` | 参考后重写 | 在 protobuf message 与 native field struct 之间做字段转换 | 字段映射、owned wrapper、repeated wrapper、oneof/error 测试关注点可参考；旧实现绑定旧 generator plan 和 unsigned ABI，需要按新版 signed ABI 与 generated service runtime 重写 |
| 旧 `native_bridge` 中 message/native 映射片段 | 参考后重写 | 承接 native request/response 与 protobuf request/response 的转换顺序 | 转换顺序有参考价值；旧 bridge 混入 client/server dispatch，Stage 4B 只生成 codec 和 mismatch 调用 glue |
| 旧 `message_client` mismatch fixture | 迁移测试思路 | 覆盖 cgo message client 调 native server | 端到端场景属于 roadmap Stage 4 验收目标，适合改写为新版 dispatcher + converter fixture |
| 旧 `message_server` mismatch fixture | 迁移测试思路 | 覆盖 cgo native client 调 message server | 端到端场景属于 roadmap Stage 4 验收目标，适合改写为新版 dispatcher + converter fixture |
| 旧 binding/provider/bootstrap/framework selector | 不迁移 | 旧多 registry、多 provider 和 framework 选择模型 | 与新版单 dispatcher、单 active server、单插件 renderer pipeline 冲突，不能进入 Stage 4B |
| 旧 connect/grpc handler adapter | 不迁移 | 本地 RPC transport adapter | Stage 4B 只处理 native/message converter，connect/grpc adapter 留给 Stage 5 |

## 输出模型

Stage 4B 结束后，converter 至少生成以下文件或等价能力：

- `<service>.codec.rpccgo.go`：service-specific protobuf/native 双向转换函数。
- `<service>.runtime.rpccgo.go`：single dispatcher 在 contract mismatch 分支调用 converter。
- `<service>.client.cgo.rpccgo.go`：cgo native client 的 mismatch 调用路径复用 dispatcher。
- `<service>.client.message.cgo.rpccgo.go`：cgo message client 的 mismatch 调用路径复用 dispatcher。
- `<service>.server.message.cgo.rpccgo.go`：cgo message server callback adapter 能被 native client 经 converter 调用。
- `<service>.server.native.rpccgo.go`：Go native server adapter 能被 message client 经 converter 调用。
- `<service>.server.cgo.rpccgo.go`：cgo native server callback adapter 能被 message client 经 converter 调用。

Stage 4B 只处理 contract mismatch conversion。contract 匹配路径仍由 Stage 3 和 Stage 4A direct path 负责，不能为了 converter 重写 direct path 语义。

## Task 1：建立 codec renderer pipeline 与文件布局

**Files:**

- Create: `internal/generator/render_codec.go`
- Create: `internal/generator/render_codec_test.go`
- Modify: `internal/generator/render.go`
- Modify: `internal/generator/render_message_plan.go`
- Modify: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 不迁旧 codec renderer 输出代码。
- 参考旧 `native_codec` 的字段覆盖测试思路，但按新版 `<service>.codec.rpccgo.go` 和 signed ABI 重建。

- [x] 定义 codec generated file family plan，输出 `<service>.codec.rpccgo.go`。
- [x] codec 生成只依赖 service-specific protobuf 类型和 Stage 1 native field plan。
- [x] codec 文件不 import connect/grpc，不接触 remote adapter。
- [x] direct path renderer 不依赖 codec 文件。
- [x] 添加测试：converter-enabled service 输出 codec 文件。
- [x] 添加测试：Stage 4B 不输出 connect/grpc/remote 文件。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -run 'TestRenderCodec|TestGenerate' -count=1`。
- [x] 验收：codec 文件布局与架构文档一致，且没有引入旧 framework selector。
- [x] 提交：`feat: add native message codec renderer`

## Task 2：生成 protobuf message 到 native field struct 转换

**Files:**

- Modify: `internal/generator/render_codec.go`
- Modify: `internal/generator/render_codec_test.go`
- Create: `internal/integration/codec_message_to_native_test.go`

**迁移内容与理由：**

- 参考旧 `native_codec` 的 scalar、string、bytes、repeated wrapper 转换关注点。
- 重写为 generated service-specific converter，不放入 `rpcruntime`。

- [x] 生成 request protobuf message 到 native request struct 的转换函数。
- [x] 生成 response protobuf message 到 native response struct 的转换函数。
- [x] string、bytes 使用 Stage 0 owned wrapper，并明确 release 责任。
- [x] repeated scalar 使用对应 `RpcRepeat` wrapper。
- [x] repeated bool 使用 byte 编码 wrapper，不使用 Go `[]bool` 作为 C ABI 表示。
- [x] nil message、非法字段状态和 wrapper 构造失败返回明确错误。
- [x] 添加 codec focused test 覆盖 scalar、string、bytes、repeated、repeated bool。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestCodecMessageToNative|TestRenderCodec' -count=1`。
- [x] 验收：protobuf message 可以稳定转换为 native field struct，且释放责任清晰。
- [x] 提交：`feat: render message to native codec`

## Task 3：生成 native field struct 到 protobuf message 转换

**Files:**

- Modify: `internal/generator/render_codec.go`
- Modify: `internal/generator/render_codec_test.go`
- Create: `internal/integration/codec_native_to_message_test.go`

**迁移内容与理由：**

- 参考旧 `native_codec` 的 native wrapper 读取和 protobuf 赋值顺序。
- 重写为 generated service-specific converter，避免把 protobuf 类型引入 `rpcruntime`。

- [x] 生成 native request struct 到 protobuf request message 的转换函数。
- [x] 生成 native response struct 到 protobuf response message 的转换函数。
- [x] string、bytes wrapper 读取失败返回明确错误。
- [x] repeated wrapper 读取失败返回明确错误。
- [x] repeated bool byte 编码解码错误返回明确错误。
- [x] protobuf marshal 前保持 message 类型完整，marshal error 直接传播。
- [x] 添加 codec focused test 覆盖 scalar、string、bytes、repeated、repeated bool。
- [x] 运行 `rtk go test ./internal/generator ./internal/integration -run 'TestCodecNativeToMessage|TestRenderCodec' -count=1`。
- [x] 验收：native field struct 可以稳定转换为 protobuf message，且不泄漏 wrapper 生命周期。
- [x] 提交：`feat: render native to message codec`

## Task 4：dispatcher mismatch glue：message client 调 Go native server

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_native_server.go`
- Create: `internal/integration/message_client_to_go_native_test.go`

**迁移内容与理由：**

- 不迁旧 bridge dispatch 代码。
- 复用 Stage 3 Go native server adapter 和 Stage 4A message client ABI，在 dispatcher mismatch 分支调用 generated codec。

- [x] message unary request bytes unmarshal 后转换为 native request。
- [x] native response 转换为 protobuf response bytes 返回给 message client。
- [x] converter error、native server error、marshal error 都返回明确 `int32` error id。
- [x] client streaming 的 `Send` payload 从 protobuf request 转 native request。
- [x] client streaming 的 `Finish` payload 从 native response 转 protobuf response。
- [x] server streaming 与 bidi streaming 的 read payload 从 native response 转 protobuf response。
- [x] 添加 integration fixture：cgo message client 调 Go native server，覆盖 unary 与三类 streaming。
- [x] 运行 `rtk go test ./internal/integration -run 'TestMessageClientToGoNative' -count=1`。
- [x] 验收：cgo message client 可以通过 dispatcher + converter 调 Go native server。
- [x] 提交：`feat: route message client to go native server`

## Task 5：dispatcher mismatch glue：message client 调 cgo native server

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Create: `internal/integration/message_client_to_cgo_native_test.go`

**迁移内容与理由：**

- 参考旧 message/native mismatch fixture 的端到端场景。
- 重写为 dispatcher mismatch branch，不绕过 active server snapshot。

- [x] message unary request bytes unmarshal 后转换为 cgo native server request wrapper。
- [x] cgo native server response wrapper 转换为 protobuf response bytes。
- [x] cgo native callback error id、converter error、marshal error 都稳定传播。
- [x] streaming `Start` 捕获 cgo native server adapter snapshot。
- [x] streaming send/read payload 全部经过 generated codec。
- [x] terminal cleanup 释放 converter 创建的 owned wrapper。
- [x] 添加 integration fixture：cgo message client 调 cgo native server，覆盖 unary 与三类 streaming。
- [x] 运行 `rtk go test ./internal/integration -run 'TestMessageClientToCGONative' -count=1`。
- [x] 验收：cgo message client 可以通过 dispatcher + converter 调 cgo native server。
- [x] 提交：`feat: route message client to cgo native server`

## Task 6：dispatcher mismatch glue：native client 调 cgo message server

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Create: `internal/integration/native_client_to_cgo_message_test.go`

**迁移内容与理由：**

- 参考旧 native client 调 message server 的 fixture 语义。
- 重写为 dispatcher mismatch branch，server adapter 仍是 Stage 4A cgo message server adapter。

- [x] native unary request 转换为 protobuf request bytes 后调用 cgo message server。
- [x] cgo message server response bytes 转换为 native response wrapper。
- [x] converter error、message callback error、unmarshal error 都返回明确 `int32` error id。
- [x] client streaming 的 `Send` payload 从 native request 转 protobuf request。
- [x] client streaming 的 `Finish` payload 从 protobuf response 转 native response。
- [x] server streaming 与 bidi streaming 的 read payload 从 protobuf response 转 native response。
- [x] 添加 integration fixture：cgo native client 调 cgo message server，覆盖 unary 与三类 streaming。
- [x] 运行 `rtk go test ./internal/integration -run 'TestNativeClientToCGOMessage' -count=1`。
- [x] 验收：cgo native client 可以通过 dispatcher + converter 调 cgo message server。
- [x] 提交：`feat: route native client to cgo message server`

## Task 7：converter lifecycle、错误语义与 active server snapshot 覆盖

**Files:**

- Modify: `internal/generator/render_codec.go`
- Modify: `internal/generator/render_runtime.go`
- Create: `internal/integration/converter_lifecycle_test.go`
- Create: `internal/integration/converter_snapshot_test.go`

**迁移内容与理由：**

- 迁移旧 wrapper lifecycle、stream cancel/finalize、error propagation 的测试关注点。
- 不迁旧 stream registry；使用 Stage 2 `StreamHandle` 和 dispatcher stream helpers。

- [x] converter 创建的 owned string/bytes wrapper 在成功、错误、cancel、finish 后都有明确释放路径。
- [x] repeated wrapper 读取和释放覆盖成功与失败路径。
- [x] stream `Start` 后重新注册 active server 不影响当前 session 的 converter 方向。
- [x] converter error 不调用下游 server callback。
- [x] 下游 server error 不被 converter 覆盖。
- [x] `Cancel` 传播到底层 adapter 并 finalize converter session state。
- [x] 添加测试：unary 与三类 streaming 的 snapshot、cancel、finalize、error precedence。
- [x] 运行 `rtk go test ./internal/integration -run 'TestConverterLifecycle|TestConverterSnapshot' -count=1`。
- [x] 验收：converter 生命周期与 Stage 2/3/4A stream lifecycle 一致。
- [x] 提交：`test: cover converter lifecycle and snapshot semantics`

## Task 8：Stage 4B acceptance tests 与迁移清单

**Files:**

- Create: `internal/integration/converter_stage4b_acceptance_test.go`
- Create: `docs/plans/2026-04-30-stage-4b-migration-inventory.md`
- Modify: `docs/plans/2026-04-30-stage-4b-native-message-converter-plan.md`

**迁移内容与理由：**

- 迁移旧 native/message codec 与 mismatch integration 的测试关注点。
- 记录旧 codec 代码为什么参考后重写，以及旧 bootstrap、framework selector 和 connect/grpc adapter 为什么不进入 Stage 4B。

- [x] 添加 Stage 4B acceptance test，覆盖 cgo message client 调 Go native server。
- [x] 添加 Stage 4B acceptance test，覆盖 cgo message client 调 cgo native server。
- [x] 添加 Stage 4B acceptance test，覆盖 cgo native client 调 cgo message server。
- [x] acceptance test 覆盖 unary、client streaming、server streaming、bidi streaming。
- [x] acceptance test 证明 mismatch 调用都进入 dispatcher 并经过 generated codec。
- [x] 写入已迁移、参考后重写、不迁移清单。
- [x] 明确旧多 registry、多 provider bootstrap、framework selector 不迁移。
- [x] 明确 Stage 4B 不实现 connect/grpc local adapter 或 remote adapter。
- [x] 记录验证命令：generator focused、integration focused、runtime focused、全仓测试、AGENTS.md 中的 forbidden unsigned scan。
- [x] 不记录机器环境处理。
- [x] 更新本计划 checkbox。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`。
- [x] 运行 `rtk go test ./internal/integration -count=1`。
- [x] 运行 `rtk go test ./rpcruntime -count=1`。
- [x] 运行 `rtk go test ./... -count=1`。
- [x] 运行 AGENTS.md 中的 forbidden unsigned scan。
- [x] 验收：阶段 4B “迁移了什么、为什么参考后重写、为什么不迁旧架构和 connect/grpc adapter”有明确记录。
- [x] 提交：`docs: record stage 4b migration inventory`

## Stage 4B 完成标准

- generated `<service>.codec.rpccgo.go` 覆盖 protobuf message 与 native field struct 双向转换。
- converter 覆盖 request、response 和 streaming send/read payload。
- cgo message client 能通过 dispatcher 调用 Go native server。
- cgo message client 能通过 dispatcher 调用 cgo native server。
- cgo native client 能通过 dispatcher 调用 cgo message server。
- native/message mismatch 的 unary、client streaming、server streaming、bidi streaming 都有端到端验证。
- contract 匹配路径继续由 Stage 3 和 Stage 4A direct path 负责。
- stream handle 统一使用 `rpcruntime.StreamHandle`，底层类型保持 `int32`。
- error id 使用 `int32`。
- converter 不进入 `rpcruntime`，不依赖 connect/grpc。
- Stage 4B 不引入 connect/grpc adapter、remote adapter 或旧 bootstrap 模型。
- Stage 4B 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无命中。
