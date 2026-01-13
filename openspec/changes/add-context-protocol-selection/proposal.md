# 变更：为生成的 CGO adaptor 增加基于 `context.Context` 的 protocol 动态选择（支持按 option 列表 fallback）

## 背景与动机（Why）
当前 `protoc-gen-rpc-cgo-adaptor` 的生成结果会在生成期固定选择单一 protocol（`grpc` 或 `connectrpc`），导致调用方无法在运行时动态选择 dispatch 目标。

事实上 `rpcruntime` 已支持同一个 service 同时注册多个 protocol 的 handler。本变更希望让生成的 adaptor 充分利用该能力：通过 `context.Context` 传入 protocol，从而实现运行时选择。

## 变更内容（What Changes）
- 插件参数 `protocol` 支持逗号分隔的有序列表（例如 `grpc,connectrpc`）。
- 如果没有传入 `protocol` 参数，默认行为 SHALL 等价于 `protocol=connectrpc`。
- 当需要与 `protoc-gen-connect-go` 的默认输出（默认 `package_suffix=connect`，生成到独立子包）配合时，插件可通过额外选项 `connect_package_suffix` 指定该 suffix，从而推导 connect handler interface 的 import path（详见 design/spec；规则要点：子包名为 `<current-go-package-name><suffix>`）。
- 插件仍然只生成同一个文件 `*_cgo_adaptor.go`（不因协议拆分文件）。
- 生成代码的 handler 查找逻辑由 `protocol` 列表驱动：
  - 当 `protocol` 仅有一个值时，生成的代码 SHALL 只尝试获取该 protocol 的 handler（不尝试其他 protocol）。
  - 当 `protocol` 有多个值时：
    - 若 `ctx` 携带 protocol，则仅按该 protocol 进行 lookup；
    - 若 `ctx` 未携带 protocol，则按列表顺序依次尝试 lookup，直到命中可用 handler。
- 若按上述规则最终未找到可用 handler，则返回确定性的错误。

## 影响范围（Impact）
- 涉及 specs：
  - `rpc-cgo-adaptor`
  - `rpc-dispatch`
- 涉及代码（仅实现阶段）：
  - `cmd/protoc-gen-rpc-cgo-adaptor/*`
  - `rpcruntime/*`（新增/统一 context key 与 helper，用于 protocol 选择）
  - `cgotest/*`（生成物与测试用例调整，包含混合注册场景）

## 兼容性说明（Compatibility Notes）
- 现有单值用法（`protocol=grpc` 或 `protocol=connectrpc`）仍然有效。
- `protocol` 省略时默认仍为 `connectrpc`。
- **行为变化**：若 `ctx` 显式指定了一个不在生成物支持列表中的 protocol，adaptor 将返回错误（不会 fallback 到其他 protocol）。
