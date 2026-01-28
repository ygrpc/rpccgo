# cgotest

这个目录用于验证 `protoc-gen-rpc-cgo` 生成的 CGO C ABI 是否能在真实 C 程序里端到端调用（构建 `.so` + C 代码链接运行）。

## 前置条件

- `protoc`
- C 编译器：`cc`/`gcc`/`clang`
- Go 工具链

脚本会自动安装（从当前 workspace）:
- `protoc-gen-rpc-cgo-adaptor`
- `protoc-gen-rpc-cgo`

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