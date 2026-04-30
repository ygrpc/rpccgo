# Stage 4A Message Contract Migration Inventory

Stage 4A 建立 generated service 级 message contract direct path。cgo message client 通过 protobuf bytes ABI 进入 generated dispatcher；active server 是 cgo message server 时，dispatcher 直接路由到 cgo message callback adapter。

## 已实现

- 新增 message file family plan，复用 `<service>.runtime.rpccgo.go`，并生成独立的 `<service>.client.message.cgo.rpccgo.go` 与 `<service>.server.message.cgo.rpccgo.go`。
- 插件入口按 service adapter selection 渲染 native 或 message direct path。
- 2026-04-30 follow-up 后，message cgo 文件使用唯一文件族名：`<service>.client.message.cgo.rpccgo.go` 与 `<service>.server.message.cgo.rpccgo.go`，避免 mixed native/message service 与 native cgo 文件冲突。
- generated runtime 增加 `MessageAdapter`、message active server registration、message stream session interface 和 `rpcruntime.StreamHandle` helper；native/message registration 写入同一个 `rpcruntime.Dispatcher[<Service>ActiveAdapter]`。
- cgo message unary client 生成 `Call<Service><Method>MessageUnary`，请求和响应都使用 protobuf bytes ABI。
- cgo message client streaming、server streaming、bidi streaming 生成 Start/Send/Finish/Read/Done/CloseSend/Cancel ABI。
- cgo message server 生成 callback table、registration API、unary callback adapter 和三类 streaming callback session adapter。
- message client 边界对 request/response protobuf bytes 执行 `proto.Unmarshal` 校验；失败通过 `rpcruntime.StoreError` 返回 `int32` error id。
- active server 只有 native contract 时，message runtime 返回明确 contract mismatch error，说明 native/message converter 未启用。

## 参考后重写

- 旧 message client renderer：只参考 protobuf bytes ABI、error id 和 response bytes ownership 关注点；新版 cgo message client 总是进入 generated dispatcher。
- 旧 message server renderer：只参考 callback table 与 streaming callback matrix；新版 registration 只写入 service message active server。
- 旧 message export shim：只参考 C callback trampoline 与 error id 传播；新版使用 signed `int32` error id 和 `rpcruntime.StreamHandle`。
- 旧 message mode integration：只迁移 unary/streaming/错误传播/active snapshot 的测试关注点；Stage 4A acceptance 先覆盖 generated-source contract。

## 不迁移

- 旧多 registry、多 provider bootstrap。
- 旧 framework selector。
- 旧 connect/grpc handler 生成路径。
- 旧 connect/grpc remote adapter。
- 旧 native/message codec 与 converter。

这些模块与新版单 generated service dispatcher、单 active server、signed ABI 和 Stage 4A message-to-message direct path 边界冲突。converter 留给 Stage 4B。

## 明确不生成

- `<service>.codec.rpccgo.go`
- connect adapter generated file
- grpc adapter generated file
- remote adapter generated file

## 验证命令

- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`
- `rtk go test ./internal/integration -count=1`
- `rtk go test ./rpcruntime -count=1`
- `rtk go test ./... -count=1`
- AGENTS.md 中的 forbidden unsigned scan。

## 剩余测试缺口

2026-04-30 follow-up 已补充真实 protobuf Go message 生成链下的 cgo message client 到 cgo message server direct-path compile/runtime coverage，覆盖 unary、client streaming、server streaming、bidi streaming、invalid protobuf bytes、callback error 和 terminal lifecycle。Stage 4B 剩余缺口是 native/message contract mismatch converter 路径。
