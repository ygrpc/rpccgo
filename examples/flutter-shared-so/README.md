# Flutter + Kotlin/JNI Shared `.so` Example

这个 example 验证同一个 Android `.so` 同时被两条调用路径消费：

- Flutter 通过 `protoc-gen-rpc-cgo-dart` 生成的 Dart FFI client 调用 rpccgo message cgo exports。
- Kotlin 通过 `System.loadLibrary(...)` 和 JNI 调用同一个 Go `c-shared` runtime。

Dart/JNI 插件只生成平台侧 message client 绑定；Go runtime、server contract、cgo exports 和注册 helper 仍由 `protoc-gen-rpc-cgo` 生成。

两条路径最终都进入同一个 Go service：`SharedSoDemo.ComposeGreeting`。

## 目录

- `proto/shared_so.proto`：全新的 proto 定义。
- `cmd/rpc/`：用于构建 `-buildmode=c-shared` 的 Go package。
- `internal/backend/`：Go service 实现。
- `flutter_app/`：Android-only Flutter app，包含 Kotlin `MainActivity`、build hook 和 UI。

## Android JNI 生成边界

Android JNI 形态是：

```text
Kotlin -> Android C++ JNI shim -> Go c-shared rpccgo C ABI -> Go runtime
```

也就是说，Go shared library 只暴露 `rpccgoMsg...` 等 C ABI；Android 端 JNI adapter 由 C++ 编写并编译成独立 `.so`，再动态链接同一个 Go c-shared `.so`。这样 JNI 头文件、`JNIEnv`、`JavaVM*` 和 `Java_...` export 都留在 Android native 层，Go 侧保持普通 cgo C ABI。

`protoc-gen-rpc-cgo-jni` 的目标生成物是：

- Kotlin typed shim，例如 `SharedSoDemoJni.kt`
- C++ JNI shim，例如 `shared_so.shared_so_demo.jni.cpp`

它不生成或覆盖 `CMakeLists.txt`。Android 工程应自行维护 CMake 配置，把生成的 C++ shim 编译成 JNI adapter library，并链接 Go `-buildmode=c-shared` 产出的 `.so` 和 `.h`。


## 关键验证点

Flutter 侧没有再打包第二份独立 native asset。`flutter_app/hook/build.dart` 为 `gen/rpccgo.dart` 注册 `DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so'))`，因此 Dart `@Native` 绑定会按 SONAME 获取目标库的 handle，再从该 handle 解析符号。`MainActivity` 在启动时先执行：

```kotlin
System.loadLibrary("rpccgo_flutter_shared")
```

Android 已加载该 SONAME 后，再次 `dlopen` 会返回同一个已加载 ELF 对象的 handle。这样 Kotlin/JNI 和 Flutter FFI 使用的是同一份已加载进进程的 `.so`，同时避免 `LookupInProcess()` 依赖 `RTLD_DEFAULT`、无法看到 `RTLD_LOCAL` 符号的问题。

## 生成代码

先确保这些工具在 `PATH` 中：

- `protoc`
- `protoc-gen-go`
- `protoc-gen-connect-go`
- `protoc-gen-rpc-cgo`
- `protoc-gen-rpc-cgo-dart`
- `protoc-gen-rpc-cgo-jni`
- `protoc-gen-dart`

然后在当前目录执行：

```bash
go generate ./...
```

这会同时生成：

- Go protobuf / Connect / rpccgo 文件到 `proto/` 和 `cmd/rpc/`
- Dart protobuf / rpccgo 文件到 `flutter_app/lib/gen/`
- Android Kotlin / C++ JNI shim 到 `flutter_app/android/app/src/main/`

## 构建 Android `.so`

先确保 Android SDK / NDK 可用。脚本会优先读取 `ANDROID_NDK_HOME`，否则自动选取 `${ANDROID_HOME:-$HOME/Android/Sdk}/ndk` 下最新版本。

`flutter build` / `flutter run` 之前不需要再手工复制 `.so`。Android `app` 的 `preBuild` 已经依赖 `flutter_app/tool/build_android_so.sh`，会在 Gradle 打包前自动把 Go `c-shared` runtime 编译到 `android/app/src/main/jniLibs/`。

如果你只想单独验证 native runtime 交叉编译，也可以手工执行：

```bash
bash flutter_app/tool/build_android_so.sh
```

脚本当前会构建两个 ABI：

- `arm64-v8a`
- `x86_64`

产物输出到：

- `flutter_app/android/app/src/main/jniLibs/<abi>/librpccgo_flutter_shared.so`

## 运行 Flutter App

```bash
cd flutter_app
flutter pub get
flutter run -d <android-device>
```

或者直接产出可安装 APK：

```bash
cd flutter_app
flutter build apk --debug
```

然后安装：

```bash
flutter install -d <android-device>
```

界面里输入一个名字后，可以分别点：

- `Call via Flutter FFI`
- `Call via Kotlin/JNI`

两侧结果都应该包含：

- `served_by=go-connect-handler`
- `library=librpccgo_flutter_shared.so`

其中 caller 标签会分别是：

- `flutter-ffi`
- `kotlin-jni`

## 本地静态验证

当前仓库里的 Go tests 会验证：

- Go server 能被生成 runtime 正常调用
- `cmd/rpc` 能构建 host `c-shared` library
- Flutter hook、Kotlin `System.loadLibrary`、Gradle `preBuild` 自动构建和生成的 Dart asset 绑定都已经落盘
