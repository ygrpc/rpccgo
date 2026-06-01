# Preserve flat native boundary with service-local routing

rpccgo keeps the service-local active server architecture, but `native` continues to mean the old project's flat function boundary: proto request/response top-level fields become the final Go/C function parameters and return values. We deliberately do not restore the old provider/bootstrap/go_role architecture; generated caller-facing closures may route calls, but they must not turn native into request/response struct or message-pointer APIs.

## Consequences

- Go native server interfaces, Go native client APIs, C native callbacks, and streaming native operations must all expose flat field-level boundaries.
- `NativeContract`-style field plans may remain as conversion metadata, but generated `Request`/`Response` structs are not a valid native ABI.
- `@rpccgo:native` keeps the new server registration selection behavior, while its native side must satisfy the flat boundary contract.
