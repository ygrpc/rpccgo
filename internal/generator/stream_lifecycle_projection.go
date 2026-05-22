package generator

import "fmt"

type StreamLifecycleProjectionPlan struct {
	SessionKind     SessionKind
	Operations      []SessionOperationPlan
	Terminal        TerminalRenderPlan
	RequiresCodec   bool
	CancelFinalizes bool
}

func ProjectStreamLifecycle(lifecycle StreamLifecycleContractPlan, needsCodec bool) (StreamLifecycleProjectionPlan, error) {
	plan := StreamLifecycleProjectionPlan{SessionKind: SessionKindNone, RequiresCodec: needsCodec}
	if !lifecycle.HasOperation(StreamLifecycleOperationStart) {
		return plan, nil
	}

	hasSend := lifecycle.HasOperation(StreamLifecycleOperationSend)
	hasReceive := lifecycle.HasOperation(StreamLifecycleOperationReceive)
	hasFinish := lifecycle.HasOperation(StreamLifecycleOperationFinish)
	hasDone := lifecycle.HasOperation(StreamLifecycleOperationDone)
	hasCloseSend := lifecycle.HasOperation(StreamLifecycleOperationCloseSend)
	hasCancel := lifecycle.HasOperation(StreamLifecycleOperationCancel)
	if hasCancel && !lifecycle.CancelFinalizes {
		return StreamLifecycleProjectionPlan{}, fmt.Errorf("invalid lifecycle plan: cancel must finalize")
	}

	plan.CancelFinalizes = lifecycle.CancelFinalizes
	if hasSend && hasFinish {
		plan.SessionKind = SessionKindClient
		plan.Operations = []SessionOperationPlan{{Kind: SessionOperationStart}, {Kind: SessionOperationSend}, {Kind: SessionOperationFinish}}
		if hasCancel {
			plan.Operations = append(plan.Operations, SessionOperationPlan{Kind: SessionOperationCancel})
		}
		plan.Terminal = TerminalRenderPlan{Kind: TerminalKindFinish, Operation: SessionOperationFinish, ReleasesHandle: true, RequiresResponseConvert: true}
		return plan, nil
	}
	if hasReceive && hasCloseSend && hasDone {
		plan.SessionKind = SessionKindBidi
		plan.Operations = []SessionOperationPlan{{Kind: SessionOperationStart}, {Kind: SessionOperationSend}, {Kind: SessionOperationReceive}, {Kind: SessionOperationCloseSend}, {Kind: SessionOperationDone}}
		if hasCancel {
			plan.Operations = append(plan.Operations, SessionOperationPlan{Kind: SessionOperationCancel})
		}
		plan.Terminal = TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true}
		return plan, nil
	}
	if hasReceive && hasDone {
		plan.SessionKind = SessionKindServer
		plan.Operations = []SessionOperationPlan{{Kind: SessionOperationStart}, {Kind: SessionOperationReceive}, {Kind: SessionOperationDone}}
		if hasCancel {
			plan.Operations = append(plan.Operations, SessionOperationPlan{Kind: SessionOperationCancel})
		}
		plan.Terminal = TerminalRenderPlan{Kind: TerminalKindDone, Operation: SessionOperationDone, ReleasesHandle: true}
		return plan, nil
	}
	return StreamLifecycleProjectionPlan{}, fmt.Errorf("invalid lifecycle plan")
}
