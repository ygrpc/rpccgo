# 变更：为生成的 CGO adaptor 增加基于 `context.Context` 的 protocol 动态选择

## 背景与动机（Why）
当前 `protoc-gen-rpc-cgo-adaptor` 的生成结果会在生成期固定选择单一 protocol（`grpc` 或 `connectrpc`），导致调用方无法在运行时动态选择 dispatch 目标。

事实上 `rpcruntime` 已支持同一个 service 同时注册多个 protocol 的 handler。本变更希望让生成的 adaptor 充分利用该能力：通过 `context.Context` 传入 protocol，从而实现运行时选择。

## 变更内容（What Changes）
- 插件参数 `protocol` 支持逗号分隔的有序列表（例如 `grpc,connectrpc`）。
- 插件总是生成一个 **通用 adaptor 文件** `*_cgo_adaptor.go`：其入口在运行时根据 `ctx` 携带的 protocol 进行选择。
- 插件额外为每个 protocol 生成一个独立文件（例如 `*_grpc_cgo_adaptor.go`、`*_connectrpc_cgo_adaptor.go`），用于显式固定 protocol 的入口。
- 当 `ctx` 未指定 protocol 时，通用 adaptor 按照 `protocol` 参数列表的顺序依次尝试查找已注册 handler，直到命中。
- 若所有 protocol 都未找到可用 handler，则返回确定性的错误。

## 影响范围（Impact）
- 涉及 specs：
  - `rpc-cgo-adaptor`
  - `rpc-dispatch`
- 涉及代码（仅实现阶段）：
  - `cmd/protoc-gen-rpc-cgo-adaptor/*`
  - `rpcruntime/*`（新增/统一 context key 与 helper，用于 protocol 选择）
  - `test/*`（生成物与测试用例调整）

## 兼容性说明（Compatibility Notes）
- 现有单值用法（`protocol=grpc` 或 `protocol=connectrpc`）仍然有效。
- `protocol` 省略时默认仍为 `connectrpc`。
- **行为变化**：若 `ctx` 显式指定了一个不在生成物支持列表中的 protocol，通用 adaptor 将返回错误（此前会始终路由到生成期固定的单一 protocol）。
