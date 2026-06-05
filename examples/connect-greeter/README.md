# Connect Greeter Example

这个 example 使用同一个 greeter service 演示 Connect 相关 server 切换：

- Connect handler
- Connect remote server
- cgo message server
- cgo native server

service 覆盖 unary、client streaming、server streaming 和 bidi streaming。remote server 是独立 `cmd/server` 进程，demo 通过标准 Connect client 经网络栈调用它。

## 一键运行

```bash
go run github.com/magefile/mage run
```

`mage run` 会重新生成代码，构建 c-shared library，启动独立 remote server，然后按顺序切换 current registered server 并运行同一套 C native client 调用。

输出会包含四段：

```text
== switch to connect handler registered server ==
== switch to connect remote registered server ==
== switch to cgo message registered server ==
== switch to cgo native registered server ==
```

每一段都会运行同一组 native C client 调用：

```text
native unary: hello ffi from c
native collect: collect:ada,grace
native broadcast: broadcast[0]:stream
native broadcast: broadcast[1]:stream
native chat c->server: ada
native chat server->c: chat:ada
native chat c->server: grace
native chat server->c: chat:grace
```

`native chat c->server` 表示 C client 向 server 发送 bidi stream 消息；`native chat server->c` 表示 server 返回给 C client 的消息。

## 测试

```bash
go run github.com/magefile/mage test
```

也可以直接运行 Go tests：

```bash
go test ./...
```

## 手动运行 Remote Server

启动 Connect server：

```bash
go run ./cmd/server --addr 127.0.0.1:8081
```

在另一个 shell 运行标准 Connect client：

```bash
go run ./cmd/client --url http://127.0.0.1:8081
```

## 参数

example 不使用环境变量切换 server；切换通过命令行参数完成。

`mage run` 内部传递这些参数给 C demo：

- `--server=connect_handler`：注册 Connect handler。
- `--server=cgo_message`：从 C 注册 cgo message server。
- `--server=cgo_native`：从 C 注册 cgo native server。
- `--connect-url=http://host:port`：注册标准 Connect client，调用独立 remote server。
- `--route=<label>`：打印当前路由标签，方便区分输出属于哪个 registered server。

`cmd/server` 支持：

- `--addr=127.0.0.1:8081`：监听地址。

## 生成

该 example 的生成命令写在 [gen.go](gen.go)：

```bash
go generate ./...
```

cgo 文件通过 `--rpc-cgo_opt=cgo_dir=../cmd/rpc` 生成到 `cmd/rpc`，用于构建 `-buildmode=c-shared`。
