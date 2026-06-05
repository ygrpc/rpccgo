# rpccgo

write cgo like rpc

## 架构简介

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

完整设计见 [rpccgo Runtime Server Registry Architecture](docs/specs/2026-06-04-rpccgo-runtime-server-registry-architecture.md)。

## 发布前验证

发布前命令清单见 [Release Verification Checklist](docs/release/verification-checklist.md)。

## Examples 使用方法

examples 里的 greeter 服务覆盖 unary、client streaming、server streaming 和 bidi streaming。两个 example 都会构建 c-shared library，并用同一个 C native client 调用当前 registered server。

- `examples/connect-greeter`：演示 Connect handler、Connect remote server、cgo message server、cgo native server。
- `examples/grpc-greeter`：演示 gRPC server、gRPC remote server、Go native server。

### 一键运行

`mage run` 会构建需要的 Go/C artifacts，启动 remote server，然后按顺序切换 current registered server 并运行同一套 C client 调用。

```bash
cd examples/connect-greeter
rtk go run github.com/magefile/mage run

cd ../grpc-greeter
rtk go run github.com/magefile/mage run
```

connect example 会输出四段：

```text
== switch to connect handler registered server ==
== switch to connect remote registered server ==
== switch to cgo message registered server ==
== switch to cgo native registered server ==
```

gRPC example 会输出三段：

```text
== switch to grpc server registered server ==
== switch to grpc remote registered server ==
== switch to go native registered server ==
```

每一段都会运行同一组 native C client 调用：

```text
native unary: hello ffi from c
native collect: collect:ada,grace
native broadcast: broadcast[0]:stream
native broadcast: broadcast[1]:stream
native chat c->server: ada
native chat server->c: chat:ada
native chat c->server: grace
native chat server->c: chat:grace
```

### 验收测试

`mage test` 会运行 example 的 transport/streaming matrix，并构建真实 c-shared C client 做端到端验证。

```bash
cd examples/connect-greeter
rtk go run github.com/magefile/mage test

cd ../grpc-greeter
rtk go run github.com/magefile/mage test
```

也可以直接跑 Go test：

```bash
cd examples/connect-greeter
rtk go test ./...

cd ../grpc-greeter
rtk go test ./...
```

### 手动运行 Connect

启动 Connect server：

```bash
cd examples/connect-greeter
rtk go run ./cmd/server --addr 127.0.0.1:8081
```

在另一个 shell 运行标准 Connect client：

```bash
cd examples/connect-greeter
rtk go run ./cmd/client --url http://127.0.0.1:8081
```

`mage run` 中的 Connect remote server 是独立 server 进程，C client 通过 `--connect-url` 参数注册标准 Connect client 后经网络栈调用它。

### 手动运行 gRPC

启动 gRPC server：

```bash
cd examples/grpc-greeter
rtk go run ./cmd/server --addr 127.0.0.1:8080
```

在另一个 shell 运行标准 gRPC client：

```bash
cd examples/grpc-greeter
rtk go run ./cmd/client --target 127.0.0.1:8080
```

`mage run` 中的 gRPC remote server 是独立 server 进程，C client 通过 `--grpc-target` 参数注册标准 gRPC client 后经网络栈调用它。

### 参数约定

example 不使用环境变量切换 server。`mage run` 内部使用命令行参数传递配置：

- `--server`：选择当前演示的 registered server，例如 `connect_handler`、`connect_remote`、`cgo_message`、`cgo_native`、`grpc_server`、`grpc_remote`、`go_native`。
- `--route`：打印当前路由标签，方便确认输出属于哪个 registered server。
- `--connect-url`：Connect remote server 的 base URL。
- `--grpc-target`：gRPC remote server 的 target。
- `--addr`：手动启动 example server 时的监听地址。

### 关键概念

每次注册都会替换同一个 service 的 current registered server。后续 unary 调用会读取最新 registered server；stream 在 `Start` 时捕获当时的 registered server，后续 `Send`、`Read`、`Finish`、`CloseSend` 和 `Cancel` 都固定路由到该 stream session。

remote server 不是特殊 adapter 文件，也不依赖 `@remote` 注释。它是一个标准 Connect/gRPC client，被注册为 current registered server；调用仍经过对应 transport 的网络栈。
