# rpccgo

write cgo like rpc

## 架构简介

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

完整设计见 [rpccgo Modular Dispatcher Architecture](docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md)。

## 发布前验证

发布前命令清单见 [Release Verification Checklist](docs/release/verification-checklist.md)。

## 生成要求

Connect remote adapter 依赖 `protoc-gen-connect-go` 生成的同包 service client，并要求生成时启用 `package_suffix=` 和 `simple=true`，例如传入 `--connect-go_opt=package_suffix=` 与 `--connect-go_opt=simple=true`。

gRPC remote adapter 依赖 `protoc-gen-go-grpc` 生成的泛型 service client。每个 service 的 generated output 只能选择 connect 或 gRPC 中的一种 message transport。

生成文件按 service 拆分为 `<proto-prefix>.<service>.<role>[.<contract|transport>].rpccgo.go`；cgo 文件输出到 `cgo_dir` 且使用 `package main`，native/message contract token 显式写入文件名，例如 `.client.native.cgo.rpccgo.go` 与 `.client.message.cgo.rpccgo.go`。完整布局见 [rpccgo Modular Dispatcher Architecture](docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md) 的“生成物布局”章节。

## Examples

- `examples/grpc-greeter`：gRPC unary 路径，覆盖标准 gRPC server/client、gRPC remote adapter、cgo native/message client 和 c-shared C client。
- `examples/full-greeter`：完整 Connect 路径，覆盖 native/message cgo client、Connect local/remote adapter 和三类 streaming。

运行 example：

```bash
cd examples/grpc-greeter
rtk go run github.com/magefile/mage run

cd ../full-greeter
rtk go run github.com/magefile/mage run
```

运行验收测试：

```bash
cd examples/grpc-greeter
rtk go run github.com/magefile/mage test

cd ../full-greeter
rtk go run github.com/magefile/mage test
```

手动启动 full example server：

```bash
cd examples/full-greeter
rtk go run github.com/magefile/mage server
```
