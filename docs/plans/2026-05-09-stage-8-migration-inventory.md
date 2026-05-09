# Stage 8 迁移清单

## 范围结论

Stage 8 只迁移旧项目的“边界测试语义”，不迁移旧项目的“架构模型”。当前 `rpccgo` 已经以 single dispatcher + single active server slot 为核心，Stage 8 的目标是补齐兼容与发布前验收，而不是回退到旧的 provider/registry/bootstrap 体系。

## 迁移或参考

| 旧项目文件或模块 | 当前处理 | 作用 | 迁移理由 |
| --- | --- | --- | --- |
| `rpccgo-old/internal/generator/message_export_shim_cgo.go` | 参考语义后在当前 renderer 重写 | message request/response bytes 的 `proto.Unmarshal` 边界校验与 error id 传播 | 旧实现证明 cgo 边界应尽早拦截 invalid protobuf；新版路径已不同，不能直接搬代码 |
| `rpccgo-old/internal/generator/message_client.go` | 参考语义后在当前 renderer 重写 | message client 侧 response protobuf 校验与错误返回 | 继续强化 direct-path bytes ABI，同时避免“坏数据延迟在下游爆炸” |
| `rpccgo-old/internal/integration/message_mode/integration_test.go` | 迁移为 generated-source acceptance 语义 | message unary/streaming 的 bytes 输入输出、protobuf roundtrip、错误传播 | 场景价值高，且与 Stage 8 message ABI hardening 直接对应 |
| `rpccgo-old/internal/integration/native_mode/integration_test.go` | 迁移为 generated-source acceptance 语义 | native request/response wrapper、owned/borrowed release、empty input 行为 | 与 Stage 8 的 empty input + memory release 两个核心任务直接对齐 |
| `rpccgo-old/internal/integration/native_forwarding/integration_test.go` | 迁移 remote lifecycle 核心语义，不迁移旧 forwarding 架构 | cancel/onDone、stream terminal、remote 回调链路的行为断言 | Stage 8 仍需 remote terminal lifecycle hardening；保留测试价值但不保留旧模型命名 |
| `rpccgo-old/internal/integration/both_mode/integration_test.go` | 迁移“共同编译 + 单 active server”语义 | 验证 native/message 生成物可共存，但运行时保持单 active server | 能防止回流“双 provider bootstrap”的错误心智 |
| `rpccgo-old/rpcruntime/*_test.go` | 按需补强当前测试 | release/error text/cleanup/length/repeat wrapper 边界 | 当前 runtime primitive 已迁入新版，补测比重写更稳更快 |
| `rpccgo-old/docs/specs/2026-03-30-request-side-empty-input-normalization-design.md` | 参考合同并按新版落地 | request-side empty input 与 ownership 归一化语义 | 合同与 Stage 8 目标一致，且新版已具备 `EmptyRpc*` 基础设施 |

## 当前实现差距（映射 Stage 8 任务）

1. request-side empty input 语义在 message/native 路径尚未完全统一。
2. message bytes ABI 虽已有部分 `proto.Unmarshal` 校验，但需要在 unary + 三类 streaming 全覆盖并用 acceptance 固化。
3. stream terminal lifecycle 仍需发布前矩阵化验收（重复 terminal、invalid handle、EOF/Done、cancel 一次性）。
4. generated 层的 owned/borrowed request 与 error text 生命周期需要补组合验收，避免只靠 runtime 单测。
5. 文档和 release checklist 需要固定“发布前验证命令集合”，确保 root module 与两个 examples 一起验收。

## Stage 8 Task 6 对齐补充

- generated layout 合同新增旧模型禁用词：`provider registry`、`framework selector`、`multi provider`、`dual provider`、`bootstrap`、`goclient.export`、`goserver.export`、`native_forwarding_client`、`native_forwarding_server`。
- README 仅保留最小“发布前验证入口”，具体命令收敛到 Stage 8 release checklist，不在 README 展开。

## 明确不迁移

- 旧 provider registry、多 provider bootstrap、framework selector。
- 旧 forwarding 目录与命名模型（`native_forwarding_client` / `native_forwarding_server`）作为架构事实。
- 旧 generated/export 文件族（`*.goclient.export.*`、`*.goserver.export.*`）。
- 旧 debugserver example 架构作为 rpccgo 主路径（包括 `examples/connect|grpc/cmd/debugserver` 的演示模型）。
- Flutter discovery 相关 example 与 debugserver 配置链路。

## 验证入口

- root module:
  - `rtk go test ./rpcruntime -count=1`
  - `rtk go test ./internal/generator -count=1`
  - `rtk go test ./internal/integration -count=1`
  - `rtk go test ./... -count=1`
- examples 子模块:
  - `cd examples/minimal-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run`
  - `cd ../full-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run`
- 合同扫描:
  - `rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/**'`
  - `rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!AGENTS.md'`

## 审计证据摘要

- message protobuf 边界校验来源：
  - `rpccgo-old/internal/generator/message_export_shim_cgo.go`
  - `rpccgo-old/internal/generator/message_client.go`
- integration 语义来源：
  - `rpccgo-old/internal/integration/message_mode/integration_test.go`
  - `rpccgo-old/internal/integration/native_mode/integration_test.go`
  - `rpccgo-old/internal/integration/native_forwarding/integration_test.go`
  - `rpccgo-old/internal/integration/both_mode/integration_test.go`
- 不迁移模型证据：
  - `rpccgo-old/internal/generator/go_client_registry.go`
  - `rpccgo-old/internal/generator/native_forwarding_client.go`
  - `rpccgo-old/internal/generator/native_forwarding_server.go`
  - `rpccgo-old/examples/connect/cmd/debugserver/main.go`
  - `rpccgo-old/examples/grpc/cmd/debugserver/main.go`
