# rpcruntime - 运行时核心

进程内 RPC 调度运行时,提供处理器注册、协议选择、错误传递、流管理。

## 结构

```
rpcruntime/
├── dispatch.go          # Handler 注册表 (gRPC/Connect)
├── dispatch_errors.go   # 调度错误定义
├── errors.go            # 错误注册表 API
├── errors_ttl.go        # 错误 TTL 清理
├── protocol_context.go  # 协议上下文 (WithProtocol/ProtocolFromContext)
├── stream_handle.go     # 流句柄管理
├── connect_stream.go    # Connect 流式适配器
├── background_context.go # 后台上下文工具
└── *_test.go            # 单元测试
```

## 任务定位

| 任务                   | 文件                     | 备注                         |
| ---------------------- | ------------------------ | ---------------------------- |
| 修改注册逻辑           | `dispatch.go`            | RegisterGrpcHandler/Connect  |
| 添加新协议支持         | `protocol_context.go`    | Protocol 类型 + context key  |
| 错误传递机制           | `errors.go`, `errors_ttl.go` | StoreError/GetErrorMsgBytes |
| 流式 RPC 句柄          | `stream_handle.go`       | StreamHandle 生命周期        |
| Connect 流式适配       | `connect_stream.go`      | ConnectStream 接口包装       |

## 约定

**错误 TTL**: 错误记录保留 3 秒,过期自动清理 (errors_ttl.go)

**协议常量**:
- `ProtocolGrpc = "grpc"`
- `ProtocolConnectRPC = "connectrpc"`

**调度错误**:
- `ErrServiceNotRegistered` - 服务未注册
- `ErrHandlerTypeMismatch` - 处理器类型不匹配
- `ErrUnknownProtocol` - 未知协议

## 反模式

- ❌ 直接跨 CGO 返回 Go error (使用 StoreError + error ID)
- ❌ 长时间持有 StreamHandle (注意生命周期)
- ❌ 手动管理错误清理 (TTL 自动处理)

## 测试

```bash
go test ./rpcruntime/...
go test -race ./rpcruntime/...  # 检测竞态
```
