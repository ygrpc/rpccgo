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

rpccgo 用于把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

新版架构以 generated service 为边界组织运行时。每个 service 拥有 dispatcher、active server slot 和 native/message converter。所有 cgo client 调用先进入 dispatcher，再由 dispatcher 根据当前 active server 和请求 contract 完成路由或转换。

完整设计见 `docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md`。

## 架构约束

- 每次运行只有一个 server 在监听。
- 每个 generated service 同一时刻只有一个 active server。
- stream session 在 `Start` 时捕获 active server snapshot，后续 `Send`、`Finish`、`CloseSend`、`Cancel` 固定路由到该 snapshot。
- native 和 message 都必须支持 unary、client streaming、server streaming、bidi streaming。
- `rpcruntime` 只放通用 runtime primitive，不依赖 service-specific protobuf 类型，不执行 native/message 转换。
- service-specific dispatcher、adapter、converter、method metadata、cgo ABI 留在 generated service runtime。
- connect 和 grpc 保持标准 RPC transport 语义；不要重新设计 connect client 或 grpc client。
- connect/grpc remote server adapter 内部复用标准 connect/grpc client。
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
- `@rpccgo` 注释只控制 server adapter 生成，不控制 cgo client 生成。

## ABI 与类型约束

- 不要使用 unsigned 32/64 位类型作为 runtime 或 ABI 类型。
- Go 代码中不要引入 `uint32`、`uint64`、`atomic.Uint32`、`atomic.Uint64`。
- C ABI 中不要引入 `uint32_t`、`uint64_t`。
- 文档中不要使用 `u32`、`u64` 作为设计类型。
- `ErrorID` 使用 `int32`，`0` 表示 no error。
- runtime handle、scheduler key、error id 等跨语言可见数字类型默认使用 `int32`。
- `NativeArrayElem` 不支持 32/64 位 unsigned 类型。
- repeated bool 不使用 Go `[]bool` 作为 C ABI 表示；使用 byte 编码，由专门 wrapper 处理。
- `uintptr` 用于 pointer handle 时可以保留。

## 目录结构

- `cmd/protoc-gen-rpc-cgo/`：protobuf 插件入口。
- `internal/generator/`：generator 内部 parser、planner、renderer。
- `internal/integration/`：端到端测试。
- `examples/`：用户可运行示例。
- `rpcruntime/`：通用 runtime primitive。
- `docs/specs/`：定稿设计文档。
- `docs/plans/`：项目路线图和实施计划。

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
- 阶段 0 只迁移 service 无关的 `rpcruntime` primitive。
- 不要提前迁移旧 `active_slot.go`、旧 generator、旧 integration 或旧 bootstrap 模型。
- 不要把旧项目的多 registry、多 provider bootstrap、framework selector 带进新版架构。
- 如果旧代码与新版 signed ABI、单 dispatcher、单 active server 约束冲突，必须按新版约束调整。

## 验证

- 常规验证：`rtk go test ./...`。
- runtime focused 验证：`rtk go test ./rpcruntime -count=1`。
- 搜索文件优先使用 `rtk rg`。
- 修改 runtime 或 ABI 类型后，必须扫描 unsigned 32/64 类型。`AGENTS.md` 为了描述禁用规则会包含这些字符串，扫描时排除它：

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md'
```

- 如果命令因为本机临时环境问题失败，不要把本机 workaround 写入项目文档；只在当前执行记录中说明。
