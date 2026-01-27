# cgotest - 端到端测试套件

多协议端到端测试,包含 Go adaptor 测试和 C 程序集成测试。

## 结构

```
cgotest/
├── proto/               # 测试 proto 定义
│   ├── unary.proto
│   └── stream.proto
├── grpc/                # 纯 gRPC 协议测试
│   ├── *.pb.go          # protobuf 生成
│   ├── *_grpc.pb.go     # gRPC stub
│   ├── *_cgo_adaptor.go # adaptor
│   └── adaptor_test.go  # Go 单测
├── connect/             # 纯 ConnectRPC 测试
├── mix/                 # 多协议回退测试 (grpc|connectrpc)
├── connect_suffix/      # ConnectRPC + 包后缀测试
├── cgo_grpc/            # gRPC C ABI 导出 (package main)
│   ├── main.go
│   ├── *_cgo.go         # C ABI 导出
│   └── registry.go      # 注册测试 handler
├── cgo_connect/         # Connect C ABI 导出
├── cgo_mix/             # 多协议 C ABI 导出
├── cgo_connect_suffix/  # ConnectRPC+后缀 C ABI 导出
└── c_tests/             # C 端测试程序
    ├── *.c              # 各协议 C 测试
    ├── run-c-tests.sh   # C 测试运行脚本
    └── libygrpc.{so,h}  # 构建产物
```

## 任务定位

| 任务                   | 位置                        | 备注                         |
| ---------------------- | --------------------------- | ---------------------------- |
| 添加新协议测试         | `{protocol}/`               | 参考 grpc/connect            |
| 修改 Go adaptor 测试   | `{protocol}/adaptor_test.go` | 协议行为验证                 |
| 修改 C ABI 入口        | `cgo_{protocol}/main.go`    | package main + CGO           |
| 添加 C 测试用例        | `c_tests/*.c`               | Binary/Native/TakeReq 验证   |
| 修改测试脚本           | `test.sh`, `run-c-tests.sh` | 构建流程                     |

## 约定

**协议矩阵**:
- `grpc/` - 纯 gRPC
- `connect/` - 纯 ConnectRPC (simple=true)
- `mix/` - 多协议回退 (grpc|connectrpc)
- `connect_suffix/` - ConnectRPC + 包后缀

**C ABI 目录**: `cgo_{protocol}/`
- 必须 `package main`
- 包含 `registry.go` 注册测试 handler
- 独立于 pb/adaptor 包

**C 测试覆盖**:
- Unary: Binary + Native (TakeReq/free 验证)
- Client-streaming: Binary + Native
- Server-streaming: Binary + Native (callback + resp_free)
- Bidi-streaming: Binary + Native (Start/Send/CloseSend)

## 反模式

- ❌ `cgo_*/` 与 `{protocol}/` 混放 (必须独立)
- ❌ C ABI 非 package main
- ❌ 修改生成代码 (应改生成器)

## 运行

```bash
# 全协议矩阵
./test.sh

# 单协议 C 测试
./run-c-tests.sh grpc
./run-c-tests.sh connect
./run-c-tests.sh mix
./run-c-tests.sh connect_suffix
```
