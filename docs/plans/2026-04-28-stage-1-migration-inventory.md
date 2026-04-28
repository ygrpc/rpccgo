# Stage 1 Migration Inventory

## 目标

本清单记录阶段 1 从旧版 `rpccgo-old` generator 迁移、参考或明确拒绝的代码范围。阶段 1 的产物只建立 protoc 输入解析和 service plan，不生成 runtime、dispatcher、adapter、converter、cgo ABI 或 example 业务代码。

这份清单也用于后续阶段约束实现方向：可以复用稳定的 parser、命名和 streaming lifecycle 经验，但不能把旧 registry、provider bootstrap、framework selector 等与新版单 dispatcher、单 active server 约束冲突的模型带回新版架构。

## 已迁移清单

| 旧项目文件或模块 | 新版落点 | 旧代码作用 | Stage 1 处理方式 | 为什么迁移 |
|---|---|---|---|---|
| `internal/generator/streaming_kind.go` | `internal/generator/streaming_kind.go` | 根据 method descriptor 的 client/server streaming 标志识别 unary、client streaming、server streaming、bidi streaming | 迁移为 `StreamingKind` 与 `StreamingKindOf`，并由 descriptor plan 和 streaming lifecycle plan 共同使用 | 这是纯 descriptor 判定逻辑，不依赖旧 registry、provider 或 renderer，和新版 Stage 1 的 service plan 输入完全一致 |
| `internal/generator/namespace.go` 中的小型命名 helper | `internal/generator/names.go` | 提供首字母小写、snake case 等 renderer 前置命名能力 | 迁移为 generator 内部纯函数，先服务 plan 和测试命名需求，不提前固化最终文件布局 | 命名转换是稳定纯函数，迁移能减少后续 renderer 重复实现；文件族和导出符号仍留给后续阶段按新版架构定义 |
| `internal/generator/render_plan.go` 的 streaming lifecycle 矩阵 | `internal/generator/streaming_plan.go` | 描述 streaming method 的 Start、Send、Finish、CloseSend、Cancel、onRead、onDone 操作矩阵和终态 | 迁移 lifecycle 语义到 `LifecyclePlan`，并通过 `ValidateStreamingLifecyclePlan` 校验四类 method 的期望矩阵 | streaming 生命周期规则和新版 dispatcher snapshot 合同一致，迁移可避免重新推导 Finish、CloseSend、onDone、Cancel 的组合语义 |
| `internal/generator/streaming_plan.go` 的 streaming plan 思路 | `internal/generator/streaming_plan.go` 与 `internal/generator/plan.go` | 为非 unary method 建立 streaming 操作计划 | 收敛到 `MethodPlan.Lifecycle`，不再保留旧 native/message 分裂结构 | 新版 Stage 1 要让所有 adapter 共用同一个 method plan，迁移核心语义比复制旧结构更合适 |

## 参考后重写清单

| 旧项目文件或模块 | 新版落点 | 旧代码作用 | Stage 1 处理方式 | 为什么参考后重写 |
|---|---|---|---|---|
| `internal/generator/frameworks.go` | `internal/generator/service_options.go` | 解析旧 connect、grpc、native framework 选择 | 只参考 token 去重、canonical order 和错误提示思路；重写为 `@rpccgo` service 注释解析，输出 `msg-connect`、`msg-grpc`、`native` adapter selection | 旧 framework 概念绑定历史 selector，新版只让 service 注释选择 server adapter；继续迁移旧模型会干扰单插件和单 dispatcher 边界 |
| `internal/generator/options.go` | `internal/generator/generator.go` 与 `cmd/protoc-gen-rpc-cgo/main.go` | 解析旧 protoc 参数、mode、go role 和 framework 配置 | 只参考 `protogen.Options` 接入方式；Stage 1 对未知 rpccgo 参数报错，并通过 `Generate` 返回 plan | 旧 options 绑定多模式生成开关，新版阶段 1 只需要单插件 parser/planner 壳，不应保留旧 mode/go role 配置面 |
| `internal/generator/generator.go` | `internal/generator/generator.go` | 旧插件调度、文件遍历、renderer 调用 | 只参考 protogen request 遍历方式；重写为只构建 `FilePlan`，不调用 renderer、不输出最终 runtime 文件 | 阶段 1 的验收对象是 plan，不是 generated code；旧 generator 与 renderer 深度耦合，直接迁移会提前带入旧架构 |
| `internal/generator/binding.go` 的 method metadata 字段命名经验 | `internal/generator/descriptor_plan.go` 与 `internal/generator/plan.go` | 描述 service/method binding、request/response 类型和调用元数据 | 只参考 service/method/request/response 的命名维度；重写为 `ServicePlan`、`MethodPlan`、`MethodIOPlan`，不迁移 resolver、registry 或 binding runtime | descriptor metadata 本身有价值，但旧 binding 是多 registry 模型的一部分，新版 Stage 1 只需要静态 plan |
| `internal/generator/native_types.go` | `internal/generator/contract_plan.go` 与 `internal/generator/plan.go` | 将 protobuf field 分类到 native ABI 类型 | 参考分类覆盖面后重写 `FieldPlan`、`NativeFieldPlan` 和 unsupported field 报错；bool repeated 记录 byte buffer wrapper，message contract 只记录 protobuf 类型 | 旧 native 类型包含新版禁止的 unsigned 32/64 ABI 思路，必须按 Stage 0 signed ABI 和 byte-encoded bool 约束重建 |
| 旧 generator fixture 和 golden 测试组织 | `internal/generator/*_test.go` 与 `internal/generator/stage1_acceptance_test.go` | 验证插件输入、生成输出和多场景组合 | 参考 fixture 组织方式后重写为 plan-level tests；验收 service 注释、descriptor metadata、contract plan、streaming lifecycle 和 codec capability | 阶段 1 不生成 renderer 输出，旧 golden 文件不再是合适验收对象；新的测试必须直接断言 plan 行为 |

## 不迁移清单

| 旧项目文件或模块 | 旧代码作用 | Stage 1 处理方式 | 为什么不迁移 |
|---|---|---|---|
| `internal/generator/binding.go` 的 registry、resolver、binding runtime | 将 generated method 绑定到旧运行时 registry，并支持历史调用解析 | 不迁移；只保留 descriptor metadata 的参考价值 | 新版架构以 generated service dispatcher 和 active server slot 为边界，不使用旧多 registry 解析模型 |
| 旧 framework registry | 管理 connect、grpc、native 等历史 framework selector 和生成分支 | 不迁移 | 新版只支持单一 protobuf 插件，service 上的 `@rpccgo` 注释只选择 server adapter，不恢复旧 selector 层 |
| 旧 provider bootstrap | 注册多个 provider，并在 bootstrap 阶段组装服务能力 | 不迁移 | 新版约束每次运行只有一个 server 在监听，每个 generated service 同一时刻只有一个 active server；旧 provider bootstrap 与该约束冲突 |
| 旧多 provider active slot 相关 generator 假设 | 支撑历史多 provider 切换和 registry 查找 | 不迁移到 Stage 1 | Stage 1 只输出静态 plan；后续 active server snapshot 需要围绕单 dispatcher 重建，不能复用旧多 provider 假设 |
| 旧 message/native renderer 文件 | 生成旧 message server、native server、client export 和桥接代码 | 不迁移 | Stage 1 不实现 renderer；后续 renderer 必须从新版 `ServicePlan`、converter plan 和 dispatcher contract 出发重建 |
| 旧 integration bootstrap 与 examples | 运行历史端到端 demo 和手工 bootstrap | 不迁移 | 旧 demo 体现历史 registry/provider 架构，和新版单 dispatcher、单 active server 模型不一致；后续 examples 按新版用户路径重新设计 |

## 阶段 1 验证结果

本阶段记录的验证只包含仓库通用命令和 Stage 1 focused 命令，不记录本机环境 workaround。

| 验证项 | 命令 | 结果 |
|---|---|---|
| generator 与插件 focused 测试 | `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` | 通过 |
| 全仓测试 | `rtk go test ./... -count=1` | 通过 |
| 禁止的 unsigned 32/64 token 扫描 | 使用 AGENTS.md 中的扫描命令，排除 `AGENTS.md` | 无命中 |
| 工作区状态 | `rtk git status --short` | 提交前仅包含本次 Task 8 文档变更 |

## 阶段 1 结论

阶段 1 已完成 parser/planner 基线：`protoc-gen-rpc-cgo` 可编译，generator 能从 proto descriptor 构建 `FilePlan`、`ServicePlan`、`MethodPlan`、field contract 和 streaming lifecycle plan，`@rpccgo` 注释规则与架构文档保持一致。

阶段 1 没有迁移旧 registry、provider bootstrap、framework selector、旧 renderer、旧 integration bootstrap 或旧 examples。后续阶段实现 dispatcher、converter、adapter renderer 时，必须继续从新版 service plan 和单 active server 约束出发，不能让旧多 registry、多 provider 模型回流。
