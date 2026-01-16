# cgotest

这个目录用于验证 `protoc-gen-rpc-cgo` 生成的 CGO C ABI 是否能在真实 C 程序里端到端调用（构建 `.so` + C 代码链接运行）。

## 前置条件

- `protoc`
- C 编译器：`cc`/`gcc`/`clang`
- Go 工具链

脚本会自动安装（从当前 workspace）:
- `protoc-gen-rpc-cgo-adaptor`
- `protoc-gen-rpc-cgo`

## 一键运行（推荐）

在仓库根目录执行：

- `./cgotest/test.sh`

它会：
- 生成各协议矩阵的 Go 代码 + adaptor
- 跑 Go 单测（用于验证 adaptor 行为）
- 生成 rpc-cgo 导出代码，构建 `c_tests/libygrpc.so` + `c_tests/libygrpc.h`
- 编译并运行 C 端到端测试程序

## 只跑某个协议的 C 端到端

在 `cgotest/` 目录执行：

- `./run-c-tests.sh connect`
- `./run-c-tests.sh grpc`
- `./run-c-tests.sh connect_suffix`
- `./run-c-tests.sh mix`

## C 测试覆盖面

`c_tests/` 下的程序覆盖：
- Unary：Binary + Native（含 TakeReq/free 策略验证）
- Client-streaming：Binary + Native（含 TakeReq/free 策略验证）
- Server-streaming：Binary + Native（callback + resp_free 验证）
- Bidi-streaming：Binary + Native（Start 注册回调、Send、CloseSend）

另外在 `connect`/`grpc`/`connect_suffix` 里会额外编译运行 `unary_test_mismatch`：
- 通过设置 `YGRPC_PROTOCOL` 为“错误协议”来触发错误路径
- 验证非 0 error id + `Ygrpc_GetErrorMsg` 返回消息与 free 函数可用
