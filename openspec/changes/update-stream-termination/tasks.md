## 1. Implementation
- [ ] 调整 rpcruntime 的流结束机制，移除 sendCh close，改为 done/ctx 结束信号
- [ ] 更新所有 adaptor 的流接收逻辑，结束信号触发时返回 io.EOF
- [ ] 补充/更新并发 Send/Finish 的单测与回归测试

## 2. Documentation
- [ ] 更新实现相关注释与说明，明确流结束语义
