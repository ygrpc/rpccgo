# cgotest

这个目录用于验证 `protoc-gen-rpc-cgo` 生成的 CGO C ABI 是否能在真实 C 程序里端到端调用（构建 `.so` + C 代码链接运行）。

> TL;DR: 运行 `test.sh` 脚本即可执行所有测试。

## 前置条件

- `protoc`
- C 编译器：`cc`/`gcc`/`clang`
- Go 工具链
- Python 3（仅用于重新生成 nanopb 代码）

脚本会自动安装（从当前 workspace）:
- `protoc-gen-rpc-cgo-adaptor`
- `protoc-gen-rpc-cgo`

## 依赖

### nanopb (vendored)
C 测试使用 nanopb 进行 protobuf 编解码。nanopb 运行时已包含在 `c_tests/nanopb/` 目录中。

**依赖说明**：`task nanopb` 会自动创建 Python 虚拟环境（`cgotest/.venv/`）并安装所需依赖，无需手动安装。

**重要**：当 proto 文件（`proto/unary.proto` 或 `proto/stream.proto`）变更时，需要重新生成 nanopb C 代码：

```bash
task nanopb
```

或手动运行：
```bash
cd cgotest
.venv/bin/python3 c_tests/nanopb/generator/nanopb_generator.py \
  -I proto -I ../proto -D c_tests/pb proto/unary.proto proto/stream.proto
```

## go-task 安装与使用

### 安装 go-task

    go install github.com/go-task/task/v3/cmd/task@latest

### 基本用法

查看可用任务：

    task --list

运行完整测试套件：

    task test

运行特定协议的 Go adaptor 测试：

    task adaptor-test PROTOCOL=grpc
    task adaptor-test PROTOCOL=connect
    task adaptor-test PROTOCOL=connect_suffix
    task adaptor-test PROTOCOL=mix

运行特定协议的 C 端到端测试：

    task c-test PROTOCOL=grpc
    task c-test PROTOCOL=connect
    task c-test PROTOCOL=connect_suffix
    task c-test PROTOCOL=mix

构建特定协议的共享库：

    task build PROTOCOL=grpc
    task build PROTOCOL=connect
    task build PROTOCOL=connect_suffix
    task build PROTOCOL=mix

清理生成的文件和构建产物：

    task clean

### 注意事项

如果直接运行 `task` 而不指定任务名称，会显示错误。
请使用 `task --list` 查看所有可用任务，然后明确指定要运行的任务。

## 测试说明

### mix 协议特殊说明
`mix/` 目录的测试包含两类：
1. **通用测试**：使用 testutil 统一套件，与其他协议共享逻辑
2. **协议特定测试**：测试多协议回退行为（如 `TestAllAdaptor_ContextSelection`），仅存在于 mix 目录

这些协议特定测试验证了 `grpc|connectrpc` 回退机制的正确性。
