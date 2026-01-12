## 1. Specs
- [ ] 1.1 更新 `rpc-cgo-adaptor` spec delta：支持多 protocol 参数 + 通用 adaptor 的选择语义
- [ ] 1.2 更新 `rpc-dispatch` spec delta：新增/统一 protocol 选择的 context key 与 helper API
- [ ] 1.3 运行 `openspec validate add-context-protocol-selection --strict`

## 2. Implementation（批准后执行）
- [ ] 2.1 扩展插件参数解析：`protocol` 支持逗号分隔列表，默认 `connectrpc`
- [ ] 2.2 生成按 protocol 分文件的 adaptor：文件名使用 protocol 后缀
- [ ] 2.3 生成通用 adaptor `*_cgo_adaptor.go`：根据 `ctx` 选择 protocol，未指定则按列表顺序 fallback
- [ ] 2.4 在 `rpcruntime` 增加 protocol 选择相关 context key + helper functions
- [ ] 2.5 更新/补充 `test/` 下的生成物与测试：覆盖 grpc/connectrpc 以及混合注册场景
- [ ] 2.6 运行 `go test ./...`
