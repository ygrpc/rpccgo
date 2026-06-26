# Flutter + Kotlin/JNI Shared `.so` Runtime Inspector

这个 example 验证同一个 Android `.so` 同时被两条调用路径消费：

- Flutter 通过生成的 Dart FFI client 直接调用 Go `c-shared` runtime。
- Android `Service` 通过 Kotlin/JNI 调用同一个 Go `c-shared` runtime。
- Flutter 通过 Dart FFI 进入 Go `c-shared` runtime，再路由到 Kotlin message server 调用 Android 本机能力。
- Android Activity 通过 Kotlin/JNI 进入 Go `c-shared` runtime，再路由到 Flutter Dart message server。

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
Dart UI -> generated Dart FFI client -> librpccgo_flutter_shared.so -> generated Kotlin message server -> Android CameraManager
Android Activity -> generated Kotlin shim -> C++ JNI shim -> librpccgo_flutter_shared.so -> generated Dart message server -> Flutter handler
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

如果 Activity 被关闭但用户没有手动 stop，Activity 的 `onDestroy` 会 cancel 当前持有的 callback stream；generated wrapper 也会在 owner Activity destroyed 时兜底 cancel native callback stream，并屏蔽 cancel 后可能到达的 terminal callback，避免继续回调已销毁的 Activity owner。后台业务 stream 应归属 Android `Service`；这类非 Activity owner 需要在 owner 的结束逻辑中保存并 cancel 返回的 `RpccgoCallbackStream`。

## Kotlin message server

`AndroidDevice` 是 Android-owned capability service。`SharedSoRuntimeService` 启动时用 generated Kotlin API 注册 unary、server-streaming、client-streaming 和 bidi-streaming message server：

```kotlin
SharedSoDemoJni.RegisterSetTorch { req ->
    // CameraManager.setTorchMode(...)
}
SharedSoDemoJni.RegisterWatchAndroidEcho { req -> /* returns AndroidDeviceWatchAndroidEchoServerHandler */ }
SharedSoDemoJni.RegisterCollectAndroidEcho { /* returns AndroidDeviceCollectAndroidEchoServerHandler */ }
SharedSoDemoJni.RegisterChatAndroidEcho { /* returns AndroidDeviceChatAndroidEchoServerHandler */ }
```

Flutter UI 不通过 `MethodChannel` 直接开灯或跑 stream；它调用 generated Dart FFI client，进入 Go shared runtime 后再路由到 Kotlin registered server：

```dart
const AndroidDeviceRpccgoClient().SetTorch(
  SetTorchRequest(enabled: true, caller: 'dart-ffi-go-kotlin'),
);
const AndroidDeviceRpccgoClient().WatchAndroidEchoStart(
  AndroidEchoRequest(value: 7, caller: 'dart-watch-android'),
);
```

`SetTorch` 验证的是 `Dart -> Go shared .so -> Kotlin message server -> Android framework`；`WatchAndroidEcho`、`CollectAndroidEcho` 和 `ChatAndroidEcho` 不依赖硬件，用来单独覆盖 Android-owned service 的 server-streaming、client-streaming、bidi-streaming RPC shape。

## Dart message server

`FlutterDevice` 是 Flutter-owned capability service。Flutter app 启动时用 generated Dart API 注册 unary 和 server-streaming message server：

```dart
FlutterDeviceRpccgoMessageServer().RegisterDescribeFlutter((req) {
  return (
    value: FlutterEchoResponse(
      message: 'hello from Flutter Dart message server',
      caller: req.caller,
      servedBy: 'dart-message-server',
    ),
    error: null,
  );
});
FlutterDeviceRpccgoMessageServer().RegisterWatchFlutterEcho((req) {
  return (value: FlutterEchoStream(req.caller), error: null);
});
```

`Dart Server Ping` 和 `Dart Server Start Stream` 按钮通过 Android `MethodChannel` 让 Activity 调 generated Kotlin/JNI client，再进入 Go shared runtime 并路由到 Dart registered server。它验证的是 `Kotlin/JNI -> Go shared .so -> Dart message server -> Flutter handler`。

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
- `Torch On/Off`：Flutter 通过 Dart FFI 调 `AndroidDevice.SetTorch`，Go runtime 路由到 Kotlin message server，再调用 Android `CameraManager`。
- `Dart Server Ping`：Activity 通过 Kotlin/JNI 调 `FlutterDevice.DescribeFlutter`，Go runtime 路由到 Dart message server。
- `Dart Server Start Stream`：Activity 通过 Kotlin/JNI 启动 `FlutterDevice.WatchFlutterEcho` callback server stream，每次回调来自 Dart message server。
- `Dart Server Stop Stream`：Activity 通过 Kotlin/JNI cancel Dart-owned callback server stream。
- `Android Server Start Stream`：Activity 通过 Kotlin/JNI 启动 `AndroidDevice.WatchAndroidEcho` callback server stream，用来验证 Android-owned server stream 在 Activity 关闭时的 cleanup。
- `Android Server Stop Stream`：Activity 通过 Kotlin/JNI cancel Android-owned callback server stream。
- `Close Activity`：关闭 Activity。按钮本身不调用 stream stop；Dart stream 由 `RpccgoLifecycleScope` cleanup，Kotlin Activity-owned stream 由 generated owner-aware JNI wrapper cleanup。

`Kotlin Start Stream`、`Android Server Start Stream` 和 `Dart Server Start Stream` 故意让 callback listener 归属 Activity，用来验证 generated Kotlin/JNI Activity-owned callback stream 会在 Activity 销毁时自动 cancel，不继续回调已销毁的 UI owner。后台业务 stream 应该归属 Service，而不是 Activity。

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
4. 预期进程和 foreground service 仍在，logcat 出现 `kotlin stream cancelled on activity destroy`，关闭 Activity 后不再持续出现新的 `kotlin stream value=...`，也不应出现 `FlutterJNI was detached from native C++`、`FATAL EXCEPTION`、`JNI DETECTED ERROR` 或 `SIGABRT`。

### 验证 AndroidDevice stream cleanup

1. 点击 `Android Server Start Stream`，确认 `Android stream: running` 且日志持续出现 `android stream value=... seq=...`。
2. 点击 `Close Activity`，不要点击 `Android Server Stop Stream`。
3. 等待几秒后检查 logcat。
4. 预期进程和 foreground service 仍在，logcat 出现 `android stream cancelled on activity destroy`，关闭 Activity 后不再持续出现新的 `android stream value=...`，也不应出现 `FlutterJNI was detached from native C++`、`FATAL EXCEPTION`、`JNI DETECTED ERROR` 或 `SIGABRT`。

可用的 adb 检查命令：

```bash
adb shell pidof com.ygrpc.examples.rpccgofluttersharedso
adb shell dumpsys activity services com.ygrpc.examples.rpccgofluttersharedso
adb logcat -d | rg 'Callback invoked|FfiCallback|FlutterJNI was detached|FATAL EXCEPTION|JNI DETECTED ERROR|SIGABRT|data_app_native_crash|kotlin stream|android stream'
```

### 验证 AndroidDevice torch

1. 允许 app 的 Camera 权限。
2. 点击 `Torch On`，预期手电筒点亮，日志出现 `torch torch-on camera=... caller=dart-ffi-go-kotlin`。
3. 点击 `Torch Off`，预期手电筒关闭。
4. 拒绝 Camera 权限或使用无闪光灯设备时，UI 应显示明确错误，例如 `camera permission is not granted` 或 `no flash camera available`，进程不应崩溃。
