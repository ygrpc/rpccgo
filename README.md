# rpccgo

write cgo like rpc

## 架构简介

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

新版架构让所有 cgo 调用先进入 generated dispatcher，再路由到当前服务实现；运行时保持单监听入口和单 active server。

完整设计见 [rpccgo Modular Dispatcher Architecture](docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md)。
