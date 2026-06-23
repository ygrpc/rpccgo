# Android Foreground Service `.so` Example

这个 example 验证 Android `ForegroundService` 持有 JNI callback listener 时，Activity 关闭后 server stream callback 会发生什么。

它是普通 Android Gradle 工程，不使用 Flutter、Dart FFI、Flutter hook 或预构建 `.so`。构建步骤都在当前目录显式执行。

## 结构

- `proto/foreground_service.proto`：一个 unary 健康检查和一个无限 server stream。
- `internal/backend/`：Go service，每秒发送一个 `Tick`，直到 client cancel。
- `cmd/rpc/`：`go build -buildmode=c-shared` 的入口。
- `android_app/`：原生 Android app、foreground service、CMake JNI adapter。

## 生成

```bash
mage generate
```

这会生成：

- Go protobuf / Connect / rpccgo 文件到 `proto/` 和 `cmd/rpc/`
- Kotlin JNI shim 和 C++ JNI adapter 到 `android_app/app/src/main/`

## 构建 Android `.so`

确保 `ANDROID_HOME` 或 `ANDROID_SDK_ROOT` 指向 Android SDK。脚本优先使用 `ANDROID_NDK_HOME`，否则选择 SDK `ndk/` 下最新版本。

```bash
bash android_app/tool/build_android_so.sh
```

输出：

- `android_app/app/src/main/jniLibs/arm64-v8a/librpccgo_android_foreground_service.so`
- `android_app/app/src/main/jniLibs/armeabi-v7a/librpccgo_android_foreground_service.so`
- `android_app/app/src/main/jniLibs/x86_64/librpccgo_android_foreground_service.so`

## 构建 APK

```bash
./android_app/gradlew -p android_app assembleDebug
```

安装：

```bash
adb install -r android_app/app/build/outputs/apk/debug/app-debug.apk
```

## 运行实验

先开 logcat：

```bash
adb logcat | rg RpccgoForegroundService
```

再操作 app：

1. 打开 app；Activity 会启动并绑定 foreground service。
2. 点 `Start normal request`。
3. 确认 logcat 每秒出现 `tick seq=...`。
4. 点 `Finish activity`，或从系统最近任务关闭 Activity。
5. 继续观察 logcat 和通知；此时 stream 仍由 `StreamForegroundService` 持有。
6. 重新打开 app，点 `Stop foreground service`，或点通知里的 `Stop`。
7. 确认 logcat 出现 cancel/done，tick 停止。

Activity 销毁后，logcat 应出现：

```text
bad ui callback failed: activity is not alive
```

或：

```text
bad ui callback failed: captured activity is not alive
```

这模拟正常使用中“所有请求都调用 Service，但一次长生命周期请求把 Activity UI callback 留在 Service 里”时会遇到的问题。即使之后系统或用户重新打开了新的 Activity，Service callback 仍在尝试更新旧 Activity。

## 验证

```bash
mage test
```
