# rpccgo Project Roadmap

## 目标

本计划描述新版 rpccgo 的项目级建设阶段。计划只定义每个阶段的任务范围、旧项目可迁移部分和验收目标，不展开具体实现步骤。

新版架构以 generated service 为边界，核心模型是 cgo client 调用进入 dispatcher，dispatcher 根据当前 active server 和 native/message contract 完成路由与转换。完整架构见 `docs/specs/2026-04-27-rpccgo-modular-dispatcher-architecture.md`。

## 阶段 0：项目骨架与迁移基线

### 任务

- 建立新版仓库的基础目录和模块边界。
- 建立 generator、runtime、integration test、example 的入口结构。
- 从旧项目中筛选可迁移代码，并标记只能参考不能照搬的旧架构部分。
- 建立最小 `go test ./...` 验证入口。

### 旧项目可迁移部分

- `rpcruntime` 的错误存储、长度校验、free/cleanup。
- `RpcBytes`、`RpcString`、`RpcRepeat` 及其测试。
- runtime wrapper 相关的基础测试用例。

### 验收目标

- 新版仓库具备清晰的 generator/runtime/test/example 目录边界。
- runtime 基础能力迁移后测试通过。
- 旧项目迁移清单明确，旧的多 registry、多 provider bootstrap 模型不进入新版基线。

## 阶段 1：Proto 解析与生成计划层

### 任务

- 实现 protoc 插件输入解析。
- 建立 service、method、field、framework、streaming kind 的 metadata 模型。
- 建立 native contract、message contract 和生成计划层。
- 为 unary、client streaming、server streaming、bidi streaming 生成稳定的中间 plan。

### 旧项目可迁移部分

- `options`。
- `binding`。
- `namespace`。
- `frameworks`。
- `streaming_kind`。
- `native_types`。
- `native_codec`。
- `streaming_plan` 的核心思路。

### 验收目标

- generator 能从 proto 生成稳定、可测试的中间 plan。
- unary 和三类 streaming method 都能被正确识别。
- native/message contract 信息足够支撑后续 dispatcher、converter 和 adapter 生成。

## 阶段 2：Runtime Dispatcher Foundation

### 任务

- 实现 active server slot。
- 实现 dispatcher 抽象。
- 实现 server adapter 抽象。
- 实现 stream session 生命周期底座。
- 建立 active server capture、handle lookup、cancel/finalize 的统一语义。

### 旧项目可迁移部分

- `active_slot` 可作为参考。
- stream handle/session 相关代码可择优迁移。
- 旧项目的 streaming lifecycle 测试思路可复用。

### 验收目标

- 单个 service 可以注册一个 active server。
- 后注册 server 覆盖后续调用。
- 已启动 stream 绑定启动时捕获的 server adapter。
- runtime foundation 不依赖 service-specific protobuf 类型。

## 阶段 3：Go Native Server 与 cgo Native Server

### 任务

- 实现 go native server adapter。
- 实现 cgo native server callback ABI。
- 实现 cgo native client 到 dispatcher 的调用路径。
- 接入 native unary 和 native streaming。

### 旧项目可迁移部分

- `native_server`。
- `native_bridge`。
- `native_bridge_cgo`。
- `native_runtime_cgo`。
- native codec 相关测试。

### 验收目标

- cgo native client 能通过 dispatcher 调用 go native server。
- cgo native client 能通过 dispatcher 调用 cgo native server。
- native unary、client streaming、server streaming、bidi streaming 都具备端到端验证。

## 阶段 4：cgo Message Server 与 Message Client

### 任务

- 实现 protobuf bytes ABI。
- 实现 cgo message client 到 dispatcher 的调用路径。
- 实现 cgo message server adapter。
- 接入 message unary 和 message streaming。
- 接入 native/message contract 不匹配时的双向转换。

### 旧项目可迁移部分

- `message_client`。
- `message_server`。
- `message_export_shim_cgo` 的 ABI 与 streaming callback 思路。
- message mode integration fixture。

### 验收目标

- cgo message client 能通过 dispatcher 调用 cgo message server。
- cgo message client 能通过 dispatcher 调用 native server，并完成 protobuf 到 native 的转换。
- cgo native client 能通过 dispatcher 调用 message server，并完成 native 到 protobuf 的转换。
- message unary、client streaming、server streaming、bidi streaming 都具备端到端验证。

## 阶段 5：Connect 与 gRPC 本地 Server Adapter

### 任务

- 实现 connect handler adapter。
- 实现 grpc server adapter。
- 让标准 connect/grpc 入站请求进入 dispatcher。
- 保持 connect/grpc 接口复用，不引入私有 transport contract。

### 旧项目可迁移部分

- `message_server` 中 connect/grpc handler 适配逻辑。
- connect/grpc 相关 integration fixture。
- framework 选择与注册的测试思路。

### 验收目标

- 一个 generated service 可以通过 connect 监听入口接收请求并进入 dispatcher。
- 一个 generated service 可以通过 grpc 监听入口接收请求并进入 dispatcher。
- 本地 connect/grpc 入站请求可以路由到当前 active server。
- connect/grpc streaming 请求使用统一 stream lifecycle。

## 阶段 6：Connect 与 gRPC Remote Server Adapter

### 任务

- 实现 connect remote server adapter。
- 实现 grpc remote server adapter。
- adapter 内部复用标准 connect/grpc client。
- 支持 unary 与三类 streaming remote 调用。

### 旧项目可迁移部分

- `forwarding_plan`。
- `native_forwarding_client`。
- `native_forwarding_server`。
- native forwarding integration tests 中的 transport adapter 与 streaming 覆盖思路。

### 验收目标

- dispatcher 可以把调用路由到远端 connect server。
- dispatcher 可以把调用路由到远端 grpc server。
- remote unary、client streaming、server streaming、bidi streaming 都具备端到端验证。
- remote adapter 不重新定义 connect/grpc client 模型。

## 阶段 7：统一生成物与端到端示例

### 任务

- 整理最终生成文件布局。
- 固化 public API 命名。
- 建立最小但完整的 example 工程。
- 提供从 proto 到生成、注册 server、启动监听、发起 cgo 调用的完整路径。

### 旧项目可迁移部分

- 旧 examples 的 proto。
- 部分 backend fixture。
- 部分 integration tests。

### 验收目标

- 用户可以从 proto 生成新版 rpccgo 代码。
- 用户可以启动一个监听 server。
- 用户可以注册一种 active server。
- 用户可以用 cgo native client 和 cgo message client 完成调用。
- example 不保留旧项目的双 provider bootstrap 模型。

## 阶段 8：兼容性、清理与发布准备

### 任务

- 补齐错误语义、内存释放、stream cancel/finalize、空输入、repeated wrapper 等验证。
- 清理不符合新版架构的旧概念。
- 对齐 README、架构文档、项目计划和实际行为。
- 建立发布前验证命令集合。

### 旧项目可迁移部分

- runtime wrapper 测试。
- message/native/native_forwarding integration tests 中的高价值用例。
- repeated wrapper、empty input、marshal/unmarshal error propagation 相关测试思路。

### 验收目标

- `go test ./...` 通过。
- 核心 examples 通过。
- 文档与实际行为一致。
- 不存在多 active server、多 provider bootstrap 或绕过 dispatcher 的新调用路径。
