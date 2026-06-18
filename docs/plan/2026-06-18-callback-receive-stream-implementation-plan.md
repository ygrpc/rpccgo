# Callback Receive Stream Implementation Plan

## 目标

为 C client export 的 server streaming 与 bidi streaming `Start` 增加可选 callback receive 模式：

- `onRecv` 与 `onDone` 同时非 nil 时启用后台接收。
- 任一 callback 为 nil 时不启用，保持现有手动 `Recv` 模式。
- 启用后用户不再手动调用 `Recv`；`Recv` export 必须返回显式错误。
- `Cancel` 仍有效，用于主动取消后台接收，并通过 `onDone` 返回非 0 error id。
- server streaming client 侧不再暴露 `Finish`。
- C server callback ABI 保留 server-stream `Finish`；它是 server implementation 的 cleanup / natural completion 回调，不是 client-side `Finish`。

## 范围

本计划只修改 generated C client projection 与相关 stream capability：

- message C client export
- native C client export
- generated stream operation projection
- 必要的 `rpcruntime` client-side server stream interface
- generator tests 与必要 integration tests

不修改：

- Go / Connect / gRPC server handler contract
- C message/native server callback registration 形状
- server registry 与 stream registry 架构
- Kotlin/Dart 高层 API，除非生成签名变化导致编译需要同步

## ABI 形状

message contract 的 callback typedef 使用 serialized protobuf bytes：

```c
typedef int32_t (*RpccgoOnRecvCallback)(int32_t stream, uintptr_t response_ptr, int32_t response_len);
typedef void (*RpccgoOnDoneCallback)(int32_t stream, int32_t err_id);
```

native contract 的 `onRecv` 使用对应 native response C ABI slot，沿用现有 native `Recv` export 的 response lowering 与 ownership slot。

`onRecv` 收到的 response buffer/ownership 释放语义沿用现有 `Recv` export：用户处理完后调用 generated shared release API。

## 行为

### Start

server streaming:

```go
func rpccgoMsgGreeterv1GreeterBroadcastStart(
    requestPtr C.uintptr_t,
    requestLen C.int32_t,
    handle *C.int32_t,
    onRecv C.RpccgoOnRecvCallback,
    onDone C.RpccgoOnDoneCallback,
) C.int32_t
```

bidi streaming:

```go
func rpccgoMsgGreeterv1GreeterChatStart(
    handle *C.int32_t,
    onRecv C.RpccgoOnRecvCallback,
    onDone C.RpccgoOnDoneCallback,
) C.int32_t
```

启用条件：

- `onRecv != nil && onDone != nil`：启用 callback receive，`Start` 成功后启动后台 goroutine 循环 `Recv`。
- 其他组合：不启用 callback receive，不报错，保持手动 `Recv`。

### Recv

启用 callback receive 的 stream handle 再调用 `Recv` 时返回显式错误，例如：

```text
rpccgo: stream receive is owned by callback receive mode
```

未启用 callback receive 时，`Recv` 行为保持现状。

### Cancel

`Cancel` 始终有效。

启用 callback receive 时：

- `Cancel` 取消 stream context。
- 后台接收循环停止。
- `onDone(stream, err_id)` 必须调用一次。
- cancel 导致的 `err_id` 必须非 0，便于用户区分正常结束与主动取消。

### Done

`onDone` 调用规则：

- 正常 EOF / server natural completion：`err_id == 0`
- `onRecv` 返回非 0：停止接收，并将该 error id 传给 `onDone`
- `Cancel` / stream error / decode or encode error：存入 `rpcruntime.StoreError` 后以非 0 error id 传给 `onDone`

`onDone` 必须最多调用一次。

## 实现步骤

1. 调整 stream capability
   - 修改 server streaming client-side capability，移除 client `Finish`。
   - 保留 server-side / C server callback 的 server-stream `Finish`。
   - 更新相关 projection tests。

2. 调整 runtime interface
   - `rpcruntime.ServerStreamingClient` 移除 `Finish(context.Context) error`。
   - 本地 server-stream client 删除或隐藏 client `Finish` 能力。
   - `ServerStreamingServer.FinishRequested` 如仍只由旧 client `Finish` 驱动，需要改为不依赖 client `Finish`；server natural completion 与 cancel 继续通过现有 context/done 通道表达。

3. 调整 generated stream operation
   - 不再生成 `*MessageServerStreamFinish` / `*NativeServerStreamFinish` 这类 client-side server-stream finish operation。
   - bidi `Finish` 保留。
   - client streaming `Finish` 保留。

4. 调整 message C client renderer
   - server stream `Start` 增加 `onRecv/onDone` 参数与 callback typedef。
   - bidi stream `Start` 增加 `onRecv/onDone` 参数与 callback typedef。
   - server stream 不再生成 `Finish` export。
   - `Recv` export 检查 callback receive 标记。
   - `Cancel` export 保持有效，并触发后台流程结束。

5. 调整 native C client renderer
   - 按 native response ABI 生成 `onRecv` typedef 与 trampoline。
   - server stream / bidi stream `Start` 增加 callback 参数。
   - server stream 不再生成 `Finish` export。
   - `Recv` / `Cancel` 规则与 message projection 一致。

6. 增加 callback receive 状态
   - 在 stream session 中记录是否启用 callback receive。
   - 最小实现优先：generated code 包装 typed client endpoint，保存 `callbackReceive bool` 与 cancel/done 协调状态。
   - 后台 goroutine 串行调用 `Recv`、encode/lower response、`onRecv`、`onDone`。
   - 避免新增通用 runtime lifecycle state machine。

7. 更新高层生成器
   - Kotlin/Dart 如果直接绑定 C `Start` 签名，传入 nil callback 以保持手动 `Recv` 行为。
   - 已有 Kotlin `RecvEach` 可先保留；它是高层手动 `Recv` loop，不等同于 C callback receive。

8. 测试
   - generator tests 覆盖 `Start` 签名包含 `onRecv/onDone`。
   - generator tests 覆盖 server stream client 不生成 `Finish`。
   - generator tests 覆盖 C server callback 仍保留 server-stream `Finish`。
   - runtime/generator tests 覆盖 callback receive 启用后 `Recv` 返回显式错误。
   - integration test 覆盖 server stream callback receive 正常 EOF 调 `onDone(0)`。
   - integration test 覆盖 bidi callback receive 与 `Send/CloseSend` 共存。
   - integration test 覆盖 `Cancel` 后 `onDone(non-zero)`。

## 验收

- `rtk go test ./...`
- 涉及 ABI / runtime 类型变化后运行 unsigned 32/64 扫描：

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/release/verification-checklist.md'
```

发布级改动按 `docs/release/verification-checklist.md` 执行。

## 风险

- 删除 server-stream client `Finish` 会影响已有 generated examples / JNI / Dart / Kotlin bindings，需要同步生成输出。
- `Cancel` 与后台 `Recv` loop 的竞态必须保证 `onDone` 只调用一次。
- message/native 两套 response lowering 不能共享错误的 bytes-only callback ABI。
