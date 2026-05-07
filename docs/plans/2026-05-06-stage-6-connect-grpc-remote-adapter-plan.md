# 阶段 6 Connect 与 gRPC Remote Server Adapter 实施计划

> **给 agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` 或 `superpowers:executing-plans` 按任务逐项执行。步骤使用 checkbox (`- [x]`) 语法跟踪。

**目标:** 为 generated service 生成 Connect 与 gRPC remote server adapter，让当前 active server 可以指向远端标准 RPC 服务，并继续通过现有 dispatcher 与 Stage 4B converter 服务 native/message mismatch。

**架构:** 阶段 6 只实现 remote server adapter：`<service>.remote.connect.rpccgo.go` 内部持有标准 `connect.Client`，`<service>.remote.grpc.rpccgo.go` 内部持有标准 `grpc.ClientConnInterface`。两类 adapter 都注册为 message contract active server，不新增 rpccgo client 类型，不引入 framework selector、多 registry 或旧 bootstrap。

**技术栈:** Go 1.24、`connectrpc.com/connect`、`google.golang.org/grpc`、`google.golang.org/protobuf/proto`、现有 `rpcruntime.Dispatcher`、generated message adapter interface 与 Stage 4B converter。

---

## 范围

阶段 6 实现：

- Connect remote server adapter。
- gRPC remote server adapter。
- remote unary、client streaming、server streaming、bidi streaming。
- remote adapter 注册到 active server slot，contract 为 `message`。
- cgo message client 调 remote server 的 message direct path。
- cgo native client 调 remote server 时复用 Stage 4B converter。
- generator focused、integration focused、runtime focused、全仓测试和 forbidden unsigned scan。
- 阶段 6 迁移清单。

阶段 6 不实现：

- connect/grpc 标准 client 生成物。
- 自定义 connect/grpc client 模型。
- connect/grpc local handler adapter 的行为改造。
- 旧项目 framework selector、多 provider registry、多 bootstrap。
- remote 负载均衡、重试、服务发现或连接生命周期托管。
- `rpcruntime` 中的 protobuf、Connect、gRPC 或 service-specific 依赖。

## 设计边界

- remote adapter 是 server adapter，不是 rpccgo client。
- remote adapter 的输入输出 contract 是 protobuf message bytes。
- remote adapter 文件留在 protobuf Go package，不能生成到 `cgo_dir` 或 `package main`。
- `@rpccgo:msg-connect` 生成 Connect local adapter 与 Connect remote adapter。
- `@rpccgo:msg-grpc` 生成 gRPC local adapter 与 gRPC remote adapter。
- `@rpccgo:native` 继续展开为 `msg-connect|native`，因此会生成 Connect remote adapter，并让 native client 通过 converter 调 remote。
- remote adapter 内部只复用标准 Connect/gRPC client API，不暴露成 rpccgo client 类型。
- stream `Start` 时由 dispatcher 捕获 remote adapter snapshot，后续 `Send`、`Recv`、`Finish`、`Done`、`CloseSend`、`Cancel` 固定使用该 snapshot。

## 外部 API 依据

- Connect 当前依赖版本为 `connectrpc.com/connect v1.19.1`。`connect.Client` 提供 `CallUnary`、`CallClientStream`、`CallServerStream`、`CallBidiStream`；stream client 暴露 `Send`、`CloseAndReceive`、`Receive`、`Msg`、`Err` 等方法。
- gRPC 当前依赖版本为 `google.golang.org/grpc v1.79.3`。remote adapter 使用 `grpc.ClientConnInterface.Invoke` 和 `NewStream`，stream client 使用 `grpc.GenericClientStream` 的 `Send`、`Recv`、`CloseAndRecv`。

## 文件结构

- Modify: `internal/generator/plan.go`  
  扩展 `MessageFileFamilyPlan`，加入 `ConnectRemote` 和 `GRPCRemote`。
- Modify: `internal/generator/render_message_plan.go`  
  根据 `@rpccgo` token 规划 `.remote.connect.rpccgo.go` 与 `.remote.grpc.rpccgo.go`。
- Modify: `internal/generator/render.go`  
  将 remote renderer 接入 `RenderMessageStageFiles` 与 `RenderStageFiles`。
- Create: `internal/generator/render_connect_remote.go`  
  生成 Connect remote server adapter。
- Create: `internal/generator/render_grpc_remote.go`  
  生成 gRPC remote server adapter。
- Create: `internal/generator/render_connect_remote_test.go`  
  覆盖 Connect remote 文件规划、生成内容、token gating、无 rpccgo client 模型。
- Create: `internal/generator/render_grpc_remote_test.go`  
  覆盖 gRPC remote 文件规划、生成内容、token gating、无 rpccgo client 模型。
- Modify: `internal/generator/generator_test.go`、`internal/generator/render_message_plan_test.go`、`internal/generator/render_codec_test.go`  
  更新 generated filename 与 Stage 5 “不生成 remote”断言。
- Create: `internal/integration/remote_transport_stage6_acceptance_test.go`  
  生成临时模块，用 Stage 5 local adapter 起远端 Connect/gRPC 服务，再注册 Stage 6 remote adapter，覆盖 unary 与三类 streaming。
- Create: `docs/plans/2026-05-06-stage-6-migration-inventory.md`  
  记录迁移、参考和不迁移范围。
- Modify: `docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md`  
  执行时更新 checkbox 与验证结果。

## Task 1：规划 remote 文件族

**Files:**

- Modify: `internal/generator/plan.go`
- Modify: `internal/generator/render_message_plan.go`
- Modify: `internal/generator/render_message_plan_test.go`

**迁移内容与理由:** 参考旧 `forwarding_plan.go` 的“把 forwarding/remote 作为独立生成文件族”的测试思路，不迁移旧 planner 代码。新版已有 `MessageFileFamilyPlan`，直接扩展它更符合单插件 parser/planner/renderer 架构。

- [x] **Step 1: 写失败测试**

在 `internal/generator/render_message_plan_test.go` 增加：

```go
func TestRenderMessageFileFamilyPlanIncludesRemoteTransportAdapters(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}
	service := ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC}},
	}

	got := BuildMessageFileFamilyPlan(file, service)

	assertGeneratedFilePlan(t, got.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", true)
	assertGeneratedFilePlan(t, got.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", true)
}

func TestRenderMessageFileFamilyPlanGatesRemoteTransportAdaptersByToken(t *testing.T) {
	file := FilePlan{GeneratedFilenamePrefix: "test/v1/greeter"}

	connectOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}},
	})
	assertGeneratedFilePlan(t, connectOnly.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", true)
	assertGeneratedFilePlan(t, connectOnly.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", false)

	grpcOnly := BuildMessageFileFamilyPlan(file, ServicePlan{
		GoName:   "Greeter",
		Adapters: AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageGRPC}},
	})
	assertGeneratedFilePlan(t, grpcOnly.ConnectRemote, "test/v1/greeter.greeter.remote.connect.rpccgo.go", false)
	assertGeneratedFilePlan(t, grpcOnly.GRPCRemote, "test/v1/greeter.greeter.remote.grpc.rpccgo.go", true)
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessageFileFamilyPlanIncludesRemoteTransportAdapters|TestRenderMessageFileFamilyPlanGatesRemoteTransportAdaptersByToken' -count=1
```

Expected: 编译失败，提示 `MessageFileFamilyPlan` 没有 `ConnectRemote` / `GRPCRemote` 字段。

- [x] **Step 3: 实现文件族计划**

在 `internal/generator/plan.go` 中把 `MessageFileFamilyPlan` 扩展为：

```go
type MessageFileFamilyPlan struct {
	Runtime          GeneratedFilePlan
	CGOMessageServer GeneratedFilePlan
	CGOMessageClient GeneratedFilePlan
	ConnectServer    GeneratedFilePlan
	GRPCServer       GeneratedFilePlan
	ConnectRemote    GeneratedFilePlan
	GRPCRemote       GeneratedFilePlan
}
```

在 `internal/generator/render_message_plan.go` 的 `BuildMessageFileFamilyPlan` 中加入：

```go
ConnectRemote: GeneratedFilePlan{
	Filename: fmt.Sprintf("%s.%s.remote.connect.rpccgo.go", prefix, serviceName),
	Enabled:  service.Adapters.Has(AdapterTokenMessageConnect),
},
GRPCRemote: GeneratedFilePlan{
	Filename: fmt.Sprintf("%s.%s.remote.grpc.rpccgo.go", prefix, serviceName),
	Enabled:  service.Adapters.Has(AdapterTokenMessageGRPC),
},
```

同步更新 `assertMessageFileFamilyDoesNotUseAdapterOrCodecFiles` 的文件列表，纳入 `got.ConnectRemote` 与 `got.GRPCRemote`。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessageFileFamilyPlan' -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/generator/plan.go internal/generator/render_message_plan.go internal/generator/render_message_plan_test.go
rtk git commit -m "feat: plan remote transport adapter files"
```

## Task 2：接入 remote renderer pipeline

**Files:**

- Modify: `internal/generator/render.go`
- Modify: `internal/generator/generator_test.go`
- Modify: `internal/generator/render_codec_test.go`

**迁移内容与理由:** 不迁移旧 `render_plan.go`，只沿用它“生成计划决定文件开关，renderer 只负责输出”的分层思路。新版 `RenderStageFiles` 已经解决 native/message 文件族合并问题，继续在这里接入 remote 文件能避免同名冲突和旧 bootstrap 回流。

- [x] **Step 1: 写失败测试**

在 `internal/generator/generator_test.go` 添加：

```go
func TestRenderStageFilesEmitsRemoteTransportAdaptersByServiceToken(t *testing.T) {
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
		"test/v1/greeter.greeter.remote.connect.rpccgo.go",
		"test/v1/greeter.greeter.remote.grpc.rpccgo.go",
	})
}
```

将 `TestRenderStageFilesEmitsCodecWithoutRemoteAdapterFiles` 重命名或调整为只覆盖没有 `msg-connect` / `msg-grpc` token 的服务，确保它不再否定 Stage 6 文件。

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderStageFilesEmitsRemoteTransportAdaptersByServiceToken|TestRenderStageFilesEmitsCodecWithoutRemoteAdapterFiles' -count=1
```

Expected: 第一个测试失败，generated filenames 缺少 `.remote.connect.rpccgo.go` 与 `.remote.grpc.rpccgo.go`。

- [x] **Step 3: 接入 renderer 分发**

在 `RenderMessageStageFiles` 的 `files` 列表中加入：

```go
family.ConnectRemote,
family.GRPCRemote,
```

并增加分发：

```go
if file == family.ConnectRemote {
	if err := renderConnectRemoteFile(plugin, plan, service, file); err != nil {
		return err
	}
	continue
}
if file == family.GRPCRemote {
	if err := renderGRPCRemoteFile(plugin, plan, service, file); err != nil {
		return err
	}
	continue
}
```

在 `RenderStageFiles` 中对 remote 文件调用 `markRendered`：

```go
markRendered(rendered, messageService.MessageFileFamily.ConnectRemote)
markRendered(rendered, messageService.MessageFileFamily.GRPCRemote)
```

在 `messageStageMarker` 中增加 remote marker：

```go
case strings.Contains(name, ".remote.connect.rpccgo.go"):
	return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "connect remote server adapter"}, " ")
case strings.Contains(name, ".remote.grpc.rpccgo.go"):
	return strings.Join([]string{"rpccgo message direct stage file for", service.GoName, "grpc remote server adapter"}, " ")
```

先创建最小 renderer 占位实现，保证 pipeline 编译：

```go
func renderConnectRemoteFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// ", messageStageMarker(service, file))
	return nil
}

func renderGRPCRemoteFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedFilePlan) error {
	g := plugin.NewGeneratedFile(file.Filename, protogen.GoImportPath(plan.GoImportPath))
	g.P("package ", plan.GoPackageName)
	g.P()
	g.P("// ", messageStageMarker(service, file))
	return nil
}
```

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderStageFilesEmitsRemoteTransportAdaptersByServiceToken|TestRenderStageFilesEmitsCodecWithoutRemoteAdapterFiles' -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/generator/render.go internal/generator/generator_test.go internal/generator/render_codec_test.go internal/generator/render_connect_remote.go internal/generator/render_grpc_remote.go
rtk git commit -m "feat: route remote transport adapter rendering"
```

## Task 3：生成 Connect remote server adapter

**Files:**

- Modify: `internal/generator/render_connect_remote.go`
- Create: `internal/generator/render_connect_remote_test.go`

**迁移内容与理由:** 参考旧 `native_forwarding_client.go` 的“远端调用被包装成本地 adapter 方法”思路，以及旧 native forwarding integration 的 streaming 顺序覆盖；不迁移旧代码，因为旧实现围绕 native forwarding、provider registry 和旧生成命名，和新版 message contract remote adapter 不匹配。

- [x] **Step 1: 写失败测试**

创建 `internal/generator/render_connect_remote_test.go`：

```go
package generator

import "testing"

func TestRenderConnectRemoteFileEmitsMessageAdapter(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const remoteFile = "test/v1/stage1_acceptance.all_service.remote.connect.rpccgo.go"
	for _, fragment := range []string{
		`connect "connectrpc.com/connect"`,
		`http "net/http"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceConnectRemoteServer struct {",
		"func NewAllServiceConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (*AllServiceConnectRemoteServer, error) {",
		"func RegisterAllServiceConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)",
		"func (s *AllServiceConnectRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"resp, err := s.unary.CallUnary(ctx, connect.NewRequest(request))",
		"func (s *AllServiceConnectRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"stream := s.clientStream.CallClientStream(ctx)",
		"func (s *AllServiceConnectRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.serverStream.CallServerStream(ctx, connect.NewRequest(request))",
		"func (s *AllServiceConnectRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream := s.bidiStream.CallBidiStream(ctx)",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, "grpc \"google.golang.org/grpc\"", "panic(", "ClientModel")
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderConnectRemoteFileEmitsMessageAdapter -count=1
```

Expected: FAIL，当前 remote renderer 只有 marker，没有 Connect client 字段和方法。

- [x] **Step 3: 实现 Connect remote adapter 生成**

`render_connect_remote.go` 生成以下结构：

```go
type <Service>ConnectRemoteServer struct {
	unary *connect.Client[<Req>, <Resp>]
	clientStream *connect.Client[<Req>, <Resp>]
	serverStream *connect.Client[<Req>, <Resp>]
	bidiStream *connect.Client[<Req>, <Resp>]
}
```

构造函数：

```go
func New<Service>ConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (*<Service>ConnectRemoteServer, error) {
	if httpClient == nil {
		return nil, errors.New("rpccgo: connect remote http client is nil")
	}
	if baseURL == "" {
		return nil, errors.New("rpccgo: connect remote base URL is empty")
	}
	return &<Service>ConnectRemoteServer{
		unary: connect.NewClient[Req, Resp](httpClient, strings.TrimRight(baseURL, "/")+<Service><Method>ConnectProcedure, options...),
	}, nil
}
```

注册函数：

```go
func Register<Service>ConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[<Service>MessageAdapter], error) {
	adapter, err := New<Service>ConnectRemoteServer(httpClient, baseURL, options...)
	if err != nil {
		return rpcruntime.AdapterSnapshot[<Service>MessageAdapter]{}, err
	}
	return Register<Service>CGOMessageActiveServer(rpcruntime.ServerKindConnectRemote, adapter)
}
```

unary 方法生成：

```go
func (s *<Service>ConnectRemoteServer) <Method>Message(ctx context.Context, req []byte) ([]byte, error) {
	if s == nil || s.<methodClient> == nil {
		return nil, errors.New("rpccgo: connect remote server is nil")
	}
	request := new(<Req>)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, fmt.Errorf("rpccgo: connect remote request protobuf unmarshal failed: %w", err)
	}
	resp, err := s.<methodClient>.CallUnary(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	return proto.Marshal(resp.Msg)
}
```

stream session 生成规则：

- client streaming session 持有 `*connect.ClientStreamForClient[Req, Resp]`，`Send` 先 unmarshal 再 `stream.Send`，`Finish` 调 `CloseAndReceive` 后 marshal response，`Cancel` 优先使用 `stream.Conn()` 关闭连接，若不可用则返回 nil。
- server streaming session 持有 `*connect.ServerStreamForClient[Resp]`，`Recv` 在 `Receive()` 为 true 时 marshal `Msg()`；`Receive()` 为 false 时若 `Err()` 非 nil 返回该错误，否则返回 `io.EOF`；`Done` 调 `Close()`。
- bidi streaming session 持有 `*connect.BidiStreamForClient[Req, Resp]`，`Send` unmarshal 后发送，`Recv` 读取 response，`CloseSend` 调 `CloseRequest`，`Done` 在 EOF 后返回 nil，`Cancel` 优先关闭底层连接。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestRenderConnectRemoteFileEmitsMessageAdapter -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/generator/render_connect_remote.go internal/generator/render_connect_remote_test.go
rtk git commit -m "feat: generate connect remote server adapter"
```

## Task 4：生成 gRPC remote server adapter

**Files:**

- Modify: `internal/generator/render_grpc_remote.go`
- Create: `internal/generator/render_grpc_remote_test.go`

**迁移内容与理由:** 参考旧 gRPC forwarding 示例里用标准 gRPC client 进入远端 transport 的测试思路；不迁移旧 generated forwarding 文件，因为旧文件绑定 native goclient registry，新版 remote adapter 应直接实现 generated message adapter interface。

- [x] **Step 1: 写失败测试**

创建 `internal/generator/render_grpc_remote_test.go`：

```go
package generator

import "testing"

func TestRenderGRPCRemoteFileEmitsMessageAdapter(t *testing.T) {
	file := stage1AcceptanceFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)

	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	const remoteFile = "test/v1/stage1_acceptance.all_service.remote.grpc.rpccgo.go"
	for _, fragment := range []string{
		`grpc "google.golang.org/grpc"`,
		`proto "google.golang.org/protobuf/proto"`,
		"type AllServiceGRPCRemoteServer struct {",
		"conn grpc.ClientConnInterface",
		"func NewAllServiceGRPCRemoteServer(conn grpc.ClientConnInterface) (*AllServiceGRPCRemoteServer, error) {",
		"func RegisterAllServiceGRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[AllServiceMessageAdapter], error) {",
		"return RegisterAllServiceCGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)",
		"func (s *AllServiceGRPCRemoteServer) UnaryMessage(ctx context.Context, req []byte) ([]byte, error) {",
		"err := s.conn.Invoke(ctx, AllServiceUnaryGRPCFullMethodName, request, response)",
		"func (s *AllServiceGRPCRemoteServer) StartClientStreamMessage(ctx context.Context) (AllServiceClientStreamMessageStreamSession, error) {",
		"stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true}, AllServiceClientStreamGRPCFullMethodName)",
		"func (s *AllServiceGRPCRemoteServer) StartServerStreamMessage(ctx context.Context, req []byte) (AllServiceServerStreamMessageStreamSession, error) {",
		"stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, AllServiceServerStreamGRPCFullMethodName)",
		"func (s *AllServiceGRPCRemoteServer) StartBidiStreamMessage(ctx context.Context) (AllServiceBidiStreamMessageStreamSession, error) {",
		"stream, err := s.conn.NewStream(ctx, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, AllServiceBidiStreamGRPCFullMethodName)",
	} {
		assertGeneratedContentContains(t, plugin, remoteFile, fragment)
	}
	assertGeneratedFileContentDoesNotContain(t, plugin, remoteFile, "connectrpc.com/connect", "panic(", "ClientModel")
}
```

- [x] **Step 2: 运行测试确认失败**

Run:

```bash
rtk go test ./internal/generator -run TestRenderGRPCRemoteFileEmitsMessageAdapter -count=1
```

Expected: FAIL，当前 remote renderer 只有 marker，没有 gRPC client 字段和方法。

- [x] **Step 3: 实现 gRPC remote adapter 生成**

`render_grpc_remote.go` 生成：

```go
type <Service>GRPCRemoteServer struct {
	conn grpc.ClientConnInterface
}

func New<Service>GRPCRemoteServer(conn grpc.ClientConnInterface) (*<Service>GRPCRemoteServer, error) {
	if conn == nil {
		return nil, errors.New("rpccgo: grpc remote client connection is nil")
	}
	return &<Service>GRPCRemoteServer{conn: conn}, nil
}

func Register<Service>GRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[<Service>MessageAdapter], error) {
	adapter, err := New<Service>GRPCRemoteServer(conn)
	if err != nil {
		return rpcruntime.AdapterSnapshot[<Service>MessageAdapter]{}, err
	}
	return Register<Service>CGOMessageActiveServer(rpcruntime.ServerKindGRPCRemote, adapter)
}
```

unary 方法使用 `Invoke`：

```go
func (s *<Service>GRPCRemoteServer) <Method>Message(ctx context.Context, req []byte) ([]byte, error) {
	if s == nil || s.conn == nil {
		return nil, errors.New("rpccgo: grpc remote server is nil")
	}
	request := new(<Req>)
	if err := proto.Unmarshal(req, request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "rpccgo: grpc remote request protobuf unmarshal failed: %v", err)
	}
	response := new(<Resp>)
	if err := s.conn.Invoke(ctx, <Service><Method>GRPCFullMethodName, request, response); err != nil {
		return nil, err
	}
	respData, err := proto.Marshal(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rpccgo: grpc remote response protobuf marshal failed: %v", err)
	}
	return respData, nil
}
```

stream session 生成规则：

- client streaming：`NewStream(ctx, &grpc.StreamDesc{ClientStreams: true}, fullMethod)`，用 `grpc.GenericClientStream[Req, Resp]`，`Finish` 调 `CloseAndRecv`。
- server streaming：`NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, fullMethod)` 后 `Send` request，再 `CloseSend`，`Recv` marshal response，`io.EOF` 作为正常结束。
- bidi streaming：`NewStream(ctx, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, fullMethod)`，`Send`、`Recv`、`CloseSend` 映射到 `GenericClientStream`。
- `Cancel(ctx)` 对 gRPC remote session 返回 nil；取消由传入 `ctx` 控制，计划不新增自定义 cancellation channel。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestRenderGRPCRemoteFileEmitsMessageAdapter -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/generator/render_grpc_remote.go internal/generator/render_grpc_remote_test.go
rtk git commit -m "feat: generate grpc remote server adapter"
```

## Task 5：阶段 6 generated-source acceptance

**Files:**

- Create: `internal/integration/remote_transport_stage6_acceptance_test.go`

**迁移内容与理由:** 迁移旧 native forwarding integration 的验收思路：用真实 transport server、真实 generated client/adapter、真实 streaming 顺序做端到端验证。实现必须按新版 generated dispatcher 和 Stage 5 local adapter 重写，不复用旧 bootstrap、debugserver 或 provider registry。

- [x] **Step 1: 写 acceptance 测试**

创建 `internal/integration/remote_transport_stage6_acceptance_test.go`。测试结构复用 `local_transport_stage5_acceptance_test.go` 的临时模块生成方式：

```go
func TestStage6RemoteTransportAcceptance(t *testing.T) {
	tmp := t.TempDir()
	plugin := newLocalTransportTestPlugin(t, "example.com/remote/test/v1;testv1")
	if _, err := generator.GenerateWithOptions(plugin, generator.GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions() error = %v", err)
	}

	writeMessageDirectPathGeneratedModule(t, tmp, plugin, "example.com/remote")
	writeFile(t, filepath.Join(tmp, "test/v1/message_integration_reset.go"), messageDirectPathResetSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/message_direct_path_callbacks.go"), messageDirectPathFixtureCallbackSource)
	writeFile(t, filepath.Join(tmp, "test/v1/cgo/remote_transport_stage6_test.go"), remoteTransportStage6FixtureTestSource)

	cmd := exec.Command("go", "test", "./test/v1/cgo", "-run", "^TestRemoteTransportStage6Acceptance$", "-count=1")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("remote transport fixture failed: %v\n%s", err, out)
	}
}
```

fixture 覆盖以下 subtests：

```go
func TestRemoteTransportStage6Acceptance(t *testing.T) {
	t.Run("connect remote routes message client to remote cgo message server", testConnectRemoteToMessageServer)
	t.Run("grpc remote routes message client to remote cgo message server", testGRPCRemoteToMessageServer)
	t.Run("connect remote reuses converter for native client", testConnectRemoteFromNativeClient)
	t.Run("grpc remote reuses converter for native client", testGRPCRemoteFromNativeClient)
	t.Run("connect remote stream snapshot stays on original remote", testConnectRemoteSnapshot)
	t.Run("grpc remote surfaces downstream errors", testGRPCRemoteErrors)
}
```

每个 transport 至少覆盖 unary、client streaming、server streaming、bidi streaming。Connect remote 用 `httptest.Server` + `New<Service>ConnectHandler()` 起远端；gRPC remote 用 `grpc.Server` + `Register<Service>GRPCServer()` + `bufconn` 起远端。远端 active server 先注册 cgo message callbacks 或 Go native server，本地 active server 注册 Stage 6 remote adapter。

- [x] **Step 2: 运行 acceptance 确认失败**

Run:

```bash
rtk go test ./internal/integration -run TestStage6RemoteTransportAcceptance -count=1
```

Expected: FAIL。若 Task 3/4 尚未实现完整 remote adapter，失败点应为 undefined generated remote API 或 fixture remote 调用失败。

- [x] **Step 3: 补齐 fixture helper**

在 fixture 中实现：

```go
func registerConnectRemoteServer(t *testing.T, httpClient connect.HTTPClient, baseURL string) {
	t.Helper()
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterConnectRemoteServer(httpClient, baseURL); err != nil {
		t.Fatalf("RegisterGreeterConnectRemoteServer() error = %v", err)
	}
}

func registerGRPCRemoteServer(t *testing.T, conn grpc.ClientConnInterface) {
	t.Helper()
	v1.ResetGreeterDispatcherForIntegrationTest()
	if _, err := v1.RegisterGreeterGRPCRemoteServer(conn); err != nil {
		t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
	}
}
```

远端服务启动 helper 必须和本地 active server reset 分开，避免远端和本地 dispatcher 状态互相覆盖；如果同一 generated package 的全局 dispatcher 让单进程 fixture 无法表达“双进程远端”，则用两个临时 Go test 进程或两个不同 Go package 生成同一 proto 的方式隔离远端/本地 dispatcher。优先选择两个不同 Go package 的 generated module fixture，因为它最贴近 remote process 语义。

- [x] **Step 4: 运行 acceptance 确认通过**

Run:

```bash
rtk go test ./internal/integration -run TestStage6RemoteTransportAcceptance -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/integration/remote_transport_stage6_acceptance_test.go
rtk git commit -m "test: verify stage 6 remote transport adapters"
```

## Task 6：迁移清单与全仓验证

**Files:**

- Create: `docs/plans/2026-05-06-stage-6-migration-inventory.md`
- Modify: `docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md`

**迁移内容与理由:** 阶段 6 必须把旧代码参考与不迁移项写清楚，尤其要说明为什么不迁移旧 provider/registry/bootstrap。这样后续实现不会把已排除的旧架构概念带回新版。

- [x] **Step 1: 写迁移清单**

创建 `docs/plans/2026-05-06-stage-6-migration-inventory.md`：

```markdown
# Stage 6 迁移清单

## 范围结论

阶段 6 实现 Connect 与 gRPC remote server adapter。两者都作为 message contract active server adapter，把当前 service 的调用转发到远端标准 RPC 服务。

阶段 6 不生成 connect/grpc 标准 client，不引入旧 framework selector、多 provider registry 或 bootstrap，也不改变 Stage 5 local handler adapter。

## 迁移或参考

1. 参考旧 `forwarding_plan.go`
   - 作用：旧 planner 把 forwarding 文件作为独立生成物规划。
   - 新版落点：`internal/generator/render_message_plan.go` 的 `ConnectRemote` / `GRPCRemote` 文件族。
   - 为什么参考而不是迁移：新版已经有 `MessageFileFamilyPlan`，只需要复用“remote 是独立文件族”的结构思路，旧 planner 的 framework/provider 概念不能迁入。

2. 参考旧 `native_forwarding_client.go` / `native_forwarding_server.go`
   - 作用：旧代码把远端 transport 调用包装成本地可注册 adapter。
   - 新版落点：`internal/generator/render_connect_remote.go`、`internal/generator/render_grpc_remote.go`。
   - 为什么参考而不是迁移：旧代码围绕 native forwarding 和 Go client registry；新版 remote adapter 是 message contract server adapter，必须直接实现 `<Service>MessageAdapter`。

3. 参考旧 native forwarding integration tests
   - 作用：覆盖真实 transport、streaming 顺序、错误传播。
   - 新版落点：`internal/integration/remote_transport_stage6_acceptance_test.go`。
   - 为什么值得迁移测试思路：remote adapter 的最大风险在端到端 stream lifecycle，仅靠 renderer 字符串测试不够。

## 明确不迁移的内容

1. framework selector。
2. 多 provider registry。
3. 旧 bootstrap。
4. GoClientMessageProvider / GoClientNativeProvider server kind。
5. connect/grpc 标准 client 生成模型。

## 验证结果

- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`：`Go test: 164 passed in 2 packages`。
- `rtk go test ./internal/integration -count=1`：`Go test: 49 passed in 1 packages`。
- `rtk go test ./rpcruntime -count=1`：`Go test: 167 passed in 1 packages`。
- `rtk go test ./... -count=1`：`Go test: 380 passed in 5 packages`。
- AGENTS.md 中的 forbidden unsigned scan：`rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-migration-inventory.md'` 退出码 `1` 且无输出。
```

- [x] **Step 2: 全仓验证**

Run:

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1
rtk go test ./internal/integration -count=1
rtk go test ./rpcruntime -count=1
rtk go test ./... -count=1
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md'
```

Expected:

- 所有 `go test` PASS。
- forbidden unsigned scan 只允许命中文档中记录命令的文字；代码、C ABI 和 generated renderer 不允许新增 forbidden unsigned 32/64 类型。

- [x] **Step 3: 更新计划与迁移清单验证结果**

将本计划所有已完成任务 checkbox 更新为 `[x]`，并把 `docs/plans/2026-05-06-stage-6-migration-inventory.md` 的 “待执行” 改为实际命令结果。

- [x] **Step 4: 提交**

```bash
rtk git add docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md docs/plans/2026-05-06-stage-6-migration-inventory.md
rtk git commit -m "docs: record stage 6 remote transport adapters"
```

## 阶段 6 完成标准

- `@rpccgo:msg-connect` 生成 `<service>.remote.connect.rpccgo.go`。
- `@rpccgo:msg-grpc` 生成 `<service>.remote.grpc.rpccgo.go`。
- `@rpccgo:native` 继续展开为 `msg-connect|native`，因此生成 Connect remote adapter 与 native/message converter。
- Connect remote adapter 内部使用标准 `connect.Client`，不暴露新的 rpccgo client 类型。
- gRPC remote adapter 内部使用标准 `grpc.ClientConnInterface`，不暴露新的 rpccgo client 类型。
- Connect/gRPC remote adapter 都注册为 message contract active server。
- remote unary、client streaming、server streaming、bidi streaming 都进入远端标准 RPC server。
- remote adapter 能路由到远端 cgo message server。
- native client 调 remote adapter 时复用 Stage 4B converter。
- stream Start 捕获 remote adapter snapshot，后续 Send/Recv/Finish/Done/CloseSend/Cancel 固定路由到该 snapshot。
- 阶段 6 不生成 connect/grpc client，不引入旧 bootstrap、framework selector 或 provider registry。
- `rpcruntime` 不引入 protobuf、connect、grpc 或 `internal/generator` 依赖。
- 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无代码命中。

## 阶段 6 后续风险

- 单进程 acceptance 如果复用同一个 generated package，远端和本地 dispatcher 可能共享全局 active server；实现时必须用独立 package 或独立进程隔离 remote process 语义。
- Connect streaming client 的底层 `Conn()` 关闭行为依赖当前 Connect 版本；如版本 API 不适合表达 cancel，本阶段只要求 context cancellation 和正常终态。
- gRPC remote `Cancel` 不主动关闭 shared `ClientConnInterface`；连接生命周期由调用方持有，Stage 6 不托管连接。
- 复杂 payload 的 native/message converter 广覆盖仍主要由 Stage 4B 测试承担；Stage 6 acceptance 重点证明 remote transport 复用 converter。
