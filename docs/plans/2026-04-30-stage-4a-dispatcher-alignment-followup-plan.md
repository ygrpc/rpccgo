# Stage 4A Dispatcher Alignment Follow-up Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `subagent-driven-development` or `executing-plans` task-by-task. This plan fixes the drift found before Stage 4B; do not start Stage 4B converter work until this plan is complete.

## Goal

Bring Stage 3/4A generated runtime back to the architecture contract: each generated service has one active server decision point, mixed native/message services render both cgo client families, and Stage 4A message direct path has real compile/runtime coverage before Stage 4B converter work starts.

## Status

Completed on 2026-04-30. Stage 4B can now build on the single generated service dispatcher shape instead of the earlier dual-dispatcher drift.

## Problem Summary

Current Stage 0/1/2 are aligned with their plans. The drift starts in Stage 3/4A:

- Generated runtime currently emits native and message dispatchers separately.
- Mixed services can skip message rendering because `RenderStageFiles` stops after native rendering.
- Message unary mismatch does not return the same explicit mismatch error as streaming paths.
- Stage 4A acceptance mostly asserts generated source fragments, not real cgo message direct-path behavior.

## Architecture Decision

Keep `rpcruntime.Dispatcher[T]` generic and service-agnostic. Generated service runtime will define a service-specific union adapter type that can hold either a native adapter or a message adapter. A single `rpcruntime.Dispatcher[<Service>ActiveAdapter]` becomes the only active server slot for that service.

Generated bridges route by `snapshot.Contract`:

- Native client + native active server: direct native path.
- Message client + message active server: direct message path.
- Message client + native active server: return explicit converter-disabled mismatch error for now.
- Native client + message active server: return explicit converter-disabled mismatch error for now.

Stage 4B will replace the mismatch error branches with generated codec conversion.

## Task 1: Generate a single service dispatcher

**Files:**

- Modify: `internal/generator/render_runtime.go`
- Modify: `internal/generator/render_runtime_test.go`

**Checklist:**

- [x] Replace separate native/message dispatcher variables with one `rpcruntime.Dispatcher[<Service>ActiveAdapter]`.
- [x] Generate `<Service>ActiveAdapter` with `Native <Service>NativeAdapter` and `Message <Service>MessageAdapter` fields.
- [x] Keep native and message adapter interfaces separate.
- [x] Registration helpers write to the same dispatcher with different `ServerContract` values.
- [x] Native bridge checks `snapshot.Contract == rpcruntime.ServerContractNative` before direct call.
- [x] Message bridge checks `snapshot.Contract == rpcruntime.ServerContractMessage` before direct call.
- [x] Mismatch branches return stable converter-disabled errors in both directions.
- [x] Stream helpers load/take/delete from the single dispatcher and type-check wrapper sessions.
- [x] Tests assert only one dispatcher variable is generated and no `<service>MessageDispatcher` remains.

## Task 2: Fix mixed service rendering

**Files:**

- Modify: `internal/generator/render.go`
- Modify: `internal/generator/generator_test.go`
- Modify: `internal/generator/render_message_plan_test.go`
- Modify: `internal/generator/render_native_plan_test.go`

**Checklist:**

- [x] `RenderStageFiles` renders shared runtime once.
- [x] Native server/cgo native server render when `AdapterTokenNative` is enabled.
- [x] Cgo native client always renders according to native client policy.
- [x] Cgo message client always renders according to message client policy.
- [x] Cgo message server renders when message server adapter is enabled.
- [x] Mixed `@rpccgo:native` service outputs native and message cgo clients without duplicate runtime files.
- [x] No connect/grpc/remote files are generated.
- [x] File naming collision between native/message cgo files is resolved by unique message cgo file family names.

## Task 3: Add Stage 4A real message direct-path coverage

**Files:**

- Create or modify: `internal/integration/message_unary_test.go`
- Create or modify: `internal/integration/message_client_streaming_test.go`
- Create or modify: `internal/integration/message_server_streaming_test.go`
- Create or modify: `internal/integration/message_bidi_streaming_test.go`
- Modify: `internal/integration/message_stage4a_acceptance_test.go`

**Checklist:**

- [x] Compile generated runtime, cgo message client, and cgo message server in a temp module.
- [x] Cover unary message client -> cgo message server direct path.
- [x] Cover client streaming direct path.
- [x] Cover server streaming direct path.
- [x] Cover bidi streaming direct path.
- [x] Cover invalid request bytes and callback error propagation.
- [x] Cover handle finalization after finish/done/cancel.
- [x] Cover native active server mismatch returns converter-disabled error for message unary and streaming.

## Task 4: Add message cgo_dir compile coverage

**Files:**

- Modify: `internal/integration/cgo_dir_generation_test.go`

**Checklist:**

- [x] Default `cgo/` directory compile fixture includes message cgo files.
- [x] External `cgo_dir=../cmd/rpc` compile fixture includes message cgo files.
- [x] Fixtures prove `package main` message files import generated service package correctly.

## Task 5: Update plan and inventory documentation

**Files:**

- Modify: `docs/plans/2026-04-28-stage-3-native-contract-plan.md`
- Modify: `docs/plans/2026-04-30-stage-4a-message-contract-plan.md`
- Modify: `docs/plans/2026-04-30-stage-4a-migration-inventory.md`
- Modify: `docs/plans/2026-04-30-stage-4b-native-message-converter-plan.md`

**Checklist:**

- [x] Record that Stage 3/4A follow-up restored the single dispatcher contract.
- [x] Record real Stage 4A direct-path verification commands.
- [x] Update Stage 4B plan to depend on this follow-up instead of assuming the old state.
- [x] Do not write machine environment workarounds into docs.

## Verification

- [ ] `rtk go test ./internal/generator ./cmd/protoc-gen-rpc-cgo -count=1`
- [ ] `rtk go test ./internal/integration -count=1`
- [ ] `rtk go test ./rpcruntime -count=1`
- [ ] `rtk go test ./... -count=1`
- [ ] AGENTS.md forbidden unsigned scan.

## Done Criteria

- Generated runtime has one dispatcher per service.
- Mixed native/message service renders both cgo client families.
- Message direct path has compile/runtime integration coverage.
- Mismatch branches are explicit and ready for Stage 4B converter replacement.
- Stage 4B plan no longer rests on the stale dual-dispatcher implementation.
