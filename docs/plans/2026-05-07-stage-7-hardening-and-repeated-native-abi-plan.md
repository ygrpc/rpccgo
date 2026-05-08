# Stage 7 Hardening And Repeated Native ABI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 Stage 0-7 审计发现的 repeated native ABI、runtime panic、remote stream cancel、example server lifecycle 和 example error handling 缺口，让当前实现从 happy-path 可运行提升到边界条件可验收。

**Architecture:** 本计划不改变单 dispatcher、单 active server slot、native/message contract、Connect/gRPC local/remote adapter 的模型。repeated native ABI 仍由 generated service runtime 负责转换，`rpcruntime` 只补 checked wrapper，并复用现有 `PinSlice` 处理 fixed-width repeated 输出；remote stream cancel 只增强 session lifecycle，不托管调用方持有的 gRPC `ClientConnInterface`。

**Tech Stack:** Go 1.24、cgo、protobuf/protogen、Connect、gRPC、`rpcruntime`、现有 generated-source acceptance、`examples/minimal-greeter`、`examples/full-greeter`。

---

## 范围

本计划修复：

- repeated numeric、repeated enum、repeated bool 的 native cgo ABI 生成、decode、encode、release 与 converter 验收。
- repeated string、repeated bytes 的 planner 早失败，避免 renderer 半实现。
- `rpcruntime` 中可由 ABI 输入触发的 panic constructor，改为 checked constructor 并让 generated code 使用显式错误返回。
- remote Connect/gRPC stream cancel 语义，确保 `Cancel` 能主动取消 stream context，并尽力关闭 request/response sides。
- example 和 example acceptance 中长期 server 进程的 `go run` cleanup 风险，统一改成先 `go build -o` 再执行二进制。
- example client 中的 `panic(err)`，改为显式返回错误并用 `log.Fatal` 或 `os.Exit(1)` 收口。
- Stage 7 文档与迁移清单中的验证记录和后续风险。

本计划不实现：

- 新 transport、新 server kind、新 rpccgo client 类型。
- remote retry、负载均衡、服务发现、连接池或 gRPC `ClientConnInterface` 生命周期托管。
- repeated message native ABI；仍保持 planner 层明确报错。
- map、oneof、unsigned 32/64 ABI。
- Stage 8 发布命令、CI 环境安装 `protoc` 或兼容性清理。

## 文件结构

- Modify: `rpcruntime/rpc_type.go`  
  新增 `NewRpcBytesChecked`、`NewRpcStringChecked`，保留旧 constructor 但不再让 generated code 依赖 panic 路径。
- Modify: `rpcruntime/rpc_repeat.go`  
  新增 `NewRpcRepeatChecked`、`NewRpcBoolRepeatChecked`，保留 `MustAt` 作为显式 hard-failure helper。
- Modify: `rpcruntime/rpc_type_test.go`
- Modify: `rpcruntime/rpc_repeat_test.go`
- Modify: `internal/generator/render_native_client_cgo.go`  
  生成 native cgo client 对 repeated numeric、enum、bool 输入/输出字段的 ABI struct、decode、encode、checked wrapper 与 release。
- Modify: `internal/generator/render_native_server_cgo.go`  
  生成 cgo native server adapter 对 repeated numeric、enum、bool 字段的 request encode、response decode 和 cleanup。
- Modify: `internal/generator/render_codec.go`  
  如果当前 codec 仍只是 protobuf marshal/unmarshal，补充 generator tests 证明 repeated 字段走完整 protobuf message roundtrip；若实现中新增 native wrapper 结构，则使用 checked wrapper。
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`
- Modify: `internal/generator/render_codec_test.go`
- Create: `internal/integration/repeated_native_abi_stage7_hardening_test.go`  
  真实 generated-source acceptance，覆盖 repeated numeric、enum、bool 的 native/message mismatch 与 cgo native direct path。
- Modify: `internal/generator/render_connect_remote.go`
- Modify: `internal/generator/render_grpc_remote.go`
- Modify: `internal/generator/render_connect_remote_test.go`
- Modify: `internal/generator/render_grpc_remote_test.go`
- Modify: `internal/integration/remote_transport_stage6_acceptance_test.go`
- Modify: `examples/minimal-greeter/magefile.go`
- Modify: `examples/full-greeter/magefile.go`
- Modify: `examples/full-greeter/cmd/rpc/full_matrix_test.go`
- Modify: `examples/minimal-greeter/cmd/client/main.go`
- Modify: `examples/full-greeter/cmd/client/main.go`
- Modify: `examples/minimal-greeter/example_test.go`
- Modify: `examples/full-greeter/example_test.go`
- Modify: `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`
- Modify: `docs/plans/2026-05-07-stage-7-migration-inventory.md`

## Task 1：新增 checked runtime wrapper 并隔离 panic 路径

**Files:**

- Modify: `rpcruntime/rpc_type.go`
- Modify: `rpcruntime/rpc_repeat.go`
- Modify: `rpcruntime/rpc_type_test.go`
- Modify: `rpcruntime/rpc_repeat_test.go`

**迁移内容与理由:** 现有 `NewRpcBytes`、`NewRpcString`、`NewRpcRepeat`、`NewRpcBoolRepeat` 对负长度使用 panic。对 runtime 内部测试 helper 可以保留 hard-failure API，但 generated cgo ABI 必须使用显式错误返回，避免外部输入导致进程崩溃。

- [x] **Step 1: 写 failing tests**

在 `rpcruntime/rpc_type_test.go` 增加：

```go
func TestNewRpcBytesCheckedRejectsNegativeLength(t *testing.T) {
	values := []byte("abc")
	got, err := NewRpcBytesChecked(&values[0], -1, false)
	if err == nil {
		t.Fatal("NewRpcBytesChecked() error = nil, want negative length error")
	}
	if got != nil {
		t.Fatalf("NewRpcBytesChecked() wrapper = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "NewRpcBytes") || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("NewRpcBytesChecked() error = %q, want NewRpcBytes negative length", err.Error())
	}
}

func TestNewRpcStringCheckedRejectsNegativeLength(t *testing.T) {
	values := []byte("abc")
	got, err := NewRpcStringChecked(&values[0], -1, false)
	if err == nil {
		t.Fatal("NewRpcStringChecked() error = nil, want negative length error")
	}
	if got != nil {
		t.Fatalf("NewRpcStringChecked() wrapper = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "NewRpcString") || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("NewRpcStringChecked() error = %q, want NewRpcString negative length", err.Error())
	}
}
```

在 `rpcruntime/rpc_repeat_test.go` 增加：

```go
func TestNewRpcRepeatCheckedRejectsNegativeLength(t *testing.T) {
	values := []int32{1, 2, 3}
	got, err := NewRpcRepeatChecked(&values[0], -1, false)
	if err == nil {
		t.Fatal("NewRpcRepeatChecked() error = nil, want negative length error")
	}
	if got != nil {
		t.Fatalf("NewRpcRepeatChecked() wrapper = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "NewRpcRepeat") || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("NewRpcRepeatChecked() error = %q, want NewRpcRepeat negative length", err.Error())
	}
}

func TestNewRpcBoolRepeatCheckedRejectsNegativeLength(t *testing.T) {
	values := []byte{1, 0, 1}
	got, err := NewRpcBoolRepeatChecked(&values[0], -1, false)
	if err == nil {
		t.Fatal("NewRpcBoolRepeatChecked() error = nil, want negative length error")
	}
	if got != nil {
		t.Fatalf("NewRpcBoolRepeatChecked() wrapper = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "NewRpcBoolRepeat") || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("NewRpcBoolRepeatChecked() error = %q, want NewRpcBoolRepeat negative length", err.Error())
	}
}
```

- [x] **Step 2: Run failing tests**

Run:

```bash
rtk go test ./rpcruntime -run 'TestNewRpc(Bytes|String|Repeat|BoolRepeat)CheckedRejectsNegativeLength' -count=1
```

Expected: FAIL with undefined checked constructor names.

- [x] **Step 3: Implement checked constructors**

In `rpcruntime/rpc_type.go`, add:

```go
func NewRpcBytesChecked(ptr *byte, length int32, ownership bool) (*RpcBytes, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcBytes: %w", err)
	}
	return newRpcBytesUnchecked(ptr, length, ownership), nil
}

func NewRpcStringChecked(ptr *byte, length int32, ownership bool) (*RpcString, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcString: %w", err)
	}
	return newRpcStringUnchecked(ptr, length, ownership), nil
}

func NewRpcBytes(ptr *byte, length int32, ownership bool) *RpcBytes {
	rpc, err := NewRpcBytesChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func NewRpcString(ptr *byte, length int32, ownership bool) *RpcString {
	rpc, err := NewRpcStringChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func newRpcBytesUnchecked(ptr *byte, length int32, ownership bool) *RpcBytes {
	rpc := &RpcBytes{ptr: ptr, length: length, ownership: ownership}
	rpc.attachCleanup(rpcBytesLabel)
	return rpc
}

func newRpcStringUnchecked(ptr *byte, length int32, ownership bool) *RpcString {
	rpc := &RpcString{ptr: ptr, length: length, ownership: ownership}
	rpc.attachCleanup(rpcStringLabel)
	return rpc
}
```

In `rpcruntime/rpc_repeat.go`, add the analogous generic helpers:

```go
func NewRpcRepeatChecked[T NativeRepeatElem](ptr *T, length int32, ownership bool) (*RpcRepeat[T], error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcRepeat: %w", err)
	}
	return newRpcRepeatUnchecked(ptr, length, ownership), nil
}

func NewRpcBoolRepeatChecked(ptr *byte, length int32, ownership bool) (*RpcBoolRepeat, error) {
	if _, err := LengthFromInt32(length); err != nil {
		return nil, fmt.Errorf("NewRpcBoolRepeat: %w", err)
	}
	return newRpcBoolRepeatUnchecked(ptr, length, ownership), nil
}

func NewRpcRepeat[T NativeRepeatElem](ptr *T, length int32, ownership bool) *RpcRepeat[T] {
	rpc, err := NewRpcRepeatChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func NewRpcBoolRepeat(ptr *byte, length int32, ownership bool) *RpcBoolRepeat {
	rpc, err := NewRpcBoolRepeatChecked(ptr, length, ownership)
	if err != nil {
		panic(err)
	}
	return rpc
}

func newRpcRepeatUnchecked[T NativeRepeatElem](ptr *T, length int32, ownership bool) *RpcRepeat[T] {
	rpc := &RpcRepeat[T]{ptr: ptr, length: length, ownership: ownership}
	rpc.attachCleanup(rpcRepeatLabel)
	return rpc
}

func newRpcBoolRepeatUnchecked(ptr *byte, length int32, ownership bool) *RpcBoolRepeat {
	rpc := &RpcBoolRepeat{ptr: ptr, length: length, ownership: ownership}
	rpc.attachCleanup(rpcBoolRepeatLabel)
	return rpc
}
```

- [x] **Step 4: Run runtime tests**

Run:

```bash
rtk go test ./rpcruntime -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
rtk git add rpcruntime/rpc_type.go rpcruntime/rpc_repeat.go rpcruntime/rpc_type_test.go rpcruntime/rpc_repeat_test.go
rtk git commit -m "feat: add checked native ABI wrappers"
```

## Task 2：实现 repeated native cgo ABI 生成和 focused renderer tests

**Files:**

- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`

**迁移内容与理由:** Stage 4B 计划已经把 repeated scalar 和 repeated bool 标成完成，但当前 renderer 对 `NativeABIShapeRepeated` 仍返回 unsupported。这里补上 generated cgo ABI 的结构字段、decode、encode、release，保证 planner 能声明的 native contract 都能生成可用代码。

- [x] **Step 1: Add renderer tests for repeated input and output fields**

In `internal/generator/render_native_client_cgo_test.go`, add a test using a fixture message with:

```proto
message RepeatedRequest {
  repeated int32 scores = 1;
  repeated bool flags = 2;
  repeated int64 counts = 3;
  repeated double ratios = 4;
  repeated Mood moods = 5;
}

message RepeatedReply {
  repeated int32 scores = 1;
  repeated bool flags = 2;
  repeated int64 counts = 3;
  repeated double ratios = 4;
  repeated Mood moods = 5;
}

enum Mood {
  MOOD_UNSPECIFIED = 0;
  MOOD_OK = 1;
  MOOD_BUSY = 2;
}
```

Assert generated cgo native client content contains these fragments:

```go
"ScoresPtr uintptr"
"ScoresLen int32"
"ScoresOwnership int32"
"FlagsPtr uintptr"
"FlagsLen int32"
"FlagsOwnership int32"
"CountsPtr uintptr"
"CountsLen int32"
"CountsOwnership int32"
"RatiosPtr uintptr"
"RatiosLen int32"
"RatiosOwnership int32"
"MoodsPtr uintptr"
"MoodsLen int32"
"MoodsOwnership int32"
"rpcruntime.NewRpcRepeatChecked"
"rpcruntime.NewRpcBoolRepeatChecked"
"rpcruntime.LengthFromInt32(input.ScoresLen)"
"rpcruntime.LengthFromInt32(input.FlagsLen)"
"rpcruntime.Release(ScoresPtr)"
```

Also assert the file does not contain `native unary client field bridge is not implemented` inside repeated decode/encode paths for the repeated fixture method.

In `internal/generator/render_native_server_cgo_test.go`, add equivalent assertions for generated C structs:

```go
"uintptr_t ScoresPtr;"
"int32_t ScoresLen;"
"int32_t ScoresOwnership;"
"uintptr_t FlagsPtr;"
"int32_t FlagsLen;"
"int32_t FlagsOwnership;"
"rpcruntime.NewRpcRepeatChecked"
"rpcruntime.NewRpcBoolRepeatChecked"
"rpcruntime.ReleaseC(unsafe.Pointer(uintptr(output.ScoresPtr)), true"
```

- [x] **Step 2: Run focused tests and confirm failure**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderNative(Client|Server)CGO.*Repeated' -count=1
```

Expected: FAIL because repeated fields are currently rendered as unsupported `uintptr` fallback or missing checked wrapper calls.

- [x] **Step 3: Implement repeated field ABI shape in native client renderer**

In `renderNativeClientFields`, handle `NativeABIShapeRepeated` and `NativeABIShapeBoolByteBufferWrapper` with `Ptr uintptr`, `Len int32`, and `Ownership int32` for input fields; output fields should use `Ptr uintptr` and `Len int32`, with ownership implied by pinned Go memory and released by caller through `rpcruntime.Release`.

In `renderNativeRequestFieldDecode`:

- For repeated signed int32 and enum, call `rpcruntime.NewRpcRepeatChecked((*int32)(unsafe.Pointer(input.<Field>Ptr)), input.<Field>Len, input.<Field>Ownership > 0)`.
- For repeated signed int64, float, double, use `*int64`, `*float32`, `*float64`.
- For repeated bool, call `rpcruntime.NewRpcBoolRepeatChecked((*byte)(unsafe.Pointer(input.<Field>Ptr)), input.<Field>Len, input.<Field>Ownership > 0)`.
- Copy with `SafeSlice()` and release the wrapper.
- Do not render repeated string/bytes; Task 4 makes planner reject them before rendering because current `RpcRepeat` element constraint does not support variable-width elements.

In `renderNativeResponseFieldValidate`, `renderNativeResponseFieldStage`, and `renderNativeResponseFieldCommit`:

- For repeated numeric and enum, pin a contiguous typed slice using existing `rpcruntime.PinSlice`; convert enum values to an `[]int32` before pinning.
- For repeated bool, encode `[]bool` as `[]byte` with `0` or `1`, then pin with `rpcruntime.PinBytes`.

- [x] **Step 4: Implement repeated field ABI shape in cgo native server renderer**

In `renderCGONativeServerCField`, emit `uintptr_t <Field>Ptr`, `int32_t <Field>Len`, and output ownership when the field is repeated or repeated bool.

In request encode:

- Pin repeated numeric/enum/bool request slices before invoking C callbacks.
- For repeated bool, encode `[]bool` as `[]byte` with `0` or `1`.
- Append pinned pointers to the existing `pinned []uintptr` cleanup stack.

In response decode:

- Validate output length with `rpcruntime.LengthFromInt32`.
- Use checked repeated wrappers to copy callback-owned buffers into protobuf repeated fields.
- Run existing cleanup logic when callback marks ownership.

- [x] **Step 5: Run focused tests**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderNative(Client|Server)CGO.*Repeated|TestRenderNative(Client|Server)CGO' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
rtk git add internal/generator/render_native_client_cgo.go internal/generator/render_native_server_cgo.go internal/generator/render_native_client_cgo_test.go internal/generator/render_native_server_cgo_test.go
rtk git commit -m "feat: generate repeated native cgo ABI"
```

## Task 3：用 generated-source acceptance 锁住 repeated native ABI

**Files:**

- Create: `internal/integration/repeated_native_abi_stage7_hardening_test.go`
- Modify: `internal/generator/render_codec.go`
- Modify: `internal/generator/render_codec_test.go`

**迁移内容与理由:** Renderer fragment tests 只能证明代码片段存在，不能证明 generated package 能编译和端到端转换。这里新增临时模块 acceptance，覆盖 repeated 字段在 native direct path、message-to-native converter path、native-to-message converter path 中的真实行为。

- [x] **Step 1: Write acceptance test**

Create `internal/integration/repeated_native_abi_stage7_hardening_test.go` with the existing generated-source fixture style:

```go
func TestRepeatedNativeABIStage7HardeningAcceptance(t *testing.T) {
	t.Parallel()
	dir := newGeneratedModule(t, generatedModuleConfig{
		ModulePath: "example.com/stage7repeated",
		ProtoPath:  "repeated/v1/repeated.proto",
		GoPackage:  "example.com/stage7repeated/repeated/v1;repeatedv1",
		CGODir:     "../../cmd/rpc",
		ProtoBody: repeatedNativeABIProto(),
	})
	runGeneratedModuleTest(t, dir, "./cmd/rpc", "-run", "^TestRepeatedNativeABI$")
}
```

The proto must include:

```proto
syntax = "proto3";

package stage7.repeated.v1;

option go_package = "example.com/stage7repeated/repeated/v1;repeatedv1";

message RepeatedRequest {
  repeated int32 scores = 1;
  repeated bool flags = 2;
}

message RepeatedReply {
  repeated int32 scores = 1;
  repeated bool flags = 2;
}

// @rpccgo: msg-connect|native
service RepeatedGreeter {
  rpc Echo(RepeatedRequest) returns (RepeatedReply);
}
```

The generated module test must:

- Register Go native server and call cgo native client with repeated int32 and repeated bool.
- Register cgo message server and call cgo native client, proving native-to-message conversion.
- Register Go native server and call cgo message client, proving message-to-native conversion.
- Pass negative repeated lengths through cgo native client and assert non-zero error id containing `negative`, not panic.
- Release all output pointers with `rpcruntime.Release`.

- [x] **Step 2: Run acceptance and confirm failure**

Run:

```bash
rtk go test ./internal/integration -run TestRepeatedNativeABIStage7HardeningAcceptance -count=1
```

Expected: FAIL until Task 2 implementation and codec behavior are complete.

- [x] **Step 3: Align codec renderer tests**

If `render_codec.go` remains protobuf marshal/unmarshal only, add tests proving repeated fields are preserved through message/native converter roundtrip by compiling generated source. If codec begins to inspect native wrappers directly, add exact tests for repeated bool byte encoding and repeated numeric wrapper errors.

Run:

```bash
rtk go test ./internal/generator -run TestRenderCodec -count=1
```

Expected: PASS.

- [x] **Step 4: Run acceptance**

Run:

```bash
rtk go test ./internal/integration -run TestRepeatedNativeABIStage7HardeningAcceptance -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
rtk git add internal/integration/repeated_native_abi_stage7_hardening_test.go internal/generator/render_codec.go internal/generator/render_codec_test.go
rtk git commit -m "test: verify repeated native ABI hardening"
```

## Task 4：若 repeated string/bytes 不实现，则在 planner 明确拒绝

**Files:**

- Modify: `internal/generator/contract_plan.go`
- Modify: `internal/generator/contract_plan_test.go`
- Modify: `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`

**迁移内容与理由:** 当前 `RpcRepeat[T]` 只支持 fixed-width 元素。repeated string/bytes 是 variable-width ABI，需要额外 offset table 或 message bytes representation。为了不制造半实现，本计划把 repeated string/bytes 标为当前阶段不支持，并在 planner 早失败。若 Task 2 已实现完整 variable-width ABI，本 Task 改为记录支持合同和测试，不拒绝。

- [x] **Step 1: Add planner tests**

In `contract_plan_test.go`, add:

```go
func TestBuildContractPlanRejectsRepeatedStringNativeABI(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedStringContractTestFile())
	_, err := BuildDescriptorPlan(plugin)
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want repeated string unsupported error")
	}
	for _, want := range []string{"repeated string", "not supported"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want %q", err.Error(), want)
		}
	}
}

func TestBuildContractPlanRejectsRepeatedBytesNativeABI(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedBytesContractTestFile())
	_, err := BuildDescriptorPlan(plugin)
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want repeated bytes unsupported error")
	}
	for _, want := range []string{"repeated bytes", "not supported"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want %q", err.Error(), want)
		}
	}
}
```

- [x] **Step 2: Run tests and confirm failure**

Run:

```bash
rtk go test ./internal/generator -run 'TestBuildContractPlanRejectsRepeated(String|Bytes)NativeABI' -count=1
```

Expected: FAIL until planner rejects repeated string/bytes explicitly.

- [x] **Step 3: Implement explicit planner errors**

In `nativeFieldPlan`, for `FieldKindString` and `FieldKindBytes` with `field.Repeated == true`, return:

```go
return NativeFieldPlan{}, fmt.Errorf("repeated string fields are not supported in native ABI")
```

and:

```go
return NativeFieldPlan{}, fmt.Errorf("repeated bytes fields are not supported in native ABI")
```

Keep repeated numeric, enum, and bool supported.

- [x] **Step 4: Update Stage 7 plan note**

In `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`, add a hardening note under `阶段 7 后续风险`:

```markdown
- Stage 7 hardening 后，repeated numeric、enum 和 bool native ABI 有 generated-source acceptance；repeated string/bytes 仍不进入 native ABI，planner 会明确报错，避免 renderer 半实现。
```

- [x] **Step 5: Run planner tests**

Run:

```bash
rtk go test ./internal/generator -run 'TestBuildContractPlanRejectsRepeated(String|Bytes)NativeABI|TestNativeFieldPlanMarksRepeatedBoolAsByteBufferWrapper|TestBuildContractPlanRejectsRepeatedMessage' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
rtk git add internal/generator/contract_plan.go internal/generator/contract_plan_test.go docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md
rtk git commit -m "fix: reject unsupported repeated variable native ABI"
```

## Task 5：强化 remote stream cancel 语义

**Files:**

- Modify: `internal/generator/render_connect_remote.go`
- Modify: `internal/generator/render_grpc_remote.go`
- Modify: `internal/generator/render_connect_remote_test.go`
- Modify: `internal/generator/render_grpc_remote_test.go`
- Modify: `internal/integration/remote_transport_stage6_acceptance_test.go`

**迁移内容与理由:** Stage 6 remote adapter 已能完成 happy path，但 cancel 主要关闭 stream send side，不能稳定证明远端上下文被取消。这里为 remote stream session 保存 cancel func，`Cancel` 先取消 context，再关闭 stream sides，并用 acceptance 证明远端能观察到 cancel。

- [x] **Step 1: Add renderer assertions**

In `render_connect_remote_test.go`, assert generated streaming sessions contain:

```go
"ctx context.Context"
"cancel context.CancelFunc"
"stream := s."
"context.WithCancel(ctx)"
"defer cancel()"
"s.cancel()"
"closeConnectRemoteConn"
```

In `render_grpc_remote_test.go`, assert generated streaming sessions contain:

```go
"ctx context.Context"
"cancel context.CancelFunc"
"stream, err := s.conn.NewStream(streamCtx"
"s.cancel()"
"return s.stream.CloseSend()"
```

- [x] **Step 2: Run renderer tests and confirm failure**

Run:

```bash
rtk go test ./internal/generator -run 'TestRender(Connect|GRPC)Remote' -count=1
```

Expected: FAIL until generated session structs save cancel functions.

- [x] **Step 3: Implement Connect remote cancel context**

In `render_connect_remote.go`:

- In each streaming `Start`, create `streamCtx, cancel := context.WithCancel(ctx)` and pass `streamCtx` to Connect call.
- Add `cancel context.CancelFunc` to generated session structs.
- `Cancel(ctx)` should call `s.cancel()` if non-nil before closing stream sides.
- `Finish`, `Done`, and successful terminal paths should call `cancel()` after normal completion to avoid leaks.

Generated client-stream session shape:

```go
type <Session> struct {
	stream *connect.ClientStreamForClient[Req, Resp]
	cancel context.CancelFunc
}
```

- [x] **Step 4: Implement gRPC remote cancel context**

In `render_grpc_remote.go`:

- In each streaming `Start`, create `streamCtx, cancel := context.WithCancel(ctx)` and pass `streamCtx` into `NewStream`.
- Add `cancel context.CancelFunc` to generated session structs.
- `Cancel(ctx)` calls `s.cancel()` and then `CloseSend()`.
- `Finish`, `Done`, and EOF terminal paths call `cancel()` after normal completion.

Do not close caller-owned `grpc.ClientConnInterface`.

- [x] **Step 5: Add integration acceptance**

In `remote_transport_stage6_acceptance_test.go`, add subtests:

- `connect_remote_client_stream_cancel_notifies_remote_context`
- `connect_remote_bidi_cancel_notifies_remote_context`
- `grpc_remote_client_stream_cancel_notifies_remote_context`
- `grpc_remote_bidi_cancel_notifies_remote_context`

The remote server fixture should block on `ctx.Done()` and send a signal channel when cancellation is observed. The test should:

1. Register remote adapter.
2. Start a stream.
3. Send one request.
4. Call generated cgo message/native cancel function.
5. Wait for remote cancel signal with a 2 second timeout.

- [x] **Step 6: Run remote acceptance**

Run:

```bash
rtk go test ./internal/generator -run 'TestRender(Connect|GRPC)Remote' -count=1
rtk go test ./internal/integration -run TestRemoteTransportStage6Acceptance -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

```bash
rtk git add internal/generator/render_connect_remote.go internal/generator/render_grpc_remote.go internal/generator/render_connect_remote_test.go internal/generator/render_grpc_remote_test.go internal/integration/remote_transport_stage6_acceptance_test.go
rtk git commit -m "fix: cancel remote stream contexts"
```

## Task 6：修复 example server lifecycle 和 client error handling

**Files:**

- Modify: `examples/minimal-greeter/magefile.go`
- Modify: `examples/full-greeter/magefile.go`
- Modify: `examples/full-greeter/cmd/rpc/full_matrix_test.go`
- Modify: `examples/minimal-greeter/cmd/client/main.go`
- Modify: `examples/full-greeter/cmd/client/main.go`
- Modify: `examples/minimal-greeter/example_test.go`
- Modify: `examples/full-greeter/example_test.go`

**迁移内容与理由:** Stage 6 acceptance 已证明 `go run` 启长期 server 会留下真实子进程。当前 example `mage run` 和 full matrix test 又出现同类模式。用户示例也不应展示 `panic(err)`。

- [x] **Step 1: Add example lifecycle tests**

In `examples/minimal-greeter/example_test.go`, add a test that runs:

```go
cmd := exec.Command("go", "run", "github.com/magefile/mage", "run")
cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
output, err := cmd.CombinedOutput()
if err != nil {
	t.Fatalf("mage run error = %v\n%s", err, output)
}
if bytes.Contains(output, []byte("panic:")) {
	t.Fatalf("mage run output contains panic:\n%s", output)
}
```

In `examples/full-greeter/example_test.go`, add the same `panic:` guard.

- [x] **Step 2: Run example tests**

Run:

```bash
cd examples/minimal-greeter && rtk go test ./... -count=1
cd examples/full-greeter && rtk go test ./... -count=1
```

Expected: Current tests may pass, but implementation still needs lifecycle hardening.

- [x] **Step 3: Build server binaries before starting**

In both `magefile.go` files:

- Create `serverBin := filepath.Join(os.TempDir(), "rpccgo-<example>-server-"+strconv.FormatInt(time.Now().UnixNano(), 10))`.
- Run `go build -o serverBin ./cmd/server`.
- Start `exec.Command(serverBin)` instead of `go run ./cmd/server`.
- Cleanup with `server.Process.Kill()` and `server.Wait()`, plus `os.Remove(serverBin)`.
- Keep `Server()` target as `go run ./cmd/server`, because it is intentionally user-facing long-running development entrypoint.

In `examples/full-greeter/cmd/rpc/full_matrix_test.go`, replace `exec.Command("go", "run", "./cmd/server")` with build-then-run binary in `t.TempDir()`.

- [x] **Step 4: Replace panic in example clients**

In `examples/minimal-greeter/cmd/client/main.go`:

```go
func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

Move the existing client call into `run(ctx context.Context) error`.

In `examples/full-greeter/cmd/client/main.go`, use:

```go
func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

and move current body into `run(ctx context.Context) error`.

- [x] **Step 5: Run example commands**

Run:

```bash
cd examples/minimal-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
cd examples/full-greeter && rtk go test ./... -count=1 && rtk go run github.com/magefile/mage run
```

Expected: PASS. Output must not include `panic:`.

- [x] **Step 6: Commit**

```bash
rtk git add examples/minimal-greeter examples/full-greeter
rtk git commit -m "fix: harden example process lifecycle"
```

## Task 7：文档收口、总验证与提交

**Files:**

- Modify: `docs/plans/2026-05-07-stage-7-hardening-and-repeated-native-abi-plan.md`
- Modify: `docs/plans/2026-05-07-stage-7-migration-inventory.md`
- Modify: `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`

**迁移内容与理由:** 这次是 Stage 7 后的 hardening，需要把“发现了哪些边界、哪些已修复、哪些仍明确不支持”写回阶段文档，避免后续继续以为 repeated string/bytes 或 remote cancel 已经完整覆盖。

- [x] **Step 1: Update migration inventory**

In `docs/plans/2026-05-07-stage-7-migration-inventory.md`, add:

```markdown
## Hardening 补充

- repeated numeric、enum、bool native ABI 已由 Stage 7 hardening acceptance 覆盖。
- repeated string/bytes 不进入当前 native ABI；planner 明确报错。
- example `Run` 和 full matrix acceptance 使用构建后的 server 二进制，避免 `go run` cleanup 只杀父进程。
- example client 不再使用 `panic(err)` 展示错误处理。
- remote stream cancel 会取消 stream context；gRPC connection 生命周期仍由调用方持有。
```

- [x] **Step 2: Record verification results**

Append verification results to this plan:

```markdown
## 验证结果

- `rtk go test ./rpcruntime -count=1`：PASS。
- `rtk go test ./internal/generator -count=1`：PASS。
- `rtk go test ./internal/integration -count=1`：PASS。
- `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- forbidden unsigned scan：PASS，无命中。
```

- [x] **Step 3: Run full verification**

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
```

Expected:

- All `go test` and `mage run` commands PASS.
- The forbidden unsigned scan exits with code 1 and no output.

- [x] **Step 4: Update checkboxes**

Mark all completed steps in this plan with `[x]` only after the corresponding command or implementation is actually done.

- [x] **Step 5: Commit docs**

```bash
rtk git add docs/plans/2026-05-07-stage-7-hardening-and-repeated-native-abi-plan.md docs/plans/2026-05-07-stage-7-migration-inventory.md docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md
rtk git commit -m "docs: record stage 7 hardening verification"
```

## 完成标准

- [x] repeated numeric、enum、bool native ABI 在 generated cgo client/server 中不再落入 unsupported fallback。
- [x] repeated string/bytes native ABI 由 planner 明确拒绝。
- [x] cgo ABI 输入中的负长度返回 error id 或 Go error，不触发 runtime panic。
- [x] generated code 使用 checked runtime wrappers。
- [x] remote Connect/gRPC stream `Cancel` 能取消 stream context，并由 integration acceptance 证明远端观察到取消。
- [x] example `Run` 和 full matrix test 不再用 `go run` 启长期 server。
- [x] example client 不再使用 `panic(err)`。
- [x] Stage 7 迁移清单记录 hardening 后的新边界。
- [x] runtime、generator、integration、全仓、两个 example、两个 `mage run` 和 forbidden unsigned scan 全部通过。

## 提交边界

计划执行时按以下边界提交，不要 squash 成一个大 commit：

1. `feat: add checked native ABI wrappers`
2. `feat: generate repeated native cgo ABI`
3. `test: verify repeated native ABI hardening`
4. `fix: reject unsupported repeated variable native ABI`
5. `fix: cancel remote stream contexts`
6. `fix: harden example process lifecycle`
7. `docs: record stage 7 hardening verification`

## 后续风险

- repeated string/bytes 若未来要支持，需要设计 variable-width ABI，不能复用 fixed-width `RpcRepeat[T]`。
- gRPC remote cancel 不关闭调用方持有的 `ClientConnInterface`，这是当前设计边界，不是泄漏。
- `MustAt` 仍保留 hard-failure 语义；业务路径和 generated code 不应调用它。
- CI 若没有 `protoc`，example generation 仍需要额外环境准备；本计划不处理 CI 安装。

## 验证结果

- `rtk go test ./rpcruntime -count=1`：PASS。
- `rtk go test ./internal/generator -count=1`：PASS。
- `rtk go test ./internal/integration -count=1`：PASS。
- `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS。
- forbidden unsigned scan：PASS，无命中。
