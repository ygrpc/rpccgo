# Stage 1 Proto Parser and Planning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## 目标

阶段 1 建立新版 rpccgo 的 protoc 输入解析与生成计划层。阶段结束后，generator 能从 proto descriptor 构建稳定、可测试的 `FilePlan`、`ServicePlan`、`MethodPlan`、`FieldPlan`、native/message contract 和 streaming lifecycle plan，为后续 dispatcher、converter、adapter renderer 提供统一输入。

阶段 1 只产出中间 plan，不生成最终 service runtime、dispatcher、cgo ABI、connect/grpc adapter 或 example 业务代码。

## 架构边界

- 单一 protobuf 插件：`protoc-gen-rpc-cgo`。
- 插件内部拆分 parser、planner、renderer；阶段 1 只实现 parser/planner。
- `@rpccgo` service 注释只控制 server adapter 生成选择，不控制 cgo native client 或 cgo message client。
- `native` 单独出现等价于 `msg-connect|native`。
- 未标注 service 默认等价于 `msg-connect`。
- unknown token 和常见拼写错误必须报错，不能静默忽略。
- 生成计划只描述能力和 contract，不持有 active server，不做 runtime dispatch。

## 旧项目迁移判定

| 旧项目文件 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| `internal/generator/streaming_kind.go` | 直接迁移并扩展测试 | 识别 unary、client streaming、server streaming、bidi streaming | 逻辑纯粹、稳定，和新版架构完全一致 |
| `internal/generator/render_plan.go` | 择优迁移 lifecycle 与 IO plan | 描述 streaming 操作矩阵和终态语义 | lifecycle 规则已经贴近新版 streaming 合同，迁移可避免重复踩 Finish/CloseSend/onDone 矩阵 |
| `internal/generator/streaming_plan.go` | 择优迁移核心思路 | 为非 unary method 生成 streaming plan | 旧结构可作为 starting point，但需要改成 service-level `MethodPlan` 的组成部分 |
| `internal/generator/frameworks.go` | 参考后重写 | 旧 connect/grpc framework token 解析 | 旧模型包含 historical native framework 概念，新版应使用 `msg-connect`、`msg-grpc`、`native` adapter token |
| `internal/generator/options.go` | 参考后重写 | protoc 参数解析与默认值 | 旧 options 绑定 mode/go_role/framework，不符合新版单插件和 service 注释模型 |
| `internal/generator/namespace.go` | 迁移命名 helper | lower/upper、file/service/method 命名 | 命名工具纯函数可复用，但生成文件布局需按新版文件族重新定义 |
| `internal/generator/native_types.go` | 参考后重写 | proto field 到 native ABI 类型分类 | 旧实现包含新版禁止的 unsigned 32/64 ABI，需要按 Stage 0 runtime 约束重建 |
| `internal/generator/native_codec.go` | 只参考 | native/message 转换 renderer | 阶段 1 只需要 contract metadata，不生成 codec code |
| `internal/generator/binding.go` | 不迁移 | 旧多 registry binding | 与新版 active server slot + dispatcher 架构冲突，只能作为不要迁移的旧模型 |
| `internal/generator/message_server.go` 等 renderer | 不迁移 | 旧 message/native renderer | 阶段 1 不实现 renderer，后续阶段按新版 `ServicePlan` 分 renderer 重建 |

## 输出模型

阶段 1 结束后至少包含以下 plan 类型：

- `FilePlan`
- `ServicePlan`
- `MethodPlan`
- `FieldPlan`
- `AdapterSelection`
- `NativeContractPlan`
- `MessageContractPlan`
- `StreamingPlan`
- `LifecyclePlan`
- `MethodIOPlan`

`ServicePlan` 必须能回答：

- service 名称、Go 名称、proto full name。
- 该 service 启用哪些 server adapter。
- 每个 method 的 streaming kind。
- 每个 method 的 request/response message 类型。
- 每个 request/response field 的 message contract 和 native contract。
- method 是否需要 native/message 转换。
- 非 unary method 的 lifecycle 操作矩阵。

## Task 1：建立 generator plan 基础包

**Files:**

- Create: `internal/generator/names.go`
- Create: `internal/generator/streaming_kind.go`
- Create: `internal/generator/plan.go`
- Create: `internal/generator/plan_test.go`
- Modify: `go.mod`

**迁移内容与理由：**

- 从旧 `streaming_kind.go` 迁移 `StreamingKind` 与 `StreamingKindOf`，因为它是纯 descriptor 逻辑。
- 从旧 `namespace.go` 迁移基础命名 helper，避免后续 renderer 重复命名规则。
- 新建 plan struct，先不依赖 protogen renderer。

- [x] 定义 `StreamingKind`、`AdapterSelection`、`FilePlan`、`ServicePlan`、`MethodPlan`、`FieldPlan`、`LifecyclePlan`、`MethodIOPlan`。
- [x] 定义 adapter token 常量：`msg-connect`、`msg-grpc`、`native`。
- [x] 保持 plan 类型不依赖 `rpcruntime`，只依赖 protobuf descriptor/protogen 必要类型。
- [x] 添加基础单元测试：streaming kind、命名 helper、plan 零值不可被误认为有效 plan。
- [x] 运行 `rtk go test ./internal/generator -run 'TestStreamingKind|TestNames|TestPlan' -count=1`。
- [x] 验收：基础 plan 类型和 streaming kind 测试通过。
- [x] 提交：`feat: add generator plan primitives`

## Task 2：实现 `@rpccgo` service 注释解析

**Files:**

- Create: `internal/generator/service_options.go`
- Create: `internal/generator/service_options_test.go`

**迁移内容与理由：**

- 参考旧 `frameworks.go` 的 token 去重、排序和错误提示方式。
- 不迁旧 `FrameworkNative` 模型，改成新版 server adapter token。

- [x] 从 service leading comments 中提取 `@rpccgo:` 指令。
- [x] 未标注 service 默认生成 `msg-connect`。
- [x] 支持 `msg-connect`、`msg-grpc`、`msg-connect|msg-grpc`、`msg-connect|native`、`msg-connect|msg-grpc|native`。
- [x] `native` 单独出现展开为 `msg-connect|native`。
- [x] token 去重，并按 canonical order 输出。
- [x] unknown token 报错，并包含合法 token 提示。
- [x] `msg-conenct` 等拼写错误必须报错。
- [x] 空指令、重复冒号、多条互相冲突指令必须报错。
- [x] 运行 `rtk go test ./internal/generator -run 'TestParseServiceRPCCGO|TestAdapterSelection' -count=1`。
- [x] 验收：注释规则完全匹配架构文档。
- [x] 提交：`feat: parse rpccgo service annotations`

## Task 3：实现 protoc 插件入口与 request 解析壳

**Files:**

- Create: `cmd/protoc-gen-rpc-cgo/main.go`
- Create: `internal/generator/generator.go`
- Create: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 参考旧 `generator.go` 的 protogen 调用方式。
- 不迁旧 options 的 mode/go_role/framework 模型，阶段 1 只保留单插件必要参数。

- [x] 实现 `protoc-gen-rpc-cgo` main，读取 `protogen.Options` 并调用 generator。
- [x] `internal/generator.Generate` 能接收 protogen plugin，并返回 plan 或错误。
- [x] 支持 protoc 标准 `paths` 参数透传。
- [x] 对未知 rpccgo 参数报错。
- [x] 阶段 1 不输出最终 generated runtime 文件；可输出空结果或 plan-only 测试入口。
- [x] 构造内存 descriptor fixture 测试 request parsing。
- [x] 运行 `rtk go test ./cmd/protoc-gen-rpc-cgo ./internal/generator -run 'TestGenerate|TestPlugin' -count=1`。
- [x] 验收：插件入口可编译，request parsing 可测试。
- [x] 提交：`feat: add protoc plugin parser shell`

## Task 4：构建 service/method descriptor metadata

**Files:**

- Create: `internal/generator/descriptor_plan.go`
- Create: `internal/generator/descriptor_plan_test.go`

**迁移内容与理由：**

- 参考旧 `binding.go` 的 `ServiceMethodBinding` 字段命名，但不迁 registry 和 resolver。
- 复用 Task 1 的 streaming kind。

- [x] 从 protogen file 构建 `FilePlan`。
- [x] 为每个 service 构建 `ServicePlan`。
- [x] 为每个 method 构建 `MethodPlan`，包括 request/response Go identifier、proto full name、streaming kind。
- [x] 保持 service 顺序和 method 顺序稳定。
- [x] 覆盖 unary、client streaming、server streaming、bidi streaming。
- [x] 覆盖多 service proto。
- [x] 运行 `rtk go test ./internal/generator -run 'TestBuildDescriptorPlan|TestMethodStreamingPlan' -count=1`。
- [x] 验收：descriptor metadata 稳定可断言。
- [x] 提交：`feat: build descriptor service plans`

## Task 5：构建 native/message contract field plan

**Files:**

- Create: `internal/generator/contract_plan.go`
- Create: `internal/generator/contract_plan_test.go`

**迁移内容与理由：**

- 参考旧 `native_types.go` 的分类思路。
- 重写 native 类型映射，遵守 Stage 0 的跨语言兼容约束，不生成 unsigned 32/64 ABI。
- message contract 只记录 protobuf message 类型和 marshal/unmarshal 需求。

- [x] 为 request/response message field 构建 `FieldPlan`。
- [x] 字段 plan 记录 proto 名称、Go 名称、kind、是否 repeated、是否 enum、是否 message。
- [x] native contract 支持 signed numeric、float、bool byte encoding、string、bytes、message-as-bytes、enum。
- [x] 遇到 unsupported native field 时在 plan 构建阶段报错，错误包含 service/method/field。
- [x] repeated bool 明确标记为 byte buffer wrapper。
- [x] repeated message 若阶段 1 暂不支持，必须给出明确错误。
- [x] message contract 记录 request/response protobuf 类型，不做 codegen。
- [x] 运行 `rtk go test ./internal/generator -run 'TestBuildContractPlan|TestNativeFieldPlan' -count=1`。
- [x] 验收：native/message contract 信息足够支撑后续 converter 和 ABI renderer。
- [x] 提交：`feat: build native and message contract plans`

## Task 6：构建 streaming lifecycle plan

**Files:**

- Create: `internal/generator/streaming_plan.go`
- Create: `internal/generator/streaming_plan_test.go`

**迁移内容与理由：**

- 从旧 `render_plan.go`、`streaming_plan.go` 迁移 lifecycle 矩阵和验证逻辑。
- 将旧 native/message 分离的 streaming plan 收敛到 `MethodPlan.Streaming`，保证后续 dispatcher 使用同一生命周期。

- [x] unary method 不生成 streaming lifecycle。
- [x] client streaming 包含 `Start`、`Send`、`Finish`、`Cancel`，终态为 finish result。
- [x] server streaming 包含 `Start`、`Cancel`、`onRead`、`onDone`，终态为 onDone。
- [x] bidi streaming 包含 `Start`、`Send`、`CloseSend`、`Cancel`、`onRead`、`onDone`，终态为 onDone。
- [x] `Cancel` 必须标记为 finalizes。
- [x] 验证非法 lifecycle matrix 会报错。
- [x] 运行 `rtk go test ./internal/generator -run 'TestBuildStreamingPlan|TestValidateStreamingLifecyclePlan' -count=1`。
- [x] 验收：四类 method 的 lifecycle 与架构文档一致。
- [x] 提交：`feat: build streaming lifecycle plans`

## Task 7：整合 ServicePlan 与阶段 1 验收 fixture

**Files:**

- Modify: `internal/generator/generator.go`
- Modify: `internal/generator/descriptor_plan.go`
- Modify: `internal/generator/contract_plan.go`
- Create: `internal/generator/testdata/` fixtures as needed
- Create: `internal/generator/stage1_acceptance_test.go`

**迁移内容与理由：**

- 参考旧 `generator_generate_test.go` 的 fixture 组织方式。
- 不迁旧生成 golden，因为阶段 1 验收对象是 plan，不是 rendered Go code。

- [x] 将 service 注释解析、descriptor metadata、contract plan、streaming lifecycle 汇总成完整 `ServicePlan`。
- [x] 覆盖未标注 service 默认 `msg-connect`。
- [x] 覆盖 `msg-connect`、`msg-grpc`、两种 message adapter 同开、message + native、message + native 全开。
- [x] 覆盖 `native` 单独出现展开默认 message adapter。
- [x] 覆盖 unknown token 和拼写错误。
- [x] 覆盖 unary 和三类 streaming。
- [x] 覆盖 native/message contract 不匹配时后续需要 converter 的标记。
- [x] 确认 plan 构建不输出 dispatcher、renderer、adapter 或 example 业务代码。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo`。
- [x] 运行 `rtk go test ./...`。
- [x] 验收：Stage 1 acceptance tests 全部通过。
- [x] 提交：`test: verify stage 1 service planning`

## Task 8：记录阶段 1 迁移清单与验证结果

**Files:**

- Create: `docs/plans/2026-04-28-stage-1-migration-inventory.md`
- Modify: `docs/plans/2026-04-28-stage-1-proto-planning-plan.md`

**迁移内容与理由：**

- 阶段 1 会从旧 generator 迁移一批纯 parser/planner 逻辑，也会明确拒绝旧 registry/provider 代码。
- 迁移清单用于后续阶段防止 renderer 实现时回流旧架构。

- [x] 写入已迁移、参考后重写、不迁移清单。
- [x] 明确旧 `binding.go`、旧 framework registry、旧 provider bootstrap 不进入新版。
- [x] 记录验证命令：`rtk go test ./...`。
- [x] 记录 forbidden unsigned ABI token 扫描结果。
- [x] 运行 `rtk git status --short`。
- [x] 验收：文档能回答阶段 1 “迁移了什么、为什么迁移、为什么不迁旧架构”。
- [x] 提交：`docs: record stage 1 migration inventory`

## 阶段 1 完成标准

- `protoc-gen-rpc-cgo` 入口可编译。
- generator 能从 proto descriptor 构建完整 `ServicePlan`。
- `@rpccgo` 注释规则与架构文档一致。
- 未标注 service 默认生成 `msg-connect`。
- `native` 单独出现会展开为 `msg-connect|native`。
- unknown token 和拼写错误会报错。
- unary、client streaming、server streaming、bidi streaming 都被正确识别。
- native/message contract plan 足够支撑后续 converter、dispatcher 和 adapter renderer。
- 阶段 1 不生成最终 runtime、dispatcher、adapter 或 example 业务代码。
- 旧多 registry、多 provider bootstrap 模型没有被迁入。
- `rtk go test ./...` 通过。
