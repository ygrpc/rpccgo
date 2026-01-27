# RPCCGO 项目知识库

**生成时间:** 2026-01-27 10:40
**Commit:** 04cdaab
**分支:** main

## 概览

rpccgo - 进程内 CGO-RPC 桥接工具。将 C/C++ 通过 C ABI 直接桥接到 Go gRPC/Connect handler,无网络开销。

**核心技术栈**: Go 1.24 + protobuf + gRPC/ConnectRPC + CGO

## 架构

```
rpccgo/
├── rpcruntime/          # 运行时(dispatch、errors、stream、protocol)
├── cmd/
│   ├── protoc-gen-rpc-cgo/          # C ABI 导出生成器
│   └── protoc-gen-rpc-cgo-adaptor/  # Go adaptor 生成器
├── proto/ygrpc/cgo/     # protobuf option 扩展定义
├── cgotest/             # 端到端测试(grpc/connect/mix/suffix)
│   ├── proto/           # 测试 proto 定义
│   ├── grpc/            # 纯 gRPC 协议测试
│   ├── connect/         # 纯 ConnectRPC 测试
│   ├── mix/             # 多协议回退测试
│   └── c_tests/         # C 端测试程序
├── go.mod               # 根模块
└── cgotest/go.mod       # 测试子模块(replace 根)
```

## 任务定位表

| 任务                       | 位置                        | 备注                         |
| -------------------------- | --------------------------- | ---------------------------- |
| 修改运行时核心逻辑         | `rpcruntime/`               | dispatch/errors/stream       |
| C ABI 生成逻辑             | `cmd/protoc-gen-rpc-cgo/`   | generate*.go, native.go      |
| Go adaptor 生成逻辑        | `cmd/protoc-gen-rpc-cgo-adaptor/` | generate.go            |
| 添加 proto option          | `proto/ygrpc/cgo/`          | options.proto                |
| 测试协议适配               | `cgotest/{protocol}/`       | grpc/connect/mix             |
| C 端集成测试               | `cgotest/c_tests/`          | *.c, run-c-tests.sh          |
| 入口点(main.go)            | `cmd/*/main.go`, `cgotest/cgo_*/main.go` | 6 个入口点 |

## 关键约定

**多模块仓库**:
- 根模块: `github.com/ygrpc/rpccgo`
- 测试子模块: `github.com/ygrpc/rpccgo/cgotest` (通过 replace 指向根)
- 已用 `go.work` 统一管理

**命名模式**:
- Adaptor 生成文件: `*_cgo_adaptor.go` (纯 Go,非 CGO)
- C ABI 导出文件: `*_cgo.go` (package main,含 //export)
- protoc 插件: `protoc-gen-rpc-{variant}` 格式

**协议支持**:
- gRPC: 需 `protoc-gen-go-grpc`
- ConnectRPC: 需 `simple=true` 模式
- 多协议: `protocol=grpc\|connectrpc` (回退机制)

## 反模式(禁止)

- ❌ 混淆 adaptor 与 C ABI 导出(adaptor 是纯 Go,不导出 C 符号)
- ❌ C ABI 导出代码与 pb/adaptor 放同一包(必须独立 package main)
- ❌ ConnectRPC 使用非 simple 模式
- ❌ 修改生成代码手动编辑(应改生成器)

## 独特风格

**生成代码分层**:
1. pb + stub (protoc-gen-go / protoc-gen-go-grpc / protoc-gen-connect-go)
2. Go adaptor (protoc-gen-rpc-cgo-adaptor,调度到 rpcruntime)
3. C ABI export (protoc-gen-rpc-cgo,调用 adaptor)

**错误传递**: 不直接跨 CGO 传递 Go error,用 error registry + TTL(3s)

## 构建与测试

```bash
# 安装插件
go install ./cmd/protoc-gen-rpc-cgo-adaptor
go install ./cmd/protoc-gen-rpc-cgo

# 运行测试
cd cgotest && ./test.sh  # 全协议矩阵端到端

# 仅 C 测试
cd cgotest && ./run-c-tests.sh {grpc|connect|mix|connect_suffix}

# 单元测试
go test ./rpcruntime/...
go test ./cgotest/{protocol}/...
```

## 注意事项

- `cgotest/` 下的 `main.go` 入口点是测试 harness,非标准 cmd 结构
- C ABI 必须构建为 `c-shared` buildmode
- 流式 RPC 句柄生命周期由 runtime 管理
- 协议选择: context 优先,回退到默认协议
