# rpccgo

write cgo like rpc

## 架构简介

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

完整设计见 [rpccgo Runtime Server Registry Architecture](docs/specs/2026-06-04-rpccgo-runtime-server-registry-architecture.md)。

## 发布前验证

发布前命令清单见 [Release Verification Checklist](docs/release/verification-checklist.md)。

## 生成要求

Connect remote registered server 依赖 `protoc-gen-connect-go` 生成的同包 service client，并要求生成时启用 `package_suffix=` 和 `simple=true`，例如传入 `--connect-go_opt=package_suffix=` 与 `--connect-go_opt=simple=true`。

gRPC remote registered server 依赖 `protoc-gen-go-grpc` 生成的泛型 service client。每个 service 的 generated output 只能选择 connect 或 gRPC 中的一种 message transport：使用 `@rpccgo:msg-connect`、`@rpccgo:msg-connect|native`、`@rpccgo:msg-grpc` 或 `@rpccgo:msg-grpc|native`。不要在同一个 service 上写 `@rpccgo:msg-connect|msg-grpc` 或 `@rpccgo:msg-connect|msg-grpc|native`。

connect 和 gRPC 不能在同一个 protobuf Go package 中同时按当前合同生成。按项目要求，connect-go 使用同包 simple client，grpc-go 也在同包生成 `GreeterClient`、`NewGreeterClient` 等符号；两者同时生成会发生 Go 符号重声明。因此当前 parser 会拒绝同时选择 `msg-connect` 和 `msg-grpc`。

不需要也不支持 `@remote` 注释。`Register<Service>ConnectRemoteServer` 和 `Register<Service>GRPCRemoteServer` 保留现有命名，但它们接收标准 connect/gRPC client，注册为 runtime server registry 中的 current registered server，并返回 `error`。generated runtime 在调用阶段根据 `rpcruntime.ServerKind` 执行 typed dispatch 和 native/message 转换，不再生成独立的 `remote.connect.rpccgo.go` 或 `remote.grpc.rpccgo.go` adapter 文件。

生成文件按 service 拆分为 `<proto-prefix>.<service>.<role>[.<contract|transport>].rpccgo.go`；cgo 文件输出到 `cgo_dir` 且使用 `package main`，native/message contract token 显式写入文件名，例如 `.client.native.cgo.rpccgo.go` 与 `.client.message.cgo.rpccgo.go`。完整布局见 [rpccgo Runtime Server Registry Architecture](docs/specs/2026-06-04-rpccgo-runtime-server-registry-architecture.md) 的“Generated Artifacts”章节。

## Examples

- `examples/grpc-greeter`：完整 gRPC 路径，覆盖标准 gRPC server/client、gRPC remote registered server、native/message cgo client、c-shared C client 和四类 streaming。
- `examples/connect-greeter`：完整 Connect 路径，覆盖 native/message cgo client、Connect local handler/remote registered server 和三类 streaming。

运行 example：

```bash
cd examples/grpc-greeter
rtk go run github.com/magefile/mage run

cd ../connect-greeter
rtk go run github.com/magefile/mage run
```

运行验收测试：

```bash
cd examples/grpc-greeter
rtk go run github.com/magefile/mage test

cd ../connect-greeter
rtk go run github.com/magefile/mage test
```

手动启动 connect example server：

```bash
cd examples/connect-greeter
rtk go run github.com/magefile/mage server
```
