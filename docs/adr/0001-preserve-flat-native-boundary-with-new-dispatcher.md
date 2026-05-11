# Preserve flat native boundary with the new dispatcher

rpccgo keeps the new dispatcher / active server architecture, but `native` continues to mean the old project's flat function boundary: proto request/response top-level fields become the final Go/C function parameters and return values. We deliberately do not restore the old provider/bootstrap/go_role architecture; the new dispatcher may route calls, but it must not turn native into request/response struct or message-pointer APIs.

## Consequences

- Go native server interfaces, Go native client APIs, C native callbacks, and streaming native operations must all expose flat field-level boundaries.
- `NativeContract`-style field plans may remain as conversion metadata, but generated `Request`/`Response` structs are not a valid native ABI.
- `@rpccgo:native` keeps the new adapter selection behavior, while its native side must satisfy the flat boundary contract.
