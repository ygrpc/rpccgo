# Stage 4B Migration Inventory

## 已迁移

- cgo message client 到 Go native server 的 mismatch 调用路径：在新版 single dispatcher 上通过 generated codec 转换 request/response。
- cgo message client 到 cgo native server 的 mismatch 调用路径：复用 cgo native server adapter，stream session 在 Start 时捕获 active server snapshot。
- cgo native client 到 cgo message server 的 mismatch 调用路径：复用 cgo message server adapter，native/message payload 由 generated codec 包装。
- converter error precedence、downstream error precedence、stream snapshot、cancel finalize 的测试关注点。

## 参考后重写

- 旧 `native_codec`：只参考字段转换和 wrapper 生命周期测试关注点。新版 codec 留在 generated service runtime，不迁入 `rpcruntime`，并遵守 signed ABI。
- 旧 message/native mismatch fixture：只参考场景，重写为新版 dispatcher、active server snapshot 和 generated codec 测试。
- 旧 wrapper lifecycle 测试：只迁移成功、错误、cancel、finalize 的验收意图，不迁移旧 registry 或 provider 模型。

## 不迁移

- 旧多 registry、多 provider bootstrap。
- 旧 framework selector。
- 旧 active slot/bootstrap 绑定模型。
- 旧 connect/grpc handler adapter 和 remote adapter。
- 与 signed ABI 冲突的 unsigned runtime 或 C ABI 类型。

## Stage 4B 边界

- Stage 4B 不实现 connect/grpc local adapter。
- Stage 4B 不实现 connect/grpc remote adapter。
- Stage 4B 不改变 connect/grpc 标准 client 语义。
- Stage 4B 只在 generated service runtime 内处理 native/message mismatch conversion。

## 验证命令

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -run 'TestRenderCodec|TestRenderRuntime|TestRenderNativeServerCGO|TestGenerate' -count=1
rtk go test ./internal/integration -run 'TestStage4B|TestMessageClientToCGONative|TestNativeContractMismatch|TestMessageContractMismatch|TestConverterLifecycle|TestConverterSnapshot' -count=1
rtk go test ./rpcruntime -count=1
rtk go test ./...
```

另需运行 `AGENTS.md` 中定义的 forbidden unsigned scan 命令；该命令文本不在本文档重复写出，避免文档自身触发扫描。
