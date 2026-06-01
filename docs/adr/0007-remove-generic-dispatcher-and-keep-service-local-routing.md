# Remove generic dispatcher and keep service-local routing

rpccgo removes `rpcruntime.Dispatcher[T]`, runtime bridge structs, stream executors, `StreamEntry`, and registry lifecycle helpers. Generated service runtime owns service-local registration, caller-facing closure construction, native/message conversion, final sessions, and stream registry binding. Runtime core only provides reusable primitives.

Each service has one generated typed atomic pointer storing its immutable service-local active server record and one non-generic stream registry storing final session values directly. Each final session embeds the small `StreamLifecycle` state primitive and the operation closures required by its capability set. Generated package-level functions load the record or session and perform the operation directly.

## Consequences

- `rpcruntime.Dispatcher[T]`, runtime bridge structs, stream lifecycle executors, `StreamEntry`, and `StreamRegistry*` lifecycle helpers should be deleted instead of expanded.
- Runtime core keeps only non-generic `StreamRegistry`, `StreamHandle`, and `StreamLifecycle` stream primitives.
- Generated service runtime uses a typed `atomic.Pointer[record]` directly; `rpcruntime.ActiveServerSlot`, `ServerKind`, `ServerContract`, `AdapterSnapshot`, and version metadata should be deleted.
- One service-local stream registry stores multiple method-specific final sessions directly.
- Runtime core should treat a handle that cannot be used for the requested stream operation as invalid, including wrong method or wrong contract lookups. A separate public session type mismatch error is not part of the stream contract.
- Generated service runtime should keep service-local active slot and stream registry variables instead of introducing a runtime core global registry or restoring the old Provider bootstrap model.
- `Finish` is the only graceful terminal operation. `CloseSend` remains the bidi half-close operation. `Cancel` remains the abort terminal operation.
