# Dart Connect Greeter Client

这个 console client 使用 `examples/connect-greeter/proto/greeter.proto` 生成的 Dart protobuf 类型和 rpccgo Dart client，并通过 `hook/build.dart` 把 connect-greeter 已经构建好的 c-shared runtime 暴露成 native asset。

当前约定是：

- `protoc-gen-rpc-cgo-dart` 总是生成 native-assets 风格的 Dart 代码。
- `dart_package` 是必填参数，生成器会把 raw bindings 的 asset ID 固定推导成 `package:rpccgo_connect_greeter_dart_client/rpccgo.dart`。
- generator 需要在 generated `@Native` 声明里显式使用这个 asset ID；仅靠 `lib/rpccgo.dart` 的 re-export 不会改变其他 library 的默认 asset ID。
- `hook/build.dart` 只消费现有 runtime artifact，不会重新执行 `go build`。

下面的命令默认都在 `examples/connect-greeter/dart-client` 目录下执行。

生成 Dart 文件：

```bash
protoc \
  -I ../proto \
  --dart_out=lib/gen \
  --rpc-cgo-dart_out=lib/gen \
  --rpc-cgo-dart_opt=paths=source_relative,dart_package=rpccgo_connect_greeter_dart_client \
  ../proto/greeter.proto
```

构建配套 shared library。这个产物会被 Go/C 示例和 Dart build hook 共用：

```bash
cd ..
mkdir -p build
go build -buildmode=c-shared -o build/librpccgo_connect_greeter.so ./cmd/rpc
```

安装依赖：

```bash
dart pub get
```

运行。`dart run` 会自动执行 `hook/build.dart`，把 `../build/librpccgo_connect_greeter.so` 复制到 Dart SDK 的 code asset 输出目录，再由 native-assets 风格 generated client 在运行时加载：

```bash
dart run bin/main.dart
```

也可以打包 CLI；`dart build` 同样会自动执行 build hook：

```bash
dart build cli -t bin/main.dart
```

## 当前限制

- 需要 Dart SDK `>= 3.10.0`，因为 build hooks 从 Dart 3.10 才开始支持。
- 当前 `hook/build.dart` 是 host-only 方案，并且代码里直接限制为 `linux` target：它固定读取 `../build/librpccgo_connect_greeter.so`，不负责按 target OS/ABI 重新编译或挑选不同 runtime artifact。
- 不再支持 `--library` 参数或 `RPCCGO_CONNECT_GREETER_LIB` 环境变量覆盖；runtime 位置固定由 build hook 决定。
- 这个 package 依赖当前 monorepo 目录布局：`hook/build.dart` 必须能从 `examples/connect-greeter/dart-client` 找到同仓库下的 `../build/librpccgo_connect_greeter.so`。
- 如果未来要让 Android app 和 Flutter app 共享同一个 rpccgo runtime，需要把 `../build/` 收敛成按 target OS/ABI 分目录的预编译产物，再让 build hook 按 `targetOS/targetArchitecture` 选择对应文件；这个示例目前还没做到这一步。
