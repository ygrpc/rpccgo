# Use Native C ABI plan as the C boundary slot source

rpccgo treats **Native** as one contract. Go native and C native are projections of that same contract, not separate native contracts. The Go native projection expresses Go function parameters and return values. The C native projection expresses cross Go/C ABI slots.

`Native C ABI plan` is the sole source of C boundary slot shape. It must be derived from `NativeContractPlan` / method render planning data, and renderers may format `CABISlot` values but must not reinterpret `FieldPlan` into C ABI slots.

## Consequences

- C native client exports, C native server callback typedefs/trampolines, registration callback parameters, handle slots, error id slots, length/count slots, ownership slots, output pointer slots, and cleanup capability all come from `Native C ABI plan`.
- Go native server/client flat function surfaces continue to come from the **Native** Go-level projection, not from `Native C ABI plan`.
- `CABISlot.Source` records the minimal Native field facts needed to prove where a field-derived C slot came from. Proto-unrelated slots such as handle and error id keep nil source and are identified by role.
- Renderers can keep small formatters for `CABISlot`, but C ABI shape decisions must remain in the plan so C client and C server output cannot drift apart.
