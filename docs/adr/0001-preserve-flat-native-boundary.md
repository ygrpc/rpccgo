# Preserve flat native boundary

`native` means the old project's flat function boundary: proto request/response top-level fields become the final Go/C function parameters and return values. The runtime server registry may route a call to native or message-shaped servers, but generated native APIs must not turn native into request/response struct or message-pointer APIs.

## Consequences

- Go native server interfaces, Go native client APIs, C native callbacks, and streaming native operations must all expose flat field-level boundaries.
- `NativeContract`-style field plans may remain as conversion metadata, but generated `Request`/`Response` structs are not a valid native ABI.
- `@rpccgo:native` keeps the new service generation selection behavior, while its native side must satisfy the flat boundary contract.
