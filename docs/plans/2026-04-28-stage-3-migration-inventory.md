# Stage 3 Native Contract Migration Inventory

## 目标

阶段 3 建立 generated service 级 native contract 调用链。cgo native client 通过 generated dispatcher 调用 Go native server 或 cgo native server，并覆盖 unary、client streaming、server streaming、bidi streaming。

## 已迁移或重写的能力

| 来源 | 新实现位置 | 处理方式 | 原因 |
|---|---|---|---|
| 旧 native server renderer 的接口形态 | `internal/generator/render_native_server.go` | 参考后重写 | Go native server interface 和 streaming session 语义仍有价值，但旧 renderer 绑定旧 registry/provider 模型，不能直接进入新版 dispatcher 架构 |
| 旧 native bridge 的字段映射和错误传播关注点 | `internal/generator/render_native_client_cgo.go` | 参考后重写 | 字段 native ABI、owned wrapper release、error id 返回顺序仍是质量门槛；新版调用必须从 cgo native client 进入 generated dispatcher |
| 旧 cgo native server callback table 形态 | `internal/generator/render_native_server_cgo.go` | 参考后重写 | callback table 适合作为 ABI 表达，但注册目标改为单 active server，并使用 `int32` error id 与 `rpcruntime.StreamHandle` |
| 旧 native streaming 生命周期测试思路 | `internal/integration/native_*_streaming_test.go` | 迁移测试关注点 | Start snapshot、Send、Recv、Finish、CloseSend、Done、Cancel 的生命周期仍需端到端验证 |
| 阶段 2 dispatcher/session primitive | `rpcruntime` + generated runtime glue | 复用 | 通用 active server、stream handle、typed load/take/delete 已由 runtime 提供，阶段 3 只生成 service-specific glue |

## 未迁移的旧架构

| 旧内容 | 处理 | 原因 |
|---|---|---|
| 多 registry、多 provider bootstrap | 不迁移 | 新架构每个 generated service 只有一个 active server，stream 在 Start 时捕获 snapshot |
| framework selector / connect-grpc selector | 不迁移 | 阶段 3 只实现 native direct call，不生成 connect/grpc adapter 或 remote adapter |
| message contract 和 native/message converter | 不迁移 | 留给后续阶段；阶段 3 不处理 protobuf bytes ABI |
| 旧 stream registry | 不迁移 | 已由 `rpcruntime.StreamHandle` 和 dispatcher stream helpers 统一表达 |
| 旧 generated runtime 文件结构 | 不迁移 | 新输出按 `<service>.runtime.rpccgo.go`、native server、cgo server、cgo client 文件族组织 |

## 验收覆盖

- `internal/integration/native_stage3_acceptance_test.go` 汇总执行 Go native server 与 cgo native server 两条路径。
- unary、client streaming、server streaming、bidi streaming 都有临时 generated module 编译和端到端调用。
- cgo native client 调用入口均通过 generated dispatcher 的 `Invoke` 或 `StartStream`。
- streaming 使用 `rpcruntime.StreamHandle`，terminal 操作用 typed take 释放 handle。
- cgo native server error text 通过 `rpcruntime` error store 返回 `int32` error id。

## 验证命令

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1
rtk go test ./internal/integration -count=1
rtk go test ./rpcruntime -count=1
rtk go test ./... -count=1
rtk <AGENTS.md 中的 forbidden unsigned scan>
```
