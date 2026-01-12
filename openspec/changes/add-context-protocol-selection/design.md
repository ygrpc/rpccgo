## 背景（Context）
目前生成的 CGO adaptor 会根据插件参数在生成期硬编码 dispatch protocol，导致即使 `rpcruntime` 能同时承载 `grpc`/`connectrpc` 的 handler，调用方依然无法在运行时做选择。

本变更引入一个统一且稳定的机制：把 protocol 选择携带在 `context.Context` 中，并调整 codegen，让通用 adaptor 能在运行时动态选择合适的 protocol。

## 目标 / 非目标（Goals / Non-Goals）
- Goals:
  - 允许调用方通过 `context.Context` 显式指定 `grpc` 或 `connectrpc`。
  - 当未指定时，按配置的有序列表进行 fallback。
  - 在未显式指定时保持默认行为不变。
  - 当路由失败时返回确定性、可调试的错误。
- Non-Goals:
  - 引入 `grpc` 与 `connectrpc` 之外的新 protocol。
  - 改变底层 handler registry 的模型或注册 API。

## 建议 API 与术语（Proposed API & Terminology）
- 统一使用术语 `protocol`，并与 `rpcruntime.Protocol` 1:1 对齐。
- 选中的 protocol 通过 `ctx` 携带，使用稳定 key：
  - `rpcruntime.ContextKeyProtocol`
- 建议提供 helper（实现阶段）：
  - `rpcruntime.WithProtocol(ctx, rpcruntime.Protocol) context.Context`
  - `rpcruntime.ProtocolFromContext(ctx) (rpcruntime.Protocol, bool)`

## 选择算法（Selection Algorithm：Universal Adaptor）
Inputs:
- `ctx`: 可能包含显式 protocol。
- `supported`: 由插件参数 `protocol` 派生的有序列表。

Algorithm:
1. 如果 `ctx` 中包含 protocol：
   - 若该值不在 `supported` 中，返回 `ErrUnknownProtocol`。
   - 否则仅按该 protocol 做一次 dispatch。
2. 如果 `ctx` 中不包含 protocol：
   - 按 `supported` 顺序遍历。
   - 对每个 protocol 尝试查找 `(protocol, serviceName)` 的 handler。
   - 首次命中则类型断言并调用。
3. 若所有查找都失败，返回 `ErrServiceNotRegistered`。

## 插件参数与生成文件（Plugin Option & Generated Files）
- 插件参数：`protocol` 为逗号分隔列表。
  - 示例：`protocol=grpc,connectrpc`。
  - 省略时默认列表为 `connectrpc`。

每个 proto 输入文件的生成输出：
- 必生成：`*_cgo_adaptor.go`（通用入口；CGO 侧最终调用此处的函数）。
- 额外生成（每个 protocol 一份）：
  - `*_grpc_cgo_adaptor.go`
  - `*_connectrpc_cgo_adaptor.go`

每个“按 protocol 分文件”的 adaptor 可以是薄封装（wrapper），用于在不读取 `ctx` 的情况下强制固定 protocol。

## 错误策略（Error Strategy）
- `ctx` 指定未知/不支持的 protocol：`rpcruntime.ErrUnknownProtocol`。
- 显式 protocol 但未注册 handler：`rpcruntime.ErrServiceNotRegistered`。
- fallback 遍历列表仍未命中任何 handler：`rpcruntime.ErrServiceNotRegistered`。
- handler 类型断言失败：`rpcruntime.ErrHandlerTypeMismatch`。

## 权衡（Trade-offs）
- Pros:
  - 在保持稳定 protocol 标识符的前提下支持运行时选择。
  - 默认行为不变。
- Cons:
  - 配置多个 protocol 时会生成更多文件（代码体积增加）。
  - 需要在 `rpcruntime` 中提供稳定的 context key 以及少量 helper API。
