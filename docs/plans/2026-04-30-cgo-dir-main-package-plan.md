# cgo_dir Main Package Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修正 generator 的 cgo 文件生成合同：所有 cgo 文件生成到 `package main`，输出目录通过 `cgo_dir` 设置，路径相对 protobuf Go package 生成目录解析，允许指向 Go package 目录之外，默认落到 Go package 目录下的 `cgo` 子目录。

**Architecture:** 该修复属于 generator-wide 输出布局修正，不改变 dispatcher、active server、native/message converter 或 runtime lifecycle 语义。非 cgo generated service 文件继续生成在 protobuf Go package 中；cgo client/server export 文件生成在独立 `package main` 目录，并通过稳定的跨包 API 调用 generated service runtime。

**Tech Stack:** Go 1.24、cgo、protobuf `protogen`、现有 `internal/generator`、标准库 `testing`。

---

## 范围

本计划聚焦 cgo 文件输出布局与 package 边界：

- `cgo_dir` plugin 参数解析。
- 默认 cgo 输出目录为 Go package 目录下的 `cgo`。
- `cgo_dir` 相对 protobuf Go package 生成目录解析。
- `cgo_dir` 支持 `../cmd/rpc` 这类指向 Go package 目录之外的路径。
- cgo generated files 使用 `package main`。
- cgo generated files 通过 import 调用 protobuf Go package 中的 generated runtime/server/client glue。
- 生成文件名和多 package 场景避免互相覆盖。
- focused generator tests 与 compile fixture。

本计划不实现：

- message contract direct path。
- native/message converter。
- connect/grpc local adapter。
- connect/grpc remote adapter。
- 监听 server 启动模型。
- example 业务工程。
- 旧多 registry、多 provider bootstrap、framework selector。

## 当前实现问题

当前代码没有满足目标合同：

- `parseRPCCGOParameter` 对所有自定义参数报 `unknown rpccgo parameter`，因此不接受 `cgo_dir`。
- `BuildNativeFileFamilyPlan` 使用 `file.GeneratedFilenamePrefix` 生成 cgo client/server 文件，cgo 文件与 runtime/native server 文件落在同一 Go package 目录。
- `render_native_client_cgo.go` 和 `render_native_server_cgo.go` 生成 `package <go package name>`，不是 `package main`。
- 当前测试断言 cgo 文件跟随 `GeneratedFilenamePrefix`，没有覆盖默认 `cgo/`、`cgo_dir=../cmd/rpc` 或 `package main`。

## 旧项目迁移判定

| 旧项目文件或模块 | 本阶段处理 | 作用 | 为什么迁移或参考 |
|---|---|---|---|
| 旧 `cgo_dir` 参数解析 | 参考后重写 | 支持 cgo 输出目录配置 | 参数语义可参考；新版 generator 参数必须接入当前 `protogen.Options` 与 `FilePlan` |
| 旧 `package main` cgo export 生成 | 参考后重写 | 让 cgo export 文件编译成独立 main package | 输出 package 边界可参考；旧代码绑定旧 bootstrap/provider，需要按新版 generated service runtime 重建跨包调用 |
| 旧 shared `cgo_dir` 文件名冲突修复 | 参考测试思路 | 避免多 package 同名 service 在共享 cgo 目录覆盖 | 冲突场景仍成立；文件名策略必须结合当前 `<service>.<role>.rpccgo.go` 文件族重算 |
| 旧多 registry、多 provider bootstrap、framework selector | 不迁移 | 旧运行时发现与框架选择模型 | 与新版单 dispatcher、单 active server、单插件 renderer pipeline 冲突 |

## 输出模型

假设 protobuf Go package 生成目录为 `test/v1`，proto basename 为 `greeter`，service 为 `Greeter`：

- 非 cgo 文件继续生成在 `test/v1/`：
  - `test/v1/greeter.greeter.runtime.rpccgo.go`
  - `test/v1/greeter.greeter.server.native.rpccgo.go`
- 默认 cgo 文件生成在 `test/v1/cgo/`，且文件内容为 `package main`：
  - `test/v1/cgo/greeter.greeter.server.cgo.rpccgo.go`
  - `test/v1/cgo/greeter.greeter.client.cgo.rpccgo.go`
- `cgo_dir=../cmd/rpc` 时，cgo 文件路径相对 `test/v1` 解析：
  - `test/cmd/rpc/greeter.greeter.server.cgo.rpccgo.go`
  - `test/cmd/rpc/greeter.greeter.client.cgo.rpccgo.go`

路径清理必须使用 slash path 语义，生成路径保持 protoc/protogen 期望的正斜杠形式。

## Task 1：用失败测试锁定 `cgo_dir` 参数合同

**Files:**

- Modify: `internal/generator/generator_test.go`
- Modify: `internal/generator/render_native_plan_test.go`

**迁移内容与理由：**

- 不迁旧参数解析代码。
- 先以当前 generator 测试锁定公开合同，避免后续 renderer 修复时混淆路径和 package 失败原因。

- [x] 添加测试：`cgo_dir` 是合法 rpccgo 参数。
- [x] 添加测试：空 `cgo_dir` 报明确错误。
- [x] 添加测试：默认 `cgo_dir` 为 `cgo`。
- [x] 添加测试：`cgo_dir=../cmd/rpc` 被保存在 generator 配置或 file plan 中。
- [x] 添加测试：未知参数仍报错。
- [x] 运行 `rtk go test ./internal/generator -run 'TestGenerate.*CGODir|TestPluginOptions|TestRenderNativeFileFamilyPlan' -count=1`。
- [x] 验收：测试能明确描述 cgo 输出目录合同，当前实现下失败点清晰。
- [x] 提交：`test: cover cgo_dir generation contract`

## Task 2：实现 `cgo_dir` 参数解析与默认值

**Files:**

- Modify: `internal/generator/generator.go`
- Modify: `internal/generator/plan.go`
- Modify: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 参考旧 `cgo_dir` 参数名和相对路径语义。
- 重写到当前单插件 `ProtogenOptions` 参数解析中。

- [x] 增加 generator config，包含 `CGODir`。
- [x] 默认 `CGODir` 为 `cgo`。
- [x] `parseRPCCGOParameter` 接受 `cgo_dir=<relative path>`。
- [x] 空 `cgo_dir` 返回明确错误。
- [x] 禁止绝对路径，避免 protoc 输出越过生成根目录。
- [x] 保留标准 `paths` 和 `M...` 参数由 `protogen` 原生处理。
- [x] 将 `CGODir` 写入 `FilePlan` 或 cgo file family plan。
- [x] 运行 `rtk go test ./internal/generator -run 'TestGenerate.*CGODir|TestPluginOptions' -count=1`。
- [x] 验收：generator 能接受并保存 `cgo_dir`，默认值稳定。
- [x] 提交：`feat: parse cgo_dir generator option`

## Task 3：调整 cgo 文件路径规划

**Files:**

- Modify: `internal/generator/render_native_plan.go`
- Modify: `internal/generator/render_native_plan_test.go`
- Modify: `internal/generator/generator_test.go`

**迁移内容与理由：**

- 不迁旧文件名构造代码。
- 参考旧 shared `cgo_dir` 文件名冲突测试思路，重建当前文件族路径规则。

- [x] 从 `GeneratedFilenamePrefix` 取 protobuf Go package 生成目录。
- [x] 用 `path.Join(goPackageDir, cgoDir)` 计算 cgo 输出目录。
- [x] 用 `path.Clean` 清理 `../cmd/rpc` 这类路径。
- [x] `Runtime` 和 `NativeServer` 继续使用原 `GeneratedFilenamePrefix`。
- [x] `CGONativeServer` 和 `CGONativeClient` 使用 cgo 输出目录。
- [x] cgo 文件 stem 保留 proto basename、service name、role，避免同 service 多文件覆盖。
- [x] 添加测试：默认输出到 `test/v1/cgo/`。
- [x] 添加测试：`cgo_dir=../cmd/rpc` 输出到清理后的 `test/cmd/rpc/`。
- [x] 添加测试：非 source-relative paths 模式也能基于 Go package 生成目录计算 cgo 输出路径。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderNativeFileFamilyPlan|TestGenerateWithNativeRenderer' -count=1`。
- [x] 验收：cgo 文件路径符合 `cgo_dir` 合同，非 cgo 文件不受影响。
- [x] 提交：`feat: route cgo files through cgo_dir`

## Task 4：把 cgo renderer 切到 `package main`

**Files:**

- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`

**迁移内容与理由：**

- 参考旧 cgo export 在 `package main` 中生成的边界。
- 重写当前 cgo renderer 的 package/import 输出，不带入旧 bootstrap。

- [x] cgo client generated file 输出 `package main`。
- [x] cgo server generated file 输出 `package main`。
- [x] cgo files 使用 import alias 引用 protobuf Go package。
- [x] cgo files 不再假设自己与 generated runtime 同包。
- [x] 添加 source assertion 测试：cgo 文件包含 `package main`。
- [x] 添加 source assertion 测试：cgo 文件 import protobuf Go package。
- [x] 添加 source assertion 测试：runtime/native server 文件仍使用 protobuf Go package name。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderNative.*CGO|TestGenerateWithNativeRenderer' -count=1`。
- [x] 验收：cgo generated files 的 package 边界符合 `package main` 合同。
- [x] 提交：`feat: generate cgo files in package main`

## Task 5：修正 cgo 跨包调用可见性

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_native_client_cgo.go`
- Modify: `internal/generator/render_native_server_cgo.go`
- Modify: `internal/generator/render_native_server.go`
- Modify: `internal/generator/render_runtime_test.go`
- Modify: `internal/generator/render_native_client_cgo_test.go`
- Modify: `internal/generator/render_native_server_cgo_test.go`

**迁移内容与理由：**

- 不迁旧 runtime export glue。
- 当前 cgo renderer 大量依赖同包未导出符号，切到 `package main` 后必须通过最小跨包 API 调用 generated service runtime。

- [x] 审计 cgo renderer 使用的 runtime helper、dispatcher helper、encoder/decoder、adapter/session helper。
- [x] 对 cgo bridge 必须调用的 service runtime 入口做最小导出。
- [x] 保持内部 helper 未导出，避免扩大 public API 面。
- [x] cgo client 通过 service package alias 调用 exported dispatcher/client bridge。
- [x] cgo server registration 通过 service package alias 调用 exported registration bridge。
- [x] error store 仍使用 `rpcruntime`，error id 保持 `int32`。
- [x] 添加 compile-oriented source assertion，证明 cgo file 不引用 service package 未导出符号。
- [x] 运行 `rtk go test ./internal/generator -run 'TestRenderRuntime|TestRenderNative.*CGO' -count=1`。
- [x] 验收：`package main` cgo 文件不依赖同包未导出符号。
- [x] 提交：`feat: expose minimal cgo bridge API`

## Task 6：补 compile fixture 覆盖默认目录与外部目录

**Files:**

- Create: `internal/integration/cgo_dir_generation_test.go`
- Modify: `internal/integration/native_stage3_acceptance_test.go`

**迁移内容与理由：**

- 迁移旧 `cgo_dir=../cmd/rpc` 边界测试思路。
- 使用当前 generated output 编译验证，而不是只检查字符串。

- [x] 添加 fixture：默认 `cgo_dir` 生成到 Go package 的 `cgo/` 子目录。
- [x] 添加 fixture：`cgo_dir=../cmd/rpc` 生成到 Go package 目录之外。
- [x] fixture 编译包含 protobuf Go package 和 cgo `package main`。
- [x] fixture 验证 cgo `package main` 能 import generated service package。
- [x] fixture 验证 native runtime/native server 文件仍在 protobuf Go package。
- [x] 不纳入 `.vscode/` 或本机环境 workaround。
- [x] 运行 `rtk go test ./internal/integration -run 'TestCGODirGeneration' -count=1`。
- [x] 验收：默认目录和外部目录两种 cgo 生成形态都能编译。
- [x] 提交：`test: cover cgo_dir compile fixtures`

## Task 7：全仓验收与文档收口

**Files:**

- Modify: `docs/plans/2026-04-30-cgo-dir-main-package-plan.md`
- Modify: `AGENTS.md`

**迁移内容与理由：**

- 将实现结果回填到计划 checklist。
- `AGENTS.md` 记录长期生成规则，防止后续阶段把 cgo 文件重新生成到 protobuf Go package。

- [x] 确认 `AGENTS.md` 已记录：cgo 文件必须生成到 `package main`。
- [x] 确认 `AGENTS.md` 已记录：`cgo_dir` 相对 protobuf Go package 生成目录解析。
- [x] 确认 `AGENTS.md` 已记录：`cgo_dir` 默认是 Go package 目录下的 `cgo` 子目录。
- [x] 更新本计划 checkbox。
- [x] 运行 `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`。
- [x] 运行 `rtk go test ./internal/integration -count=1`。
- [x] 运行 `rtk go test ./rpcruntime -count=1`。
- [x] 运行 `rtk go test ./... -count=1`。
- [x] 运行 AGENTS.md 中的 forbidden unsigned scan。
- [x] 验收：生成器输出布局、package 边界、测试和长期规则一致。
- [x] 提交：`docs: record cgo_dir main package contract`

## 完成标准

- generator 接受 `cgo_dir` 参数。
- 默认 `cgo_dir` 为 `cgo`。
- `cgo_dir` 相对 protobuf Go package 生成目录解析。
- `cgo_dir` 可以指向 protobuf Go package 目录之外。
- cgo generated files 生成到 `package main`。
- 非 cgo generated files 继续生成到 protobuf Go package。
- cgo generated files 通过 import alias 调用 generated service package，不依赖同包未导出符号。
- 默认 `cgo/` 和 `cgo_dir=../cmd/rpc` 都有 compile fixture。
- 不引入旧多 registry、多 provider bootstrap、framework selector。
- 不引入 forbidden unsigned 32/64 ABI 类型。
- `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1` 通过。
- `rtk go test ./internal/integration -count=1` 通过。
- `rtk go test ./rpcruntime -count=1` 通过。
- `rtk go test ./... -count=1` 通过。
- AGENTS.md 中的 forbidden unsigned scan 无命中。
