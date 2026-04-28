# Stage 0 Migration Inventory

## 目标

本清单记录阶段 0 从旧版 `rpccgo-old` 迁移到新版 rpccgo 的 runtime 基础代码。每项都说明代码作用、迁移理由和后续风险，避免后续阶段把旧项目的多 registry、多 provider bootstrap 架构误带入新版设计。

## 已迁移代码

| 旧项目文件 | 新版落点 | 作用 | 迁移理由 | 风险点 |
|---|---|---|---|---|
| `rpcruntime/length.go` | `rpcruntime/length.go` | 统一 `int` 和 `int32` 长度转换，拒绝负数和溢出 | ABI 长度边界稳定，旧测试已覆盖负数、零值和 `int32` 上限 | 后续 generator 必须统一调用该入口，不能在 renderer 中重复写转换 |
| `rpcruntime/release.go` | `rpcruntime/release.go` | pin Go memory，并通过 exported pointer 调用 `Release` 解除 pin | `runtime.Pinner` 生命周期容易写错，旧实现已经封装重复 pin、空值和 release 语义 | 只能用于 Go 返回给 C 的 pinned memory，不能混用为 C input 的 free 机制 |
| `rpcruntime/cleanup_scheduler.go` | `rpcruntime/cleanup_scheduler.go` | 为短生命周期 runtime 状态提供低成本延迟清理 | 旧实现避免每条记录启动独立 goroutine/timer，并有并发测试 | 后续新增使用方必须保证 key 类型继续使用 signed `int32` 语义 |
| `rpcruntime/errors.go` | `rpcruntime/errors.go` | 将 Go error 存为 `ErrorID`，支持一次性取回 pinned error text | 旧实现已覆盖存取、过期清理、并发 take 和 export 输出指针语义 | `ErrorID` 已调整为 `int32`，后续 cgo ABI 不能退回 unsigned id |
| `rpcruntime/free.go` | `rpcruntime/free.go` | 管理 C free callback，释放 owned C input，并支持 callback 后注册时 retry | C input ownership 是 runtime 基础状态机，旧实现已覆盖 once-only release、retry 和 reset | 只负责 C input；不能用 `Release` 释放，也不能在 callback 未注册时吞掉需要 retry 的状态 |
| `rpcruntime/rpc_type.go` | `rpcruntime/rpc_type.go` | 提供 `RpcBytes`、`RpcString`，支持 unsafe view、safe copy、显式 release 和 GC cleanup | native/message/streaming 都会复用输入 wrapper，语义稳定且旧测试覆盖充分 | unsafe view 只在 wrapper 可达且底层 input 未释放时有效；跨边界保存必须使用 safe copy |
| `rpcruntime/rpc_repeat.go` | `rpcruntime/rpc_repeat.go` | 提供 repeated numeric wrapper 与 byte-encoded bool wrapper | repeated wrapper 的 empty singleton、zero-length owned pointer、bool 编码细节多，迁移比重写更可靠 | 新版禁止 32/64 unsigned ABI，因此只保留 signed/float 类型；bool repeated 必须继续走 byte buffer |

## 已迁移测试

| 旧项目测试 | 新版落点 | 迁移内容 | 作用 |
|---|---|---|---|
| `length_test.go` | `rpcruntime/length_test.go` | 长度转换边界测试 | 防止负长度、溢出和空值语义漂移 |
| `release_test.go` | `rpcruntime/release_test.go` | pin/release、重复 pin、重复 release 测试 | 防止 Go 返回内存泄漏或重复 unpin |
| `cleanup_scheduler_test.go` | `rpcruntime/cleanup_scheduler_test.go` | schedule、cancel、并发清理测试 | 防止 error store 清理路径泄漏或死锁 |
| `errors_test.go` | `rpcruntime/errors_test.go` | error 存取、TTL、export 输出测试 | 防止 error text 丢失、重复读取或 pinned pointer 泄漏 |
| `rpc_type_test.go` | `rpcruntime/rpc_type_test.go` | bytes/string wrapper 单元测试 | 覆盖 safe/unsafe、owned/borrowed、release retry 和零长度 owned input |
| `integration_test.go` | `rpcruntime/integration_test.go` | runtime wrapper 与 cleanup 集成测试 | 覆盖 GC cleanup、显式 release 防双释放和 error text release |
| `rpc_repeat_test.go` | `rpcruntime/rpc_repeat_test.go` | repeated wrapper 单元测试 | 覆盖 signed/float/enum、bool byte wrapper、empty wrapper 和 owned release |

## 明确调整

- `ErrorID` 使用 `int32`，`0` 表示没有 error。
- runtime、ABI 和测试中不使用 `uint32`、`uint64`、`atomic.Uint32`、`atomic.Uint64`、`uint32_t`、`uint64_t`、`u32`、`u64`。
- `NativeArrayElem` 与 `NativeRepeatElem` 不包含 32/64 unsigned 类型。
- `RpcBoolRepeat` 使用 byte buffer 表示 bool repeated，不使用 Go `[]bool` 底层指针。
- `TakeErrorTextForExport` 使用 signed `int32` status 和 signed `int32` error id。

## 暂不迁移代码

| 旧项目文件或模块 | 处理方式 | 原因 | 后续使用方式 |
|---|---|---|---|
| `rpcruntime/active_slot.go` | 暂不迁移 | 旧 active slot 服务于历史多 provider/provider bootstrap 模型，新版需要围绕 dispatcher active server snapshot 重建 | 阶段 2 仅参考测试思路和并发语义 |
| `internal/generator/*` | 暂不迁移 | 阶段 0 不实现 generator；旧 renderer 与新版单插件、多 renderer、`@rpccgo` 注释模型不完全匹配 | 阶段 1 按新版 plan layer 重新设计，择优迁移小型纯函数 |
| `internal/integration/*` | 只迁测试思路 | 阶段 0 只有 runtime package，还没有 dispatcher、adapter、generated service | 后续 adapter 成型后迁移端到端场景，而不是迁旧 bootstrap |
| 旧 examples | 暂不迁移 | 旧 example 体现历史多 server/provider 架构，不符合新版单监听 server 和 dispatcher 模型 | 阶段 7 按新版用户路径重建 example，可复用 proto 和 fixture 数据 |

## 阶段 0 边界

阶段 0 的迁移产物只建立 service 无关的 runtime 基线。dispatcher、active server slot、generator renderer、connect/grpc adapter、native/message converter 和业务 example 都留到后续阶段实现。
