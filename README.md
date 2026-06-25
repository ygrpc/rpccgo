# rpccgo

rpccgo 把 C/FFI 调用接入 Go、Connect 或 gRPC 服务，并在 native 字段 ABI 与 protobuf message ABI 之间做转换。

它的目标是让一份 protobuf service 同时暴露：

- Go 侧可实现的 native server contract。
- C 侧可注册的 native/message server callback。
- C 侧可调用的 native/message client ABI。
- Connect 或 gRPC 的标准 server/client transport 接入。

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

本仓库提供三个 protoc 插件。发布版安装：

```bash
go install github.com/ygrpc/rpccgo/cmd/protoc-gen-rpc-cgo@latest
go install github.com/ygrpc/rpccgo/cmd/protoc-gen-rpc-cgo-dart@latest
go install github.com/ygrpc/rpccgo/cmd/protoc-gen-rpc-cgo-jni@latest
```

- `protoc-gen-rpc-cgo`：生成 Go runtime、server contracts、cgo native/message client/server ABI。
- `protoc-gen-rpc-cgo-dart`：生成 Dart/Flutter message FFI client，只覆盖平台侧客户端绑定。
- `protoc-gen-rpc-cgo-jni`：生成 Android Kotlin typed shim 和 C++ JNI shim，只覆盖 Android 侧 message client adapter。

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
which protoc-gen-rpc-cgo-dart
which protoc-gen-rpc-cgo-jni
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
- 必须设置 `--connect-go_opt=package_suffix=`，让 Connect generated code 与 protobuf Go package 保持同一个 package。rpccgo 生成的 Connect registration helper 会直接引用该 package 内的 standard Connect handler 和 client 类型；具体命名统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。
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

生成文件和 symbol 的命名规则统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。

## 注册 Server

生成代码暴露 service-specific registration helper；具体 helper 命名统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。

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

Streaming 的关键点是 `Start` 决定方向：`Start` 时捕获当前 registered server，后续同一个 stream handle 的 `Send`、`Recv`、`Finish`、`CloseSend` 和 `Cancel` 都继续进入这个 server。重新注册 server 只影响新的 unary 调用和新的 stream `Start`。

## Dart/Flutter 接入 (protoc-gen-rpc-cgo-dart)

`protoc-gen-rpc-cgo-dart` 是一个独立的 protoc 插件，专门用于为 Dart/Flutter 平台生成符合 Native Assets 规范的 FFI 客户端。它与 Go 端的 `protoc-gen-rpc-cgo` 配合工作，允许 Dart 侧直接调用由 Go 编译出的 C-shared 动态库（`.so`/`.dylib`/`.dll`）。

它不是完整 rpccgo 生成器的替代品：当前只生成基于 `rpccgoMsg...` C ABI 的 message client 绑定，不生成 Go runtime、server contract、cgo server、native ABI 或 registration helper。这些仍由 `protoc-gen-rpc-cgo` 生成。

### 安装 Dart 插件

```bash
go install github.com/ygrpc/rpccgo/cmd/protoc-gen-rpc-cgo-dart@latest
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
- `dart_package` (必填)：目标 Dart/Flutter Package 的名字。生成器会为生成的类绑定 native asset ID；命名规则统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。
- `paths=source_relative`：控制生成文件的目录结构与输入 proto 一致。

#### 生成产物
在指定的 `--rpc-cgo-dart_out` 目录下，除了标准的 `.pb.dart` 文件外，还会生成 Dart FFI 客户端和共享入口文件；命名规则统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。

### 使用方法与动态库绑定

生成的 Dart 代码使用了 Dart 2.19+ 的 Native Assets 规范。在使用时：
1. **基础依赖**：项目 `pubspec.yaml` 至少需要依赖 `ffi` 库。
2. **动态库路径映射**：您需要在 Flutter/Dart 的构建生命周期中（例如通过 build hook 的 `build.dart` 脚本或 `code_assets` 库），将生成的 native asset ID 映射到您通过 Go 编译的动态库（如 `librpccgo_service.so`）。
3. **调用 API**：
   ```dart
   final client = GreeterRpccgoClient();
   final result = client.SayHello(
     SayHelloRequest(name: 'World'),
   );
   if (result.error != null) {
     throw Exception(result.error);
   }
   print(result.value!.message);
   ```
   Streaming API 显式使用 cgo operation 语义：先调用 stream start method 获取 stream，再调用 `Send()`、`Recv()`、`Finish()`、`CloseSend()` 或 `Cancel()`；命名规则统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。

#### Flutter callback stream 生命周期

server-streaming 和 bidi-streaming 的 Dart callback receive API 会持有 Dart callback。Flutter app 应把生成的 `RpccgoLifecycleScope` 放在 `runApp` 最外层，让生成代码在 Flutter tree dispose 或 app lifecycle `detached` 时自动 cancel 仍注册的 callback stream：

```dart
void main() {
  runApp(const RpccgoLifecycleScope(child: MyApp()));
}
```

用户仍可手动调用 stream 的 `Cancel()`；手动 cancel 后 stream 会从 generated registry 移除，后续 lifecycle cleanup 不会再次 cancel 同一个 stream。

*注：关于如何在 Android 下使 Flutter 和 Kotlin JNI 共享同一个 Go `.so` 运行时和内存状态，请参考 [examples/flutter-shared-so](examples/flutter-shared-so/README.md) 示例。*

## Android JNI 接入 (protoc-gen-rpc-cgo-jni)

`protoc-gen-rpc-cgo-jni` 是面向 Android 的 JNI adapter 生成器。它生成 Kotlin typed shim 和 C++ JNI shim；C++ shim 通过 Go `-buildmode=c-shared` 产出的 C ABI 调用 `rpccgoMsg...` 符号，不在 Go 文件中直接生成 `Java_...` JNI export。

它不是完整 rpccgo 生成器的替代品：当前只生成 Android 侧 message client adapter，不生成 Go runtime、server contract、cgo server、native ABI 或 registration helper。这些仍由 `protoc-gen-rpc-cgo` 生成。

安装 JNI 插件：

```bash
go install github.com/ygrpc/rpccgo/cmd/protoc-gen-rpc-cgo-jni@latest
```

调用形态：

```text
Kotlin -> Android C++ JNI shim -> Go c-shared rpccgo C ABI -> Go runtime
```

生成示例：

```bash
protoc \
  -I proto \
  --rpc-cgo-jni_out=android/app/src/main \
  --rpc-cgo-jni_opt=paths=source_relative \
  --rpc-cgo-jni_opt=jni_class=com.example.app.GreeterJni \
  --rpc-cgo-jni_opt=cpp_dir=cpp/rpccgo \
  --rpc-cgo-jni_opt=kotlin_dir=kotlin \
  --rpc-cgo-jni_opt=rpccgo_header=librpccgo_service.h \
  proto/greeter.proto
```

生成参数说明：

- `jni_class` (必填)：Kotlin JNI facade 的完全限定类名，例如 `com.example.app.GreeterJni`。生成器用它决定 Kotlin 文件路径、Kotlin object 名称，以及 C++ JNI 函数名。
- `rpccgo_header` (必填)：Go `-buildmode=c-shared` 生成的 C header 文件名，例如 `librpccgo_service.h`。C++ JNI shim 会 include 这个 header 来调用 `rpccgoMsg...`、`rpccgoRelease` 和错误读取函数。
- `cpp_dir` (默认 `cpp/rpccgo`)：相对 `--rpc-cgo-jni_out` 的 C++ 输出目录。例如 `--rpc-cgo-jni_out=android/app/src/main` 时，默认输出到 `android/app/src/main/cpp/rpccgo/`。
- `kotlin_dir` (默认 `kotlin`)：相对 `--rpc-cgo-jni_out` 的 Kotlin source 根目录。例如默认输出到 `android/app/src/main/kotlin/<jni_class package>/`。
- `paths=source_relative`：沿用 protoc 常规路径语义，控制按 proto source path 组织生成文件。

生成产物包括 Android C++ JNI shim 和 Kotlin typed shim；命名规则统一记录在 [CONTEXT.md](CONTEXT.md) 的 `Naming Rules`。

生成器不会生成或覆盖 `CMakeLists.txt`。Android 工程应自行维护 CMake 配置，把生成的 C++ 文件编译成独立 JNI `.so`，并动态链接 Go c-shared `.so`。典型运行时会有两个 `.so`：

- `librpccgo_service.so`：Go c-shared library，包含 Go runtime 和 rpccgo C ABI。
- `lib<jni_adapter>.so`：Android C++ JNI adapter，依赖并调用 `librpccgo_service.so`。

Kotlin 侧应确保先加载 Go c-shared library，再加载 JNI adapter：

```kotlin
System.loadLibrary("rpccgo_service")
System.loadLibrary("greeter_jni")
```

### Kotlin callback stream 生命周期

server-streaming 和 bidi-streaming 的 Kotlin `StartCallback` API 会让 JNI 持有 listener 的 global reference。若 listener 归属 Activity，应使用 generated owner-aware overload，把 Activity 传给生成代码：

```kotlin
val stream = GreeterJni.ListStartCallback(
    this,
    ListRequest.newBuilder().build(),
    listener,
)
if (!stream.ok) {
    // handle stream.error
}
```

生成代码会注册 `Application.ActivityLifecycleCallbacks`。当 owner Activity destroyed 时，它会自动 cancel native callback stream，并屏蔽 cancel 之后可能到达的 terminal callback，避免继续回调已销毁的 Activity owner。返回的 `RpccgoCallbackStream` 也可用于手动停止：

```kotlin
stream.value?.cancel()
```

如果 callback stream 归属 Android `Service` 或其他非 Activity owner，应保存返回的 stream handle，并在 owner 的结束逻辑中调用 `cancel()`。

维护 `CMakeLists.txt` 时，不要用 `IMPORTED_LOCATION` 直接链接 Go 生成的 `.so`。Go `-buildmode=c-shared` 产物通常没有 `SONAME`，Android linker 可能会把构建机绝对路径写入 JNI adapter 的 `DT_NEEDED`，导致安装到设备后 `dlopen` 失败。应通过 link search path 加文件名链接：

```cmake
set(RPCCGO_SHARED_LIB_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}")

target_include_directories(greeter_jni PRIVATE
    "${RPCCGO_SHARED_LIB_DIR}"
)

target_link_libraries(greeter_jni PRIVATE
    "-L${RPCCGO_SHARED_LIB_DIR}"
    "-l:librpccgo_service.so"
    log
)
```

C++ JNI shim 默认应包含 `JNI_OnLoad` 和 `JNI_OnUnload`，保存并清除 `JavaVM*`，为后续 stream、callback 或跨线程场景保留 JVM attachment 能力。生成器当前返回 `JNI_VERSION_1_6`，这是 Android NDK 支持的 JNI interface version，不是用户必须配置的生成参数：

```cpp
static JavaVM* javaVM = nullptr;

JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM* vm, void*) {
    javaVM = vm;
    // Android supports JNI 1.6; this declares the JNI interface version requested by this library.
    return JNI_VERSION_1_6;
}

JNIEXPORT void JNICALL JNI_OnUnload(JavaVM*, void*) {
    javaVM = nullptr;
}
```

## Examples

可运行示例放在 `examples/`，example 运行步骤和输出说明放在各自目录的 README 中：

- [Connect Greeter](examples/connect-greeter/README.md)
- [gRPC Greeter](examples/grpc-greeter/README.md)
- [Flutter + Kotlin/JNI Shared .so](examples/flutter-shared-so/README.md)

## 开发与验证

常规验证：

```bash
go test ./...
```

发布级验证流程见 [Release Verification Checklist](docs/release/verification-checklist.md)。
