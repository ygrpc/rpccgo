# 阶段 7 统一生成物与端到端示例实施计划

> **给 agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` 或 `superpowers:executing-plans` 按任务逐项执行。步骤使用 checkbox (`- [x]`) 语法跟踪。

**目标:** 固化最终生成文件布局与 public API 命名，并提供两个用户可运行 example：一个最小路径 example，一个全 transport / 全 streaming example。

**架构:** 阶段 7 不改变 dispatcher、converter、local adapter 或 remote adapter 语义；它把 Stage 1-6 已实现能力整理成稳定用户入口。生成物仍以 protobuf Go package 为 service package，cgo 生成物仍进入 `cgo_dir` 的 `package main`，所有调用路径仍经过 generated dispatcher 和单 active server slot。

**技术栈:** Go 1.24、cgo、protobuf/protoc、`protoc-gen-go`、`protoc-gen-rpc-cgo`、Connect、gRPC、现有 `rpcruntime` 与 generated service runtime。

---

## 范围

阶段 7 实现：

- 最终 generated file family 与 public API 命名合同测试。
- `examples/minimal-greeter`：从 proto 到生成、注册 server、启动监听、发起 cgo native 与 cgo message unary 调用的最小路径。
- `examples/full-greeter`：全 transport、全 streaming 的完整能力 example，覆盖 Connect/gRPC local、Connect/gRPC remote、native/message cgo client、unary/client streaming/server streaming/bidi streaming。
- example 生成与验证脚本或 `go generate` 入口。
- example-focused acceptance，证明 example 不依赖 integration-only helper。
- 阶段 7 迁移清单与验证记录。

阶段 7 不实现：

- 新 transport、新 client 类型或新 adapter 语义。
- remote 负载均衡、重试、服务发现或连接生命周期托管。
- Stage 8 的兼容性清理、发布命令集合、复杂错误矩阵扩展。
- 旧项目的双 provider bootstrap、multi registry、framework selector、debugserver 架构。
- 把 cgo 生成物放回 protobuf Go package，或把 service adapter 文件生成到 `cgo_dir`。

## 设计边界

- 两个 example 都必须使用真实 `protoc` + `protoc-gen-go` + `protoc-gen-rpc-cgo` 生成路径，而不是只调用 `internal/generator.GenerateWithOptions`。
- `examples/minimal-greeter` 只覆盖 unary 主路径，让用户最快理解：proto -> generated service -> active server -> listener -> cgo client call。
- `examples/full-greeter` 才承担全矩阵展示，避免最小 example 变成集成测试大杂烩。
- example 可以复用旧项目 greeter proto 的字段和业务行为，但必须按新版 `@rpccgo` token、单 dispatcher、单 active server slot 重写。
- example 中允许测试文件通过 Go/cgo 直接调用 generated `package main` cgo export 函数；不要求本阶段编写真正外部 C 可执行程序。
- public API 冻结以 generator tests 表达，README 只加极简 example 导航，不把 spec 概念搬进去。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
| --- | --- | --- | --- |
| `examples/connect/proto/greeter.proto` | 参考后重写 | 提供 greeter 业务、scalar/repeated/message 字段和 streaming method 形状 | proto 场景贴近用户示例，值得保留语义；旧文件含旧注释和 skip 语义，需要改写为新版 `@rpccgo` token |
| `examples/connect/internal/backend/backend.go` | 参考后重写 | 提供 greeting、inspect、streaming demo 行为 | 业务行为可读，适合迁移为 example backend；旧代码依赖旧 generated API、forwarding client 和 bootstrap，不能直接迁移 |
| 旧 `cmd/rpc` generated/export 文件 | 不迁移 | 旧 cgo export 与 bootstrap 产物 | 与新版 `cgo_dir/package main` 和 single dispatcher 合同冲突，必须由当前 generator 重新生成 |
| 旧 provider registry / framework selector / debugserver | 不迁移 | 旧多 provider、多 registry、多入口调试模型 | 与新版单 active server、标准 Connect/gRPC transport 复用冲突，不能进入阶段 7 |
| Stage 3-6 integration fixtures | 参考测试思路 | 已覆盖 direct path、converter、local/remote transport、stream lifecycle | 可提炼 example acceptance 的验证方式；不能把 integration-only reset/helper 当作用户 API |

## 文件结构

- Create: `internal/generator/generated_layout_contract_test.go`  
  固化 generated file family、package placement、public registration API 和禁止旧命名。
- Create: `examples/minimal-greeter/proto/greeter.proto`  
  最小 unary proto，含 `@rpccgo: msg-connect|native`。
- Create: `examples/minimal-greeter/go.mod`  
  example 模块，`replace rpccgo => ../..`。
- Create: `examples/minimal-greeter/gen.go`  
  `go generate` 入口，调用真实 `protoc`。
- Create: `examples/minimal-greeter/internal/backend/backend.go`  
  Go native server 实现。
- Create: `examples/minimal-greeter/cmd/server/main.go`  
  注册 Go native server 并启动 Connect handler。
- Create: `examples/minimal-greeter/cmd/rpc/minimal_unary_test.go`  
  在 generated cgo package 中验证 cgo native client 与 cgo message client unary 调用。
- Create: `examples/minimal-greeter/example_test.go`  
  运行生成命令和 example package 测试。
- Create: `examples/minimal-greeter/magefile.go`  
  提供 `Generate`、`Run`、`Test`、`Server` target，`Run` 执行真实 demo，`Test` 跑验收。
- Create: `examples/full-greeter/proto/greeter.proto`  
  全 transport / 全 streaming proto，含 unary、client streaming、server streaming、bidi streaming。
- Create: `examples/full-greeter/go.mod`
- Create: `examples/full-greeter/gen.go`
- Create: `examples/full-greeter/internal/backend/backend.go`
- Create: `examples/full-greeter/cmd/server/main.go`
- Create: `examples/full-greeter/cmd/rpc/full_matrix_test.go`
- Create: `examples/full-greeter/example_test.go`
- Create: `examples/full-greeter/magefile.go`
- Modify: `README.md`  
  只增加极简 example 导航。
- Create: `docs/plans/2026-05-07-stage-7-migration-inventory.md`
- Modify: `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`  
  执行时更新 checkbox 和验证结果。

## Task 1：固化 generated layout 与 public API 合同

**Files:**

- Create: `internal/generator/generated_layout_contract_test.go`

**迁移内容与理由:** 不迁移旧生成物命名；本任务把 Stage 1-6 当前生成物命名冻结为测试合同，避免阶段 7 写 example 时意外依赖即将变化的 API。

- [x] **Step 1: 写失败测试**

创建 `internal/generator/generated_layout_contract_test.go`：

```go
package generator

import (
	"strings"
	"testing"
)

func TestStage7GeneratedLayoutContract(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", streamingTestFileWithServiceComment("@rpccgo: msg-connect|msg-grpc|native\n"))
	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.runtime.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.codec.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.server.connect.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.server.grpc.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.remote.connect.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/greeter.greeter.remote.grpc.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/../cmd/rpc/greeter.greeter.server.cgo.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/../cmd/rpc/greeter.greeter.client.cgo.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/../cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go")
	assertGeneratedFileExists(t, plugin, "test/v1/../cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go")
}

func TestStage7PublicAPIContract(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", streamingTestFileWithServiceComment("@rpccgo: msg-connect|msg-grpc|native\n"))
	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}

	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.native.rpccgo.go",
		"func RegisterGreeterGoNativeServer(server GreeterNativeServer) (rpcruntime.AdapterSnapshot[GreeterNativeAdapter], error)")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.connect.rpccgo.go",
		"func NewGreeterConnectHandler(options ...connect.HandlerOption) (string, http.Handler)")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.server.grpc.rpccgo.go",
		"func RegisterGreeterGRPCServer(registrar grpc.ServiceRegistrar) error")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.remote.connect.rpccgo.go",
		"func RegisterGreeterConnectRemoteServer(httpClient connect.HTTPClient, baseURL string, options ...connect.ClientOption) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error)")
	assertGeneratedContentContains(t, plugin, "test/v1/greeter.greeter.remote.grpc.rpccgo.go",
		"func RegisterGreeterGRPCRemoteServer(conn grpc.ClientConnInterface) (rpcruntime.AdapterSnapshot[GreeterMessageAdapter], error)")
	assertGeneratedContentContains(t, plugin, "test/v1/../cmd/rpc/greeter.greeter.server.message.cgo.rpccgo.go",
		"func RegisterGreeterCGOMessageServer")
}

func TestStage7GeneratedLayoutRejectsOldBootstrapNames(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative,cgo_dir=../cmd/rpc", streamingTestFileWithServiceComment("@rpccgo: msg-connect|msg-grpc|native\n"))
	if _, err := GenerateWithOptions(plugin, GenerateOptions{RenderStageFiles: true}); err != nil {
		t.Fatalf("GenerateWithOptions(RenderStageFiles) error = %v", err)
	}
	for _, generated := range plugin.Response().GetFile() {
		name := generated.GetName()
		content := generated.GetContent()
		for _, banned := range []string{"provider", "registry", "bootstrap", "framework selector", "goclient.export", "goserver.export"} {
			if strings.Contains(strings.ToLower(name), banned) || strings.Contains(strings.ToLower(content), banned) {
				t.Fatalf("generated %s contains old bootstrap token %q", name, banned)
			}
		}
	}
}
```

- [x] **Step 2: 运行测试确认失败或暴露缺口**

Run:

```bash
rtk go test ./internal/generator -run 'TestRenderMessageClientCGO|TestStage7' -count=1
```

Result: PASS。

Expected: 若当前 helper 缺失或文件名不完全匹配，测试失败并指出具体合同缺口。

- [x] **Step 3: 修正测试 helper 或合同断言**

如果 `streamingTestFileWithServiceComment` 不存在，改用已有 `newTestPlugin` fixture helper 组合一个包含 unary、client streaming、server streaming、bidi streaming 的 service descriptor。保持断言目标不变。

- [x] **Step 4: 运行测试确认通过**

Run:

```bash
rtk go test ./internal/generator -run TestStage7 -count=1
```

Expected: PASS。

- [x] **Step 5: 提交**

```bash
rtk git add internal/generator/generated_layout_contract_test.go
rtk git commit -m "test: lock stage 7 generated layout contract"
```

## Task 2：建立 minimal-greeter proto 与真实生成入口

**Files:**

- Create: `examples/minimal-greeter/go.mod`
- Create: `examples/minimal-greeter/gen.go`
- Create: `examples/minimal-greeter/proto/greeter.proto`
- Create: `examples/minimal-greeter/example_test.go`

**迁移内容与理由:** 参考旧 greeter proto 的 unary greeting 场景，但重写为最小字段集合，降低新用户理解成本；使用真实 `protoc` 证明用户可以从 proto 生成新版代码。

- [x] **Step 1: 写最小 proto**

创建 `examples/minimal-greeter/proto/greeter.proto`：

```proto
syntax = "proto3";

package examples.minimal.greeter.v1;

option go_package = "example.com/rpccgo-minimal/gen/greeter/v1;greeterv1";

message SayHelloRequest {
  string name = 1;
}

message SayHelloResponse {
  string message = 1;
}

// @rpccgo: msg-connect|native
service Greeter {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
}
```

- [x] **Step 2: 写 example module 与 go generate 入口**

创建 `examples/minimal-greeter/go.mod`：

```go
module example.com/rpccgo-minimal

go 1.24.4

require (
	connectrpc.com/connect v1.19.1
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
	rpccgo v0.0.0
)

replace rpccgo => ../..
```

创建 `examples/minimal-greeter/gen.go`：

```go
package minimal

//go:generate protoc -I proto --go_out=. --go_opt=paths=source_relative --rpc-cgo_out=. --rpc-cgo_opt=paths=source_relative --rpc-cgo_opt=cgo_dir=../../cmd/rpc proto/greeter.proto
```

- [x] **Step 3: 写失败测试**

创建 `examples/minimal-greeter/example_test.go`：

```go
package minimal

import (
	"os"
	"os/exec"
	"testing"
)

func TestMinimalGreeterGenerate(t *testing.T) {
	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}
	for _, path := range []string{
		"proto/greeter.pb.go",
		"proto/greeter.greeter.runtime.rpccgo.go",
		"proto/greeter.greeter.server.native.rpccgo.go",
		"proto/greeter.greeter.server.connect.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.cgo.rpccgo.go",
		"cmd/rpc/greeter.greeter.client.message.cgo.rpccgo.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s missing: %v", path, err)
		}
	}
}
```

- [x] **Step 4: 运行测试确认失败**

Run:

```bash
rtk go test ./examples/minimal-greeter -run TestMinimalGreeterGenerate -count=1
```

Expected: 初次运行可能因为插件二进制未在 `PATH`、`protoc-gen-go` 缺失或生成路径不对而失败；按失败信息修正 example 生成入口。

- [x] **Step 5: 修正生成入口**

若 `protoc-gen-rpc-cgo` 不在 `PATH`，在测试中先运行：

```go
install := exec.Command("go", "install", "../../cmd/protoc-gen-rpc-cgo")
```

并把 `GOBIN` 指向 `t.TempDir()`，再把该目录加入 `PATH`。若 `protoc-gen-go` 不在 `PATH`，使用同样方式安装 `google.golang.org/protobuf/cmd/protoc-gen-go`。

- [x] **Step 6: 运行测试确认通过**

Run:

```bash
rtk go test ./examples/minimal-greeter -run TestMinimalGreeterGenerate -count=1
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
rtk git add examples/minimal-greeter
rtk git commit -m "feat: add minimal greeter generation example"
```

## Task 3：实现 minimal-greeter 运行路径

**Files:**

- Create: `examples/minimal-greeter/internal/backend/backend.go`
- Create: `examples/minimal-greeter/cmd/server/main.go`
- Create: `examples/minimal-greeter/cmd/rpc/minimal_unary_test.go`
- Modify: `examples/minimal-greeter/example_test.go`

**迁移内容与理由:** 参考旧 backend 的 greeting 行为，重写为新版 Go native server。最小 example 只展示一个 active server 和一个 Connect listener，cgo native/message client 都通过 dispatcher 到同一个 server。

- [x] **Step 1: 写 backend**

创建 `examples/minimal-greeter/internal/backend/backend.go`：

```go
package backend

import (
	"context"
	"fmt"

	greeterv1 "example.com/rpccgo-minimal/proto"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	name := req.GetName()
	if name == "" {
		name = "world"
	}
	return &greeterv1.SayHelloResponse{Message: fmt.Sprintf("hello, %s", name)}, nil
}
```

- [x] **Step 2: 写 server 入口**

创建 `examples/minimal-greeter/cmd/server/main.go`：

```go
package main

import (
	"log"
	"net/http"

	greeterv1 "example.com/rpccgo-minimal/proto"
	"example.com/rpccgo-minimal/internal/backend"
)

func main() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
	path, handler := greeterv1.NewGreeterConnectHandler()
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
```

- [x] **Step 3: 写 cgo package 测试**

创建 `examples/minimal-greeter/cmd/rpc/minimal_unary_test.go`：

```go
package main

import (
	"testing"

	greeterv1 "example.com/rpccgo-minimal/proto"
	"example.com/rpccgo-minimal/internal/backend"
	"google.golang.org/protobuf/proto"
	rpcruntime "rpccgo/rpcruntime"
)

func TestMinimalNativeAndMessageClients(t *testing.T) {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}

	nameBytes := []byte("native")
	req := GreeterSayHelloNativeRequest{Name: rpcruntime.NewRpcString(&nameBytes[0], int32(len(nameBytes)), false)}
	var resp GreeterSayHelloNativeResponse
	if errID := GreeterSayHelloNative(&req, &resp); errID != 0 {
		t.Fatalf("GreeterSayHelloNative() error id = %d", errID)
	}
	got, err := resp.Message.String()
	if err != nil {
		t.Fatalf("native response string error = %v", err)
	}
	if got != "hello, native" {
		t.Fatalf("native response = %q, want hello, native", got)
	}

	msgReq, err := proto.Marshal(&greeterv1.SayHelloRequest{Name: "message"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var outPtr uintptr
	var outLen int32
	if errID := GreeterSayHelloMessage(msgReq, &outPtr, &outLen); errID != 0 {
		t.Fatalf("GreeterSayHelloMessage() error id = %d", errID)
	}
	out := rpcruntime.BytesFromPtr(outPtr, outLen)
	var msgResp greeterv1.SayHelloResponse
	if err := proto.Unmarshal(out, &msgResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if msgResp.GetMessage() != "hello, message" {
		t.Fatalf("message response = %q, want hello, message", msgResp.GetMessage())
	}
}
```

- [x] **Step 4: 更新 example test 跑完整最小路径**

在 `examples/minimal-greeter/example_test.go` 增加：

```go
func TestMinimalGreeterExample(t *testing.T) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("minimal example failed: %v\n%s", err, out)
	}
}
```

- [x] **Step 5: 运行测试确认失败并修正 API 名称**

Run:

```bash
在 examples/minimal-greeter 下执行 `rtk go test ./... -count=1`：PASS
```

Expected: 若 generated cgo export 函数或 wrapper helper 名称和计划不一致，测试失败；按实际 generated symbol 修正 `minimal_unary_test.go`，不要修改 generator 合同。

- [x] **Step 6: 运行测试确认通过**

Run:

```bash
rtk go test ./examples/minimal-greeter/... -count=1
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
rtk git add examples/minimal-greeter
rtk git commit -m "feat: add minimal greeter runnable path"
```

## Task 4：建立 full-greeter proto、backend 与生成入口

**Files:**

- Create: `examples/full-greeter/go.mod`
- Create: `examples/full-greeter/gen.go`
- Create: `examples/full-greeter/proto/greeter.proto`
- Create: `examples/full-greeter/internal/backend/backend.go`
- Create: `examples/full-greeter/example_test.go`

**迁移内容与理由:** 参考旧 greeter proto 的 unary、repeated、streaming 形状，以及 Stage 5/6 integration 的 transport 验证思路；重写为用户 example，不引入 integration reset helper 或旧 bootstrap。

- [x] **Step 1: 写 full proto**

创建 `examples/full-greeter/proto/greeter.proto`：

```proto
syntax = "proto3";

package examples.full.greeter.v1;

option go_package = "example.com/rpccgo-full/proto;greeterv1";

message SayHelloRequest {
  string name = 1;
  string city = 2;
  repeated int32 ids = 3;
  repeated bool flags = 4;
}

message SayHelloResponse {
  string message = 1;
}

// @rpccgo: msg-connect|msg-grpc|native
service Greeter {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
  rpc Collect(stream SayHelloRequest) returns (SayHelloResponse);
  rpc Broadcast(SayHelloRequest) returns (stream SayHelloResponse);
  rpc Chat(stream SayHelloRequest) returns (stream SayHelloResponse);
}
```

- [x] **Step 2: 写 module 与 go generate**

创建 `examples/full-greeter/go.mod`：

```go
module example.com/rpccgo-full

go 1.24.4

require (
	connectrpc.com/connect v1.19.1
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
	rpccgo v0.0.0
)

replace rpccgo => ../..
```

创建 `examples/full-greeter/gen.go`：

```go
package full

//go:generate protoc -I proto --go_out=. --go_opt=paths=source_relative --rpc-cgo_out=. --rpc-cgo_opt=paths=source_relative --rpc-cgo_opt=cgo_dir=../../cmd/rpc proto/greeter.proto
```

- [x] **Step 3: 写 backend**

创建 `examples/full-greeter/internal/backend/backend.go`：

```go
package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	greeterv1 "example.com/rpccgo-full/proto"
)

type Greeter struct{}

func (Greeter) SayHello(_ context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	return &greeterv1.SayHelloResponse{Message: format(req.GetName(), req.GetCity(), len(req.GetIds()), len(req.GetFlags()))}, nil
}

func (Greeter) Collect(_ context.Context, stream greeterv1.GreeterCollectNativeClientStream) (*greeterv1.SayHelloResponse, error) {
	var names []string
	for {
		req, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, req.GetName())
	}
	return &greeterv1.SayHelloResponse{Message: "collect:" + strings.Join(names, ",")}, nil
}

func (Greeter) Broadcast(_ context.Context, req *greeterv1.SayHelloRequest, stream greeterv1.GreeterBroadcastNativeServerStream) error {
	for i := 0; i < 2; i++ {
		if err := stream.Send(&greeterv1.SayHelloResponse{Message: fmt.Sprintf("broadcast[%d]:%s", i, req.GetName())}); err != nil {
			return err
		}
	}
	return nil
}

func (Greeter) Chat(_ context.Context, stream greeterv1.GreeterChatNativeBidiStream) error {
	for {
		req, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&greeterv1.SayHelloResponse{Message: "chat:" + req.GetName()}); err != nil {
			return err
		}
	}
}

func format(name, city string, ids, flags int) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("hello %s from %s ids=%d flags=%d", name, city, ids, flags)
}
```

- [x] **Step 4: 写生成测试**

创建 `examples/full-greeter/example_test.go`：

```go
package full

import (
	"os"
	"os/exec"
	"testing"
)

func TestFullGreeterGenerate(t *testing.T) {
	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go generate failed: %v\n%s", err, out)
	}
}
```

- [x] **Step 5: 运行测试确认失败并修正接口名**

Run:

```bash
rtk go test ./examples/full-greeter -run TestFullGreeterGenerate -count=1
```

Expected: 生成通过后，若 backend interface 名称与 generated native streaming interface 不一致，在后续 compile 中暴露；本步只要求真实生成成功。

- [x] **Step 6: 运行生成测试确认通过**

Run:

```bash
rtk go test ./examples/full-greeter -run TestFullGreeterGenerate -count=1
```

Expected: PASS。

- [x] **Step 7: 提交**

```bash
rtk git add examples/full-greeter
rtk git commit -m "feat: add full greeter generation example"
```

## Task 5：实现 full-greeter 全 transport / 全 streaming 运行矩阵

**Files:**

- Create: `examples/full-greeter/cmd/server/main.go`
- Create: `examples/full-greeter/cmd/rpc/full_matrix_test.go`
- Modify: `examples/full-greeter/example_test.go`

**迁移内容与理由:** 迁移 Stage 5/6 acceptance 的验证思路：使用真实 Connect/gRPC local server 和 remote adapter 证明 transport 复用标准 API。示例测试重写为用户侧 package，不复用 integration-only module writer 或 dispatcher reset helper。

- [x] **Step 1: 写 server 入口**

创建 `examples/full-greeter/cmd/server/main.go`：

```go
package main

import (
	"log"
	"net"
	"net/http"

	greeterv1 "example.com/rpccgo-full/proto"
	"example.com/rpccgo-full/internal/backend"
	"google.golang.org/grpc"
)

func main() {
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		log.Fatal(err)
	}
	go func() {
		path, handler := greeterv1.NewGreeterConnectHandler()
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		log.Fatal(http.ListenAndServe("127.0.0.1:8081", mux))
	}()
	lis, err := net.Listen("tcp", "127.0.0.1:8082")
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	if err := greeterv1.RegisterGreeterGRPCServer(server); err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Serve(lis))
}
```

- [x] **Step 2: 写 full matrix cgo test**

创建 `examples/full-greeter/cmd/rpc/full_matrix_test.go`，测试结构如下：

```go
package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	greeterv1 "example.com/rpccgo-full/proto"
	"example.com/rpccgo-full/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestFullGreeterTransportAndStreamingMatrix(t *testing.T) {
	t.Run("cgo native client to go native server", func(t *testing.T) {
		registerGoNative(t)
		callNativeUnary(t)
		callNativeClientStreaming(t)
		callNativeServerStreaming(t)
		callNativeBidiStreaming(t)
	})
	t.Run("cgo message client to go native server", func(t *testing.T) {
		registerGoNative(t)
		callMessageUnary(t)
		callMessageClientStreaming(t)
		callMessageServerStreaming(t)
		callMessageBidiStreaming(t)
	})
	t.Run("connect local to go native server", func(t *testing.T) {
		registerGoNative(t)
		baseURL, client := startConnectLocal(t)
		callConnectUnary(t, client, baseURL)
		callConnectClientStreaming(t, client, baseURL)
		callConnectServerStreaming(t, client, baseURL)
		callConnectBidiStreaming(t, client, baseURL)
	})
	t.Run("grpc local to go native server", func(t *testing.T) {
		registerGoNative(t)
		conn := startGRPCLocal(t)
		callGRPCUnary(t, conn)
		callGRPCClientStreaming(t, conn)
		callGRPCServerStreaming(t, conn)
		callGRPCBidiStreaming(t, conn)
	})
	t.Run("connect remote active server", func(t *testing.T) {
		remoteURL, client := startConnectLocal(t)
		if _, err := greeterv1.RegisterGreeterConnectRemoteServer(client, remoteURL); err != nil {
			t.Fatalf("RegisterGreeterConnectRemoteServer() error = %v", err)
		}
		callMessageUnary(t)
		callNativeUnary(t)
	})
	t.Run("grpc remote active server", func(t *testing.T) {
		conn := startGRPCLocal(t)
		if _, err := greeterv1.RegisterGreeterGRPCRemoteServer(conn); err != nil {
			t.Fatalf("RegisterGreeterGRPCRemoteServer() error = %v", err)
		}
		callMessageUnary(t)
		callNativeUnary(t)
	})
}

func registerGoNative(t *testing.T) {
	t.Helper()
	if _, err := greeterv1.RegisterGreeterGoNativeServer(backend.Greeter{}); err != nil {
		t.Fatalf("RegisterGreeterGoNativeServer() error = %v", err)
	}
}

func startConnectLocal(t *testing.T) (string, *http.Client) {
	t.Helper()
	registerGoNative(t)
	path, handler := greeterv1.NewGreeterConnectHandler()
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL, server.Client()
}

func startGRPCLocal(t *testing.T) grpc.ClientConnInterface {
	t.Helper()
	registerGoNative(t)
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	if err := greeterv1.RegisterGreeterGRPCServer(server); err != nil {
		t.Fatalf("RegisterGreeterGRPCServer() error = %v", err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

var _ = connect.NewClient[greeterv1.SayHelloRequest, greeterv1.SayHelloResponse]
```

在同文件补齐 `callNative*`、`callMessage*`、`callConnect*`、`callGRPC*` helper。优先从 Stage 3-6 acceptance 中提炼最短调用代码，断言每条路径返回非空且包含输入 name。

- [x] **Step 3: 更新 full example test**

在 `examples/full-greeter/example_test.go` 增加：

```go
func TestFullGreeterExample(t *testing.T) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("full example failed: %v\n%s", err, out)
	}
}
```

- [x] **Step 4: 运行测试确认失败并补齐 helper**

Run:

```bash
在 examples/full-greeter 下执行 `rtk go test ./... -count=1`：PASS
```

Expected: helper 未补齐或 generated symbol 名称不匹配时失败；逐步补齐直到所有 transport/streaming 子测试通过。

- [x] **Step 5: 运行测试确认通过**

Run:

```bash
rtk go test ./examples/full-greeter/... -count=1
```

Expected: PASS。

- [x] **Step 6: 提交**

```bash
rtk git add examples/full-greeter
rtk git commit -m "feat: add full greeter transport matrix example"
```

## Task 6：README 导航、迁移清单与阶段验收

**Files:**

- Modify: `README.md`
- Create: `docs/plans/2026-05-07-stage-7-migration-inventory.md`
- Modify: `docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md`

**迁移内容与理由:** 阶段 7 需要记录旧 examples 中迁移了什么、为什么只参考不照搬旧 bootstrap；README 只做入口导航，避免重复架构 spec。

- [x] **Step 1: 更新 README**

在 `README.md` 架构简介后增加：

```markdown
## Examples

- `examples/minimal-greeter`：最小路径，从 proto 生成代码并通过 cgo native/message client 调用 Go native server。
- `examples/full-greeter`：完整路径，覆盖 native/message cgo client、Connect/gRPC local adapter、Connect/gRPC remote adapter 和三类 streaming。
```

- [x] **Step 2: 写迁移清单**

创建 `docs/plans/2026-05-07-stage-7-migration-inventory.md`：

```markdown
# Stage 7 迁移清单

## 范围结论

阶段 7 固化 generated layout 与 public API，并新增 `examples/minimal-greeter` 与 `examples/full-greeter`。example 使用真实 `protoc` 生成路径，所有调用仍经过 generated dispatcher 和单 active server slot。

## 迁移或参考

| 旧项目文件或模块 | 本阶段处理 | 作用 | 迁移理由 |
| --- | --- | --- | --- |
| `examples/connect/proto/greeter.proto` | 参考后重写 | greeter unary、repeated、streaming 场景 | 业务场景适合用户示例，但旧 token 与 skip 语义不适合新版 |
| `examples/connect/internal/backend/backend.go` | 参考后重写 | greeting 与 streaming backend 行为 | 行为可读，API 必须按新版 generated service 重写 |
| Stage 3-6 integration fixtures | 参考测试思路 | direct path、converter、local/remote transport、stream lifecycle 验证 | 用于 example acceptance，但不暴露 integration-only helper |

## 明确不迁移

- 旧 `cmd/rpc` generated/export 文件。
- 旧 provider registry、多 provider bootstrap、framework selector。
- 旧 debugserver 与 forwarding bootstrap。

## 验证结果

已执行阶段 7 验证：

- `rtk go test ./internal/generator -run 'TestRenderMessageClientCGO|TestStage7' -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/full-greeter` 下执行 `rtk go test ./... -count=1`：PASS。
- 在 `examples/minimal-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS，打印真实 demo 输出并自动退出。
- 在 `examples/full-greeter` 下执行 `rtk go run github.com/magefile/mage run`：PASS，打印 Connect/gRPC demo 输出并自动退出。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`：PASS。
- `rtk go test ./internal/integration -count=1`：PASS。
- `rtk go test ./rpcruntime -count=1`：PASS。
- `rtk go test ./... -count=1`：PASS。
- forbidden unsigned scan：PASS，无输出，退出码 `1`。
```

- [x] **Step 3: 全仓验证**

Run:

```bash
rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1
rtk go test ./internal/integration -count=1
rtk go test ./rpcruntime -count=1
cd examples/minimal-greeter && rtk go test ./... -count=1
cd examples/full-greeter && rtk go test ./... -count=1
rtk go test ./... -count=1
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/plans/2026-05-06-stage-6-connect-grpc-remote-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-connect-grpc-local-adapter-plan.md' -g '!docs/plans/2026-05-06-stage-5-migration-inventory.md' -g '!docs/plans/2026-05-06-stage-6-migration-inventory.md'
```

Expected:

- 所有 `go test` 通过。
- forbidden unsigned scan 退出码 `1` 且无输出。

- [x] **Step 4: 更新计划 checkbox 和验证结果**

把本计划已完成项更新为 `[x]`，并把迁移清单中的“待执行”改为实际结果。不要记录本机环境 workaround。

- [x] **Step 5: 提交**

```bash
rtk git add README.md docs/plans/2026-05-07-stage-7-generated-layout-and-examples-plan.md docs/plans/2026-05-07-stage-7-migration-inventory.md
rtk git commit -m "docs: record stage 7 generated examples"
```

## 阶段 7 完成标准

- [x] generated file family 和 public API 命名有 focused contract tests。
- [x] `examples/minimal-greeter` 可以从 proto 生成代码。
- [x] `examples/minimal-greeter` 可以注册 Go native server、启动 Connect handler，并通过 cgo native client 与 cgo message client 完成 unary 调用。
- [x] `examples/full-greeter` 可以从 proto 生成代码。
- [x] `examples/full-greeter` 覆盖 native/message cgo client、Connect/gRPC local adapter、Connect/gRPC remote adapter。
- [x] `examples/full-greeter` 覆盖 unary、client streaming、server streaming、bidi streaming。
- [x] example 不保留旧项目双 provider bootstrap、多 registry、framework selector 或 debugserver 模型。
- [x] `README.md` 有简短 example 导航。
- [x] example 可以通过 Magefile 运行生成和验收入口。
- [x] 迁移清单说明旧 examples 中迁移、参考和不迁移内容。
- [x] generator focused、integration focused、runtime focused、example focused、全仓测试通过。
- [x] forbidden unsigned scan 无命中。

## 阶段 7 后续风险

- example 使用真实 `protoc`，开发机需要 `protoc` 可用；若 CI 没有 `protoc`，阶段 8 需要决定是否在 CI 安装或把 example generation 拆成可选验证。
- full example 的 helper 可能和现有 integration fixture 有重复；阶段 7 接受少量重复以换取用户可读性，阶段 8 再决定是否抽公共 test helper。
- 本阶段固化 public API 后，后续重命名成本会上升；Task 1 必须仔细确认命名就是想长期保留的用户 API。
