# Stage 5 迁移清单

## 范围结论

阶段 5 完成了本地 Connect handler adapter 与 gRPC server adapter。两者都只作为入站 transport adapter，把标准 RPC 请求转成 message contract，然后进入现有 dispatcher 与 Stage 4B converter。

阶段 5 没有迁移 remote adapter、framework selector、bootstrap，也没有把 Connect/gRPC 监听端注册成新的 active server 类型。

## 直接复用与迁移

1. 复用现有 generated runtime / dispatcher / bridge 结构
   - 文件：`internal/generator/render.go`、`internal/generator/render_message_plan.go`、`internal/generator/render_connect_server.go`、`internal/generator/render_grpc_server.go`
   - 作用：Connect/gRPC 本地 adapter 不直接实现路由，而是统一调用 `New<Service>CGOMessageClientBridge()` 进入 dispatcher。
   - 为什么复用而不是重写：Stage 4A/4B 已经把 snapshot、converter、message stream lifecycle 放进 bridge + dispatcher，继续复用能保证 transport 只是入站壳，不会复制一套路由逻辑。

2. 复用 Stage 4B 的 native/message converter
   - 文件：generated `<service>.codec.rpccgo.go` 与 `<service>.runtime.rpccgo.go`
   - 作用：当本地 Connect/gRPC 请求命中 go native server 或 cgo native server 时，仍然通过已有 converter 完成 message/native ABI 转换。
   - 为什么复用而不是重写：converter 已覆盖 unary 与三类 streaming 的 mismatch path，再额外做 transport 专用转换层只会引入重复和回归面。

3. 扩展 cgo message server 的 EOF helper
   - 文件：`internal/generator/render_message_server_cgo.go`
   - 作用：为本地 server-streaming / bidi handler 提供可识别的 `io.EOF` error id，让 transport adapter 能区分“正常结束”和“失败返回”。
   - 为什么补在这里：EOF 是 cgo message callback 和 Go transport 之间的边界问题，放在 message server generator 里最靠近真实来源，也不会污染 `rpcruntime`。

## 参考但未迁移

1. 旧仓库 Connect/gRPC transport 思路
   - 参考对象：`/home/zenghp/github.com/ygrpc/rpccgo-old`
   - 使用方式：只参考“标准 Connect/gRPC handler / ServiceDesc 应该长什么样”，没有迁移旧实现代码。
   - 不迁移原因：旧仓库包含多 registry、多 provider bootstrap、framework selector，与新版单 dispatcher、单 active server 约束冲突。

2. 标准库与官方 API
   - Connect：`connect.NewUnaryHandler`、`NewClientStreamHandler`、`NewServerStreamHandler`、`NewBidiStreamHandler`
   - gRPC：`grpc.ServiceDesc`、`grpc.ServiceRegistrar.RegisterService`、`grpc.GenericServerStream`
   - 使用原因：阶段 5 的目标是“标准 transport 入站适配”，直接按官方 API 生成，比迁移旧项目里包着额外框架层的代码更稳。

## 明确不迁移的内容

1. `connect remote server adapter`
2. `grpc remote server adapter`
3. 旧项目的 framework selector / provider registry / bootstrap
4. 自定义 Connect client 或 gRPC client 包装层

## 验证结果

- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`
- `rtk go test ./internal/integration -count=1`
- `rtk go test ./rpcruntime -count=1`
- `rtk go test ./... -count=1`
- `rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md'`

结果：全部通过；unsigned 扫描无命中。
