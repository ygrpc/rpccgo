# Release Verification Checklist

发布前用这份清单确认仓库、生成器、runtime、integration 和 examples 仍然保持同一个可用合同。

## 必跑命令

```bash
rtk env GOCACHE=/tmp/rpccgo-go-build go test ./... -count=1
```

## 重点入口

- `rtk env GOCACHE=/tmp/rpccgo-go-build go test ./rpcruntime -count=1`
- `rtk env GOCACHE=/tmp/rpccgo-go-build go test ./internal/generator -count=1`
- `rtk env GOCACHE=/tmp/rpccgo-go-build go test ./internal/integration -count=1`
- `cd examples/grpc-greeter && rtk go run github.com/magefile/mage generate`
- `cd examples/grpc-greeter && rtk go run github.com/magefile/mage test`
- `cd examples/grpc-greeter && rtk go run github.com/magefile/mage run`
- `cd examples/connect-greeter && rtk go run github.com/magefile/mage generate`
- `cd examples/connect-greeter && rtk go run github.com/magefile/mage test`
- `cd examples/connect-greeter && rtk go run github.com/magefile/mage run`
- `cd examples/flutter-shared-so && rtk go run github.com/magefile/mage generate`
- `cd examples/flutter-shared-so && rtk go run github.com/magefile/mage test`

## 合同扫描

修改 runtime 或 ABI 类型后，扫描 unsigned 32/64 位类型；proto 字段本身允许 unsigned，重点确认 length/count/handle/error id 等 proto 无关 helper 未使用 unsigned：

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/release/verification-checklist.md'
```

若命令因为本机 Go build cache 权限失败，可以临时使用 `GOCACHE=/tmp/rpccgo-go-build`。不要把本机 workaround 写入项目设计文档。
