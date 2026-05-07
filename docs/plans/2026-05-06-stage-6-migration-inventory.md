# Stage 6 迁移清单

## 范围结论

阶段 6 实现 Connect 与 gRPC remote server adapter。两者都作为 message contract active server adapter，把当前 generated service 的调用转发到远端标准 RPC 服务。

阶段 6 不生成 connect/grpc 标准 client，不引入旧 framework selector、多 provider registry 或 bootstrap，也不改变 Stage 5 local handler adapter。

## 迁移或参考

1. 参考旧 `forwarding_plan.go`
   - 作用：旧 planner 把 forwarding 文件作为独立生成物规划。
   - 新版落点：`internal/generator/render_message_plan.go` 的 `ConnectRemote` / `GRPCRemote` 文件族。
   - 为什么参考而不是迁移：新版已经有 `MessageFileFamilyPlan`，只需要复用“remote 是独立文件族”的结构思路，旧 planner 的 framework/provider 概念不能迁入。

2. 参考旧 `native_forwarding_client.go` / `native_forwarding_server.go`
   - 作用：旧代码把远端 transport 调用包装成本地可注册 adapter。
   - 新版落点：`internal/generator/render_connect_remote.go`、`internal/generator/render_grpc_remote.go`。
   - 为什么参考而不是迁移：旧代码围绕 native forwarding 和 Go client registry；新版 remote adapter 是 message contract server adapter，必须直接实现 `<Service>MessageAdapter`。

3. 参考旧 native forwarding integration tests
   - 作用：覆盖真实 transport、streaming 顺序、错误传播。
   - 新版落点：`internal/integration/remote_transport_stage6_acceptance_test.go`。
   - 为什么值得迁移测试思路：remote adapter 的最大风险在端到端 stream lifecycle，仅靠 renderer 字符串测试不够。
   - 本轮实际落地：acceptance 用两个不同 generated package 隔离本地/远端 dispatcher，并在外层先 `go build` 远端服务二进制，再由 fixture 直接启动它，避免 `go run` 父子进程生命周期导致测试挂起。

## 明确不迁移的内容

1. framework selector。
2. 多 provider registry。
3. 旧 bootstrap。
4. GoClientMessageProvider / GoClientNativeProvider server kind。
5. connect/grpc 标准 client 生成模型。

## 验证结果

- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`：`Go test: 164 passed in 2 packages`。
- `rtk go test ./internal/integration -count=1`：`Go test: 49 passed in 1 packages`。
- `rtk go test ./rpcruntime -count=1`：`Go test: 167 passed in 1 packages`。
- `rtk go test ./... -count=1`：`Go test: 380 passed in 5 packages`。
- AGENTS.md 中的 forbidden unsigned scan：`rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-migration-inventory.md'` 退出码 `1` 且无输出。
