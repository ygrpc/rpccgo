# rpccgo

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

它的目标是让一份 protobuf service 同时暴露：

- Go 侧可实现的 native server contract。
- C 侧可注册的 native/message server callback。
- C 侧可调用的 native/message client ABI。
- Connect 或 gRPC 的标准 server/client transport 接入。

项目当前尚未发布，API 仍以收敛架构和术语为优先。

## 适用场景

rpccgo 适合这些场景：

- 你有 Go service，希望从 C、C++、Rust、Swift 或其他 FFI 运行时调用它。
- 你希望 C 侧按 native 字段 ABI 调用，而不是手写 protobuf marshal/unmarshal。
- 你需要把 C callback 注册成 Go service 的当前实现。
- 你希望同一套 C client 可以切换调用本地 Go native server、cgo server、Connect/gRPC server 或 remote server。
- 你需要 unary、client streaming、server streaming 和 bidi streaming 都走同一套生成合同。

## 核心模型

每个 protobuf service 在运行时只有一个 **current registered server**。用户不直接操作 runtime registry，而是调用 generated registration helper。

支持注册的 server 类型：

- Go native server
- cgo native server
- cgo message server
- Connect handler
- gRPC server
- Connect remote server
- gRPC remote server

Unary 调用每次从 `rpcruntime` server registry 读取当前 registered server。重新注册 server 后，后续 unary 调用会进入新的 server。

Streaming 在 `Start` 时读取一次 current registered server，并创建 `{ServerKind, session}` stream session。后续 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 都固定路由到这个 session，不会因为后续重新注册 server 而改变方向。

更完整的架构说明见 [Runtime Server Registry Architecture](docs/specs/2026-06-04-rpccgo-runtime-server-registry-architecture.md)。

## 安装工具

前置条件：

- Go 1.24+
- cgo 可用的 C toolchain
- `protoc`

rpccgo 是一个 protoc 插件。项目尚未发布时，在本仓库内安装：

```bash
go install ./cmd/protoc-gen-rpc-cgo
```

通常还需要安装 protobuf 的 Go 插件，以及你选择的 transport 插件：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

确保这些命令安装出的二进制在 `PATH` 中：

```bash
which protoc-gen-go
which protoc-gen-rpc-cgo
which protoc-gen-connect-go
which protoc-gen-go-grpc
```

## 选择生成能力

rpccgo 通过 service leading comment 中的 `@rpccgo` 指令选择生成能力：

```proto
// @rpccgo: msg-connect|native
service Greeter {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
}
```

可用 token：

- `msg-connect`：生成 Connect message transport 接入。
- `msg-grpc`：生成 gRPC message transport 接入。
- `native`：生成 native contract、native converter 和 native cgo ABI。

规则：

- 没有 `@rpccgo` 时，默认等价于 `@rpccgo:msg-connect`。
- `native` 单独出现时，默认等价于 `@rpccgo:msg-connect|native`。
- `msg-connect` 和 `msg-grpc` 必须二选一，不能同时选择。
- 未知 token 会报错，例如 `msg-conenct` 不会被静默忽略。
- 没有 `native` token 时，不生成 native server、cgo native server 或 cgo native client artifact。

## 生成代码

Connect service 示例：

```bash
protoc \
  -I proto \
  --go_out=gen --go_opt=paths=source_relative \
  --connect-go_out=gen \
  --connect-go_opt=paths=source_relative \
  --connect-go_opt=package_suffix= \
  --connect-go_opt=simple=true \
  --rpc-cgo_out=gen --rpc-cgo_opt=paths=source_relative \
  --rpc-cgo_opt=cgo_dir=../cmd/rpc \
  proto/greeter.proto
```

Connect 生成要求：

- 使用 `protoc-gen-connect-go` `v1.19.1`，这是当前验证版本。
- 必须设置 `--connect-go_opt=package_suffix=`，让 Connect generated code 与 protobuf Go package 保持同一个 package。rpccgo 生成的 `Register<Service>ConnectHandler` 会直接引用该 package 内的 `<Service>Handler` 和 `<Service>Client`。
- 必须设置 `--connect-go_opt=simple=true`，让 Connect generated code 使用 simple handler/client stream API。rpccgo 的 Connect direct path 按这个签名生成 typed dispatch。

gRPC service 示例：

```bash
protoc \
  -I proto \
  --go_out=gen --go_opt=paths=source_relative \
  --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
  --rpc-cgo_out=gen --rpc-cgo_opt=paths=source_relative \
  --rpc-cgo_opt=cgo_dir=../cmd/rpc \
  proto/greeter.proto
```

`cgo_dir` 用于设置 cgo生成文件目录，路径相对 protobuf Go package 的生成目录解析。cgo 文件会生成到 `package main`，通常放在用于构建 `-buildmode=c-shared` 的 Go package 中。

生成文件按 service 和能力拆分，常见文件包括：

- `<proto>.<service>.runtime.rpccgo.go`
- `<proto>.<service>.server.native.rpccgo.go`
- `<proto>.<service>.server.message.rpccgo.go`
- `<proto>.<service>.codec.rpccgo.go`
- `<proto>.<service>.client.native.cgo.rpccgo.go`
- `<proto>.<service>.client.message.cgo.rpccgo.go`
- `<proto>.<service>.server.native.cgo.rpccgo.go`
- `<proto>.<service>.server.message.cgo.rpccgo.go`
- `rpccgo.exports.cgo.rpccgo.go`

## 注册 Server

生成代码暴露 service-specific registration helper。以 `Greeter` 为例：

```go
err := greeterv1.RegisterGreeterGoNativeServer(server)
err := greeterv1.RegisterGreeterCGONativeServer(server)
err := greeterv1.RegisterGreeterCGOMessageServer(server)
err := greeterv1.RegisterGreeterConnectHandler(handler)
err := greeterv1.RegisterGreeterGRPCServer(server)
err := greeterv1.RegisterGreeterConnectRemoteServer(client)
err := greeterv1.RegisterGreeterGRPCRemoteServer(client)
```

注册成功会替换该 service 的 current registered server。注册失败会清空该 service 的 current registered server 并返回错误。

Connect remote server 和 gRPC remote server 不是特殊 adapter 文件。它们分别是标准 Connect/gRPC client，被注册成 current registered server；调用会经过对应 transport 的网络栈。

## 从 C 调用

生成的 cgo package 需要构建成 shared library：

```bash
go build -buildmode=c-shared -o librpccgo_service.so ./cmd/rpc
```

C 侧 include 生成的 header，然后调用 flat ABI 函数。函数名由 contract、proto Go package namespace、service、method 和 operation 组成。

`native` token 会生成 C native client ABI，C 侧按字段传参：

```c
int32_t err = rpccgoNativeGreeterv1GreeterSayHello(
    name_ptr, name_len, name_ownership,
    city_ptr, city_len, city_ownership,
    &out_message_ptr, &out_message_len, &out_message_ownership);
```

message transport 会生成 C message client ABI，C 侧传入 protobuf encoded bytes：

```c
int32_t err = rpccgoMsgGreeterv1GreeterSayHello(
    request_ptr, request_len,
    &response_ptr, &response_len);
```

这里的 `request_ptr/request_len` 是 borrowed input，不携带 `request_ownership`。Go 会在导出函数内立即把 bytes 解码成 typed protobuf message；调用方只需保证这块 request buffer 在本次调用返回前保持可读，返回后可以自行释放或复用。

返回值 `0` 表示成功，非 `0` 是 runtime error id。错误文本通过 shared exports 读取：

```c
uintptr_t text_ptr = 0;
int32_t text_len = 0;
rpccgoTakeErrorText(err, &text_ptr, &text_len);
rpccgoRelease(text_ptr);
```

这里的 `response_ptr/response_len` 是 Go 返回给 C 的 output buffer；使用完成后调用 `rpccgoRelease` 释放。stream handle 使用 `int32_t`，后续操作通过 handle 继续调用对应 generated stream operation。

## 从 C 注册 Server

生成的 cgo server ABI 允许 C 侧注册 callback，作为 current registered server。

cgo native server 使用 native 字段 ABI callback；cgo message server 使用 protobuf message bytes callback。callback 支持按 method 局部注册：

- unary callback 为 nil 表示该 method 未实现。
- streaming method 的 operation callbacks 必须全 nil 或全非 nil。
- 同一个 kind 的 per-method register 会累积到现有 cgo adapter。
- 不同 kind 的注册会替换 current registered server。

cgo message callback 的 `request_ptr/request_len` 也采用 borrowed input 语义，不生成 ownership 参数。Go 侧只保证 request bytes 在本次同步 callback 调用期间可读；如果 C 侧要跨 callback 或跨 stream 保存内容，必须自行复制。callback 写回的 `response_ptr/response_len` 同样不带 ownership，必须保证返回的 bytes 在本次 rpccgo 调用完成前持续可读，不能返回指向 callback 栈内临时 buffer 的指针。

C 侧传入或返回 `ownership > 0` 的内存前，必须通过 shared export 注册对应的释放函数。使用标准 `malloc` 分配时可以直接注册 `free`：

```c
if (rpccgoRegisterFree(free) != 0) {
    /* handle registration failure */
}
```

C callback 返回 `0` 表示成功。返回错误时，使用 shared export 把错误文本存入 runtime，并返回得到的 error id：

```c
return rpccgoStoreErrorText(message, message_len);
```

## Streaming 行为

rpccgo 支持四类 RPC：

- unary
- client streaming
- server streaming
- bidi streaming

Streaming 的关键点是 `Start` 决定方向：`Start` 时捕获当前 registered server，后续同一个 stream handle 的 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 都继续进入这个 server。重新注册 server 只影响新的 unary 调用 and 新的 stream `Start`。

## Dart/Flutter 接入 (protoc-gen-rpc-cgo-dart)

`protoc-gen-rpc-cgo-dart` 是一个独立的 protoc 插件，专门用于为 Dart/Flutter 平台生成符合 Native Assets 规范的 FFI 客户端。它与 Go 端的 `protoc-gen-rpc-cgo` 配合工作，允许 Dart 侧直接调用由 Go 编译出的 C-shared 动态库（`.so`/`.dylib`/`.dll`）。

### 安装 Dart 插件

```bash
go install ./cmd/protoc-gen-rpc-cgo-dart
```

确保 `protoc-gen-rpc-cgo-dart` 和官方的 `protoc-gen-dart` 插件均在系统的 `PATH` 路径中。

### 生成 Dart 代码

运行以下命令以同时生成 Dart protobuf 协议类与 rpccgo Dart FFI 客户端（目前只有message模式）：

```bash
protoc \
  -I proto \
  --dart_out=lib/gen \
  --rpc-cgo-dart_out=lib/gen \
  --rpc-cgo-dart_opt=paths=source_relative,dart_package=my_dart_package \
  proto/greeter.proto
```

#### 生成参数说明
- `dart_package` (必填)：目标 Dart/Flutter Package 的名字。生成器会为生成的类绑定 native asset ID，格式为 `package:<dart_package>/gen/rpccgo.dart`。
- `paths=source_relative`：控制生成文件的目录结构与输入 proto 一致。

#### 生成产物
在指定的 `--rpc-cgo-dart_out` 目录下，除了标准的 `.pb.dart` 文件外，还会生成：
- `<proto>.<service>.rpccgo.dart`：包含生成的 Dart FFI 客户端类（例如 `GreeterRpccgoClient`），内部使用 `@ffi.DefaultAsset` 和 `@ffi.Native` 绑定 rpccgo 导出的 cgo 符号。
- `rpccgo.dart`：共享的入口与库文件定义。

### 使用方法与动态库绑定

生成的 Dart 代码使用了 Dart 2.19+ 的 Native Assets 规范。在使用时：
1. **基础依赖**：项目 `pubspec.yaml` 至少需要依赖 `ffi` 库。
2. **动态库路径映射**：您需要在 Flutter/Dart 的构建生命周期中（例如通过 build hook 的 `build.dart` 脚本或 `code_assets` 库），将库对应的 asset ID（如 `package:<dart_package>/gen/rpccgo.dart`）映射到您通过 Go 编译的动态库（如 `librpccgo_service.so`）。
3. **调用 API**：
   ```dart
   final client = GreeterRpccgoClient();
   final response = client.SayHello(
     SayHelloRequest(name: 'World'),
   );
   print(response.message);
   ```

*注：关于如何在 Android 下使 Flutter 和 Kotlin JNI 共享同一个 Go `.so` 运行时和内存状态，请参考 [examples/flutter-shared-so](examples/flutter-shared-so/README.md) 示例。*

## Examples

可运行示例放在 `examples/`，example 运行步骤和输出说明放在各自目录的 README 中：

- [Connect Greeter](examples/connect-greeter/README.md)
- [gRPC Greeter](examples/grpc-greeter/README.md)

## 开发与验证

常规验证：

```bash
go test ./...
```

发布级验证流程见 [Release Verification Checklist](docs/release/verification-checklist.md)。
