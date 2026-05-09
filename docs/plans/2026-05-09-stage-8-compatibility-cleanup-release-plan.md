# Stage 8 兼容性清理与发布准备实施计划

> **给 agentic workers:** 按任务逐项执行本计划。步骤使用 checkbox (`- [ ]`) 语法跟踪。每个任务完成后先运行对应验证，再更新 checkbox；不要把多个提交点 squash 成一个大提交。

**Goal:** 把 Stage 0-7 已经可运行的新版架构推进到发布前可验收状态：补齐 request-side empty input、message bytes 错误语义、stream terminal lifecycle、内存释放、旧模型清理、文档一致性和发布验证命令集合。

**Architecture:** Stage 8 不改变新版核心模型。所有 cgo native/message client 调用仍进入 generated dispatcher；每个 generated service 仍只有一个 active server slot；native/message 转换仍在 generated service runtime；`rpcruntime` 只承载通用 wrapper、error store、stream/session primitive。Connect/gRPC client 仍只是 remote server adapter 内部细节，不进入 rpccgo client 类型模型。

**Tech Stack:** Go 1.24、cgo、protobuf/protogen、Connect、gRPC、`rpcruntime`、generated-source acceptance、`examples/minimal-greeter`、`examples/full-greeter`。

---

## 范围

Stage 8 实现：

- request-side empty input normalization：`ptr == 0 || len/count == 0` 统一视为 empty request input；负长度仍报错。
- request-side ownership normalization：request decode 中 `ownership > 0` 表示转移 ownership，`ownership == 0` 表示 borrowed。
- message bytes ABI 错误语义：cgo message client/server 的 unary 与三类 streaming 都先校验 protobuf bytes，把 invalid protobuf 转成 error id 或 Go error。
- stream terminal lifecycle hardening：`Finish`、`Done`、`CloseSend`、`Cancel`、EOF、重复 terminal operation、invalid handle 的行为在 native/message/local/remote 路径中一致。
- 内存释放与 error text 生命周期验收：owned request 释放一次，borrowed request 不释放，output pointer 可释放，error text 可消费且不会泄漏。
- 清理旧项目心智模型：旧 provider registry、多 provider bootstrap、framework selector、旧 forwarding client/server 术语不得回流。
- README、计划文档、迁移清单、发布前验证命令集合与实际行为对齐。

Stage 8 不实现：

- 新 transport、新 server kind、新 rpccgo client 类型。
- remote retry、负载均衡、服务发现、连接池或 gRPC `ClientConnInterface` 生命周期托管。
- map、oneof、unsigned 32/64 native ABI。
- repeated string、repeated bytes、repeated message native ABI；这些仍由 planner 明确拒绝。
- 旧项目的 provider registry、多 provider bootstrap、framework selector、debugserver、Flutter discovery example。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
| --- | --- | --- | --- |
| `rpccgo-old/rpcruntime/*_test.go` | 参考并补强 | wrapper、release、error store、cleanup、length 边界 | runtime primitive 已迁入新版；Stage 8 只补当前缺少的 release/error text 边界，不回滚 signed ABI |
| `rpccgo-old/internal/generator/message_export_shim_cgo.go` | 参考测试语义后重写 | message request bytes decode、`proto.Unmarshal`、error id | 旧实现证明 message ABI 应在边界校验 protobuf；新版 renderer 已重写，不能直接迁移代码 |
| `rpccgo-old/internal/generator/message_client.go` | 参考测试语义后重写 | message response bytes unmarshal 与 error propagation | 旧代码可提供 invalid bytes 用例；新版必须走 generated dispatcher 与 current file family |
| `rpccgo-old/internal/integration/message_mode` | 参考用例 | message unary/streaming bytes ABI 与 protobuf roundtrip | 适合迁移成 generated-source acceptance；不迁移旧 bootstrap |
| `rpccgo-old/internal/integration/native_mode` | 参考用例 | native request/response wrapper、owned/borrowed input、release | 适合补 native ABI 边界；需按新版 signed ABI 与 `cgo_dir/package main` 重写 |
| `rpccgo-old/internal/integration/native_forwarding` | 参考用例 | remote forwarding、cancel、onDone、marshal/unmarshal 错误 | 只迁移高价值 remote lifecycle 场景；不迁移旧 forwarding client/server 模型 |
| `rpccgo-old/internal/integration/both_mode` | 参考并改写 | native/message 生成物共同编译 | 用于确认新版 file family 可共存；不得恢复双 provider bootstrap |
| `rpccgo-old/docs/specs/2026-03-30-request-side-empty-input-normalization-design.md` | 参考合同 | request-side empty input 与 ownership 语义 | 合同与 Stage 8 roadmap 匹配；实现必须落到当前 generator/runtime，而不是旧路径 |
| 旧 `examples/connect`、`examples/grpc` generated/export 文件 | 不迁移 | 旧生成物与旧导出 API | 与新版 generated layout、single dispatcher、single active slot 冲突 |
| 旧 provider registry、framework selector、debugserver、Flutter discovery example | 不迁移 | 旧多入口/调试/业务示例 | 超出 Stage 8 发布准备范围，且部分概念与新版架构冲突 |

### Task 1 审计补充（2026-05-09）

- message bytes ABI 的边界校验证据主要来自 `rpccgo-old/internal/generator/message_export_shim_cgo.go` 与 `rpccgo-old/internal/generator/message_client.go`，两者都在 cgo 边界执行 `proto.Unmarshal`。
- streaming terminal / cancel 语义的高价值回归主要来自 `rpccgo-old/internal/integration/native_forwarding/integration_test.go` 与 `rpccgo-old/internal/integration/both_mode/integration_test.go`。
- request-side empty input 与 ownership 归一化合同以 `rpccgo-old/docs/specs/2026-03-30-request-side-empty-input-normalization-design.md` 为参考，但实现必须遵循当前 signed ABI 与 `EmptyRpc*` 合同。
- 旧模型禁入的核心证据包括 `rpccgo-old/internal/generator/go_client_registry.go`、`native_forwarding_client.go`、`native_forwarding_server.go` 以及 `examples/*/cmd/debugserver`，这些只可作为“明确不迁移”样本。

## 文件结构

- Create: `docs/plans/2026-05-09-stage-8-migration-inventory.md`
- Create: `docs/plans/2026-05-09-stage-8-release-checklist.md`
- Modify: `rpcruntime/rpc_type_test.go`
- Modify: `rpcruntime/rpc_repeat_test.go`
- Modify: `rpcruntime/errors_test.go`
- Modify: `rpcruntime/release_test.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_client_cgo_test.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_message_server_cgo_test.go`
- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`
- Modify: `internal/generator/generated_layout_contract_test.go`
- Create: `internal/integration/stage8_empty_input_normalization_test.go`
- Create: `internal/integration/stage8_message_bytes_hardening_test.go`
- Create: `internal/integration/stage8_stream_terminal_lifecycle_test.go`
- Create: `internal/integration/stage8_memory_release_hardening_test.go`
- Modify: `internal/integration/remote_transport_stage6_acceptance_test.go`
- Modify: `examples/minimal-greeter/example_test.go`
- Modify: `examples/full-greeter/example_test.go`
- Modify: `README.md`
- Modify: `docs/plans/2026-04-27-rpccgo-project-roadmap.md`

## Task 1：审计旧项目边界并写 Stage 8 迁移清单

**Files:**

- Create: `docs/plans/2026-05-09-stage-8-migration-inventory.md`
- Modify: `docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md`

**迁移内容与理由:** 旧项目的价值主要是边界用例，不是旧架构。先把 message/native/native_forwarding/both_mode 中值得迁移的测试语义写成清单，避免执行时把 provider registry、framework selector 或 debugserver 带回新版。

- [ ] **Step 1: 盘点旧项目高价值用例**

检查：

```bash
rtk rg -n "proto.Unmarshal|proto.Marshal|Release|ownership|Cancel|CloseSend|Finish|Done|empty|nil pointer|forwarding|both_mode" /home/zenghp/github.com/ygrpc/rpccgo-old/internal /home/zenghp/github.com/ygrpc/rpccgo-old/docs /home/zenghp/github.com/ygrpc/rpccgo-old/examples -g '*.go' -g '*.md'
```

记录四类用例：

- message bytes invalid protobuf 与 zero-length request/response。
- native wrapper owned/borrowed release 与 empty input。
- streaming terminal lifecycle 与 cancel/onDone。
- both-mode 共同编译但单 active server bootstrap。

- [ ] **Step 2: 写迁移清单**

创建 `docs/plans/2026-05-09-stage-8-migration-inventory.md`，包含：

- `迁移或参考` 表格，写明旧文件、当前处理、作用、迁移理由。
- `明确不迁移` 列表，包含 provider registry、framework selector、debugserver、旧 generated/export 文件、Flutter discovery example。
- `当前实现差距` 列表，映射到本计划 Task 2-7。
- `验证入口` 列表，映射到 focused tests、全仓测试、examples、unsigned scan。

- [ ] **Step 3: 回填计划文档审计结果**

在本计划 `旧项目迁移判定` 或后续风险处补充审计过程中发现的新增边界。只补真实长期边界；不要把临时本机环境问题写进项目文档。

- [ ] **Step 4: 验证**

Run:

```bash
rtk rg -n "T[B]D|T[O]DO" docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md docs/plans/2026-05-09-stage-8-migration-inventory.md
rtk git diff -- docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md docs/plans/2026-05-09-stage-8-migration-inventory.md
```

Expected:

- 占位词扫描无命中。
- diff 只包含 Stage 8 计划与迁移清单。

- [ ] **Step 5: 验收**

- 迁移清单能回答“迁移什么、有什么用、为什么迁移而不是重写、什么明确不迁移”。
- 每个后续任务都能从迁移清单找到对应边界来源。

- [ ] **Step 6: 提交**

```bash
rtk git add docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md docs/plans/2026-05-09-stage-8-migration-inventory.md
rtk git commit -m "docs: plan stage 8 compatibility cleanup"
```

## Task 2：统一 request-side empty input 与 ownership 合同

**Files:**

- Modify: `rpcruntime/rpc_type_test.go`
- Modify: `rpcruntime/rpc_repeat_test.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_client_cgo_test.go`
- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`
- Create: `internal/integration/stage8_empty_input_normalization_test.go`

**迁移内容与理由:** 旧 empty input normalization 设计把 request-side `ptr == 0 || len/count == 0` 固定为 empty input，并把 `ownership > 0` 固定为 ownership transfer。当前 runtime 已有 canonical empty wrapper；Stage 8 要把 generated request decode 统一到这条合同，并用 generated-source acceptance 证明 message/native 路径一致。

- [x] **Step 1: 写 focused renderer 红测**

在 generator tests 中断言：

- message request decode 中 `length == 0` 或 `ptr == 0` 返回 empty bytes，不再把 `ptr == 0 && length > 0` 当 pointer error。
- native string/bytes/message-bytes/repeated numeric/repeated enum/repeated bool request decode 在 empty path 使用 `rpcruntime.EmptyRpcString()`、`rpcruntime.EmptyRpcBytes()`、`rpcruntime.EmptyRpcRepeat[T]()`、`rpcruntime.EmptyRpcBoolRepeat()`。
- request-side non-empty path 使用 `Ownership > 0`。
- negative length/count 仍返回 error id 或 Go error。

- [x] **Step 2: 写 generated-source acceptance**

创建 `internal/integration/stage8_empty_input_normalization_test.go`，覆盖：

- cgo message unary request：`ptr=0, len>0` 进入默认 protobuf request，而不是 pointer error。
- cgo native unary request：string/bytes/repeated bool/repeated numeric 的 `ptr=0, len>0` 被视为 empty。
- non-empty request 且 `ownership=2` 会调用 free callback 一次。
- negative length/count 返回 error id，错误文本包含 `negative`。

- [x] **Step 3: Run failing tests**

Run:

```bash
rtk go test ./internal/generator -run 'TestRender(MessageClientCGO|NativeClientCGO|NativeServerCGO).*Empty|TestRender.*Ownership' -count=1
rtk go test ./internal/integration -run TestStage8EmptyInputNormalization -count=1
```

Expected: 在实现前失败，指出 request decode 尚未统一的路径。

- [x] **Step 4: 实现 message request empty normalization**

在 `render_message_client_cgo.go` 中调整 message request bytes helper：

- `length < 0` 保持错误。
- `ptr == 0 || length == 0` 返回 `nil, nil`，作为 empty protobuf request bytes。
- 非 empty path 复制 `unsafe.Slice`。

保留 response-side pointer required 语义；本任务只改 request-side。

- [x] **Step 5: 实现 native request empty normalization**

在 `render_native_client_cgo.go` 和 `render_native_server_cgo.go` 的 request decode 中：

- string/bytes/message-bytes empty path 使用 `rpcruntime.EmptyRpcString()` 或 `rpcruntime.EmptyRpcBytes()`。
- repeated numeric/enum empty path 使用 `rpcruntime.EmptyRpcRepeat[T]()`。
- repeated bool empty path 使用 `rpcruntime.EmptyRpcBoolRepeat()`。
- non-empty path 继续使用 checked constructor，并传入 `Ownership > 0`。
- negative length/count 先报错，不进入 empty path。

- [x] **Step 6: Run focused tests**

Run:

```bash
rtk go test ./rpcruntime -run 'TestEmptyRpc|TestNewRpc.*Checked' -count=1
rtk go test ./internal/generator -run 'TestRender(MessageClientCGO|NativeClientCGO|NativeServerCGO).*Empty|TestRender.*Ownership' -count=1
rtk go test ./internal/integration -run TestStage8EmptyInputNormalization -count=1
```

Expected: PASS。

- [x] **Step 7: 验收**

- request-side empty input 合同在 message/native 生成物中一致。
- response-side pointer required 语义没有被静默放宽。
- owned non-empty request 仍释放一次，borrowed request 不释放。

- [x] **Step 8: 提交**

```bash
rtk git add rpcruntime/rpc_type_test.go rpcruntime/rpc_repeat_test.go internal/generator/render_message_client_cgo.go internal/generator/render_message_client_cgo_test.go internal/generator/render_native_client_cgo.go internal/generator/render_native_client_cgo_test.go internal/generator/render_native_server_cgo.go internal/generator/render_native_server_cgo_test.go internal/integration/stage8_empty_input_normalization_test.go
rtk git commit -m "fix: normalize request-side empty ABI inputs"
```

## Task 3：锁住 message bytes ABI 的 protobuf 错误语义

**Files:**

- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_client_cgo_test.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_message_server_cgo_test.go`
- Create: `internal/integration/stage8_message_bytes_hardening_test.go`

**迁移内容与理由:** 旧 message export/message client 代码在 ABI 边界执行 `proto.Unmarshal`，把坏 protobuf bytes 转成错误。新版已在部分 message client path 加了校验；Stage 8 要把 unary、client streaming、server streaming、bidi streaming、cgo message server callback response 全部锁成 acceptance，防止某条 stream path 只传 raw bytes 而延迟炸在下游。

- [x] **Step 1: 写 renderer 断言**

在 message client/server renderer tests 中断言：

- 每个 request bytes 入口都包含 `protobuf.Unmarshal` 或等价 helper。
- 每个 response bytes 写出前都包含 `protobuf.Unmarshal` 或等价 helper。
- 错误文本包含 `message request protobuf unmarshal failed` 或 `message response protobuf unmarshal failed`。
- cgo message server callback 返回非零 `errID` 时通过 `rpcruntime.TakeErrorText` 转成 Go error。
- unknown error id 返回包含 id 的错误，不被吞掉。

- [x] **Step 2: 写 generated-source acceptance**

创建 `internal/integration/stage8_message_bytes_hardening_test.go`，覆盖：

- invalid unary request bytes 返回非零 error id，active server 未被调用。
- invalid client-streaming send bytes 返回非零 error id，stream handle 仍可 cancel。
- invalid server-streaming start request bytes 返回 start error id 且 handle 为 0。
- invalid bidi send bytes 返回非零 error id。
- cgo message server callback 返回 invalid response bytes 时，cgo message client read/finish/unary 得到 response unmarshal error。
- callback unknown error id 返回明确错误文本。

- [x] **Step 3: Run failing tests**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessage(Client|Server)CGO.*Protobuf|TestRenderMessage(Client|Server)CGO.*ErrorID' -count=1
rtk go test ./internal/integration -run TestStage8MessageBytesHardening -count=1
```

Expected: 若当前某条路径未校验，测试失败并指出方法类型。

- [x] **Step 4: 实现缺失校验**

补齐 `render_message_client_cgo.go` 与 `render_message_server_cgo.go` 中缺失的 protobuf request/response 校验。保持 direct-path bytes ABI：校验通过后仍把 raw protobuf bytes 传给 dispatcher 或 callback，不引入新的 message wrapper 模型。

- [x] **Step 5: Run focused tests**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessage(Client|Server)CGO' -count=1
rtk go test ./internal/integration -run TestStage8MessageBytesHardening -count=1
```

Expected: PASS。

- [x] **Step 6: 验收**

- invalid protobuf bytes 在 cgo boundary 变成 error id 或 Go error。
- valid zero-length protobuf request 仍能表示默认 protobuf message。
- direct-path message bytes ABI 没有被替换成 service-specific runtime 类型。

- [x] **Step 7: 提交**

```bash
rtk git add internal/generator/render_message_client_cgo.go internal/generator/render_message_client_cgo_test.go internal/generator/render_message_server_cgo.go internal/generator/render_message_server_cgo_test.go internal/integration/stage8_message_bytes_hardening_test.go
rtk git commit -m "fix: harden message protobuf ABI errors"
```

## Task 4：补齐 stream terminal lifecycle 矩阵

**Files:**

- Create: `internal/integration/stage8_stream_terminal_lifecycle_test.go`
- Modify: `internal/integration/remote_transport_stage6_acceptance_test.go`
- Modify: `internal/generator/render_connect_remote.go`
- Modify: `internal/generator/render_grpc_remote.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_native_client_cgo.go`

**迁移内容与理由:** Stage 2-7 已覆盖各类 streaming happy path 和部分 cancel 行为。Stage 8 要补发布前最容易漏的 terminal lifecycle：重复 terminal operation、terminal 后继续 send/read、invalid handle、EOF 后 done、registration override 后 in-flight stream snapshot 不漂移。

- [x] **Step 1: 写 terminal lifecycle acceptance**

创建 `internal/integration/stage8_stream_terminal_lifecycle_test.go`，覆盖 native client、message client、Connect remote、gRPC remote 的共同场景：

- client streaming：`Finish` 后再次 `Finish` 或 `Send` 返回 error id。
- server streaming：EOF 后 `Done` 成功，`Done` 后再次 `Read` 返回 error id。
- bidi streaming：`CloseSend` 后再次 `Send` 返回 error id，`Cancel` 后 `Read` 返回 error id。
- `Cancel` 后重复 `Cancel` 返回 error id，且下游 cancel callback 只调用一次。
- invalid handle 对 `Send`、`Read`、`Finish`、`Done`、`CloseSend`、`Cancel` 都返回明确错误。
- stream start 捕获 active server snapshot；注册新 active server 后，旧 stream 后续操作仍走旧 adapter。

- [x] **Step 2: Run failing tests**

Run:

```bash
rtk go test ./internal/integration -run TestStage8StreamTerminalLifecycle -count=1
```

Expected: 暴露缺失的 terminal guard 或 handle finalization 行为。

- [x] **Step 3: 修复 lifecycle 缺口**

按失败路径做最小修复：

- 优先在 generated stream wrapper 中使用已有 `rpcruntime` lifecycle primitive。
- 只在通用 lifecycle 语义缺失时修改 `rpcruntime`。
- 不让 generated service 维护平行 session registry。
- remote stream `Cancel` 只取消 stream context 和关闭 send side，不关闭调用方持有的 gRPC conn。

- [x] **Step 4: Run focused stream suites**

Run:

```bash
rtk go test ./rpcruntime -run 'TestStreamSession|TestDispatcherStream|TestStreamRegistry' -count=1
rtk go test ./internal/integration -run 'TestStage8StreamTerminalLifecycle|TestRemoteTransportStage6Acceptance|TestNative(Client|Server|Bidi)Streaming|TestMessageDirectPath' -count=1
```

Expected: PASS。

- [x] **Step 5: 验收**

- terminal operation 只有一次成功。
- terminal 后继续操作返回 error id，不 panic，不泄漏 handle。
- in-flight stream snapshot 不受后续 active server registration 影响。

- [x] **Step 6: 提交**

```bash
rtk git add internal/integration/stage8_stream_terminal_lifecycle_test.go internal/integration/remote_transport_stage6_acceptance_test.go internal/generator/render_connect_remote.go internal/generator/render_grpc_remote.go internal/generator/render_message_client_cgo.go internal/generator/render_native_client_cgo.go
rtk git commit -m "test: harden stream terminal lifecycle"
```

## Task 5：补齐内存释放与 error text 生命周期验收

**Files:**

- Modify: `rpcruntime/errors_test.go`
- Modify: `rpcruntime/release_test.go`
- Modify: `rpcruntime/rpc_type_test.go`
- Modify: `rpcruntime/rpc_repeat_test.go`
- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_message_client_cgo.go`
- Modify: `internal/generator/render_message_server_cgo.go`
- Create: `internal/integration/stage8_memory_release_hardening_test.go`

**迁移内容与理由:** runtime wrapper 已有大量 release 测试，Stage 7 也补了 repeated release。Stage 8 要补 generated ABI 层的组合验收：owned/borrowed request、output pointer release、callback error text、unknown error id、失败路径 cleanup。

- [x] **Step 1: 写 memory release acceptance**

创建 `internal/integration/stage8_memory_release_hardening_test.go`，覆盖：

- native string/bytes/repeated request：borrowed input 不调用 free，owned input 调用 free 一次。
- decode 失败时，已经接管 ownership 的 wrapper 会释放，未接管的 borrowed pointer 不释放。
- output pointer 由 `rpcruntime.Release` 释放一次，重复释放返回 false 或 no-op 合同中的失败值。
- cgo message server response pointer 在 Go 侧复制后不被 Go 自动释放；所有权仍由 C callback 合同定义。
- error text 通过 `TakeErrorText` 消费一次，导出的 pointer 可 release。

- [x] **Step 2: 补 runtime 单测缺口**

如果 acceptance 暴露 runtime 层缺口，在 `rpcruntime/*_test.go` 中先补 failing tests，再做最小实现。重点覆盖：

- `TakeErrorText` pin 失败时记录可重试。
- release failure 不吞错误。
- empty singleton release no-op。
- owned zero-length pointer 的释放合同保持现有行为。

- [x] **Step 3: Run failing tests**

Run:

```bash
rtk go test ./rpcruntime -run 'Test.*Release|Test.*ErrorText|TestEmptyRpc' -count=1
rtk go test ./internal/integration -run TestStage8MemoryReleaseHardening -count=1
```

Expected: 实现前可能暴露 generated cleanup 缺口。

- [x] **Step 4: 修复 generated cleanup 缺口**

只修复 acceptance 指出的路径：

- decode 成功后释放 owned request wrapper。
- decode 失败时释放已经构造成功且需要 release 的 wrapper。
- response output pin 失败时不留下半写 output。
- error id helper 使用 `TakeErrorText` 后释放 error text pointer。

- [x] **Step 5: Run focused tests**

Run:

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./internal/integration -run TestStage8MemoryReleaseHardening -count=1
```

Expected: PASS。

- [x] **Step 6: 验收**

- owned request 内存释放一次。
- borrowed request 不释放。
- output/error text 生命周期有明确测试。
- 失败路径不会产生明显 handle 或 pinned pointer 泄漏。

- [x] **Step 7: 提交**

```bash
rtk git add rpcruntime/errors_test.go rpcruntime/release_test.go rpcruntime/rpc_type_test.go rpcruntime/rpc_repeat_test.go internal/generator/render_native_client_cgo.go internal/generator/render_native_server_cgo.go internal/generator/render_message_client_cgo.go internal/generator/render_message_server_cgo.go internal/integration/stage8_memory_release_hardening_test.go
rtk git commit -m "test: verify ABI memory release semantics"
```

## Task 6：清理旧模型残留并锁住 generated layout

**Files:**

- Modify: `internal/generator/generated_layout_contract_test.go`
- Modify: `README.md`
- Modify: `docs/plans/2026-04-27-rpccgo-project-roadmap.md`
- Modify: `docs/plans/2026-05-09-stage-8-migration-inventory.md`

**迁移内容与理由:** 新版明确不保留 provider registry、多 provider bootstrap、framework selector、旧 forwarding client/server 心智模型。Stage 8 要用测试和文档扫描锁住这些旧概念不回流，同时让 README 的可用性描述保持简洁。

- [ ] **Step 1: 扩展 generated layout 禁用词测试**

在 `generated_layout_contract_test.go` 中确认生成物不包含旧模型 token：

- `provider registry`
- `framework selector`
- `bootstrap`
- `goclient.export`
- `goserver.export`
- `native_forwarding_client`
- `native_forwarding_server`

允许文档中作为“明确不迁移”出现这些词；生成物和用户 example 不允许。

- [ ] **Step 2: 文档扫描**

Run:

```bash
rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!AGENTS.md'
```

Expected: 只允许测试断言或历史说明中的受控命中；真实生成物、README、examples 不出现旧模型作为推荐路径。

- [ ] **Step 3: 更新 README 与 roadmap**

README 只补发布前验证入口和 examples 入口，不展开 spec 细节。Roadmap Stage 8 更新为实际收口项：

- empty input normalization。
- message bytes hardening。
- stream terminal lifecycle。
- memory release。
- release checklist。

- [ ] **Step 4: Run focused tests**

Run:

```bash
rtk go test ./internal/generator -run TestStage7GeneratedLayout -count=1
rtk go test ./internal/generator -run TestStage7GeneratedLayoutRejectsOldBootstrapNames -count=1
```

Expected: PASS。

- [ ] **Step 5: 验收**

- 生成物和 examples 不回流旧 provider/bootstrap/forwarding 模型。
- README 保持极简心智模型，不复制 spec 大段概念。
- Roadmap 与 Stage 8 实际行为一致。

- [ ] **Step 6: 提交**

```bash
rtk git add internal/generator/generated_layout_contract_test.go README.md docs/plans/2026-04-27-rpccgo-project-roadmap.md docs/plans/2026-05-09-stage-8-migration-inventory.md
rtk git commit -m "docs: align release-ready architecture wording"
```

## Task 7：建立发布前验证命令集合

**Files:**

- Create: `docs/plans/2026-05-09-stage-8-release-checklist.md`
- Modify: `README.md`
- Modify: `docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md`

**迁移内容与理由:** Stage 8 的验收不能只依赖一次本机记录。需要固定一组发布前命令，覆盖 root module、examples 子模块、生成路径、unsigned scan、旧模型扫描和 git 状态。

- [ ] **Step 1: 写 release checklist**

创建 `docs/plans/2026-05-09-stage-8-release-checklist.md`，包含以下命令和期望：

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./internal/generator -count=1
rtk go test ./internal/integration -count=1
rtk go test ./... -count=1
cd examples/minimal-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../full-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../..
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/**'
rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!AGENTS.md'
rtk git status --short
```

说明：

- unsigned scan exit code 1 且无输出表示 PASS。
- 旧模型扫描允许测试断言命中时，checklist 必须列明允许文件。
- examples 是独立 Go module，root `go test ./...` 不覆盖它们。

- [ ] **Step 2: README 添加最小验证入口**

README 只增加一个短小的“发布前验证”段落，指向 `docs/plans/2026-05-09-stage-8-release-checklist.md`，不要把 checklist 全文复制到 README。

- [ ] **Step 3: Run checklist locally**

按 checklist 顺序执行所有命令，并把结果写回本计划 `验证结果`。

- [ ] **Step 4: 验证**

Run:

```bash
rtk rg -n "T[B]D|T[O]DO" docs/plans/2026-05-09-stage-8-release-checklist.md README.md
```

Expected: 无命中。

- [ ] **Step 5: 验收**

- checklist 可独立指导发布前验证。
- README 保持简洁。
- 本计划记录实际验证结果。

- [ ] **Step 6: 提交**

```bash
rtk git add docs/plans/2026-05-09-stage-8-release-checklist.md README.md docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md
rtk git commit -m "docs: add stage 8 release checklist"
```

## Task 8：总验证、文档收口与提交

**Files:**

- Modify: `docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md`
- Modify: `docs/plans/2026-05-09-stage-8-migration-inventory.md`
- Modify: `docs/plans/2026-05-09-stage-8-release-checklist.md`

**迁移内容与理由:** 最后用 canonical plan、实际代码、fresh tests 三方交叉验证 Stage 8，避免只看 checkbox。把实际结果回填到计划和清单，形成可审计交付记录。

- [ ] **Step 1: Run full verification**

Run:

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./internal/generator -count=1
rtk go test ./internal/integration -count=1
rtk go test ./... -count=1
cd examples/minimal-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../full-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../..
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/**'
rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!AGENTS.md'
rtk git status --short
```

Expected:

- 所有 `go test`、example tests、`mage run` PASS。
- unsigned scan exit code 1 且无输出。
- 旧模型扫描没有非预期命中。
- `rtk git status --short` 只显示本计划相关文件，忽略未跟踪 `.vscode/`。

- [ ] **Step 2: Record verification results**

在本计划新增或更新 `验证结果`，逐条记录命令和 PASS/失败原因。若失败来自本机临时环境，只记录在执行结果，不把 workaround 写进长期文档。

- [ ] **Step 3: 更新完成标准**

只在对应实现与 fresh 验证都完成后，把 `完成标准` checkbox 标为 `[x]`。

- [ ] **Step 4: 最终文档扫描**

Run:

```bash
rtk rg -n "T[B]D|T[O]DO" docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md docs/plans/2026-05-09-stage-8-migration-inventory.md docs/plans/2026-05-09-stage-8-release-checklist.md README.md
```

Expected: 无命中。

- [ ] **Step 5: 提交**

```bash
rtk git add docs/plans/2026-05-09-stage-8-compatibility-cleanup-release-plan.md docs/plans/2026-05-09-stage-8-migration-inventory.md docs/plans/2026-05-09-stage-8-release-checklist.md
rtk git commit -m "docs: record stage 8 verification"
```

## 完成标准

- [ ] request-side empty input normalization 在 message/native generated-source acceptance 中通过。
- [ ] request-side `ownership > 0` 合同在 non-empty request 中通过 release 验证。
- [ ] invalid protobuf message bytes 在 unary 与三类 streaming 中都返回 error id 或 Go error。
- [ ] cgo message server callback response bytes 的 invalid protobuf 场景被验收。
- [ ] stream terminal lifecycle 对重复 terminal operation、terminal 后继续操作、invalid handle、EOF/Done、Cancel 都有验收。
- [ ] native/message/local/remote 路径的 in-flight stream snapshot 不受后续 active server registration 影响。
- [ ] owned/borrowed request、output pointer、error text 生命周期有测试覆盖。
- [ ] 生成物和 examples 不包含旧 provider registry、多 provider bootstrap、framework selector 或旧 forwarding client/server 模型。
- [ ] README、roadmap、迁移清单、release checklist 与实际行为一致。
- [ ] root module、internal focused tests、两个 examples、两个 `mage run`、unsigned scan、旧模型扫描全部通过。

## 提交边界

计划执行时按以下边界提交，不要 squash 成一个大 commit：

1. `docs: plan stage 8 compatibility cleanup`
2. `fix: normalize request-side empty ABI inputs`
3. `fix: harden message protobuf ABI errors`
4. `test: harden stream terminal lifecycle`
5. `test: verify ABI memory release semantics`
6. `docs: align release-ready architecture wording`
7. `docs: add stage 8 release checklist`
8. `docs: record stage 8 verification`

## 后续风险

- repeated string/bytes/message native ABI 仍不支持；未来若要支持，需要新的 variable-width ABI 设计，而不是扩展 fixed-width `RpcRepeat[T]`。
- map、oneof、unsigned 32/64 native ABI 仍不支持；这是当前 signed ABI 与生成器边界。
- remote retry、负载均衡、服务发现、连接池不属于 rpccgo Stage 8；应交给标准 Connect/gRPC client 或后续独立计划。
- CI 若没有 `protoc`、`protoc-gen-go` 或 cgo toolchain，example generation 会失败；Stage 8 checklist 只固定验证命令，不安装 CI 环境。
- 旧 Flutter discovery example 是业务示例，不进入新版 rpccgo 发布准备范围。

## 验证结果

- 待执行 Stage 8 时记录。
