# cmd - protoc 插件

两个 protoc 插件,分别生成 Go adaptor 和 C ABI 导出代码。

## 结构

```
cmd/
├── protoc-gen-rpc-cgo-adaptor/  # Go adaptor 生成器
│   ├── main.go                  # 插件入口
│   └── generate.go              # 生成逻辑
└── protoc-gen-rpc-cgo/          # C ABI 导出生成器
    ├── main.go                  # 插件入口
    ├── generate.go              # 公共代码生成
    ├── generate_unary.go        # Unary RPC 生成
    ├── generate_streaming.go    # 流式 RPC 生成
    └── native.go                # Native 模式工具
```

## 任务定位

| 任务                       | 位置                              | 备注                     |
| -------------------------- | --------------------------------- | ------------------------ |
| 修改 adaptor 生成逻辑      | `protoc-gen-rpc-cgo-adaptor/generate.go` | 协议选择、调度           |
| 修改 C ABI 生成逻辑        | `protoc-gen-rpc-cgo/generate*.go` | Binary/Native/TakeReq    |
| 添加新 RPC 变体            | `protoc-gen-rpc-cgo/generate_*.go` | 参考 Unary/Streaming     |
| Native 模式支持            | `protoc-gen-rpc-cgo/native.go`    | 扁平消息检测             |

## 约定

**Adaptor 生成**:
- 输出文件: `*_cgo_adaptor.go`
- 纯 Go,非 CGO,不导出 C 符号
- 调用 `rpcruntime.Lookup{Grpc|Connect}Handler`

**C ABI 导出生成**:
- 输出文件: `*_cgo.go` (必须 package main)
- 包含 `//export Ygrpc_*` 函数
- 必须独立目录,不可与 pb/adaptor 混放

**函数变体** (根据 proto option 生成):
- Binary: 标准序列化
- TakeReq: 调用方传入 reqFree
- Native: 扁平化参数直传
- NativeTakeReq: 组合变体

## 反模式

- ❌ Adaptor 代码导出 C 符号 (只有 C ABI 层导出)
- ❌ C ABI 与 pb 同包 (必须 package main 独立)
- ❌ 手动编辑生成代码 (应改生成器)

## 构建

```bash
go install ./cmd/protoc-gen-rpc-cgo-adaptor
go install ./cmd/protoc-gen-rpc-cgo
```
