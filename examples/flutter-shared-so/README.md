# Flutter + Kotlin/JNI Shared `.so` Runtime Inspector

这个 example 验证同一个 Android `.so` 同时被两条调用路径消费：

- Flutter 通过生成的 Dart FFI client 直接调用 Go `c-shared` runtime。
- Android `Service` 通过 Kotlin/JNI 调用同一个 Go `c-shared` runtime。

App 启动时会自动启动 `SharedSoRuntimeService`。UI 里的 Dart/Kotlin stream 按钮分别验证 Dart FFI callback stream 和 Activity-owned Kotlin/JNI callback stream 在关闭 Activity、重新打开后的行为。

## 目录

- `proto/shared_so.proto`：runtime state 和 stream 验证协议。
- `cmd/rpc/`：用于构建 `-buildmode=c-shared` 的 Go package。
- `internal/backend/`：Go service 实现。
- `flutter_app/`：Android-only Flutter app。

## 调用路径

```text
Dart UI -> generated Dart FFI client -> librpccgo_flutter_shared.so -> Go service
Android Service -> generated Kotlin shim -> C++ JNI shim -> librpccgo_flutter_shared.so -> Go service
```

Flutter 侧通过 `flutter_app/hook/build.dart` 注册：

```dart
DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so'))
```

因此 Dart FFI 按 SONAME 使用 Android 进程里已加载的同一个 `.so`，不会打包第二份 native asset。

## Flutter lifecycle

`protoc-gen-rpc-cgo-dart` 会在 `flutter_app/lib/gen/rpccgo.dart` 里生成通用的 `RpccgoLifecycleScope`。Flutter app 应该把它放在 `runApp` 最外层：

```dart
void main() {
  runApp(const RpccgoLifecycleScope(child: SharedSoApp()));
}
```

Generated callback stream 会自动注册到 `RpccgoStreamRegistry`。`RpccgoLifecycleScope` 在 Flutter tree dispose 或 app lifecycle `detached` 时调用 `Cancel()` 清理仍注册的 Dart FFI stream，避免 Activity 正常关闭时由用户逐个手动 cancel。

这个 scope 是 Dart lifecycle cleanup，不是 native watchdog；如果 Dart isolate 已经没有机会执行 lifecycle hook，仍需要 native 侧兜底方案。

## Kotlin/JNI callback lifecycle

`protoc-gen-rpc-cgo-jni` 会为 server-streaming 和 bidi-streaming callback API 生成 Activity-owned overload。Activity 里的 callback stream 应把 Activity 作为 owner 传给 generated Kotlin API：

```kotlin
val stream = SharedSoDemoJni.WatchRuntimeStateStartCallback(
    this,
    ReadRuntimeStateRequest.newBuilder()
        .setCaller("kotlin-activity-count-stream")
        .build(),
    listener,
)
```

返回的 `RpccgoResult<RpccgoCallbackStream>` 可用于手动停止：

```kotlin
stream.value?.cancel()
```

如果 Activity 被关闭但用户没有手动 stop，generated wrapper 会在 owner Activity destroyed 时自动 cancel native callback stream，并屏蔽 cancel 后可能到达的 terminal callback，避免继续回调已销毁的 Activity owner。后台业务 stream 应归属 Android `Service`；这类非 Activity owner 需要在 owner 的结束逻辑中保存并 cancel 返回的 `RpccgoCallbackStream`。

## 运行

```bash
go generate ./...
cd flutter_app
flutter pub get
flutter run -d <android-device>
```

`flutter run` / `flutter build` 会通过 Gradle `preBuild` 自动执行 `flutter_app/tool/build_android_so.sh`，把 Go shared library 编译到：

```text
flutter_app/android/app/src/main/jniLibs/<abi>/librpccgo_flutter_shared.so
```

也可以构建并安装 debug APK：

```bash
cd flutter_app
flutter build apk --debug
adb install -r build/app/outputs/flutter-apk/app-debug.apk
adb shell am start -n com.ygrpc.examples.rpccgofluttersharedso/.MainActivity
```

## UI 验证

- `Kotlin Read`：Flutter 发 command 给 Android Service，Service 通过 Kotlin/JNI 读 state。
- `Kotlin Increment`：Service 通过 Kotlin/JNI 修改 state。
- `Dart Read`：Flutter 通过 Dart FFI 读 state。
- `Dart Increment`：Flutter 通过 Dart FFI 修改 state。
- `Dart Start Stream`：Flutter 通过 Dart FFI 启动 server stream，每秒接收 count/state。
- `Dart Stop Stream`：Flutter 通过 Dart FFI cancel stream。
- `Kotlin Start Stream`：Activity 通过 Kotlin/JNI 启动 callback server stream，每秒把 count/state 回传给 Flutter UI。
- `Kotlin Stop Stream`：Activity 通过 Kotlin/JNI cancel callback server stream。
- `Close Activity`：关闭 Activity。按钮本身不调用 stream stop；Dart stream 由 `RpccgoLifecycleScope` cleanup，Kotlin Activity-owned stream 由 generated owner-aware JNI wrapper cleanup。

`Kotlin Start Stream` 故意让 callback listener 归属 Activity，用来验证 generated Kotlin/JNI Activity-owned callback stream 会在 Activity 销毁时自动 cancel，不继续回调已销毁的 UI owner。后台业务 stream 应该归属 Service，而不是 Activity。

两边结果中的 `pid` 和 `instance_address` 一致，且 `value` / `revision` 连续变化，即表示 Kotlin/JNI 和 Dart FFI 进入了同一个 Go runtime/service 实例。

### 验证 Dart stream cleanup

1. 点击 `Dart Start Stream`，确认 `Dart stream: running` 且日志出现 `dart count value=...`。
2. 点击 `Close Activity`。
3. 等待几秒后重新打开 Activity。
4. 预期进程和 foreground service 仍在，UI 显示 `Dart stream: stopped`，logcat 不应出现 `Callback invoked after it has been deleted`、`FfiCallbackMetadata` 或 `SIGABRT`。

### 验证 Kotlin stream cleanup

1. 点击 `Kotlin Start Stream`，确认 `Kotlin stream: running` 且日志出现 `kotlin stream value=...`。
2. 点击 `Close Activity`，不要点击 `Kotlin Stop Stream`。
3. 等待几秒后检查 logcat。
4. 预期进程和 foreground service 仍在，关闭 Activity 后不再持续出现新的 `kotlin stream value=...`，也不应出现 `FlutterJNI was detached from native C++`、`FATAL EXCEPTION`、`JNI DETECTED ERROR` 或 `SIGABRT`。

可用的 adb 检查命令：

```bash
adb shell pidof com.ygrpc.examples.rpccgofluttersharedso
adb shell dumpsys activity services com.ygrpc.examples.rpccgofluttersharedso
adb logcat -d | rg 'Callback invoked|FfiCallback|FlutterJNI was detached|FATAL EXCEPTION|JNI DETECTED ERROR|SIGABRT|data_app_native_crash|kotlin stream'
```
