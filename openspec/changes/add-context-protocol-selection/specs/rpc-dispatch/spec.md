## ADDED Requirements

### Requirement: Protocol selection is carried in context
系统 SHALL 定义一个稳定的 context key，用于在 dispatch 时携带 protocol 选择。

- The key SHALL be exposed as `rpcruntime.ContextKeyProtocol`.
- The value type SHALL be `rpcruntime.Protocol`.

系统 SHOULD 提供便捷 helper functions：
- `rpcruntime.WithProtocol(ctx, protocol)`
- `rpcruntime.ProtocolFromContext(ctx)`

#### Scenario: Caller selects grpc via context
- **GIVEN** a `ctx`
- **WHEN** the caller sets `rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolGrpc)`
- **THEN** generated adaptor code SHALL be able to read the selection and dispatch using grpc

#### Scenario: Caller selects connectrpc via context
- **GIVEN** a `ctx`
- **WHEN** the caller sets `rpcruntime.WithProtocol(ctx, rpcruntime.ProtocolConnectRPC)`
- **THEN** generated adaptor code SHALL be able to read the selection and dispatch using connectrpc
