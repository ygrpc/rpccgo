# Stage 2 Runtime Dispatcher Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 service 无关的 runtime dispatcher foundation，让后续 generated service 可以安全注册 active server、捕获 adapter snapshot、分发 unary 调用并管理 streaming session 生命周期。

**Architecture:** 阶段 2 只在 `rpcruntime` 中实现通用 primitive，不依赖 protobuf service 类型，不生成 cgo ABI、converter、connect/grpc adapter 或业务 example。generated service 后续会用这些 primitive 组装 service-specific dispatcher、adapter、converter 和注册 API。

**Tech Stack:** Go 1.24、标准库 `context` / `sync` / `sync/atomic`、`testing`、现有 `rpcruntime`。

---

## 范围

阶段 2 聚焦 runtime foundation：

- active server slot primitive。
- server adapter metadata 与 snapshot primitive。
- dispatcher shell primitive。
- stream handle allocator 与 session table。
- stream session finalization、cancel、terminal state 语义。
- runtime foundation 的并发测试和迁移清单。

阶段 2 不实现：

- service-specific protobuf converter。
- generated cgo native client 或 cgo message client ABI。
- go native、cgo native、cgo message、connect、grpc adapter renderer。
- connect/grpc 监听入口。
- remote connect/grpc server adapter。
- example 业务代码。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| `rpcruntime/active_slot.go` | 不迁移实现，只记录结论 | 旧文件为空，没有可复用实现 | 新版 active server slot 必须重建；旧文件只能证明阶段 2 需要补这个 runtime primitive |
| 旧 generated stream registry runtime | 参考后重写 | 为 generated streaming 调用分配 handle、保存 session、按 handle 查找和删除 | handle/session 表的生命周期思路可复用，但旧实现绑定 generated 输出、旧 handle 类型和旧 native/message 分裂模型，必须按新版 signed handle 与 service-agnostic runtime 重写 |
| `internal/generator/message_streaming_render.go` 的 session 状态机片段 | 参考后重写 | 处理 send、close send、cancel、onRead、onDone 的状态转换 | 状态语义有参考价值，但旧代码是 renderer 字符串输出且绑定 connect/grpc 适配细节；阶段 2 只抽取 runtime session 终态和 cancel/finalize 规则 |
| 旧 streaming lifecycle 测试思路 | 迁移测试思路 | 覆盖 unknown handle、重复 cancel、close send 后 send、terminal cleanup | 这些行为是 runtime foundation 的核心质量门槛，适合用新版 API 重新写测试 |
| 旧 registry/provider/bootstrap 代码 | 不迁移 | 支撑历史多 provider、多 registry、多 bootstrap 选择 | 与新版单 dispatcher、单 active server slot 约束冲突，不能带入阶段 2 |
| 旧 connect/grpc/native renderer | 不迁移 | 生成旧 adapter 和 cgo 调用路径 | 阶段 2 不做 renderer；后续 renderer 必须从 Stage 1 `ServicePlan` 和本阶段 runtime primitive 出发重建 |

## 输出模型

阶段 2 结束后至少包含以下 runtime 类型或等价能力：

- `ServerContract`：标识 adapter 接收 native 或 message contract。
- `ServerKind`：标识 active server 类型。
- `AdapterSnapshot[T]`：保存 server kind、contract、version、adapter。
- `ActiveServerSlot[T]`：注册并读取当前 active server snapshot。
- `Dispatcher[T]`：封装 active slot，并提供 snapshot capture 入口。
- `StreamHandle`：signed stream handle。
- `StreamRegistry[T]`：分配 handle、保存 session、load、delete、finalize。
- `StreamSessionState`：标识 active、closed send、finalized、canceled 等状态。
- `StreamFinalizer` / `CancelFunc` 等通用 callback primitive。

这些类型必须保持 service-agnostic：可以使用 generic `T` 表示 generated adapter，但不能 import protobuf、connect、grpc 或 `internal/generator`。

## Task 1：定义 runtime dispatcher contract primitive

**Files:**

- Create: `rpcruntime/dispatcher_contract.go`
- Create: `rpcruntime/dispatcher_contract_test.go`

**迁移内容与理由：**

- 不从旧项目直接迁移。旧项目没有独立 dispatcher contract primitive。
- 从架构文档提取 server kind、contract kind 和 snapshot 语义，作为后续 active slot、dispatcher、stream session 的共同语言。

- [ ] 定义 `ServerContract`，至少包含 native 与 message 两种 contract。
- [ ] 定义 `ServerKind`，覆盖 go native、cgo native、cgo message、connect handler、grpc server、connect remote、grpc remote。
- [ ] 定义 `AdapterSnapshot[T any]`，包含 server kind、server contract、version、adapter。
- [ ] 定义 `HasAdapter()` 或等价方法，确保 zero snapshot 不会被误认为有效 active server。
- [ ] 添加测试：zero snapshot invalid、native/message contract 可区分、所有 server kind 有稳定字符串表示。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestServerContract|TestServerKind|TestAdapterSnapshot' -count=1`。
- [ ] 验收：runtime contract primitive 不依赖 protobuf、connect、grpc 或 generator。
- [ ] 提交：`feat: add runtime dispatcher contracts`

## Task 2：实现 active server slot

**Files:**

- Create: `rpcruntime/active_slot.go`
- Create: `rpcruntime/active_slot_test.go`

**迁移内容与理由：**

- 旧 `rpcruntime/active_slot.go` 是空文件，不迁移实现。
- 参考旧架构中“注册覆盖后续调用”的目标语义，但按新版单 active server slot 重写。

- [ ] 定义 `ActiveServerSlot[T any]`。
- [ ] 实现 `Store(kind ServerKind, contract ServerContract, adapter T) (AdapterSnapshot[T], error)`。
- [ ] 实现 `Load() (AdapterSnapshot[T], bool)`。
- [ ] 每次成功注册递增 signed version，后注册覆盖后续 `Load`。
- [ ] nil adapter 或 zero adapter 必须报错；错误消息不依赖 service-specific 类型。
- [ ] `Load` 返回 snapshot value，后续注册不能改变旧 snapshot 中的 adapter 与 version。
- [ ] 添加并发测试：多 goroutine store/load 不 panic、不返回 zero snapshot；最终 snapshot 是某个成功注册结果。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestActiveServerSlot' -count=1`。
- [ ] 验收：单个 service 可以注册一个 active server，后注册 server 覆盖后续调用，旧 snapshot 保持稳定。
- [ ] 提交：`feat: add active server slot`

## Task 3：实现 dispatcher shell

**Files:**

- Create: `rpcruntime/dispatcher.go`
- Create: `rpcruntime/dispatcher_test.go`

**迁移内容与理由：**

- 不迁旧 binding、registry 或 provider bootstrap。
- 新 dispatcher shell 只封装 active slot 和 snapshot capture，不执行 service-specific converter。

- [ ] 定义 `Dispatcher[T any]`，内部持有 `ActiveServerSlot[T]`。
- [ ] 实现 `Register(kind ServerKind, contract ServerContract, adapter T) (AdapterSnapshot[T], error)`。
- [ ] 实现 `Capture() (AdapterSnapshot[T], error)`，没有 active server 时返回明确错误。
- [ ] 实现 `Invoke(ctx context.Context, invoke func(context.Context, AdapterSnapshot[T]) error) error`，用于 unary 调用捕获 snapshot 后执行 adapter。
- [ ] 确认 `Invoke` 在调用开始时只 capture 一次；执行期间重新注册不影响当前调用。
- [ ] 添加测试：未注册时报错、注册后可调用、后注册影响新调用、不影响已 capture 调用。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestDispatcher' -count=1`。
- [ ] 验收：dispatcher primitive 能证明 unary 调用使用启动时捕获的 active server snapshot。
- [ ] 提交：`feat: add runtime dispatcher shell`

## Task 4：实现 stream handle allocator 与 session registry

**Files:**

- Create: `rpcruntime/stream_registry.go`
- Create: `rpcruntime/stream_registry_test.go`

**迁移内容与理由：**

- 参考旧 generated stream registry 的 handle 分配、store、load、delete 思路。
- 不迁旧 generated 代码，因为旧实现绑定 generated 文件、旧 handle 类型和 native/message 双 runtime。

- [ ] 定义 signed `StreamHandle`。
- [ ] 定义 `StreamRegistry[T any]`。
- [ ] 实现 `Create(session T) (StreamHandle, error)`，返回非零 handle。
- [ ] 实现 `Load(handle StreamHandle) (T, bool)`。
- [ ] 实现 `Delete(handle StreamHandle) bool`。
- [ ] 实现 `Take(handle StreamHandle) (T, bool)`，用于 terminal 操作原子取出 session。
- [ ] nil session 或 zero session 必须报错。
- [ ] handle 分配 wrap 时不得返回 zero；如果耗尽，返回明确错误。
- [ ] 添加测试：create/load/delete/take、unknown handle、重复 delete、并发 create 产生唯一 non-zero signed handle。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestStreamRegistry|TestStreamHandle' -count=1`。
- [ ] 验收：stream session table 是 service-agnostic，并使用 signed handle。
- [ ] 提交：`feat: add stream session registry`

## Task 5：实现 stream session 状态与 finalization helper

**Files:**

- Create: `rpcruntime/stream_session.go`
- Create: `rpcruntime/stream_session_test.go`
- Modify: `rpcruntime/stream_registry.go`

**迁移内容与理由：**

- 参考旧 streaming renderer 的 send、close send、cancel、terminal cleanup 测试思路。
- 重写为 runtime session helper，不绑定 request/response protobuf 类型。

- [ ] 定义 `StreamSession` 或 `StreamLifecycle` helper，管理 active、send closed、finalized、canceled 状态。
- [ ] 实现 `MarkSendClosed() error`，重复 close send 报错。
- [ ] 实现 `EnsureCanSend() error`，send closed、finalized、canceled 后报错。
- [ ] 实现 `Finalize() bool`，只允许第一次 terminal 操作成功。
- [ ] 实现 `Cancel(cancel func() error) error`，调用 cancel 后进入 terminal 状态。
- [ ] 将 `StreamRegistry.Take` 与 finalization helper 的使用方式写入测试，证明 terminal 操作后 handle 不再可用。
- [ ] 添加测试：send after close、double close、cancel finalizes、finish finalizes、onDone finalizes、double terminal 操作只生效一次。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestStreamSession|TestStreamRegistry' -count=1`。
- [ ] 验收：client streaming、server streaming、bidi streaming 的 cancel/finalize 规则可以由同一 runtime helper 表达。
- [ ] 提交：`feat: add stream session lifecycle helpers`

## Task 6：证明 stream Start 捕获 active server snapshot

**Files:**

- Create: `rpcruntime/dispatcher_stream_test.go`
- Modify: `rpcruntime/dispatcher.go`

**迁移内容与理由：**

- 不迁旧 generated Start/Send/Finish/Cancel 代码。
- 迁移旧 streaming 测试关注点：Start 时捕获 adapter，后续 handle 操作固定路由到该 snapshot。

- [ ] 在 `Dispatcher[T]` 中提供 `StartStream(create func(AdapterSnapshot[T]) (session any, err error))` 或等价 helper；helper 必须先 capture snapshot，再创建 session。
- [ ] 将创建出的 session 存入 `StreamRegistry`，返回 signed handle。
- [ ] 测试：注册 server A，Start stream，注册 server B，随后通过 handle 取出的 session 仍绑定 server A snapshot。
- [ ] 测试：Start 时没有 active server 返回错误且不分配 handle。
- [ ] 测试：Start 创建 session 失败时不泄漏 handle。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestDispatcherStream' -count=1`。
- [ ] 验收：已启动 stream 绑定启动时捕获的 server adapter，后续注册只影响新 stream。
- [ ] 提交：`feat: capture active server for streams`

## Task 7：runtime foundation acceptance tests

**Files:**

- Create: `rpcruntime/dispatcher_foundation_test.go`
- Modify: existing `rpcruntime/*_test.go` as needed

**迁移内容与理由：**

- 不迁旧 integration bootstrap。
- 用新的 runtime primitive 写 service-agnostic acceptance fixture，证明阶段 2 完成标准。

- [ ] 构造 fake adapter 类型，不依赖 protobuf、connect、grpc。
- [ ] 验证单个 service dispatcher 可以注册一个 active server。
- [ ] 验证后注册 server 覆盖后续 unary 调用。
- [ ] 验证已启动 stream 固定使用启动时 snapshot。
- [ ] 验证 unknown stream handle、double terminal、cancel finalizes 都返回明确错误或稳定 bool。
- [ ] 验证 runtime foundation package 没有 import protobuf、connect、grpc、internal/generator。
- [ ] 运行 `rtk go test ./rpcruntime -count=1`。
- [ ] 运行 `rtk go test ./... -count=1`。
- [ ] 验收：阶段 2 runtime foundation completion criteria 全部由测试覆盖。
- [ ] 提交：`test: verify runtime dispatcher foundation`

## Task 8：记录阶段 2 迁移清单与验证结果

**Files:**

- Create: `docs/plans/2026-04-28-stage-2-migration-inventory.md`
- Modify: `docs/plans/2026-04-28-stage-2-runtime-dispatcher-foundation-plan.md`

**迁移内容与理由：**

- 阶段 2 会参考旧 stream registry 和 streaming lifecycle 测试思路，同时明确拒绝旧 registry/provider/bootstrap。
- 迁移清单用于后续 generator renderer 实现时防止旧架构回流。

- [ ] 写入已迁移、参考后重写、不迁移清单。
- [ ] 明确旧 `active_slot.go` 没有实现可迁移。
- [ ] 明确旧 stream registry 只能参考 handle/session 语义，不能迁移旧 handle 类型或 generated runtime 文件结构。
- [ ] 明确旧 binding、provider bootstrap、framework selector 不进入新版 runtime foundation。
- [ ] 记录验证命令：`rtk go test ./rpcruntime -count=1`、`rtk go test ./... -count=1`、禁止的 unsigned 32/64 token 扫描。
- [ ] 不记录机器环境处理。
- [ ] 验收：文档能回答阶段 2 “迁移了什么、为什么迁移、为什么不迁旧架构”。
- [ ] 提交：`docs: record stage 2 migration inventory`

## 阶段 2 完成标准

- `rpcruntime` 提供 service-agnostic active server slot。
- `rpcruntime` 提供 dispatcher shell，可以在 unary 调用开始时捕获 active server snapshot。
- `rpcruntime` 提供 signed stream handle allocator 和 stream session registry。
- stream `Start` 捕获 active server snapshot，后续 handle 操作固定使用该 snapshot。
- `Cancel`、`Finish`、`CloseSend`、`onDone` 的 terminal/finalize 行为有 runtime helper 和测试覆盖。
- runtime foundation 不依赖 service-specific protobuf 类型，不 import protobuf、connect、grpc 或 `internal/generator`。
- 旧多 registry、多 provider bootstrap、framework selector 没有被迁入。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- 禁止的 unsigned 32/64 token 扫描无命中。
