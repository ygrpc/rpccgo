# Stage 8 Release Checklist

## 目标

发布前用同一套命令验证 root module、examples 子模块、ABI 合同扫描与旧模型扫描。

## 命令顺序

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./internal/generator -count=1
rtk go test ./internal/integration -count=1
rtk go test ./... -count=1

cd examples/minimal-greeter
rtk go test ./... -count=1
rtk go run github.com/magefile/mage run

cd ../full-greeter
rtk go test ./... -count=1
rtk go run github.com/magefile/mage run

cd ../..
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/**'
rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!AGENTS.md'
rtk git status --short
```

## 通过标准

- 所有 `go test` 命令返回 PASS。
- 两个 example 的 `mage run` 返回 0。
- unsigned 扫描应为：`rg` 无输出（通常 exit code 1）。
- 旧模型扫描允许受控命中仅在：
  - `internal/generator/generated_layout_contract_test.go`
- `git status --short` 仅包含本阶段预期改动（可忽略本地 `.vscode/` 未跟踪项）。
