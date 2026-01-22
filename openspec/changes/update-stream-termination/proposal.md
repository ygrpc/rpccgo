# Change: Update stream termination to avoid send/close panic

## Why
当前 runtime 在并发场景下对 sendCh 的 close 与发送可能触发 panic，影响稳定性与可预测性。

## What Changes
- 将流的结束信号从 close(sendCh) 调整为显式的 done/ctx 结束信号。
- 统一 runtime 与 adaptor 的流结束处理，确保接收侧仍以 io.EOF 语义结束。
- 为并发 Send/Finish 场景提供确定性错误返回，不出现 panic。

## Impact
- Affected specs: rpc-cgo-adaptor
- Affected code: rpcruntime/stream_handle.go 与所有流相关 adaptor 代码
