# 阶段 5 Connect 与 gRPC 本地 Server Adapter 实施计划

> **给 agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` 或 `superpowers:executing-plans` 按任务逐项执行。步骤使用 checkbox (`- [ ]`) 语法跟踪。

**目标:** 为 generated service 生成本地 Connect handler adapter 与 gRPC server adapter，让标准 RPC 入站请求以 message contract 进入现有 dispatcher，并复用 Stage 4B native/message converter 路由到当前 active server。

**架构:** 阶段 5 只实现本地 transport 入站 adapter：`<service>.server.connect.rpccgo.go` 输出 `http.Handler`，`<service>.server.grpc.rpccgo.go` 输出 `grpc.ServiceDesc` 注册入口。两类 adapter 都不占用 active server slot，而是把标准 RPC 入站请求转成 message contract，统一调用 generated message bridge 进入现有 dispatcher，不引入 connect/grpc client、remote adapter、旧 framework selector 或旧 bootstrap。

**技术栈:** Go 1.24、`connectrpc.com/connect`、`google.golang.org/grpc`、`google.golang.org/protobuf/proto`、现有 `rpcruntime.Dispatcher` 与 generated message bridge。

---

## 范围

阶段 5 实现：

- Connect 本地 handler adapter。
- gRPC 本地 server adapter。
- unary、client streaming、server streaming、bidi streaming 入站请求进入 dispatcher。
- 入站 message 请求路由到 cgo message server、Go native server、cgo native server，并在 mismatch 时复用 Stage 4B converter。
- generator focused、integration focused、runtime focused、全仓测试和 forbidden unsigned scan。
- 阶段 5 迁移清单。

阶段 5 不实现：

- connect remote server adapter。
- grpc remote server adapter。
- connect/grpc 标准 client 生成。
- 多 registry、多 provider bootstrap、framework selector。
- 自定义 transport contract。

## 外部 API 依据

- Connect 官方 Go package 文档说明 `connect.Handler` 是单个 RPC 的 `http.Handler`，并提供 `NewUnaryHandler`、`NewClientStreamHandler`、`NewServerStreamHandler`、`NewBidiStreamHandler`。阶段 5 使用这些标准 handler 构造函数。
- gRPC Go 官方 package 文档说明 `grpc.ServiceRegistrar` 通过 `RegisterService(*grpc.ServiceDesc, impl any)` 注册服务，unary 与 streaming 处理分别使用 `grpc.UnaryHandler` 和 `grpc.StreamHandler` 形态。阶段 5 生成 `grpc.ServiceDesc`，不绕过 gRPC 注册模型。

## 文件结构

- Modify: `go.mod`、`go.sum`  
  添加 Connect 与 gRPC Go 依赖。
- Modify: `internal/generator/plan.go`  
  扩展 `MessageFileFamilyPlan`，加入 `ConnectServer` 和 `GRPCServer`。
- Modify: `internal/generator/render_message_plan.go`  
  根据 `@rpccgo` token 规划 `.server.connect.rpccgo.go` 与 `.server.grpc.rpccgo.go`。
- Modify: `internal/generator/render.go`  
  将新增文件接入 `RenderMessageStageFiles` 与 `RenderStageFiles`。
- Create: `internal/generator/render_connect_server.go`  
  生成 Connect 本地 handler adapter。
- Create: `internal/generator/render_grpc_server.go`  
  生成 gRPC 本地 server adapter。
- Modify: `internal/generator/render_message_server_cgo.go`  
  为 cgo message server streaming callback 生成 `io.EOF` error id helper，让本地 transport handler 能识别 server stream 正常结束。
- Create: `internal/generator/render_connect_server_test.go`  
  覆盖 Connect 文件规划、生成内容、token gating、无 remote 生成。
- Create: `internal/generator/render_grpc_server_test.go`  
  覆盖 gRPC 文件规划、生成内容、token gating、无 remote 生成。
- Modify: `internal/generator/generator_test.go`、`internal/generator/render_message_plan_test.go`、`internal/generator/render_codec_test.go`  
  更新已存在的 generated filename 断言。
- Create: `internal/integration/local_transport_stage5_acceptance_test.go`  
  生成临时模块，并用 `httptest.Server` 与 `grpc.Server` + `bufconn` 汇总覆盖 Connect/gRPC unary 与三类 streaming、错误传播与 snapshot。
- Create: `docs/plans/2026-05-06-stage-5-migration-inventory.md`  
  记录迁移、参考和不迁移范围。
- Modify: `docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md`  
  执行时更新 checkbox 与验证记录。

## Task 1：添加 transport 依赖并锁定文件族计划

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/generator/plan.go`
- Modify: `internal/generator/render_message_plan.go`
- Modify: `internal/generator/render_message_plan_test.go`

- [x] **Step 1: 写失败测试**

在 `internal/generator/render_message_plan_test.go` 添加：

```go
func TestRenderMessageFileFamilyPlanIncludesLocalTransportAdapters(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.ConnectServer, "test/v1/greeter.greeter.server.connect.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.GRPCServer, "test/v1/greeter.greeter.server.grpc.rpccgo.go", true)
}

func TestRenderMessageFileFamilyPlanGatesLocalTransportAdaptersByToken(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}

	connectOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
	})
	assertGeneratedFilePlan(t, connectOnly.ConnectServer, "test/v1/greeter.greeter.server.connect.rpccgo.go", true)
	assertGeneratedFilePlan(t, connectOnly.GRPCServer, "test/v1/greeter.greeter.server.grpc.rpccgo.go", false)

	grpcOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageGRPC}},
	})
	assertGeneratedFilePlan(t, grpcOnly.ConnectServer, "test/v1/greeter.greeter.server.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, grpcOnly.GRPCServer, "test/v1/greeter.greeter.server.grpc.rpccgo.go", true)
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessageFileFamilyPlanIncludesLocalTransportAdapters|TestRenderMessageFileFamilyPlanGatesLocalTransportAdaptersByToken' -count=1
```

Expected: 编译失败，提示 `MessageFileFamilyPlan` 没有 `ConnectServer` / `GRPCServer` 字段。

- [x] **Step 3: 添加依赖**

Run:

```bash
rtk go get connectrpc.com/connect@v1.19.1 google.golang.org/grpc
```

Expected: `go.mod` 增加 `connectrpc.com/connect` 与 `google.golang.org/grpc`，`go.sum` 更新。

- [x] **Step 4: 实现文件族计划**

在 `internal/generator/plan.go` 中把 `MessageFileFamilyPlan` 改为：

```go
type MessageFileFamilyPlan struct {
	Runtime          GeneratedFilePlan
	CGOMessageServer GeneratedFilePlan
	CGOMessageClient GeneratedFilePlan
	ConnectServer    GeneratedFilePlan
	GRPCServer       GeneratedFilePlan
}
```

在 `internal/generator/render_message_plan.go` 中扩展 `BuildMessageFileFamilyPlan`：

```go
return MessageFileFamilyPlan{
	Runtime: GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.runtime.rpccgo.go", prefix, serviceName),
		Enabled:  true,
	},
	CGOMessageServer: GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.server.message.cgo.rpccgo.go", cgoPrefix, serviceName),
		Enabled:  needsCGOMessageServerAdapter(service),
	},
	CGOMessageClient: GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.client.message.cgo.rpccgo.go", cgoPrefix, serviceName),
		Enabled:  true,
	},
	ConnectServer: GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.server.connect.rpccgo.go", prefix, serviceName),
		Enabled:  service.Adapters.Has(AdapterTokenMessageConnect),
	},
	GRPCServer: GeneratedFilePlan{
		Filename: fmt.Sprintf("%s.%s.server.grpc.rpccgo.go", prefix, serviceName),
		Enabled:  service.Adapters.Has(AdapterTokenMessageGRPC),
	},
}
```

- [x] **Step 5: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessageFileFamilyPlan' -count=1
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
rtk git add go.mod go.sum internal/generator/plan.go internal/generator/render_message_plan.go internal/generator/render_message_plan_test.go
rtk git commit -m "feat: plan local transport adapter files"
```

## Task 2：把新增文件接入 render pipeline

**Files:**

- Modify: `internal/generator/render.go`
- Modify: `internal/generator/generator_test.go`
- Modify: `internal/generator/render_codec_test.go`
- Modify: `internal/integration/message_stage4a_acceptance_test.go`

- [x] **Step 1: 写失败测试**

在 `internal/generator/generator_test.go` 添加：

```go
func TestRenderStageFilesEmitsLocalTransportAdaptersByServiceToken(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|msg-grpc|native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertGeneratedFilenames(t, plugin, []string{
		"test/v1/greeter.greeter.runtime.rpccgo.go",
		"test/v1/greeter.greeter.server.native.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go",
		"test/v1/cgo/greeter.greeter.client.message.cgo.rpccgo.go",
		"test/v1/greeter.greeter.codec.rpccgo.go",
		"test/v1/greeter.greeter.server.connect.rpccgo.go",
		"test/v1/greeter.greeter.server.grpc.rpccgo.go",
	})
	assertNoGeneratedFilenameContains(t, plugin, ".remote.")
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderStageFilesEmitsLocalTransportAdaptersByServiceToken -count=1
```

Expected: FAIL，实际 generated filenames 缺少 `.server.connect.rpccgo.go` 与 `.server.grpc.rpccgo.go`。

- [x] **Step 3: 接入 renderer 分发**

在 `RenderMessageStageFiles` 的 files 列表中加入：

```go
family.ConnectServer,
family.GRPCServer,
```

并在循环中加入分支：

```go
if file == family.ConnectServer {
	if err := renderConnectServerFile(plugin, plan, service, file); err != nil {
		return err
	}
	continue
}
if file == family.GRPCServer {
	if err := renderGRPCServerFile(plugin, plan, service, file); err != nil {
		return err
	}
	continue
}
```

在 `RenderStageFiles` 中，messageService 设置后调用：

```go
markRendered(rendered, messageService.MessageFileFamily.ConnectServer)
markRendered(rendered, messageService.MessageFileFamily.GRPCServer)
```

避免后续新增合并路径重复输出。

- [x] **Step 4: 添加临时 renderer 壳**

创建 `internal/generator/render_connect_server.go`：

```go
package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderConnectServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// rpccgo connect local server adapter stage file for ", service.GoName)
	return nil
}
```

创建 `internal/generator/render_grpc_server.go`：

```go
package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderGRPCServerFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// rpccgo grpc local server adapter stage file for ", service.GoName)
	return nil
}
```

- [x] **Step 5: 更新既有 filename 断言**

把 Stage 4A/4B 中“不生成 connect/grpc”的断言改成 token-specific：

- `@rpccgo:native` 现在应生成 connect 文件，因为 `native` 展开为 `msg-connect|native`。
- `@rpccgo:msg-grpc` 应生成 grpc 文件。
- 只保留“不生成 `.remote.`”作为阶段 5 前后的硬边界。

- [x] **Step 6: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator ./internal/integration -run 'TestRenderStageFilesEmitsLocalTransportAdaptersByServiceToken|TestMessageStage4AAcceptanceGeneratedDirectPath|TestRenderStageFilesEmitsCodecWithoutTransportAdapterFiles' -count=1
```

Expected: PASS。其中 `TestRenderStageFilesEmitsCodecWithoutTransportAdapterFiles` 需要重命名或调整为只断言不生成 remote adapter。

- [x] **Step 7: 提交**

```bash
rtk git add internal/generator/render.go internal/generator/render_connect_server.go internal/generator/render_grpc_server.go internal/generator/generator_test.go internal/generator/render_codec_test.go internal/integration/message_stage4a_acceptance_test.go
rtk git commit -m "feat: route local transport adapter rendering"
```

## Task 3：补齐 message stream EOF 合同

**Files:**

- Modify: `internal/generator/render_message_server_cgo.go`
- Modify: `internal/generator/render_message_server_cgo_test.go`
- Modify: `internal/integration/message_direct_path_test.go`

- [x] **Step 1: 写失败测试**

在 `internal/generator/render_message_server_cgo_test.go` 添加：

```go
func TestRenderMessageServerCGOFileEmitsStreamEOFHelper(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plans, err := Generate(plugin)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	AttachMessageFileFamilyPlan(&plans[0])
	if err := RenderMessageStageFiles(plugin, plans[0]); err != nil {
		t.Fatalf("RenderMessageStageFiles() error = %v", err)
	}

	const serverFile = "test/v1/cgo/greeter.greeter.server.message.cgo.rpccgo.go"
	for _, fragment := range []string{
		`io "io"`,
		"func GreeterCGOMessageStreamEOFErrorID() int32 {",
		"return int32(rpcruntime.StoreError(io.EOF))",
	} {
		assertGeneratedContentContains(t, plugin, serverFile, fragment)
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderMessageServerCGOFileEmitsStreamEOFHelper -count=1
```

Expected: FAIL，当前 cgo message server 文件没有 `io.EOF` helper。

- [x] **Step 3: 生成 EOF helper**

在 `renderMessageServerCGOFile` 的 import 中加入：

```go
g.P(`io "io"`)
```

并在文件末尾生成：

```go
func GreeterCGOMessageStreamEOFErrorID() int32 {
	return int32(rpcruntime.StoreError(io.EOF))
}
```

helper 命名用 service GoName 前缀，避免多 service 同包冲突。

- [x] **Step 4: 用 integration 证明 EOF 不破坏 direct path**

在 `internal/integration/message_direct_path_test.go` 的 cgo fixture 中新增一个 server-streaming 测试：

```go
func TestMessageServerStreamingEOFHelperIsAvailable(t *testing.T) {
	errID := GreeterCGOMessageStreamEOFErrorID()
	assertMessageErrContains(t, errID, "EOF")
}
```

这个测试只证明 generated cgo package 可以取得 `io.EOF` error id；direct path 仍由 cgo client 显式 `Done`，不改变 Stage 4A/4B 行为。

- [x] **Step 5: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestRenderMessageServerCGOFileEmitsStreamEOFHelper -count=1
rtk go test ./internal/integration -run TestMessageServerStreamingEOFHelperIsAvailable -count=1
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
rtk git add internal/generator/render_message_server_cgo.go internal/generator/render_message_server_cgo_test.go internal/integration/message_direct_path_test.go
rtk git commit -m "feat: expose message stream eof helper"
```

## Task 4：生成 Connect unary 与 streaming handler adapter

**Files:**

- Modify: `internal/generator/render_connect_server.go`
- Create: `internal/generator/render_connect_server_test.go`

- [x] **Step 1: 写失败测试**

在 `internal/generator/render_connect_server_test.go` 添加：

```go
package generator

import "testing"

func TestRenderConnectServerFileEmitsHandlers(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-connect|native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const connectFile = "test/v1/greeter.greeter.server.connect.rpccgo.go"
	for _, fragment := range []string{
		`connect "connectrpc.com/connect"`,
		`http "net/http"`,
		"const GreeterConnectServiceName = \"test.v1.Greeter\"",
		"func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {",
		"func RegisterGreeterConnectHandler(options ...connect.HandlerOption) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {",
		"return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindConnectHandler, adapter)",
		"connect.NewUnaryHandler",
		"connect.NewClientStreamHandler",
		"connect.NewServerStreamHandler",
		"connect.NewBidiStreamHandler",
	} {
		assertGeneratedContentContains(t, plugin, connectFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, connectFile, "google.golang.org/grpc", ".remote.", "panic(")
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderConnectServerFileEmitsHandlers -count=1
```

Expected: FAIL，当前 Connect 文件只有 marker。

- [x] **Step 3: 实现 Connect adapter 类型与注册函数**

在 `render_connect_server.go` 中生成：

```go
type greeterConnectAdapter struct{}

func RegisterGreeterConnectHandler(options ...connect.HandlerOption) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	adapter := &greeterConnectAdapter{}
	return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindConnectHandler, adapter)
}
```

并生成：

```go
func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	adapter := &greeterConnectAdapter{}
	mux.Handle(GreeterUnaryConnectProcedure, connect.NewUnaryHandler(GreeterUnaryConnectProcedure, adapter.UnaryConnect, options...))
	return GreeterConnectServicePathPrefix, mux
}
```

实际实现必须为每个 method 生成 procedure 常量：

```go
const GreeterUnaryConnectProcedure = "/test.v1.Greeter/Unary"
```

`GreeterConnectServicePathPrefix` 使用 `"/" + service.FullName + "/"`。

- [x] **Step 4: 实现 unary handler**

生成每个 unary method：

```go
func (a *greeterConnectAdapter) UnaryConnect(ctx context.Context, req *connect.Request[HelloRequest]) (*connect.Response[HelloReply], error) {
	if req == nil || req.Msg == nil {
		return nil, errors.New("rpccgo: Greeter.Unary connect request is nil")
	}
	data, err := proto.Marshal(req.Msg)
	if err != nil {
		return nil, fmt.Errorf("rpccgo: connect request protobuf marshal failed: %w", err)
	}
	respData, err := NewGreeterCGOMessageClientBridge().Unary(ctx, data)
	if err != nil {
		return nil, err
	}
	var resp HelloReply
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("rpccgo: connect response protobuf unmarshal failed: %w", err)
	}
	return connect.NewResponse(&resp), nil
}
```

- [x] **Step 5: 实现三类 streaming handler**

生成 client streaming：

```go
func (a *greeterConnectAdapter) UploadConnect(ctx context.Context, stream *connect.ClientStream[HelloRequest]) (*connect.Response[HelloReply], error) {
	session, err := NewGreeterCGOMessageClientBridge().StartUpload(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Cancel(ctx)
	for stream.Receive() {
		data, err := proto.Marshal(stream.Msg())
		if err != nil {
			return nil, fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)
		}
		if err := session.Send(ctx, data); err != nil {
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	respData, err := session.Finish(ctx)
	if err != nil {
		return nil, err
	}
	var resp HelloReply
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)
	}
	return connect.NewResponse(&resp), nil
}
```

生成 server streaming：

```go
func (a *greeterConnectAdapter) ListConnect(ctx context.Context, req *connect.Request[HelloRequest], stream *connect.ServerStream[HelloReply]) error {
	data, err := proto.Marshal(req.Msg)
	if err != nil {
		return fmt.Errorf("rpccgo: connect stream request protobuf marshal failed: %w", err)
	}
	session, err := NewGreeterCGOMessageClientBridge().StartList(ctx, data)
	if err != nil {
		return err
	}
	defer session.Cancel(ctx)
	for {
		respData, err := session.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return session.Done(ctx)
			}
			return err
		}
		var resp HelloReply
		if err := proto.Unmarshal(respData, &resp); err != nil {
			return fmt.Errorf("rpccgo: connect stream response protobuf unmarshal failed: %w", err)
		}
		if err := stream.Send(&resp); err != nil {
			return err
		}
	}
}
```

对 cgo message server，C callback 在最后一个响应后返回 `GreeterCGOMessageStreamEOFErrorID()`；对 Go native server，`Recv` 返回 `io.EOF`。Connect adapter 只识别 `io.EOF` 作为正常结束，并在结束时调用 `Done`。

生成 bidi streaming：

```go
func (a *greeterConnectAdapter) ChatConnect(ctx context.Context, stream *connect.BidiStream[HelloRequest, HelloReply]) error {
	session, err := NewGreeterCGOMessageClientBridge().StartChat(ctx)
	if err != nil {
		return err
	}
	defer session.Cancel(ctx)
	for {
		var req HelloRequest
		if err := stream.Receive(&req); err != nil {
			if errors.Is(err, io.EOF) {
				if err := session.CloseSend(ctx); err != nil {
					return err
				}
				break
			}
			return err
		}
		data, err := proto.Marshal(&req)
		if err != nil {
			return fmt.Errorf("rpccgo: connect bidi request protobuf marshal failed: %w", err)
		}
		if err := session.Send(ctx, data); err != nil {
			return err
		}
		respData, err := session.Recv(ctx)
		if err != nil {
			return err
		}
		var resp HelloReply
		if err := proto.Unmarshal(respData, &resp); err != nil {
			return fmt.Errorf("rpccgo: connect bidi response protobuf unmarshal failed: %w", err)
		}
		if err := stream.Send(&resp); err != nil {
			return err
		}
	}
	return session.Done(ctx)
}
```

实现时如果 Connect 的 current API 对 bidi receive 使用 `stream.Receive()` + `stream.Msg()`，按当前依赖版本调整测试期望；adapter 行为保持不变。

- [x] **Step 6: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestRenderConnectServerFileEmitsHandlers -count=1
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
rtk git add internal/generator/render_connect_server.go internal/generator/render_connect_server_test.go
rtk git commit -m "feat: generate connect local server adapter"
```

## Task 5：生成 gRPC unary 与 streaming server adapter

**Files:**

- Modify: `internal/generator/render_grpc_server.go`
- Create: `internal/generator/render_grpc_server_test.go`

- [x] **Step 1: 写失败测试**

在 `internal/generator/render_grpc_server_test.go` 添加：

```go
package generator

import "testing"

func TestRenderGRPCServerFileEmitsServiceDesc(t *testing.T) {
	file := simpleTestFile()
	setSimpleServiceComment(t, file, "@rpccgo: msg-grpc|native\n")
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const grpcFile = "test/v1/greeter.greeter.server.grpc.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`codes "google.golang.org/grpc/codes"`,
		`status "google.golang.org/grpc/status"`,
		"func RegisterGreeterGRPCServer(registrar grpc.ServiceRegistrar) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {",
		"registrar.RegisterService(&GreeterGRPCServiceDesc, adapter)",
		"var GreeterGRPCServiceDesc = grpc.ServiceDesc{",
		`ServiceName: "test.v1.Greeter"`,
		"Methods: []grpc.MethodDesc{",
		"Streams: []grpc.StreamDesc{",
		"ClientStreams: true",
		"ServerStreams: true",
	} {
		assertGeneratedContentContains(t, plugin, grpcFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, grpcFile, "connectrpc.com/connect", ".remote.", "panic(")
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderGRPCServerFileEmitsServiceDesc -count=1
```

Expected: FAIL，当前 gRPC 文件只有 marker。

- [x] **Step 3: 实现 gRPC adapter 与注册函数**

生成：

```go
type greeterGRPCAdapter struct{}

func RegisterGreeterGRPCServer(registrar grpc.ServiceRegistrar) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error) {
	if registrar == nil {
		return rpcruntime.AdapterSnapshot[GreeterMessageAdapter]{}, errors.New("rpccgo: Greeter grpc registrar is nil")
	}
	adapter := &greeterGRPCAdapter{}
	registrar.RegisterService(&GreeterGRPCServiceDesc, adapter)
	return RegisterGreeterCGOMessageActiveServer(rpcruntime.ServerKindGRPCServer, adapter)
}
```

- [x] **Step 4: 生成 `grpc.ServiceDesc`**

按 method streaming kind 生成：

```go
var GreeterGRPCServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.v1.Greeter",
	HandlerType: (*greeterGRPCAdapter)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "Unary", Handler: greeterUnaryGRPCHandler},
	},
	Streams: []grpc.StreamDesc{
		{StreamName: "Upload", Handler: greeterUploadGRPCHandler, ClientStreams: true},
		{StreamName: "List", Handler: greeterListGRPCHandler, ServerStreams: true},
		{StreamName: "Chat", Handler: greeterChatGRPCHandler, ClientStreams: true, ServerStreams: true},
	},
	Metadata: "test/v1/message_direct.proto",
}
```

如果 grpc-go 对 `HandlerType` 要求接口类型，把它生成为：

```go
type GreeterGRPCServer interface{}
```

并将 `HandlerType` 设置为 `(*GreeterGRPCServer)(nil)`。用 integration 编译测试决定最终形态。

- [x] **Step 5: 实现 unary handler**

生成：

```go
func greeterUnaryGRPCHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	var req HelloRequest
	if err := dec(&req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc request decode failed: %v", err)
	}
	handler := func(ctx context.Context, request any) (any, error) {
		typed, ok := request.(*HelloRequest)
		if !ok || typed == nil {
			return nil, status.Error(codes.InvalidArgument, "rpccgo: grpc request type mismatch")
		}
		return srv.(*greeterGRPCAdapter).UnaryGRPC(ctx, typed)
	}
	if interceptor == nil {
		return handler(ctx, &req)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/test.v1.Greeter/Unary"}
	return interceptor(ctx, &req, info, handler)
}
```

并生成 adapter method：

```go
func (a *greeterGRPCAdapter) UnaryGRPC(ctx context.Context, req *HelloRequest) (*HelloReply, error) {
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc request protobuf marshal failed: %v", err)
	}
	respData, err := NewGreeterCGOMessageClientBridge().Unary(ctx, data)
	if err != nil {
		return nil, err
	}
	var resp HelloReply
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc response protobuf unmarshal failed: %v", err)
	}
	return &resp, nil
}
```

- [x] **Step 6: 实现 streaming handler**

为 client streaming 生成 `RecvMsg` loop，调用 message client stream `Send`，最后 `Finish` 并 `SendMsg`：

```go
func greeterUploadGRPCHandler(srv any, stream grpc.ServerStream) error {
	session, err := NewGreeterCGOMessageClientBridge().StartUpload(stream.Context())
	if err != nil {
		return err
	}
	defer session.Cancel(stream.Context())
	for {
		var req HelloRequest
		if err := stream.RecvMsg(&req); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		data, err := proto.Marshal(&req)
		if err != nil {
			return status.Errorf(codes.Internal, "rpccgo: grpc stream request protobuf marshal failed: %v", err)
		}
		if err := session.Send(stream.Context(), data); err != nil {
			return err
		}
	}
	respData, err := session.Finish(stream.Context())
	if err != nil {
		return err
	}
	var resp HelloReply
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return status.Errorf(codes.Internal, "rpccgo: grpc stream response protobuf unmarshal failed: %v", err)
	}
	return stream.SendMsg(&resp)
}
```

server streaming 先 `RecvMsg` 一个 request，再循环 `session.Recv` + `stream.SendMsg`。如果 `session.Recv` 返回 `io.EOF`，调用 `session.Done` 并返回 nil；其他 error 直接返回。

bidi streaming 先 `StartChat`，循环 `RecvMsg`、`session.Send`、`session.Recv`、`stream.SendMsg`。收到客户端 `io.EOF` 后调用 `session.CloseSend`；随后继续读取 `session.Recv`，直到服务端返回 `io.EOF`，再调用 `session.Done`。

- [x] **Step 7: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestRenderGRPCServerFileEmitsServiceDesc -count=1
```

Expected: PASS。

- [x] **Step 8: 提交**

```bash
rtk git add internal/generator/render_grpc_server.go internal/generator/render_grpc_server_test.go
rtk git commit -m "feat: generate grpc local server adapter"
```

## Task 6：Connect integration 覆盖本地 handler 到 dispatcher

**Files:**

- Create: `internal/integration/connect_local_adapter_test.go`

- [x] **Step 1: 写失败集成测试**

创建 `internal/integration/connect_local_adapter_test.go`，复用 `newMessageDirectPathTestPlugin` 生成临时模块。测试源内包含：

```go
func TestConnectUnaryRoutesToCGOMessageServer(t *testing.T) {
	registerMessageServer(t)
	path, handler := v1.NewGreeterConnectHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](server.Client(), server.URL+path+"Unary")
	_, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		t.Fatalf("CallUnary() error = %v", err)
	}
	if got := greeterMessageUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("message unary calls = %d, want 1", got)
	}
}
```

并添加：

```go
func TestConnectClientStreamingRoutesToCGOMessageServer(t *testing.T)
func TestConnectServerStreamingRoutesToCGOMessageServer(t *testing.T)
func TestConnectBidiStreamingRoutesToCGOMessageServer(t *testing.T)
func TestConnectUnaryRoutesThroughConverterToGoNativeServer(t *testing.T)
func TestConnectClientStreamingStartCapturesSnapshot(t *testing.T)
```

每个测试都使用 generated Connect handler 作为唯一 HTTP 入口，断言对应 cgo message/native callback 计数。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/integration -run TestConnectLocalAdapter -count=1
```

Expected: FAIL，临时模块编译或运行失败，暴露 Connect renderer 不完整点。

- [x] **Step 3: 修复 Connect renderer 到测试通过**

按失败信息修复：

- 如果 `NewGreeterConnectHandler` 返回 path 不适合 `httptest.Server`，改为返回 service prefix 和 mux，测试使用 `server.URL + GreeterUnaryConnectProcedure`。
- 如果 Connect streaming API 签名不匹配，按当前 `connectrpc.com/connect` 版本改 renderer 与测试。
- 如果 error 不能被 Connect 正确编码，使用原始 error 返回；不要把 error 存进 `rpcruntime.ErrorID`。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/integration -run TestConnectLocalAdapter -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/integration/connect_local_adapter_test.go internal/generator/render_connect_server.go
rtk git commit -m "test: cover connect local adapter integration"
```

## Task 7：gRPC integration 覆盖本地 server adapter 到 dispatcher

**Files:**

- Create: `internal/integration/grpc_local_adapter_test.go`

- [x] **Step 1: 写失败集成测试**

创建 `internal/integration/grpc_local_adapter_test.go`，临时模块使用 `bufconn` 启动 `grpc.Server`：

```go
func TestGRPCUnaryRoutesToCGOMessageServer(t *testing.T) {
	registerMessageServer(t)

	server := grpc.NewServer()
	if _, err := v1.RegisterGreeterGRPCServer(server); err != nil {
		t.Fatalf("RegisterGreeterGRPCServer() error = %v", err)
	}
	conn := newBufconnClientConn(t, server)
	defer conn.Close()

	var resp emptypb.Empty
	if err := conn.Invoke(context.Background(), "/test.v1.Greeter/Unary", &emptypb.Empty{}, &resp); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got := greeterMessageUnaryCallsForIntegration(); got != 1 {
		t.Fatalf("message unary calls = %d, want 1", got)
	}
}
```

并添加：

```go
func TestGRPCClientStreamingRoutesToCGOMessageServer(t *testing.T)
func TestGRPCServerStreamingRoutesToCGOMessageServer(t *testing.T)
func TestGRPCBidiStreamingRoutesToCGOMessageServer(t *testing.T)
func TestGRPCUnaryRoutesThroughConverterToGoNativeServer(t *testing.T)
func TestGRPCClientStreamingStartCapturesSnapshot(t *testing.T)
```

streaming 测试使用 `conn.NewStream` 和 generated `grpc.StreamDesc` 信息发收 protobuf message。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/integration -run TestGRPCLocalAdapter -count=1
```

Expected: FAIL，临时模块暴露 gRPC `ServiceDesc` 或 handler 签名问题。

- [x] **Step 3: 修复 gRPC renderer 到测试通过**

按失败信息修复：

- `HandlerType` 必须满足 grpc-go 注册检查。
- `FullMethod` 使用 `"/" + service.FullName + "/" + method.Name`。
- handler 返回普通 error 时允许 grpc-go 映射为 `codes.Unknown`；生成器内部 marshal/unmarshal 错误使用 `status.Errorf(codes.Internal, ...)` 或 `codes.InvalidArgument`。
- stream handler 必须在 terminal 路径调用 `Done` 或 `CloseSend`，保持 Stage 2 stream lifecycle。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/integration -run TestGRPCLocalAdapter -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/integration/grpc_local_adapter_test.go internal/generator/render_grpc_server.go
rtk git commit -m "test: cover grpc local adapter integration"
```

## Task 8：阶段 5 acceptance 与回归收口

**Files:**

- Create: `internal/integration/local_transport_stage5_acceptance_test.go`
- Modify: `internal/integration/converter_stage4b_acceptance_test.go`
- Modify: `internal/generator/render_connect_server_test.go`
- Modify: `internal/generator/render_grpc_server_test.go`

- [x] **Step 1: 写阶段 5 acceptance**

创建 `internal/integration/local_transport_stage5_acceptance_test.go`：

```go
package integration

import "testing"

func TestStage5LocalTransportAcceptance(t *testing.T) {
	t.Run("connect unary and streaming enter dispatcher", func(t *testing.T) {
		runConnectLocalAdapterFixture(t, "TestConnectUnaryRoutesToCGOMessageServer")
		runConnectLocalAdapterFixture(t, "TestConnectClientStreamingRoutesToCGOMessageServer")
		runConnectLocalAdapterFixture(t, "TestConnectServerStreamingRoutesToCGOMessageServer")
		runConnectLocalAdapterFixture(t, "TestConnectBidiStreamingRoutesToCGOMessageServer")
	})
	t.Run("grpc unary and streaming enter dispatcher", func(t *testing.T) {
		runGRPCLocalAdapterFixture(t, "TestGRPCUnaryRoutesToCGOMessageServer")
		runGRPCLocalAdapterFixture(t, "TestGRPCClientStreamingRoutesToCGOMessageServer")
		runGRPCLocalAdapterFixture(t, "TestGRPCServerStreamingRoutesToCGOMessageServer")
		runGRPCLocalAdapterFixture(t, "TestGRPCBidiStreamingRoutesToCGOMessageServer")
	})
	t.Run("local transports reuse converter for native active server", func(t *testing.T) {
		runConnectLocalAdapterFixture(t, "TestConnectUnaryRoutesThroughConverterToGoNativeServer")
		runGRPCLocalAdapterFixture(t, "TestGRPCUnaryRoutesThroughConverterToGoNativeServer")
	})
}
```

- [x] **Step 2: 删除 Stage 4B fixture 的 pending 覆盖日志或转成测试**

在 `internal/integration/converter_stage4b_acceptance_test.go` 删除：

```go
t.Log("pending broader payload coverage: add non-empty scalar/string/bytes/repeated fixture messages")
```

如果本阶段没有补非空 payload converter 测试，把该风险写入 Stage 8 文档而不是保留在 acceptance 测试中。

- [x] **Step 3: 加强 renderer 禁止 remote 断言**

在 Connect/gRPC renderer tests 中加入：

```go
assertGeneratedContentDoesNotContain(t, plugin, ".remote.", "ConnectRemote", "GRPCRemote")
```

- [x] **Step 4: 运行阶段 5 focused 测试**

Run:

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -run 'TestRenderConnect|TestRenderGRPC|TestRenderStageFilesEmitsLocalTransport|TestRenderMessageFileFamilyPlan' -count=1
rtk go test ./internal/integration -run 'TestStage5|TestConnectLocalAdapter|TestGRPCLocalAdapter' -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/integration/local_transport_stage5_acceptance_test.go internal/integration/converter_stage4b_acceptance_test.go internal/generator/render_connect_server_test.go internal/generator/render_grpc_server_test.go
rtk git commit -m "test: verify stage 5 local transport adapters"
```

## Task 9：迁移清单与全仓验证

**Files:**

- Create: `docs/plans/2026-05-06-stage-5-migration-inventory.md`
- Modify: `docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md`

- [x] **Step 1: 写迁移清单**

创建 `docs/plans/2026-05-06-stage-5-migration-inventory.md`：

```markdown
# Stage 5 Migration Inventory

阶段 5 实现 Connect 与 gRPC 本地 server adapter。它只让标准 RPC 入站请求进入 generated dispatcher，不实现 remote adapter，不改变 connect/grpc 标准 client 语义。

## 迁移或参考

| 旧项目文件或模块 | 新版落点 | 旧代码作用 | Stage 5 处理方式 | 为什么迁移或参考 |
| --- | --- | --- | --- | --- |
| 旧 message server connect/grpc handler 适配逻辑 | `internal/generator/render_connect_server.go`、`internal/generator/render_grpc_server.go` | 把 RPC transport 入站消息交给 message handler | 参考后重写 | transport handler 的请求/响应顺序有价值，但旧代码绑定旧 bootstrap 和 framework selector |
| 旧 connect/grpc integration fixture | `internal/integration/connect_local_adapter_test.go`、`internal/integration/grpc_local_adapter_test.go` | 验证标准 RPC 入站路径 | 迁移测试思路 | 本阶段仍需要证明 unary 和 streaming 通过 dispatcher |

## 不迁移

| 旧项目文件或模块 | 旧代码作用 | Stage 5 处理方式 | 为什么不迁移 |
| --- | --- | --- | --- |
| framework selector | 在 connect/grpc/native 之间选择生成模式 | 不迁移 | 新版由 `@rpccgo` token 和单插件多 renderer 控制 |
| 多 provider bootstrap | 注册多个 provider 并启动 | 不迁移 | 新版每次运行只有一个监听 server，每个 generated service 同一时刻只有一个 active server |
| connect/grpc remote adapter | 远端调用 | 不迁移 | remote adapter 是 Stage 6 范围 |
| connect/grpc client 生成 | 标准 RPC client | 不迁移 | connect/grpc client 不属于 rpccgo client 类型模型 |

## 验证结果

- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`：待执行。
- `rtk go test ./internal/integration -count=1`：待执行。
- `rtk go test ./rpcruntime -count=1`：待执行。
- `rtk go test ./... -count=1`：待执行。
- forbidden unsigned scan：待执行。
```

- [x] **Step 2: 全仓验证**

Run:

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1
rtk go test ./internal/integration -count=1
rtk go test ./rpcruntime -count=1
rtk go test ./... -count=1
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md'
```

Expected:

- 所有 `go test` PASS。
- forbidden unsigned scan 无输出，`rg` exit code 为 1。

- [x] **Step 3: 更新计划与迁移清单验证结果**

将本计划所有已完成任务 checkbox 更新为 `[x]`，并把 `docs/plans/2026-05-06-stage-5-migration-inventory.md` 的 “待执行” 改为实际结果。

- [x] **Step 4: 提交**

```bash
rtk git add docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md docs/plans/2026-05-06-stage-5-migration-inventory.md
rtk git commit -m "docs: record stage 5 local transport adapters"
```

## 阶段 5 完成标准

- `@rpccgo:msg-connect` 生成 `<service>.server.connect.rpccgo.go`。
- `@rpccgo:msg-grpc` 生成 `<service>.server.grpc.rpccgo.go`。
- `@rpccgo:native` 继续展开为 `msg-connect|native`，因此生成 Connect 本地 adapter 与 native/message converter。
- Connect handler 是标准 `http.Handler`，内部使用 `connectrpc.com/connect` handler API。
- gRPC server adapter 使用标准 `grpc.ServiceRegistrar.RegisterService` 注册 `grpc.ServiceDesc`。
- Connect/gRPC 入站 unary、client streaming、server streaming、bidi streaming 都进入 dispatcher。
- Connect/gRPC 入站请求能路由到 cgo message server。
- Connect/gRPC 入站请求在 active server 为 Go native 或 cgo native 时复用 Stage 4B converter。
- stream Start 捕获 active server snapshot，后续 Send/Recv/Finish/Done/CloseSend/Cancel 固定路由到该 snapshot。
- 阶段 5 不生成 remote adapter，不生成 connect/grpc client，不引入旧 bootstrap 或 framework selector。
- `rpcruntime` 不引入 protobuf、connect、grpc 或 `internal/generator` 依赖。
- 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无命中。

## 阶段 5 后续风险

- Connect 与 gRPC streaming API 的具体 helper 签名可能随依赖版本变化；实现时以当前 `go.mod` 锁定版本和编译测试为准。
- Stage 4B converter 的复杂 payload 端到端覆盖仍可加强。阶段 5 只要求 transport adapter 正确复用 converter；非空 scalar/string/bytes/repeated 的更广覆盖可以作为 Stage 8 清理项，或者在阶段 5 执行中发现 transport 与 converter 交互风险时提前补入。
- gRPC handler error code 需要在 integration 中确认。业务 server 返回普通 error 时保持 grpc-go 默认映射；generated marshal/unmarshal 和请求类型错误使用明确 `status` error。
