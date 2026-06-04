# Use runtime server registry for current server

rpccgo uses a `rpcruntime` server registry keyed by generated `ServiceID` to store the current registered server record for each service. Registration remains exposed through generated service helpers, which validate service-specific contracts, choose a `rpcruntime.ServerKind`, and write `{Kind, Server}` into the registry; generated invoke/start facades read that record and perform service-specific type assertions, direct calls, and Native/Message conversion.

Streaming sessions are also owned by `rpcruntime`: generated `Start` facades create service-specific typed sessions, then store a runtime `{ServerKind, session}` stream session record in the global runtime stream session registry. Later generated package-level stream operation functions load that record by handle and perform service-specific typed dispatch and Native/Message conversion.

## Consequences

- Each service has exactly one current registered server. Go native, cgo native, cgo message, connect, gRPC, connect remote, and gRPC remote registrations all replace the same `ServiceID` record.
- `rpcruntime.ServerKind` is a fixed runtime enum for Go native, cgo native, cgo message, connect, gRPC, connect remote, and gRPC remote. It describes server shape only; it does not perform service-specific dispatch or protobuf/native conversion.
- Generated registration helpers remain the user API. Users do not hand-write `ServiceID` or `ServerKind`; generated helpers call runtime register/clear primitives internally.
- Registration failure clears the current registered server for that `ServiceID` and returns an error, so later calls see no registered server instead of silently continuing to use an old server.
- Generated service runtime must not generate service-local native/message active binding slots, service-wide binding closure tables, or per-contract current server values.
- Unary calls read the current registry record on each call. Stream `Start` reads the current record once, creates a stream session, and stores `{ServerKind, session}` in the global `rpcruntime` stream session registry; later stream operations use only that stream session and do not reread the server registry.
- Stream sessions do not use operation closures or a generic lifecycle state machine. Terminal operations remove the handle from the global `rpcruntime` stream session registry; later operations on that handle return invalid-handle.
- Generated code must not emit service-local stream registries, method-specific final session record types, or stream handle facade structs whose only state is `rpcruntime.StreamHandle`.
- Generated stream operations should be package-level functions that accept `rpcruntime.StreamHandle` directly. They load the runtime stream session record and keep all service-specific method calls, type assertions, and Native/Message conversion in generated code.
- Standard connect/gRPC servers and remote clients are registered directly as their standard types. rpccgo does not generate local connect/gRPC transport ingress files, remote adapter files, or HTTP loopback paths for C-to-Go calls.
- Generated server contract interfaces remain the typed user surface: `GreeterNativeServer` for Go/cgo native, `GreeterCGOMessageServer` for cgo message, and standard connect/gRPC interfaces or clients for transport sources. Unimplemented helper structs remain valid for partial user implementations.
- `rpcruntime.Dispatcher[T]`, runtime forwarding structs, stream executors, registry lifecycle helper layers, `ActiveServerSlot`, `AdapterSnapshot`, service-local stream registries, and service-local active binding records should not be introduced.
