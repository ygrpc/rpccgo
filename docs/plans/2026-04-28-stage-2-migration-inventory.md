# Stage 2 Migration Inventory

## 目标

本清单记录阶段 2 runtime dispatcher foundation 从旧版 `rpccgo-old` 迁移、参考或明确拒绝的代码范围。阶段 2 的产物只建立 service 无关的 runtime primitive：active server slot、dispatcher snapshot capture、stream handle/session registry、stream lifecycle finalization helper，以及覆盖这些语义的 runtime 测试。

这份清单也用于后续 generator、adapter、converter 和 cgo ABI 阶段约束实现方向：可以复用旧 streaming handle/session 生命周期经验，但不能把旧 binding、provider bootstrap、framework selector、旧 generated runtime 文件结构或旧 handle 类型带回新版 runtime foundation。

## 已迁移清单

| 旧项目文件或模块 | 新版落点 | 旧代码作用 | Stage 2 处理方式 | 为什么迁移 |
|---|---|---|---|---|
| 旧 streaming lifecycle 测试关注点 | `rpcruntime/stream_session_test.go`、`rpcruntime/stream_registry_test.go`、`rpcruntime/dispatcher_foundation_test.go` | 覆盖 unknown handle、重复 terminal 操作、cancel 后终止、close send 后禁止 send 等 streaming 生命周期边界 | 迁移为 service-agnostic runtime 测试思路，不复制旧 generated 测试结构 | 这些行为是新版 runtime foundation 的稳定合同，和具体 protobuf、connect、grpc 或 generated renderer 无关，适合作为阶段 2 验收测试 |

## 参考后重写清单

| 旧项目文件或模块 | 新版落点 | 旧代码作用 | Stage 2 处理方式 | 为什么参考后重写 |
|---|---|---|---|---|
| 旧 generated stream registry runtime | `rpcruntime/stream_registry.go` | 为 generated streaming 调用分配 handle、保存 session、按 handle 查找、删除或取出 session | 只参考 handle/session 表的生命周期语义，重写为 `StreamRegistry[T]` 与 `StreamHandle` | 旧实现绑定 generated 输出、旧 handle 类型和旧 native/message 分裂模型；新版必须使用 `int32` handle，并保持 runtime primitive service-agnostic |
| 旧 generated streaming Start/Send/Finish/Cancel 路径 | `rpcruntime/dispatcher.go`、`rpcruntime/dispatcher_stream_test.go` | 在 stream Start 时绑定当前 server，并在后续 handle 操作中继续使用同一 session | 只参考 Start 捕获 active server snapshot、后续操作固定路由到该 snapshot 的语义 | 旧代码混合 generated ABI、adapter 调用和 transport 细节；阶段 2 只实现 snapshot capture 和 session registry 基础能力 |
| `internal/generator/message_streaming_render.go` 中的 session 状态片段 | `rpcruntime/stream_session.go` | 在 generated 字符串输出中处理 send、close send、finish、cancel、onDone 等状态转换 | 参考状态转换规则后重写为 runtime lifecycle helper | 旧片段绑定 renderer 字符串和具体 adapter 细节；新版需要独立 helper 表达 active、send closed、finalized、canceled 等终态规则 |
| 旧架构中“后注册覆盖后续调用”的 active server 目标语义 | `rpcruntime/active_slot.go`、`rpcruntime/dispatcher.go` | 让后续调用使用最新注册的 server 能力 | 参考目标语义后重写为单 active server slot 与 dispatcher capture | 新版约束每个 generated service 同一时刻只有一个 active server；旧多 registry、多 provider 假设不能迁入 |

## 不迁移清单

| 旧项目文件或模块 | 旧代码作用 | Stage 2 处理方式 | 为什么不迁移 |
|---|---|---|---|
| 旧 `rpcruntime/active_slot.go` | 旧文件没有实际 runtime 实现 | 不迁移实现，只记录结论 | 该文件没有可迁移代码；新版 `ActiveServerSlot[T]` 必须按单 active server、signed version、snapshot value 语义重建 |
| 旧 stream registry 的 handle 类型 | 表示旧 generated streaming session handle | 不迁移 | 新版跨语言可见 stream handle 必须写成 `int32`，不能延续旧 handle 类型 |
| 旧 stream registry 的 generated runtime 文件结构 | 将 streaming registry、adapter 调用和 generated ABI 组织在旧输出文件中 | 不迁移 | 阶段 2 只提供 `rpcruntime` service-agnostic primitive；generated 文件布局属于后续 renderer 阶段，不能提前复制旧结构 |
| 旧 binding、resolver 和 registry runtime | 将 generated method 绑定到历史多 registry 调用解析模型 | 不迁移 | 新版以 generated service dispatcher 为边界，dispatcher 捕获 active server snapshot 后路由调用，不使用旧 binding/runtime resolver |
| 旧 provider bootstrap | 注册多个 provider，并在 bootstrap 阶段组装服务能力 | 不迁移 | 新版约束每次运行只有一个 server 在监听，每个 generated service 同一时刻只有一个 active server；旧 provider bootstrap 与该约束冲突 |
| 旧 framework selector | 在 connect、grpc、native 等历史 framework 分支之间选择生成路径 | 不迁移 | 新版只有一个 protobuf 插件，service 注释只选择 server adapter；runtime foundation 不持有 framework selector |
| 旧 connect/grpc/native renderer 与 generated cgo ABI | 生成旧 adapter、client export、native/message 桥接和业务调用路径 | 不迁移 | 阶段 2 不生成 adapter、converter、cgo ABI 或 example；后续阶段必须从 Stage 1 `ServicePlan` 和 Stage 2 runtime primitive 出发重建 |
| 旧 integration bootstrap 与 examples | 运行历史端到端 demo 和手工 bootstrap | 不迁移 | 旧 demo 体现历史 registry/provider 架构；阶段 2 的验收对象是 runtime foundation，不是用户可运行 example |

## 阶段 2 验证结果

本阶段记录的验证只包含仓库通用命令和 Stage 2 focused 命令，不记录本机环境处理或 workaround。

| 验证项 | 命令 | 结果 |
|---|---|---|
| runtime focused 测试 | `rtk go test ./rpcruntime -count=1` | 通过 |
| 全仓测试 | `rtk go test ./... -count=1` | 通过 |
| 禁止的 unsigned 32/64 token 扫描 | 使用 AGENTS.md 中的扫描命令，排除 `AGENTS.md` | 无命中 |
| 工作区状态 | `rtk git status --short` | 提交前仅包含本次 Task 8 文档变更 |

## 阶段 2 结论

阶段 2 已完成 runtime dispatcher foundation：`rpcruntime` 提供 service-agnostic active server slot、dispatcher shell、unary snapshot capture、stream Start snapshot capture、`int32` stream handle registry、stream session lifecycle helper，以及覆盖并发、unknown handle、double terminal、cancel/finalize 等边界的测试。

阶段 2 只迁移了旧 streaming lifecycle 的测试关注点，并参考后重写了 stream registry、stream Start snapshot 和 session 状态语义。旧 `rpcruntime/active_slot.go` 没有实现可迁移；旧 stream registry 只能参考 handle/session 生命周期，不能迁移旧 handle 类型或 generated runtime 文件结构；旧 binding、provider bootstrap、framework selector、renderer、integration bootstrap 和 examples 都没有进入新版 runtime foundation。

后续阶段实现 generated service runtime、converter、adapter renderer 和 cgo ABI 时，必须继续从 Stage 1 `ServicePlan` 和本阶段 runtime primitive 出发，保持单 dispatcher、单 active server slot、`int32` handle 和 service-specific 逻辑留在 generated service runtime 的边界。
