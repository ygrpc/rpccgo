# Use on-demand Native C ABI lowering

rpccgo treats **Native** as one contract. Go native and C native are projections of that same contract, not separate native contracts. The Go native projection expresses Go function parameters and return values. The C native projection expresses cross Go/C ABI slots.

The generator uses shared on-demand `Native C ABI lowering` for C boundary slot shape. It is derived from `NativeContractPlan` and the requested operation. C client and C server renderers share the lowering module instead of persisting a service-level or method-level ABI plan in the generator plan.

The previous persisted `Native C ABI plan` added an attach phase and a second service-level aggregation phase without hiding renderer complexity. Removing that lifecycle simplifies the generator while retaining one lowering implementation.

## Consequences

- C native client exports, C native server callback typedefs/trampolines, registration callback parameters, handle slots, error id slots, length/count slots, ownership slots, output pointer slots, and cleanup capability come from shared on-demand lowering.
- Go native server/client flat function surfaces continue to come from the **Native** Go-level projection.
- The generator does not attach C ABI structures to `MethodPlan` and does not aggregate a persisted `NativeCABIPlan`.
- Service-level callback registration ABI is assembled directly from each method's on-demand lowering result. A renderer may use an ephemeral service-level ABI value while rendering one artifact, provided it is derived on demand and not stored in `GenerationPlan`, `ServicePlan`, or `MethodPlan`.
- The lowering module owns each method's C boundary operation inventory; renderers do not maintain separate unary or streaming operation lists.
- Lowered slots keep only values consumed by renderers: name, C type, cgo Go type, role, and optional field Go name.
- Lowering returns explicit errors for unknown streaming kinds, invalid operations, and service-level registration ABI assembly failures. Renderers propagate errors instead of formatting empty ABI values.
- C native preamble, callback registration, and C export renderers iterate the lowering module's operation inventory. Streaming-kind branches remain only where generated implementations differ.
- Renderers may format lowered slots but must not implement separate field-to-slot mappings.
