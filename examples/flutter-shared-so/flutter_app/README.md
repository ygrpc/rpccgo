# Flutter & Kotlin JNI 共享 Go Runtime 指南

本项目是一个 Flutter App 示例，展示了如何配置 Flutter 和 Android Kotlin (JNI) 共同加载并调用**同一个** Go 编译出来的 `.so` 动态链接库，并共享同一个 Go Runtime 实例与内存状态。

---

## 核心机制：为什么能共享同一个 Runtime？

在 Android 平台上，当一个 `.so` 动态链接库被 `dlopen` 打开时：
1. **进程内唯一性**：如果该 `.so` 已经被加载到当前进程中，后续再次使用 `dlopen` 打开相同的 SONAME 时，操作系统的动态链接器不会重新加载一份新的二进制文件，而是直接返回先前已加载的库句柄（Handle）。
2. **状态共享**：由于返回的是同一个句柄，因此所有的全局变量、Go 运行时调度器、以及通过 `rpcruntime` 注册的服务都是全局唯一的。

**实现共享的两个关键配置**：
1. **Kotlin 优先加载**：在 Kotlin [MainActivity.kt](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt) 启动时，率先执行 `System.loadLibrary("rpccgo_flutter_shared")`，确保 `.so` 载入进程。
2. **Flutter 按系统链接加载**：通过 Dart Native Assets 的 [build.dart](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/hook/build.dart) 配置 `DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so'))`，使 Flutter FFI 在解析符号时直接从已加载的进程中解析，而不要重新打包和加载另一份独立的 `.so`。

---

## 实现步骤详解

要在您的 Flutter 项目中实现类似架构，请遵循以下配置步骤：

### 第一步：生成 Go 端 C ABI 与 Android JNI Adapter
在 Go 端的 `main` 包中，我们需要同时暴露两种 API：
1. **FFI 接口**：由 `rpccgo` 插件自动生成。
2. **Android JNI Adapter**：由 Android 端 C++ JNI shim 调用 Go `-buildmode=c-shared` 产出的 `rpccgoMsg...` C ABI。JNI 头文件、`JNIEnv`、`JavaVM*` 和 `Java_...` export 都留在 Android native 层，Go 侧保持普通 cgo C ABI。

### 第二步：配置 Android 原生侧编译与加载
1. **在 Kotlin 中加载 `.so` 并调用 JNI 垫片**：
   在 [MainActivity.kt](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/android/app/src/main/kotlin/com/ygrpc/examples/rpccgofluttersharedso/MainActivity.kt) 的 `companion object` 中加载库，业务代码调用生成的 `SharedSoDemoJni`：
   ```kotlin
   class MainActivity : FlutterActivity() {
       companion object {
           init {
               System.loadLibrary("rpccgo_flutter_shared")
           }
       }
       val result = SharedSoDemoJni.ComposeGreeting(
           ComposeGreetingRequest.newBuilder()
               .setName("Ada")
               .setCaller("kotlin-jni")
               .build()
       )
       // ... 注册 MethodChannel 暴露给 Flutter UI 触发 ...
   }
   ```

2. **配置 Gradle 自动编译 Go 代码**：
   在 [build.gradle.kts](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/android/app/build.gradle.kts) 中，注册一个 Gradle Exec 任务来自动运行 Go 交叉编译脚本 [build_android_so.sh](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/tool/build_android_so.sh)，并将生成的 `.so` 输出到 `src/main/jniLibs/<abi>/`。同时将该任务挂载到 `preBuild`：
   ```kotlin
   val buildSharedSoForAndroid by tasks.registering(Exec::class) {
       workingDir = exampleDir
       commandLine("bash", exampleDir.resolve("flutter_app/tool/build_android_so.sh").absolutePath)
       outputs.dir(projectDir.resolve("src/main/jniLibs"))
   }
   tasks.named("preBuild") {
       dependsOn(buildSharedSoForAndroid)
   }
   ```

### 第三步：配置 Flutter Native Assets 钩子
在 Flutter 项目根目录下创建或编辑 [build.dart](file:///home/zenghp/github.com/ygrpc/rpccgo/examples/flutter-shared-so/flutter_app/hook/build.dart)，指定对应的 `linkMode` 为系统动态加载模式：
```dart
import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';

const _assetName = 'gen/rpccgo.dart';

void main(List<String> args) async {
  await build(args, (input, output) async {
    if (!input.config.buildCodeAssets) return;
    
    output.assets.code.add(
      CodeAsset(
        package: input.packageName,
        name: _assetName,
        linkMode: DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so')),
      ),
    );
  });
}
```
这会通知 Flutter 编译器：`gen/rpccgo.dart` 中引用的底层 cgo 接口，直接链接到系统级加载的 `librpccgo_flutter_shared.so`，而**不要**再打包并加载额外的 native assets。

### 第四步：编写 Flutter Dart 业务代码
在 Flutter 中，直接使用生成的 `rpccgo` 客户端或 MethodChannel 调用 Kotlin 端：
```dart
// 1. 通过 Flutter FFI 直接调用 Go
final client = SharedSoDemoRpccgoClient();
final response = client.ComposeGreeting(
  ComposeGreetingRequest(name: 'Ada', caller: 'flutter-ffi'),
);

// 2. 或者通过 MethodChannel 间接触发 Kotlin JNI 路径调用 Go
final jniChannel = MethodChannel('rpccgo.shared.so/jni');
  final responseFromJNI = await jniChannel.invokeMethod<String>(
  'composeGreeting',
  {'name': 'Ada'},
);
```

---

## 运行与验证

### 1. 生成代码与资源
在 `examples/flutter-shared-so` 目录下运行：
```bash
go generate ./...
```
这会生成 Go/Dart 的契约和绑定文件。

### 2. 编译并启动 Flutter App
确保连接了 Android 设备，然后在 `flutter_app` 目录下执行：
```bash
flutter run
```

### 3. 验证共享 Runtime 状态
在 App 界面中：
1. 点击 **"Flutter write, Kotlin read"** 按钮。
2. 该操作会首先通过 **Flutter FFI** 调用 Go 写入一个全局自增状态，输出最新的状态计数和当前 Go runtime 实例地址。
3. 随后会通过 **MethodChannel** 触发 **Kotlin/JNI** 调用 Go 读取相同的状态。
4. 结果表明，两边读取到的实例地址和自增计数值完全一致，这验证了两者在使用同一个运行时。
