# StreamRegistry Choreography Deepening Plan

## Goal

Deepen `rpcruntime.StreamRegistry` usage so generated service runtime no longer repeats stream session choreography in every generated stream facade.

The current generated runtime repeats this pattern per method and caller contract:

1. `Load(handle)`
2. type assert to the method-specific session type
3. lifecycle operation check
4. `Take(handle)` for terminal operations
5. compare the taken value with the loaded session
6. finalize or cancel lifecycle
7. call the method-specific closure

The target is to move the common registry and lifecycle choreography into Runtime core while keeping method-specific native/message closures in Generated service runtime.

## Decisions

- Keep one service-local `rpcruntime.StreamRegistry`.
- Keep storing generated method-specific stream session values directly in the registry.
- Do not make `StreamRegistry` generic, because one registry stores multiple method/contract session types.
- Add generic Runtime core functions that operate on `*StreamRegistry` and `StreamHandle`.
- Rename generated internal session structs from `FinalSession` to `StreamSession`.
- Keep generated session names method-specific and contract-specific:
  - before: `greeterCollectNativeFinalSession`
  - after: `greeterCollectNativeStreamSession`
  - before: `greeterCollectMessageFinalSession`
  - after: `greeterCollectMessageStreamSession`

## Runtime Core Shape

Add generic helper functions in `rpcruntime`, near `stream_registry.go` or a new adjacent file.

Proposed operations:

- `LoadStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`
- `SendStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`
- `CloseSendStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`
- `RecvStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`
- `FinishStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`
- `CancelStreamSession[T any](registry *StreamRegistry, handle StreamHandle) (T, error)`

The exact names can change during implementation, but the important split is:

- non-terminal operations load and type-check the session
- send and close-send also apply send-side lifecycle checks
- finish takes the session and finalizes it once
- cancel takes the session and marks it canceled once
- unknown handle, zero handle, wrong session type, repeated terminal operation, and lost take all return `ErrStreamInvalidHandle` unless `StreamLifecycle` returns a more specific lifecycle error for the current operation

## Generated Runtime Shape

Generated stream facade methods should stop emitting raw registry choreography.

Before:

```go
value, ok := greeterStreamRegistry.Load(s.handle)
if !ok { return rpcruntime.ErrStreamInvalidHandle }
session, ok := value.(*greeterCollectNativeFinalSession)
if !ok { return rpcruntime.ErrStreamInvalidHandle }
if err := session.lifecycle.EnsureCanSend(); err != nil { return err }
return session.send(ctx, name)
```

After:

```go
session, err := rpcruntime.SendStreamSession[*greeterCollectNativeStreamSession](&greeterStreamRegistry, s.handle)
if err != nil { return err }
return session.send(ctx, name)
```

Finish with native response fields should use the same helper but preserve native zero-return formatting:

```go
session, err := rpcruntime.FinishStreamSession[*greeterCollectNativeStreamSession](&greeterStreamRegistry, s.handle)
if err != nil { return false, nil, err }
return session.finish(ctx)
```

Cancel should use Runtime core choreography before calling the method-specific cancel closure:

```go
session, err := rpcruntime.CancelStreamSession[*greeterCollectNativeStreamSession](&greeterStreamRegistry, s.handle)
if err != nil { return err }
return session.cancel(ctx)
```

## Implementation Steps

1. Add focused red tests in `rpcruntime/stream_registry_test.go`.
   - typed load succeeds for matching session type
   - typed load rejects unknown handle
   - typed load rejects wrong session type
   - finish takes the session and rejects repeated finish
   - cancel takes the session and rejects repeated cancel
   - send rejects after close-send/finalize/cancel using existing lifecycle errors

2. Implement Runtime core generic helpers.
   - Keep `StreamRegistry.Create/Load/Take/Delete` for low-level use.
   - Reuse existing `StreamLifecycle`.
   - Avoid `panic`; return explicit errors.

3. Update generator naming.
   - Rename `renderRuntimeFinalSessions` to a `StreamSession` naming helper.
   - Change `runtimeFinalNativeSessionName` / `runtimeFinalMessageSessionName` output suffix from `FinalSession` to `StreamSession`.
   - Update generated active record fields and start closures accordingly.

4. Update generated stream facade renderer.
   - Replace `renderRuntimeLoadSession*`, `renderRuntimeTakeSession*`, and inline lifecycle calls with Runtime core generic helpers.
   - Keep native/message return shape logic in generated code.
   - Delete renderer helpers that only emit old registry choreography.

5. Update generator tests.
   - Replace assertions expecting `FinalSession`.
   - Assert generated code calls `rpcruntime.SendStreamSession`, `RecvStreamSession`, `FinishStreamSession`, `CloseSendStreamSession`, and `CancelStreamSession`.
   - Assert generated code no longer emits `StreamRegistry.Load` / `Take` choreography inside stream facade methods.

6. Regenerate examples if generated files are tracked.
   - Update connect and grpc greeter generated runtime files.
   - Ensure generated examples no longer contain `FinalSession`.

## Verification

Run focused checks first:

```bash
rtk go test ./rpcruntime -count=1
rtk go test ./internal/generator -count=1
```

Then run regular validation:

```bash
rtk go test ./...
```

Because this touches Runtime core and generated runtime, also run the unsigned auxiliary type scan:

```bash
rtk rg -n "uint32|uint64|Uint32|Uint64|u32|u64|uint32_t|uint64_t" . -g '!AGENTS.md' -g '!docs/release/verification-checklist.md'
```

## Risks

- Generic helper names may become too operation-specific. If implementation feels noisy, collapse to fewer helpers only when generated code still stops repeating choreography.
- Finish and cancel ordering must stay exact: take/remove the session before calling terminal closure, so repeated terminal operations fail by handle lookup.
- CloseSend should not take the session; it is a half-close operation for bidi streaming.
- Recv should not take or finalize; server streaming can receive repeatedly until the source returns its terminal error.
- Native finish helpers must preserve existing zero-return behavior for response fields.
