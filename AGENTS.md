# rpccgo Agent Guide

## 交互规则

- 默认使用简体中文回复。
- 代码标识符、命令、日志、报错信息保持原始语言。
- 接到任务时先判断简单任务或复杂任务。
- 简单任务先复述需求和假设，最多提出 3 个关键问题，等用户确认后再动手；如果用户已经明确批准执行，可以直接进入实现。
- 复杂任务先用设计/计划流程拆清楚边界；已有计划时按计划执行，不重复讨论。
- 遇到 bug 先写能复现的测试，再修复到测试通过。
- shell 命令默认加 `rtk` 前缀。
- 文件编辑必须使用 `apply_patch`，不要用 shell 重定向或脚本直接写文件。

## 项目目标

**当前项目还没有发布，不需要考虑兼容性**

rpccgo 用于把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

新版架构以 generated service 为边界组织运行时。每个 service 拥有独立的 typed atomic native/message active binding slot、不可变 active binding、stream registry 和 native/message converter。注册阶段把具体 server 归一化为对应 contract 的 method closure；调用阶段只捕获 binding snapshot 并调用 closure。

完整设计见 `docs/specs/2026-04-27-rpccgo-service-local-active-server-architecture.md`。

## 架构约束

- 项目未发布前，不为兼容性保留重复概念、重复 API 或仅用于平滑迁移的中间结构；优先直接删除多余层次并收敛术语。
- 每次运行只有一个 server 在监听。
- 每个 generated service 的每个 contract 同一时刻只有一个 active binding；native active binding 与 message active binding 可独立注册。
- stream session 在 `Start` 时捕获对应 contract 的 active binding snapshot，后续 `Send`、`Finish`、`CloseSend`、`Cancel` 固定路由到该 snapshot。
- native 和 message 都必须支持 unary、client streaming、server streaming、bidi streaming。
- `rpcruntime` 只放通用 runtime primitive，不依赖 service-specific protobuf 类型，不执行 native/message 转换。
- service-specific active binding、converter、method metadata、cgo ABI 留在 generated service runtime。
- native/message active binding slot 直接使用 generated service-local typed atomic pointer；不要保留 `rpcruntime.ActiveServerSlot`、`ServerKind`、`ServerContract`、`AdapterSnapshot` 或 version metadata。
- active binding 必须在完整校验后原子发布；native/message active binding 独立保存、独立发布，互不引用；调用阶段不按 server kind 或 contract 分支。
- C native/message callback 必须按完整 service callback set 注册，不允许按 method 增量激活。
- stream registry 直接保存最终 session；不要保留 `StreamEntry`、dispatcher、executor 或 registry lifecycle helper 层。
- connect 和 grpc 保持标准 RPC transport 语义；不要重新设计 connect client 或 grpc client。
- connect/grpc remote client active server 直接注册标准 connect/grpc client，不生成独立 remote adapter 文件。
- 不允许使用 painc，所有错误必须显式传递。

## Server 与 Client 模型

支持的 server 类型：

- go native server
- cgo native server
- cgo message server
- connect handler
- grpc server
- connect remote server
- grpc remote server

rpccgo 只设计两类 cgo client：

- cgo native client
- cgo message client

connect client 和 grpc client 属于标准 RPC client，不进入 rpccgo client 类型模型。

## Protobuf 插件策略

- 只实现一个 protobuf 插件：`protoc-gen-rpc-cgo`。
- 插件内部拆分 parser、planner、renderer，不为不同 server 类型拆多个 protoc 插件。
- 插件读取 service 上的 `@rpccgo` 注释，建立统一 `ServicePlan`，再按 plan 调用不同 renderer。
- 没有 `@rpccgo` 注释时默认等价于 `@rpccgo:msg-connect`。
- 支持 token：`msg-connect`、`msg-grpc`、`native`。
- `native` 单独出现时默认生成 `msg-connect|native`。
- 未知 token 必须报错；常见拼写错误如 `msg-conenct` 不能静默忽略。
- `@rpccgo` 注释控制 message/native contract artifact family；没有 `native` token 时不得生成 native server、cgo native server 或 cgo native client artifact。
- cgo 生成文件必须生成到 `package main`。输出目录由 `cgo_dir` 参数控制，路径相对 protobuf Go package 的生成目录解析；`cgo_dir` 可以指向 Go package 目录之外，默认值是 Go package 目录下的 `cgo` 子目录。
- cgo service 文件名必须显式写出 native/message contract token：使用 `.server.native.cgo.rpccgo.go` / `.client.native.cgo.rpccgo.go` 与 `.server.message.cgo.rpccgo.go` / `.client.message.cgo.rpccgo.go`；共享 exports 按 cgo Go package 只生成一次，文件名固定为 `rpccgo.exports.cgo.rpccgo.go`。

## ABI 与类型约束

- protobuf schema 中允许使用 `uint32` / `uint64` 字段；这些字段的 generated API 应保留 proto 语义。
- protobuf schema 中的 unsigned 字段可以继续使用；proto 无关的 runtime、scheduler、handle、length、error id、计数、索引等辅助类型不要使用 unsigned 32/64 位类型。
- Go 辅助代码中不要为 proto 无关类型引入 `uint32`、`uint64`、`atomic.Uint32`、`atomic.Uint64`。
- C ABI 中 proto 无关的辅助类型不要引入 `uint32_t`、`uint64_t`。
- 文档中不要使用 `u32`、`u64` 作为 proto 无关的设计类型。
- `ErrorID` 使用 `int32`，`0` 表示 no error。
- runtime handle、scheduler key、error id 等跨语言可见数字类型默认使用 `int32`。
- repeated bool 不使用 Go `[]bool` 作为 C ABI 表示；使用 byte 编码，由专门 wrapper 处理。
- `uintptr` 用于 pointer handle 时可以保留。

## 目录结构

- `cmd/protoc-gen-rpc-cgo/`：protobuf 插件入口。
- `internal/generator/`：generator 内部 parser、planner、renderer。
- `internal/integration/`：端到端测试。
- `examples/`：用户可运行示例。
- `rpcruntime/`：通用 runtime primitive。
- `docs/specs/`：定稿设计文档。
- `docs/release/`：发布验证清单。

## 技术栈

- Go 1.24。
- cgo。
- protobuf / protogen。
- Connect。
- gRPC。
- `runtime.Pinner`。
- `runtime.AddCleanup`。
- 标准库 `testing`。

## 迁移规则

- 旧项目路径：`/home/zenghp/github.com/ygrpc/rpccgo-old`。
- 迁移旧代码前必须说明该代码的作用，以及为什么迁移比重写更合适。
- 旧代码迁移应从 service 无关的 `rpcruntime` primitive 这类低耦合能力开始。
- 不要提前迁移旧 `active_slot.go`、旧 generator、旧 integration 或旧 bootstrap 模型。
- 不要把旧项目的多 registry、多 provider bootstrap、framework selector 带进新版架构。
- 如果旧代码与新版 signed ABI、contract-specific 单 active binding、不可变 active binding 约束冲突，必须按新版约束调整。

## 验证

- 常规验证：`rtk go test ./...`。
- 发布级验证或涉及 planner / ABI / runtime / examples 合同的改动，使用 `docs/release/verification-checklist.md` 的完整流程；至少运行必跑命令：`rtk env GOCACHE=/tmp/rpccgo-go-build go test ./... -count=1`。
- 完整 checklist 还包括 runtime、generator、integration focused 测试，grpc/connect greeter 的 `mage generate`、`mage test`、`mage run`，以及 unsigned 32/64 合同扫描。
- runtime focused 验证：`rtk go test ./rpcruntime -count=1`。
- 涉及 generator planner、ABI plan 或 protogen descriptor 形状的测试，优先使用真实 `.proto` fixture 经过插件/parser/planner 路径构造输入；不要用手写 planner 结构体替代真实 descriptor，除非测试目标明确与 proto/protogen 无关。
- 搜索文件优先使用 `rtk rg`。
- 修改 runtime 或 ABI 类型后，必须扫描 unsigned 32/64 类型。`AGENTS.md` 为了描述禁用规则会包含这些字符串，扫描时排除它：

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/release/verification-checklist.md'
```

- 如果命令因为本机临时环境问题失败，不要把本机 workaround 写入项目文档；只在当前执行记录中说明。

## Agent skills

### Issue tracker

Issues and PRDs are tracked in GitHub Issues for `ygrpc/rpccgo`. See `docs/agents/issue-tracker.md`.

### Triage labels

Use the default five-label triage vocabulary. See `docs/agents/triage-labels.md`.

### Domain docs

This is a single-context repo with root `CONTEXT.md` and `docs/adr/`. See `docs/agents/domain.md`.
