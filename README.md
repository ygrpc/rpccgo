# rpccgo

write cgo like rpc

## 架构简介

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

新版架构让所有 cgo 调用先进入 generated dispatcher，再路由到当前服务实现；运行时保持单监听入口和单 active server。

完整设计见 [rpccgo Modular Dispatcher Architecture](docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md)。

## 发布前验证

发布前命令清单见 [Release Verification Checklist](docs/release/verification-checklist.md)。

## 生成要求

Connect remote adapter 依赖 `protoc-gen-connect-go` 生成的同包 service client，并要求生成时启用 `package_suffix=` 和 `simple=true`，例如传入 `--connect-go_opt=package_suffix=` 与 `--connect-go_opt=simple=true`。

gRPC remote adapter 依赖 `protoc-gen-go-grpc` 生成的泛型 service client。每个 service 的 generated output 只能选择 connect 或 gRPC 中的一种 message transport。

## Examples

- `examples/minimal-greeter`：最小路径，从 proto 生成代码并通过 cgo native/message client 调用 Go native server。
- `examples/full-greeter`：完整路径，覆盖 native/message cgo client、Connect local/remote adapter 和三类 streaming；gRPC adapter 由 generator/integration 测试覆盖。

运行 example：

```bash
cd examples/minimal-greeter
rtk go run github.com/magefile/mage run

cd ../full-greeter
rtk go run github.com/magefile/mage run
```

运行验收测试：

```bash
cd examples/minimal-greeter
rtk go run github.com/magefile/mage test

cd ../full-greeter
rtk go run github.com/magefile/mage test
```

手动启动 full example server：

```bash
cd examples/full-greeter
rtk go run github.com/magefile/mage server
```
