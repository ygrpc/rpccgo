# rpccgo
write cgo like rpc

## Runtime：错误信息注册表（Error Registry）

本仓库实现了一个可复用的运行时能力：把“错误信息文本/字节”与一个整型 `errorId` 绑定，使得 CGO 导出函数只需要返回 `errorId != 0`，C 侧即可通过稳定 ABI 拉取错误信息。

### 为什么需要它

在跨语言/跨 ABI 场景下：

- 直接返回 Go 字符串指针不安全（生命周期、GC、跨线程/跨语言释放规则不明确）。
- 直接在导出函数里把错误信息塞进 `char*` 需要定义“谁分配、谁释放、何时释放”的统一规则。

因此这里采用“`errorId` + `Ygrpc_GetErrorMsg` 拉取”的两步模型：

1. Go 侧把错误信息写入全局 registry，拿到 `errorId`。
2. C 侧用 `Ygrpc_GetErrorMsg(errorId, ...)` 拉回一段 `malloc` 分配的字节缓冲，并拿到可调用的 `free`。

### TTL 与清理策略

registry 中的 `errorId -> errorMsg(bytes)` 记录应当“保留约 3 秒”。

- 读取时会检查是否过期：过期即删除并视为 not-found。
- 运行时也会启动一个轻量后台清理循环，定期清掉过期记录。

这既避免了错误消息无限增长，也让 C 侧有一个短窗口来拉取信息。

## Go API（rpcruntime）

实现位于目录 [rpcruntime](rpcruntime)。

- `StoreError(err error) int`：存入 `err.Error()`，返回 `errorId`（`err==nil` 返回 `0`）。
- `StoreErrorMsg(msg []byte) int`：存入任意字节消息，返回 `errorId`。
- `GetErrorMsgBytes(errorId int) ([]byte, bool)`：取回消息（拷贝），不存在/过期返回 `false`。

### Go 侧最小用法示例

```go
id := rpcruntime.StoreError(err)
if id != 0 {
	// 把 id 通过你的 ABI 返回给 C 侧
}
```

## C ABI：Ygrpc_GetErrorMsg

OpenSpec 定义的 ABI 原型如下：

```c
typedef void (*FreeFunc)(void*);

int Ygrpc_GetErrorMsg(int error_id, void** msg_ptr, int* msg_len, FreeFunc* msg_free);
```

语义：

- 返回 `0`：找到消息，输出 `msg_ptr`/`msg_len`/`msg_free`。
- 返回 `1`：未找到或已过期。

缓冲区规则：

- `msg_ptr` 必须是 `malloc` 兼容分配的内存。
- `msg_free` 必须是可调用的释放函数，兼容 `free(msg_ptr)`。
