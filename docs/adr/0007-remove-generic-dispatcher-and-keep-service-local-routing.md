# Remove generic dispatcher and keep service-local routing

rpccgo will remove `rpcruntime.Dispatcher[T]` and the `DispatcherStream*` lifecycle helpers as public Runtime core modules. Generated service runtime already owns service-local Active server routing, native/message conversion, and stream registry binding, so a generic dispatcher that combines `ActiveServerSlot` and stream storage is a shallow Module that cannot express the service-specific Runtime bridge contract.

This updates the implementation direction from ADR-0003: Runtime core still owns common Stream lifecycle state and operation semantics, but the main seam is `StreamRegistry[*StreamEntry]` plus the `StreamRegistry*` lifecycle operations, not a generic dispatcher value. Runtime core keeps `ActiveServerSlot`, `StreamRegistry`, `StreamEntry`, and lifecycle operations; Generated service runtime continues to compose the active slot, stream registry, Runtime bridge, typed stream facades, and converter glue locally.

## Consequences

- `rpcruntime.Dispatcher[T]`, `DispatcherStreamSend`, `DispatcherStreamReceive`, `DispatcherStreamFinish`, `DispatcherStreamDone`, `DispatcherStreamCancel`, and `DispatcherStreamCloseSend` should be deleted instead of expanded.
- `StreamEntry` remains non-generic because one service-local stream registry stores multiple method-specific native, message, and wrapper sessions.
- Runtime core should distinguish session type mismatch from invalid handles with a dedicated sentinel such as `ErrStreamSessionTypeMismatch`.
- Generated service runtime should keep using service-local `ActiveServerSlot[any]` and `StreamRegistry[*StreamEntry]` instead of introducing a runtime core global registry or restoring the old Provider bootstrap model.
