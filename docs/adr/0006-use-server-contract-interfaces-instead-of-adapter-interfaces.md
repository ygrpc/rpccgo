# Use server contract interfaces instead of adapter interfaces

rpccgo generated service runtime does not expose service-level `NativeAdapter` or `MessageAdapter` interfaces. Generated registration accepts concrete server contract types: `GreeterNativeServer` represents the native server contract implemented by both Go native and C native servers, `GreeterCGOMessageServer` represents the C callback-backed message server contract, and standard connect/gRPC servers or clients are registered as their standard transport types. Registration validates the complete service and builds immutable service-local active bindings whose method closures bind concrete calls and required conversion. Native active bindings and message active bindings are stored and published independently. Invocation does not dispatch by `ServerKind`.

## Consequences

- C message server contracts are generated in a separate protobuf Go package file such as `<proto-prefix>.<service>.server.message.rpccgo.go`, together with their optional `Unimplemented<Service>CGOMessageServer` helper.
- C message server methods use the service method Go name without an extra `Message` suffix or streaming `Start` prefix; the contract is expressed by the server interface name.
- C message server streaming methods use handler-style stream parameters, matching the server contract shape used by native/connect/gRPC servers. Generated C callback glue may still use internal `Start` session functions to project callback operations into the runtime lifecycle.
- Go native server contracts are generated in a separate protobuf Go package file such as `<proto-prefix>.<service>.server.native.rpccgo.go`, together with their native stream interfaces and optional `Unimplemented<Service>NativeServer` helper.
- `Register<Service>GoNativeServer` and the C native registration path use the shared `<Service>NativeServer` contract instead of a generated `<Service>NativeAdapter` interface.
- `Register<Service>CGOMessageServer` uses `<Service>CGOMessageServer` instead of a generated `<Service>MessageAdapter` interface.
