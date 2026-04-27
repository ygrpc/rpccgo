# Stage 0 Runtime Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立新版 rpccgo 的项目骨架与 runtime 迁移基线，让后续 dispatcher、generator、adapter 工作有可测试的底座。

**Architecture:** 阶段 0 只迁移与 service 无关的 runtime 基础能力，并建立 generator/runtime/integration/example 的目录边界。旧项目中的多 registry、多 provider bootstrap、framework selector、generated service 代码不进入本阶段。

**Tech Stack:** Go 1.24、cgo、`runtime.AddCleanup`、`runtime.Pinner`、标准库 testing。

---

## 范围

阶段 0 只处理以下内容：

- 新版仓库基础目录。
- `rpcruntime` 的可复用基础能力。
- runtime 基础测试迁移。
- 旧项目迁移清单。
- 最小验证命令。

阶段 0 不处理以下内容：

- dispatcher 正式实现。
- active server slot 正式实现。
- protoc 插件解析。
- `@rpccgo` service 注释解析。
- generated service 代码。
- connect/grpc adapter。
- native/message converter。
- examples 的完整业务流程。

## 迁移判定

| 旧项目文件 | 本阶段处理 | 作用 | 为什么迁移而不是重写 |
|---|---|---|---|
| `rpcruntime/length.go` | 直接迁移 | 统一 int/int32 ABI 长度转换，拒绝负数和溢出 | 行为小而稳定，已有边界测试；重写容易遗漏负长度和 int32 上限语义 |
| `rpcruntime/errors.go` | 带调整迁移 | 将 Go error 转成 cgo 可取回的 error id 和 pinned error text | TTL、并发 take、过期删除已经被旧测试覆盖；重写会重新引入泄漏或重复读取风险 |
| `rpcruntime/cleanup_scheduler.go` | 直接迁移 | 为 error store 提供低成本延迟清理 | 已有时间轮实现和并发测试；用通用 timer 重写会增加 goroutine/timer churn |
| `rpcruntime/release.go` | 直接迁移 | pin Go memory 并通过 `Release` 解除 pin，支撑返回 bytes/string/slice 到 cgo | `runtime.Pinner` 生命周期和重复 pin 检测容易写错，旧实现已封装成稳定 primitive |
| `rpcruntime/free.go` | 带调整迁移 | 管理 C free callback，释放 owned cgo input，并支持 callback 后注册时重试 | 这是 cgo ownership 的核心状态机，旧实现已覆盖 once-only release、retry、reset；重写风险高 |
| `rpcruntime/rpc_type.go` | 直接迁移 | 提供 `RpcBytes`、`RpcString`，支持 unsafe view、safe copy、owned release、cleanup | 这是 native/message ABI 都会用到的输入 wrapper，语义稳定且测试覆盖多 |
| `rpcruntime/rpc_repeat.go` | 直接迁移 | 提供 repeated scalar 和 bool wrapper，支持 safe/unsafe 访问与 owned release | repeated wrapper 的 bool 编码、empty wrapper、zero-length owned release 细节多，迁移比重写更可靠 |
| `rpcruntime/active_slot.go` | 暂不迁移 | 旧 active slot 用于历史架构的 provider 状态 | 新版 active server slot 要围绕 dispatcher adapter snapshot 重建，旧实现只能在阶段 2 参考 |
| 旧 `internal/generator/*` | 暂不迁移 | generator、streaming plan、adapter 渲染 | 阶段 0 不实现 generator；后续阶段按新版架构重新拆分 |
| 旧 `internal/integration/*` | 只迁测试思路 | 覆盖 runtime、message/native、forwarding 的端到端场景 | 阶段 0 只需要 runtime 单元测试；integration 场景等 adapter 出现后再迁 |

## 文件结构

阶段 0 结束后应形成以下结构：

```text
cmd/protoc-gen-rpc-cgo/
internal/generator/
internal/integration/
examples/
rpcruntime/
docs/plans/
docs/specs/
```

各目录职责：

- `cmd/protoc-gen-rpc-cgo/`：后续单一 protobuf 插件入口，本阶段只建立目录，不实现 `@rpccgo` 注释解析。
- `internal/generator/`：后续 generator 内部实现，本阶段只建立目录。
- `internal/integration/`：后续端到端测试，本阶段只建立目录。
- `examples/`：后续示例，本阶段只建立目录。
- `rpcruntime/`：本阶段唯一迁移代码的 Go package。

## Task 1：建立项目骨架

**Files:**

- Create: `cmd/protoc-gen-rpc-cgo/.gitkeep`
- Create: `internal/generator/.gitkeep`
- Create: `internal/integration/.gitkeep`
- Create: `examples/.gitkeep`

- [ ] 建立目录占位文件。
- [ ] 运行 `rtk go test ./...`。
- [ ] 验收：命令通过，且目录边界存在。
- [ ] 提交：`chore: add project skeleton`

## Task 2：迁移长度与 pinned release primitive

**Files:**

- Create: `rpcruntime/length.go`
- Create: `rpcruntime/length_test.go`
- Create: `rpcruntime/release.go`
- Create: `rpcruntime/release_test.go`

**迁移内容与理由：**

- `LengthToInt32` / `LengthFromInt32` 是所有 cgo ABI 长度字段的统一入口。迁移它可以避免每个 ABI renderer 自己处理负数、溢出和 int32 转换。
- `PinBytes` / `PinString` / `PinSlice` / `Release` 是 Go 返回内存暴露给 cgo 的基础。旧实现已经处理空值、重复 pin、unpin，且 Go 1.24 环境与当前仓库一致。

- [ ] 从旧项目迁移 `length.go`、`length_test.go`、`release.go`、`release_test.go`。
- [ ] 调整 package import 路径，保持 package 名为 `rpcruntime`。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestLength|TestPin|TestRelease' -count=1`。
- [ ] 验收：长度边界、重复 pin、release 行为测试通过。
- [ ] 提交：`feat: migrate runtime length and pinned release primitives`

## Task 3：迁移 cleanup scheduler 与 error store

**Files:**

- Create: `rpcruntime/cleanup_scheduler.go`
- Create: `rpcruntime/cleanup_scheduler_test.go`
- Create: `rpcruntime/errors.go`
- Create: `rpcruntime/errors_test.go`

**迁移内容与理由：**

- `cleanup_scheduler` 为 error TTL 清理服务。它集中管理延迟清理，避免每个 error id 启动独立 goroutine 或 timer。
- `StoreError` / `TakeErrorText` / `TakeErrorTextForExport` 是 cgo 错误跨边界传播的基础。旧实现已经覆盖 error id 分配、过期、take 后删除、pinned text 和 export 输出指针语义。

- [ ] 从旧项目迁移 cleanup scheduler 和 error store 代码。
- [ ] 保持 error text 使用 `PinString`，避免复制出无法 release 的内存。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestCleanup|TestStoreError|TestTakeError' -count=1`。
- [ ] 验收：error 能存取一次、过期清理有效、export 输出长度正确。
- [ ] 提交：`feat: migrate runtime error store`

## Task 4：迁移 C input ownership 与 bytes/string wrapper

**Files:**

- Create: `rpcruntime/free.go`
- Create: `rpcruntime/rpc_type.go`
- Create: `rpcruntime/rpc_type_test.go`
- Create: `rpcruntime/integration_test.go`

**迁移内容与理由：**

- `RegisterFreeCallback` / `ReleaseC` / `releaseRpcInput` 是 owned C input 的释放状态机。它保证同一指针只释放一次，并支持 free callback 尚未注册时的 retry。
- `RpcBytes` / `RpcString` 是 cgo 输入在 Go 侧的标准 wrapper。它们同时支持 zero-copy unsafe view、可持久化 safe copy、显式 release 和 GC cleanup。
- 这些语义跨 native、message、streaming 都会复用，属于基础 runtime contract，不依赖旧 generator 架构。

- [ ] 从旧项目迁移 `free.go`、`rpc_type.go`、`rpc_type_test.go`、`integration_test.go` 中与 `RpcBytes`、`RpcString`、free callback 相关的测试。
- [ ] 保留 `ResetFreeCallbackForTesting`，用于隔离测试状态。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestRpcBytes|TestRpcString|TestReleaseC|TestFreeCallback' -count=1`。
- [ ] 验收：borrowed input 不释放，owned input release 一次，safe copy 可在 release 后保留，GC cleanup 能释放 owned pointer。
- [ ] 提交：`feat: migrate runtime bytes and string wrappers`

## Task 5：迁移 repeated wrapper

**Files:**

- Create: `rpcruntime/rpc_repeat.go`
- Create: `rpcruntime/rpc_repeat_test.go`

**迁移内容与理由：**

- `RpcRepeat[T]` 统一表示 repeated numeric/enum input。
- `RpcBoolRepeat` 用 byte 编码表示 repeated bool，避免 Go `[]bool` 的非 byte-addressable 布局问题。
- repeated wrapper 的 empty singleton、zero-length owned pointer release、safe copy、bool decode 细节多；旧实现已经围绕这些风险写过测试，迁移比重写更稳。

- [ ] 从旧项目迁移 `rpc_repeat.go` 和 `rpc_repeat_test.go`。
- [ ] 保持 `NativeRepeatElem` 的固定宽度类型约束。
- [ ] 运行 `rtk go test ./rpcruntime -run 'TestRpcRepeat|TestRpcBoolRepeat|TestEmptyRpcRepeat' -count=1`。
- [ ] 验收：numeric/enum repeated、bool repeated、empty wrapper、owned release 都通过测试。
- [ ] 提交：`feat: migrate runtime repeated wrappers`

## Task 6：记录迁移清单

**Files:**

- Create: `docs/plans/2026-04-27-stage-0-migration-inventory.md`

**迁移内容与理由：**

- 阶段 0 需要留下明确证据，说明哪些旧代码进入新版、哪些只作为后续参考、哪些因旧架构不匹配而暂不迁移。
- 这份清单避免后续实现时把旧项目的多 registry、多 provider bootstrap 模型带回来。

- [ ] 写入 runtime 迁移清单。
- [ ] 写入暂不迁移清单：`active_slot.go`、`internal/generator/*`、`internal/integration/*`。
- [ ] 为每项写明作用、迁移理由、风险点。
- [ ] 验收：清单能回答“迁移了什么、有什么用、为什么不重写”。
- [ ] 提交：`docs: record stage 0 migration inventory`

## Task 7：阶段 0 总验证

**Files:**

- Modify: `docs/plans/2026-04-27-stage-0-runtime-migration-plan.md`

- [ ] 运行 `rtk go test ./...`。
- [ ] 运行 `rtk git status --short`。
- [ ] 确认没有引入 dispatcher、generator renderer、adapter 或 example 业务代码。
- [ ] 在计划文档中记录验证命令和结果。
- [ ] 验收：`go test ./...` 通过，工作区只包含阶段 0 相关变更。
- [ ] 提交：`test: verify stage 0 runtime migration`

## 阶段 0 完成标准

- 项目骨架存在。
- `rpcruntime` 基础能力迁移完成。
- runtime 测试通过。
- 迁移清单明确说明每项旧代码的作用和迁移理由。
- `active_slot.go`、旧 generator、旧 integration 代码没有被提前迁入。
- `rtk go test ./...` 通过。
