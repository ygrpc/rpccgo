# Stage 7 迁移清单

## 范围结论

阶段 7 固化 generated layout 与 public API，并新增 `examples/minimal-greeter` 与 `examples/full-greeter`。两个 example 都使用真实 `protoc` 生成路径，调用仍经过 generated dispatcher 和单 active server slot。

两个 example 都新增 `magefile.go`，保留旧项目“进入 example 目录后用 Mage 运行”的操作模型；`Run` 执行真实 demo，`Test` 跑验收，`Server` 暴露长驻服务入口，不迁移旧 bootstrap。

## 迁移或参考

| 旧项目文件或模块 | 本阶段处理 | 作用 | 迁移理由 |
| --- | --- | --- | --- |
| `examples/connect/proto/greeter.proto` | 参考后重写 | greeter unary 与 streaming 场景 | 业务场景适合用户示例，但旧 token、skip 语义和生成布局不适合新版 |
| `examples/connect/internal/backend/backend.go` | 参考后重写 | greeting 与 streaming backend 行为 | 行为可读，API 必须按新版 generated service 和 stream session 接口重写 |
| Stage 3-6 integration fixtures | 参考测试思路 | direct path、converter、local/remote transport、stream lifecycle 验证 | 用于 example acceptance，但不暴露 integration-only reset/helper |

## 明确不迁移

- 旧 `cmd/rpc` generated/export 文件。
- 旧 provider registry、多 provider bootstrap、framework selector。
- 旧 debugserver 与 forwarding bootstrap。

## Hardening 补充

- repeated numeric、enum、bool native ABI 已由 Stage 7 hardening acceptance 覆盖。
- repeated string/bytes 不进入当前 native ABI；planner 明确报错。
- example `Run` 和 full matrix acceptance 使用构建后的 server 二进制，避免 `go run` cleanup 只杀父进程。
- example client 不再使用 `panic(err)` 展示错误处理。
- remote stream cancel 会取消 stream context；gRPC connection 生命周期仍由调用方持有。

## 验证结果

- `rtk go test ./internal/generator -run 'TestRenderMessageClientCGO|TestStage7' -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- 全仓收口验证见阶段 7 计划文档最终记录，当前均 PASS。
