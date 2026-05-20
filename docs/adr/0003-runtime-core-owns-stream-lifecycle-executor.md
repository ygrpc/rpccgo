# Runtime core owns stream lifecycle execution

rpccgo keeps streaming operation sets as generator planning data, but moves generic stream lifecycle execution into the Runtime core. The generator produces a contract-level Stream lifecycle plan for each streaming method. Generated service runtime projects that plan into method-specific typed facades. Those facades bind active routing, native/message conversion, session callbacks, and error mapping, but they do not compose stream registry load/take/delete primitives to implement lifecycle ordering.

The Runtime core provides a generic Stream lifecycle executor owned by the service dispatcher. It stores runtime-owned lifecycle entries rather than bare typed sessions, and centralizes handle lookup, take/release, terminal-once, invalid-handle, send-closed, finalized/canceled, and cancel/terminal finalization semantics. Generated code calls the executor through typed facades instead of directly calling stream registry primitives.

## Consequences

- Native flat function boundaries and the dispatcher / Active server architecture remain unchanged.
- `StreamLifecyclePlan` in the generator is the source of truth for method operation sets and terminal policy.
- `rpcruntime.StreamLifecycle` remains a runtime state machine, distinct from the generator plan.
- Runtime core defines common stream lifecycle errors instead of generating per-method lifecycle sentinel errors.
- Generated service runtime keeps method-specific typed facades when they bind conversion, routing, callbacks, or error mapping.
- Generated service runtime must not generate per-method `load/take/delete` wrappers or manually sequence registry primitives for lifecycle semantics.
- Existing unpublished generated output may change; compatibility with the current implementation shape is not required.
