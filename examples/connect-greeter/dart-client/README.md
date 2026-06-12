# Dart Connect Greeter Client

这个 console client 使用 `examples/connect-greeter/proto/greeter.proto` 生成的 Dart protobuf 类型和 rpccgo Dart FFI client，直接加载 connect-greeter 的 c-shared library。

生成 Dart 文件：

```bash
protoc \
  -I ../examples/connect-greeter/proto \
  --dart_out=lib/gen \
  --rpc-cgo-dart_out=lib/gen \
  --rpc-cgo-dart_opt=paths=source_relative \
  ../examples/connect-greeter/proto/greeter.proto
```

构建配套 shared library：

```bash
cd ../examples/connect-greeter
mkdir -p build
go build -buildmode=c-shared -o build/librpccgo_connect_greeter.so ./cmd/rpc
```

运行：

```bash
dart pub get
dart run bin/main.dart
```

也可以指定 library 路径：

```bash
dart run bin/main.dart --library=/path/to/librpccgo_connect_greeter.so
```
