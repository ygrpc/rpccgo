# Test Suite Consolidation and E2E Boundaries Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 收敛迁移阶段测试，把仍有长期价值的断言迁移到通用测试，并补足真实端到端边界测试。

**Architecture:** 测试体系从 `stageN` 迁移证明转为当前架构合同证明。generator 层保留 parser/planner/renderer 的稳定合同断言；integration 层优先验证真实生成 module、cgo bridge、transport、stream lifecycle、release/error text 行为。删除阶段命名文件前必须确保等价断言已在通用测试中存在。

**Tech Stack:** Go 1.24、testing、cgo、protobuf/protogen、Connect、gRPC、rpcruntime、internal/generator、internal/integration。

---

## 文件结构

- Modify: `internal/generator/service_options_test.go`
  - 承载 `@rpccgo` 注释解析、默认 token、native-only expansion、未知 token 报错等长期 parser 合同。
- Modify: `internal/generator/descriptor_plan_test.go`
  - 承载多 service descriptor planning、ServicePlan/MethodPlan 元数据完整性合同。
- Modify: `internal/generator/streaming_plan_test.go`
  - 已承载 lifecycle matrix；只补缺失断言，不复制 stage fixture。
- Modify: `internal/generator/render_message_client_cgo_test.go`
  - 承载 message cgo client request/response protobuf 校验、streaming entrypoint、invalid handle/error id 生成合同。
- Modify: `internal/generator/render_message_server_cgo_test.go`
  - 承载 message cgo server callback response decode、`TakeErrorText`、unknown error id 生成合同。
- Modify: `internal/generator/render_native_client_cgo_test.go`
  - 承载 native request empty input、ownership、release 生成合同。
- Modify: `internal/generator/generated_layout_contract_test.go`
  - 去除 `Stage7` 命名，保留 layout/public API/旧模型禁入合同。
- Modify: `internal/integration/message_direct_path_test.go`
  - 增加真实 message bytes E2E、stream terminal E2E、message error text E2E。
- Modify: `internal/integration/native_unary_test.go`
  - 增加真实 empty input、ownership/release、output Release E2E。
- Modify or Rename: `internal/integration/local_transport_stage5_acceptance_test.go`
  - 迁移为通用 local transport 测试，不能保留 stage 命名。
- Modify or Rename: `internal/integration/remote_transport_stage6_acceptance_test.go`
  - 迁移为通用 remote transport 测试，不能保留 stage 命名。
- Modify or Rename: `internal/integration/repeated_native_abi_stage7_hardening_test.go`
  - 迁移为通用 repeated native ABI 测试，不能保留 stage 命名。
- Delete after migration:
  - `internal/generator/stage1_acceptance_test.go`
  - `internal/integration/native_stage3_acceptance_test.go`
  - `internal/integration/message_stage4a_acceptance_test.go`
  - `internal/integration/converter_stage4b_acceptance_test.go`
  - `internal/integration/stage8_empty_input_normalization_test.go`
  - `internal/integration/stage8_message_bytes_hardening_test.go`
  - `internal/integration/stage8_stream_terminal_lifecycle_test.go`
  - `internal/integration/stage8_memory_release_hardening_test.go`

## Chunk 1: 迁移 generator 阶段断言

### Task 1: 迁移 Stage 1 parser/planner 合同

**Files:**
- Modify: `internal/generator/service_options_test.go`
- Modify: `internal/generator/descriptor_plan_test.go`
- Modify: `internal/generator/streaming_plan_test.go`
- Delete: `internal/generator/stage1_acceptance_test.go`

- [ ] **Step 1: 确认现有 parser 测试覆盖 Stage 1 token 合同**

检查 `internal/generator/service_options_test.go` 是否已覆盖：

```go
// absent annotation -> msg-connect
// msg-connect
// msg-grpc
// msg-connect|msg-grpc
// msg-connect|native
// msg-connect|msg-grpc|native
// native -> msg-connect|native
// unknown token includes token text
// spelling typo msg-conenct returns error
```

Expected: 大部分已存在；若缺少 `unknown @rpccgo token "bogus"` 精确断言则补上。

- [ ] **Step 2: 补 parser 错误断言**

在 `TestParseServiceRPCCGOOptionsErrors` 中补：

```go
{
    name:        "unknown token keeps bad token in error",
    comments:    "@rpccgo:msg-connect|bogus",
    wantMessage: `unknown @rpccgo token "bogus"`,
},
{
    name:        "spelling error keeps bad token in error",
    comments:    "@rpccgo:msg-conenct",
    wantMessage: `unknown @rpccgo token "msg-conenct"`,
},
```

- [ ] **Step 3: 运行 parser 测试**

Run:

```bash
rtk go test ./internal/generator -run 'TestParseServiceRPCCGOOptions|TestAdapterSelectionHas' -count=1
```

Expected: PASS。

- [ ] **Step 4: 迁移完整 ServicePlan 断言**

在 `internal/generator/descriptor_plan_test.go` 增加非 stage 命名测试，例如：

```go
func TestBuildDescriptorPlanBuildsCompleteServicePlans(t *testing.T) {
    plugin := newTestPlugin(t, "paths=source_relative", completeServicePlanTestFile())

    plan, err := BuildDescriptorPlan(plugin.Files[0])
    if err != nil {
        t.Fatalf("BuildDescriptorPlan() error = %v", err)
    }
    if len(plan.Services) != 7 {
        t.Fatalf("Services = %d, want 7", len(plan.Services))
    }

    services := servicesByName(t, plan.Services,
        "DefaultService", "ConnectService", "GrpcService", "MessageService",
        "ConnectNativeService", "AllService", "NativeOnlyService",
    )
    assertAdapterTokens(t, services["DefaultService"].Adapters, []AdapterToken{AdapterTokenMessageConnect})
    assertAdapterTokens(t, services["ConnectService"].Adapters, []AdapterToken{AdapterTokenMessageConnect})
    assertAdapterTokens(t, services["GrpcService"].Adapters, []AdapterToken{AdapterTokenMessageGRPC})
    assertAdapterTokens(t, services["MessageService"].Adapters, []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC})
    assertAdapterTokens(t, services["ConnectNativeService"].Adapters, []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative})
    assertAdapterTokens(t, services["AllService"].Adapters, []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC, AdapterTokenNative})
    assertAdapterTokens(t, services["NativeOnlyService"].Adapters, []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative})

    for name, service := range services {
        if !service.NeedsCodec {
            t.Fatalf("%s NeedsCodec = false, want true", name)
        }
        for _, method := range service.Methods {
            if !method.NeedsCodec {
                t.Fatalf("%s.%s NeedsCodec = false, want true", name, method.Name)
            }
            assertCompleteMethodContracts(t, method)
        }
    }
}
```

实现 `completeServicePlanTestFile` 时可从 `stage1AcceptanceFile` 复制 fixture，但重命名为通用名称，避免 stage 命名。

- [ ] **Step 5: 迁移 method contract helper**

在 `descriptor_plan_test.go` 或测试 helper 文件中添加：

```go
func assertCompleteMethodContracts(t *testing.T, method MethodPlan) {
    t.Helper()
    if method.Request.GoName == "" || method.Response.GoName == "" {
        t.Fatalf("%s request/response descriptor metadata is missing", method.FullName)
    }
    if method.MessageContract.RequestType != method.Request || method.MessageContract.ResponseType != method.Response {
        t.Fatalf("%s MessageContract = %#v, want request/response IO metadata", method.FullName, method.MessageContract)
    }
    if len(method.NativeContract.RequestFields) == 0 || len(method.NativeContract.ResponseFields) == 0 {
        t.Fatalf("%s NativeContract missing request or response fields", method.FullName)
    }
    if len(method.RequestBody) != len(method.NativeContract.RequestFields) || len(method.ResponseBody) != len(method.NativeContract.ResponseFields) {
        t.Fatalf("%s request/response bodies do not match native contract fields", method.FullName)
    }
}
```

- [ ] **Step 6: 运行 descriptor/streaming 测试**

Run:

```bash
rtk go test ./internal/generator -run 'TestBuildDescriptorPlan|TestBuildStreamingPlan|TestValidateStreamingLifecyclePlan' -count=1
```

Expected: PASS。

- [ ] **Step 7: 删除 Stage 1 文件**

Delete:

```bash
rm internal/generator/stage1_acceptance_test.go
```

注意：如果使用 Claude Code 工具执行，使用 `rm` 删除文件前确认文件已迁移；不要用脚本重写文件。

- [ ] **Step 8: 确认 generator 无 Stage 1 引用**

Run:

```bash
rtk rg -n 'Stage1|stage1|stage 1|Stage 1' internal/generator -g '*_test.go'
rtk go test ./internal/generator -count=1
```

Expected: `rg` 无输出；generator tests PASS。

- [ ] **Step 9: 提交**

```bash
rtk git add internal/generator/service_options_test.go internal/generator/descriptor_plan_test.go internal/generator/streaming_plan_test.go
rtk git rm internal/generator/stage1_acceptance_test.go
rtk git commit -m "test: consolidate generator planning contracts"
```

## Chunk 2: 迁移 integration 阶段测试命名

### Task 2: 删除空包装和重命名长期 E2E 测试

**Files:**
- Delete: `internal/integration/native_stage3_acceptance_test.go`
- Delete: `internal/integration/message_stage4a_acceptance_test.go`
- Delete: `internal/integration/converter_stage4b_acceptance_test.go`
- Rename: `internal/integration/local_transport_stage5_acceptance_test.go` -> `internal/integration/local_transport_test.go`
- Rename: `internal/integration/remote_transport_stage6_acceptance_test.go` -> `internal/integration/remote_transport_test.go`
- Rename: `internal/integration/repeated_native_abi_stage7_hardening_test.go` -> `internal/integration/repeated_native_abi_test.go`

- [ ] **Step 1: 读取三个待删除 acceptance 文件**

Read:

```text
internal/integration/native_stage3_acceptance_test.go
internal/integration/message_stage4a_acceptance_test.go
internal/integration/converter_stage4b_acceptance_test.go
```

确认其断言是否已被以下文件覆盖：

```text
internal/integration/native_unary_test.go
internal/integration/native_client_streaming_test.go
internal/integration/native_server_streaming_test.go
internal/integration/native_bidi_streaming_test.go
internal/integration/message_direct_path_test.go
internal/integration/converter_lifecycle_test.go
internal/integration/converter_snapshot_test.go
```

- [ ] **Step 2: 若有独特断言，先迁移**

只迁移长期行为断言，不迁移 stage 文件名、stage proto 名或阶段描述。示例：如果 message acceptance 里独有 `RegisterGreeterCGOMessageServer` 断言，应迁移到 `message_direct_path_test.go` 或 renderer test。

- [ ] **Step 3: 删除三个 acceptance 文件**

```bash
rtk git rm internal/integration/native_stage3_acceptance_test.go
rtk git rm internal/integration/message_stage4a_acceptance_test.go
rtk git rm internal/integration/converter_stage4b_acceptance_test.go
```

- [ ] **Step 4: 重命名长期 E2E 文件**

```bash
rtk git mv internal/integration/local_transport_stage5_acceptance_test.go internal/integration/local_transport_test.go
rtk git mv internal/integration/remote_transport_stage6_acceptance_test.go internal/integration/remote_transport_test.go
rtk git mv internal/integration/repeated_native_abi_stage7_hardening_test.go internal/integration/repeated_native_abi_test.go
```

- [ ] **Step 5: 去除函数和 helper 中的 Stage 命名**

在重命名后的文件中做机械重命名：

```text
TestStage6RemoteTransportAcceptance -> TestRemoteTransportAcceptance
newRemoteTransportStage6TestPlugin -> newRemoteTransportTestPlugin
writeStage6RemoteGeneratedModule -> writeRemoteTransportGeneratedModule
stage6RemoteServerMainSource -> remoteTransportServerMainSource
TestStage7... -> TestRepeatedNativeABI...
```

不要改变测试语义。

- [ ] **Step 6: 运行 integration focused tests**

Run:

```bash
rtk go test ./internal/integration -run 'TestNative|TestMessage|TestConverter|TestLocalTransport|TestRemoteTransport|TestRepeatedNativeABI' -count=1
```

Expected: PASS。

- [ ] **Step 7: 扫描 integration 阶段命名**

Run:

```bash
rtk rg -n 'Stage[0-9]|stage[0-9]|stage [0-9]|Stage 8|Stage8' internal/integration -g '*_test.go'
```

Expected: 只允许尚未迁移的四个 stage8 文件命中。

- [ ] **Step 8: 提交**

```bash
rtk git add internal/integration
rtk git commit -m "test: remove migration-stage integration wrappers"
```

## Chunk 3: 补真实 message bytes 和 stream terminal E2E

### Task 3: 补 message bytes E2E

**Files:**
- Modify: `internal/integration/message_direct_path_test.go`
- Modify: `internal/generator/render_message_client_cgo_test.go`
- Modify: `internal/generator/render_message_server_cgo_test.go`
- Delete after migration: `internal/integration/stage8_message_bytes_hardening_test.go`

- [ ] **Step 1: 写真实 invalid protobuf request 测试**

在 `message_direct_path_test.go` 的临时 module fixture 中新增测试源码，覆盖：

```go
func TestMessageBytesRejectInvalidUnaryRequest(t *testing.T) {
    resetGreeterMessageIntegrationState()
    if err := registerGreeterMessageCallbacksForIntegration(); err != nil {
        t.Fatalf("register callbacks: %v", err)
    }

    output := &GreeterMessageOutput{}
    errID := CallGreeterUnaryMessageUnary(context.Background(), uintptr(unsafe.Pointer(&[]byte{0xff}[0])), 1, output)
    if errID == 0 {
        t.Fatal("CallGreeterUnaryMessageUnary() error id = 0, want invalid protobuf error")
    }
    assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
}
```

实际函数名以 fixture 生成物为准；优先复用现有 helper，避免新建 fixture 框架。

- [ ] **Step 2: 补 streaming invalid bytes 测试**

同一 fixture 中新增：

```go
func TestMessageBytesRejectInvalidClientStreamSend(t *testing.T) { ... }
func TestMessageBytesRejectInvalidServerStreamStart(t *testing.T) { ... }
func TestMessageBytesRejectInvalidBidiSend(t *testing.T) { ... }
```

断言：

```go
errID != 0
assertMessageErrContains(t, errID, "message request protobuf unmarshal failed")
```

若 stream 已创建，测试末尾调用 `Cancel...` 清理 handle。

- [ ] **Step 3: 补 invalid callback response bytes 测试**

在 fixture callback 源中增加开关，例如：

```go
var greeterMessageInvalidResponse bool

func setGreeterMessageInvalidResponseForIntegration(v bool) {
    greeterMessageInvalidResponse = v
}
```

当开关为 true，callback 返回非空但非法 protobuf bytes。测试断言 client 侧得到：

```text
message response protobuf unmarshal failed
```

- [ ] **Step 4: 运行 message direct path 测试，确认失败或通过**

Run:

```bash
rtk go test ./internal/integration -run 'TestMessageDirectPath|TestMessageBytes' -count=1
```

Expected: 如果当前实现已有行为，直接 PASS；否则 FAIL 指向缺失路径。

- [ ] **Step 5: 若失败，最小修复 renderer**

只修改：

```text
internal/generator/render_message_client_cgo.go
internal/generator/render_message_server_cgo.go
```

不要改 runtime 模型；不要引入新 wrapper。

- [ ] **Step 6: 迁移 Stage 8 message 片段断言到 renderer tests**

把长期生成合同迁移到：

```text
internal/generator/render_message_client_cgo_test.go
internal/generator/render_message_server_cgo_test.go
```

保留断言：

```text
rpccgo: message request protobuf unmarshal failed
rpccgo: message response protobuf unmarshal failed
rpcruntime.TakeErrorText
unknown error id
```

- [ ] **Step 7: 删除 stage8 message 文件**

```bash
rtk git rm internal/integration/stage8_message_bytes_hardening_test.go
```

- [ ] **Step 8: 验证**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessage(Client|Server)CGO' -count=1
rtk go test ./internal/integration -run 'TestMessageDirectPath|TestMessageBytes' -count=1
```

Expected: PASS。

- [ ] **Step 9: 提交**

```bash
rtk git add internal/generator/render_message_client_cgo_test.go internal/generator/render_message_server_cgo_test.go internal/generator/render_message_client_cgo.go internal/generator/render_message_server_cgo.go internal/integration/message_direct_path_test.go
rtk git rm internal/integration/stage8_message_bytes_hardening_test.go
rtk git commit -m "test: cover message bytes errors end to end"
```

### Task 4: 补 stream terminal E2E

**Files:**
- Modify: `internal/integration/message_direct_path_test.go`
- Modify: `internal/generator/render_message_client_cgo_test.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Delete after migration: `internal/integration/stage8_stream_terminal_lifecycle_test.go`

- [ ] **Step 1: 写 client-stream terminal 测试**

在真实 fixture 中新增：

```go
func TestMessageClientStreamRejectsOperationsAfterFinish(t *testing.T) {
    handle, errID := StartGreeterUploadMessageClientStream(context.Background())
    if errID != 0 { t.Fatalf(...) }
    output := &GreeterMessageOutput{}
    if errID := FinishGreeterUploadMessageClientStream(context.Background(), handle, output); errID != 0 { t.Fatalf(...) }
    if errID := FinishGreeterUploadMessageClientStream(context.Background(), handle, output); errID == 0 { t.Fatal("second finish error id = 0") }
    if errID := SendGreeterUploadMessageClientStream(context.Background(), handle, validPtr, validLen); errID == 0 { t.Fatal("send after finish error id = 0") }
}
```

- [ ] **Step 2: 写 server-stream terminal 测试**

覆盖：

```go
func TestMessageServerStreamRejectsReadAfterDone(t *testing.T) { ... }
```

执行顺序：Start -> Read until EOF path -> Done -> Read again must return error id。

- [ ] **Step 3: 写 bidi terminal 测试**

覆盖：

```go
func TestMessageBidiRejectsSendAfterCloseSendAndReadAfterCancel(t *testing.T) { ... }
```

- [ ] **Step 4: 写 invalid handle matrix 测试**

覆盖至少：

```go
invalid := rpcruntime.StreamHandle(999999)
Send... invalid -> error id
Finish... invalid -> error id
Read... invalid -> error id
Done... invalid -> error id
CloseSend... invalid -> error id
Cancel... invalid -> error id
```

- [ ] **Step 5: 运行 stream terminal tests**

Run:

```bash
rtk go test ./internal/integration -run 'TestMessage.*Stream.*Rejects|Test.*InvalidHandle' -count=1
```

Expected: PASS 或暴露缺失 guard。

- [ ] **Step 6: 若失败，最小修复 generated stream wrappers**

优先修改：

```text
internal/generator/render_message_client_cgo.go
internal/generator/render_native_client_cgo.go
```

只在通用 lifecycle 缺失时修改：

```text
rpcruntime/stream_session.go
```

- [ ] **Step 7: 迁移 stage8 stream 片段断言到 renderer tests**

保留生成合同：

```text
Load*MessageStream
Take*MessageStream
rpccgo: message client stream handle is invalid
```

- [ ] **Step 8: 删除 stage8 stream 文件**

```bash
rtk git rm internal/integration/stage8_stream_terminal_lifecycle_test.go
```

- [ ] **Step 9: 验证并提交**

Run:

```bash
rtk go test ./internal/generator -run 'TestRender(Message|Native)ClientCGO' -count=1
rtk go test ./internal/integration -run 'TestMessageDirectPath|TestMessage.*Stream|Test.*InvalidHandle' -count=1
```

Expected: PASS。

Commit:

```bash
rtk git add internal/generator/render_message_client_cgo_test.go internal/generator/render_native_client_cgo_test.go internal/generator/render_message_client_cgo.go internal/generator/render_native_client_cgo.go internal/integration/message_direct_path_test.go
rtk git rm internal/integration/stage8_stream_terminal_lifecycle_test.go
rtk git commit -m "test: cover stream terminal errors end to end"
```

## Chunk 4: 补 empty/ownership/release/error text E2E

### Task 5: 补 native empty input 和 release E2E

**Files:**
- Modify: `internal/integration/native_unary_test.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Delete after migration: `internal/integration/stage8_empty_input_normalization_test.go`
- Delete after migration: `internal/integration/stage8_memory_release_hardening_test.go`

- [ ] **Step 1: 写 ptr=0/len>0 empty input 测试**

在 native unary fixture 中新增真实调用：

```go
func TestNativeUnaryTreatsNilPointerAsEmptyRequestInput(t *testing.T) {
    register native server that records received string/bytes/repeated fields
    input := &GreeterSayHelloNativeUnaryInput{
        NamePtr: 0,
        NameLen: 5,
        PayloadPtr: 0,
        PayloadLen: 5,
        // repeated fields if fixture supports them
    }
    output := &GreeterSayHelloNativeUnaryOutput{}
    errID := CallGreeterSayHelloNativeUnary(context.Background(), input, output)
    if errID != 0 { t.Fatalf(...) }
    assert server observed empty values
}
```

如果现有 native unary fixture 不含 repeated 字段，不新增 schema；repeated 行为由 `repeated_native_abi_test.go` 覆盖。

- [ ] **Step 2: 写 negative length 测试**

```go
func TestNativeUnaryRejectsNegativeRequestLength(t *testing.T) {
    input := valid input with NameLen: -1
    errID := CallGreeterSayHelloNativeUnary(context.Background(), input, &GreeterSayHelloNativeUnaryOutput{})
    if errID == 0 { t.Fatal("error id = 0, want negative length error") }
    assertNativeErrContains(t, errID, "negative")
}
```

- [ ] **Step 3: 写 ownership release 测试**

复用 runtime free callback 计数或现有 fixture helper。覆盖：

```go
borrowed input -> free count unchanged
owned input -> free count increments once
```

- [ ] **Step 4: 写 output Release 测试**

```go
func TestNativeUnaryOutputCanBeReleasedOnce(t *testing.T) {
    call unary
    if !rpcruntime.Release(output.MessagePtr) { t.Fatal("first release = false") }
    if rpcruntime.Release(output.MessagePtr) { t.Fatal("second release = true") }
}
```

- [ ] **Step 5: 运行 native tests**

Run:

```bash
rtk go test ./internal/integration -run 'TestNativeUnary.*Empty|TestNativeUnary.*Negative|TestNativeUnary.*Release|TestRepeatedNativeABI' -count=1
```

Expected: PASS 或暴露 release/empty 缺口。

- [ ] **Step 6: 若失败，最小修复 native client/server renderer**

优先修改：

```text
internal/generator/render_native_client_cgo.go
internal/generator/render_native_server_cgo.go
```

不要修改 ABI 类型。

- [ ] **Step 7: 迁移 stage8 empty/memory 片段断言到 renderer tests**

保留生成合同：

```text
ptr == 0 || len == 0
EmptyRpcString
EmptyRpcBytes
EmptyRpcRepeat
EmptyRpcBoolRepeat
Ownership > 0
rpcruntime.Release
```

- [ ] **Step 8: 删除 stage8 empty/memory 文件**

```bash
rtk git rm internal/integration/stage8_empty_input_normalization_test.go
rtk git rm internal/integration/stage8_memory_release_hardening_test.go
```

- [ ] **Step 9: 验证并提交**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderNative(Client|Server)CGO' -count=1
rtk go test ./internal/integration -run 'TestNativeUnary|TestRepeatedNativeABI' -count=1
```

Expected: PASS。

Commit:

```bash
rtk git add internal/generator/render_native_client_cgo_test.go internal/generator/render_native_server_cgo_test.go internal/generator/render_native_client_cgo.go internal/generator/render_native_server_cgo.go internal/integration/native_unary_test.go internal/integration/repeated_native_abi_test.go
rtk git rm internal/integration/stage8_empty_input_normalization_test.go internal/integration/stage8_memory_release_hardening_test.go
rtk git commit -m "test: cover native ABI release boundaries end to end"
```

### Task 6: 补 error text E2E

**Files:**
- Modify: `internal/integration/message_direct_path_test.go`
- Modify: `internal/integration/converter_lifecycle_test.go`
- Modify: `internal/generator/render_message_server_cgo_test.go`

- [ ] **Step 1: 写 known error id text 消费测试**

在 message callback fixture 中制造 callback 返回 runtime error id。测试断言：

```go
errID := Call...
if errID == 0 { t.Fatal(...) }
text := rpcruntime.TakeErrorText(errID)
if !strings.Contains(text, "expected callback error") { t.Fatalf(...) }
second := rpcruntime.TakeErrorText(errID)
if second != 0 or text not empty depending current API contract { t.Fatalf(...) }
```

实际断言以 `rpcruntime.TakeErrorText` 当前 API 为准，先 Read `rpcruntime/errors.go`。

- [ ] **Step 2: 写 unknown error id E2E**

让 cgo message callback 返回不存在的 error id，断言上层 Go error 包含：

```text
unknown error id
```

- [ ] **Step 3: 运行 error text focused tests**

Run:

```bash
rtk go test ./rpcruntime -run 'Test.*Error' -count=1
rtk go test ./internal/integration -run 'Test.*ErrorText|Test.*UnknownErrorID|TestMessageDirectPath' -count=1
```

Expected: PASS。

- [ ] **Step 4: 若失败，最小修复 message server error helper**

优先修改：

```text
internal/generator/render_message_server_cgo.go
```

不要改 error store API，除非 runtime 测试证明 API 本身有 bug。

- [ ] **Step 5: 验证并提交**

Run:

```bash
rtk go test ./rpcruntime -run 'Test.*Error' -count=1
rtk go test ./internal/generator -run 'TestRenderMessageServerCGO' -count=1
rtk go test ./internal/integration -run 'Test.*ErrorText|Test.*UnknownErrorID|TestMessageDirectPath' -count=1
```

Expected: PASS。

Commit:

```bash
rtk git add rpcruntime/errors_test.go internal/generator/render_message_server_cgo_test.go internal/generator/render_message_server_cgo.go internal/integration/message_direct_path_test.go internal/integration/converter_lifecycle_test.go
rtk git commit -m "test: cover message error text boundaries"
```

## Chunk 5: 最终清理和验证

### Task 7: 清理所有 stage 命名测试并跑发布验证

**Files:**
- Modify: `internal/generator/generated_layout_contract_test.go`
- Modify: renamed integration tests if needed
- Delete/Rename: all remaining `*stage*test*.go`

- [ ] **Step 1: 去除 generated layout 的 Stage7 命名**

Rename tests/helpers only，不改变断言：

```text
TestStage7GeneratedLayoutContract -> TestGeneratedLayoutContract
TestStage7PublicAPIContract -> TestGeneratedPublicAPIContract
TestStage7GeneratedLayoutRejectsOldBootstrapNames -> TestGeneratedLayoutRejectsOldBootstrapNames
newStage7GeneratedLayoutPlugin -> newGeneratedLayoutPlugin
assertStage7GeneratedPackage -> assertGeneratedPackage
```

- [ ] **Step 2: 扫描剩余 stage 测试文件**

Run:

```bash
rtk rg -n 'Stage[0-9]|stage[0-9]|stage [0-9]|Stage 8|Stage8' internal/generator internal/integration -g '*_test.go'
rtk rg -n 'stage' internal/generator internal/integration -g '*stage*test*.go'
```

Expected: 无输出。

- [ ] **Step 3: 跑 focused tests**

Run:

```bash
rtk go test ./internal/generator -count=1
rtk go test ./internal/integration -count=1
```

Expected: PASS。

- [ ] **Step 4: 跑全量和 examples**

Run:

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./... -count=1
cd examples/minimal-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../full-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd ../..
```

Expected: 全部 PASS。

- [ ] **Step 5: 跑 ABI/旧模型扫描**

Run:

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/**' -g '!docs/superpowers/**'
rtk rg -n "provider registry|framework selector|multi provider|dual provider|goclient.export|goserver.export|native_forwarding_client|native_forwarding_server" . -g '!docs/plans/**' -g '!docs/specs/**' -g '!docs/superpowers/**' -g '!AGENTS.md'
```

Expected: unsigned scan 无输出；旧模型扫描最多只允许 generator 禁入测试中的受控断言。

- [ ] **Step 6: 查看 git 状态**

Run:

```bash
rtk git status --short
```

Expected: 只包含本任务相关测试/计划文件。

- [ ] **Step 7: 最终提交**

```bash
rtk git add internal/generator internal/integration docs/superpowers/plans/2026-05-09-test-suite-consolidation-e2e-boundaries.md
rtk git commit -m "test: consolidate release boundary coverage"
```

## 完成标准

- [ ] `internal/generator` 与 `internal/integration` 不再包含 `stageN`/`StageN` 命名测试文件或测试函数。
- [ ] Stage 1 中仍有价值的 token、ServicePlan、MethodPlan、lifecycle、contract 断言已迁移到通用 generator 测试。
- [ ] Stage 3-7 中真实长期能力测试保留，但文件和测试名不再绑定迁移阶段。
- [ ] Stage 8 四类边界从片段断言升级为真实端到端测试：message bytes、stream terminal、empty/ownership/release、error text。
- [ ] 删除的阶段测试没有造成覆盖缺口；focused tests、全量 tests、examples、扫描全部通过。

## 风险与注意事项

- 不要为了清理命名而删除仍唯一覆盖核心合同的测试；先迁移再删除。
- 不要把 stage fixture 名称迁移到新通用测试；新 helper 使用领域名称，例如 `messageDirectPath`、`remoteTransport`、`repeatedNativeABI`。
- 不要新增旧 provider/bootstrap/forwarding 模型。
- 不要改变 ABI 类型或 public generated API，除非端到端测试证明当前实现有 bug。
- 若真实端到端测试暴露 bug，按 TDD 处理：保留失败测试，做最小修复，再复跑 focused tests。
