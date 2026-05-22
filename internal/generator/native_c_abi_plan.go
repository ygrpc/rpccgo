package generator

import "fmt"

type NativeCABIPlan struct {
	Methods []MethodNativeCABIPlan
}

type MethodNativeCABIPlan struct {
	MethodFullName string
	Operations     []COperationABI
}

type NativeCOperation string

const (
	NativeCOperationUnary     NativeCOperation = "unary"
	NativeCOperationStart     NativeCOperation = "start"
	NativeCOperationSend      NativeCOperation = "send"
	NativeCOperationRecv      NativeCOperation = "recv"
	NativeCOperationFinish    NativeCOperation = "finish"
	NativeCOperationCloseSend NativeCOperation = "close_send"
	NativeCOperationDone      NativeCOperation = "done"
	NativeCOperationCancel    NativeCOperation = "cancel"
	NativeCOperationRegister  NativeCOperation = "register"
)

type COperationABI struct {
	Operation NativeCOperation
	Symbol    string
	TypeName  string
	Params    []CABISlot
	Return    CABISlot
}

type CABISlot struct {
	Source  *NativeFieldRef
	Name    string
	CType   string
	Role    CABISlotRole
	Cleanup CABICleanup
}

type NativeFieldRef struct {
	ProtoName string
	GoName    string
	CName     string
	GoType    string
	Kind      FieldKind
	Scalar    bool
}

type CABISlotRole string

const (
	CABISlotRoleValue      CABISlotRole = "value"
	CABISlotRolePointer    CABISlotRole = "pointer"
	CABISlotRoleLength     CABISlotRole = "length"
	CABISlotRoleCount      CABISlotRole = "count"
	CABISlotRoleOutValue   CABISlotRole = "out_value"
	CABISlotRoleOutPointer CABISlotRole = "out_pointer"
	CABISlotRoleOutLength  CABISlotRole = "out_length"
	CABISlotRoleOutCount   CABISlotRole = "out_count"
	CABISlotRoleHandle     CABISlotRole = "handle"
	CABISlotRoleErrorID    CABISlotRole = "error_id"
	CABISlotRoleCallback   CABISlotRole = "callback"
)

type CABICleanup string

const (
	CABICleanupNoCleanup       CABICleanup = "no_cleanup"
	CABICleanupFreeWithRuntime CABICleanup = "free_with_runtime"
)

func BuildNativeCABIPlan(service ServicePlan) (NativeCABIPlan, error) {
	methods := make([]MethodNativeCABIPlan, 0, len(service.Methods))
	for _, method := range service.Methods {
		if method.Contract.NativeCABI.MethodFullName == "" {
			return NativeCABIPlan{}, fmt.Errorf("method %s native C ABI plan is missing", methodPlanName(method))
		}
		methods = append(methods, method.Contract.NativeCABI)
	}
	return NativeCABIPlan{Methods: methods}, nil
}

func BuildMethodNativeCABIPlan(plan FilePlan, service ServicePlan, method MethodPlan) (MethodNativeCABIPlan, error) {
	builder := nativeCABIBuilder{file: plan, service: service, method: method}
	switch method.Streaming {
	case StreamingKindUnary:
		return MethodNativeCABIPlan{MethodFullName: method.FullName, Operations: []COperationABI{builder.unary(), builder.register()}}, nil
	case StreamingKindClientStreaming:
		return MethodNativeCABIPlan{MethodFullName: method.FullName, Operations: []COperationABI{builder.startOutHandle(), builder.send(), builder.finish(), builder.cancel(), builder.register()}}, nil
	case StreamingKindServerStreaming:
		return MethodNativeCABIPlan{MethodFullName: method.FullName, Operations: []COperationABI{builder.serverStreamStart(), builder.recv(), builder.done(), builder.cancel(), builder.register()}}, nil
	case StreamingKindBidiStreaming:
		return MethodNativeCABIPlan{MethodFullName: method.FullName, Operations: []COperationABI{builder.startOutHandle(), builder.send(), builder.recv(), builder.closeSend(), builder.done(), builder.cancel(), builder.register()}}, nil
	default:
		return MethodNativeCABIPlan{}, fmt.Errorf("method %s: unsupported native C ABI streaming kind %q", method.FullName, method.Streaming)
	}
}

type nativeCABIBuilder struct {
	file    FilePlan
	service ServicePlan
	method  MethodPlan
}

func (b nativeCABIBuilder) unary() COperationABI {
	params := b.inputSlots(b.method.Contract.Native.RequestFields)
	params = append(params, b.outputSlots(b.method.Contract.Native.ResponseFields)...)
	return COperationABI{Operation: NativeCOperationUnary, Symbol: nativeCExportFuncName(b.file, b.service, b.method, ""), TypeName: nativeCGOServerCallbackName(b.service, b.method), Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) startOutHandle() COperationABI {
	return COperationABI{Operation: NativeCOperationStart, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "start"), TypeName: b.callbackTypeName(NativeCOperationStart), Params: []CABISlot{outHandleSlot("stream")}, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) serverStreamStart() COperationABI {
	params := b.inputSlots(b.method.Contract.Native.RequestFields)
	params = append(params, outHandleSlot("stream"))
	return COperationABI{Operation: NativeCOperationStart, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "start"), TypeName: b.callbackTypeName(NativeCOperationStart), Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) send() COperationABI {
	params := []CABISlot{handleSlot("stream")}
	params = append(params, b.inputSlots(b.method.Contract.Native.RequestFields)...)
	return COperationABI{Operation: NativeCOperationSend, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "send"), TypeName: b.callbackTypeName(NativeCOperationSend), Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) recv() COperationABI {
	params := []CABISlot{handleSlot("stream")}
	params = append(params, b.outputSlots(b.method.Contract.Native.ResponseFields)...)
	return COperationABI{Operation: NativeCOperationRecv, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "read"), TypeName: b.callbackTypeName(NativeCOperationRecv), Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) finish() COperationABI {
	params := []CABISlot{handleSlot("stream")}
	params = append(params, b.outputSlots(b.method.Contract.Native.ResponseFields)...)
	return COperationABI{Operation: NativeCOperationFinish, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "finish"), TypeName: b.callbackTypeName(NativeCOperationFinish), Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) closeSend() COperationABI {
	return COperationABI{Operation: NativeCOperationCloseSend, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "close_send"), TypeName: b.callbackTypeName(NativeCOperationCloseSend), Params: []CABISlot{handleSlot("stream")}, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) done() COperationABI {
	return COperationABI{Operation: NativeCOperationDone, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "done"), TypeName: b.callbackTypeName(NativeCOperationDone), Params: []CABISlot{handleSlot("stream")}, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) cancel() COperationABI {
	return COperationABI{Operation: NativeCOperationCancel, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "cancel"), TypeName: b.callbackTypeName(NativeCOperationCancel), Params: []CABISlot{handleSlot("stream")}, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) register() COperationABI {
	params := []CABISlot{{Name: "callback", CType: b.callbackTypeName(NativeCOperationUnary), Role: CABISlotRoleCallback, Cleanup: CABICleanupNoCleanup}}
	if b.method.Streaming != StreamingKindUnary {
		params = nil
		for _, operation := range b.streamingCallbackOperations() {
			params = append(params, CABISlot{Name: nativeCABIRegisterParamName(operation), CType: b.callbackTypeName(operation), Role: CABISlotRoleCallback, Cleanup: CABICleanupNoCleanup})
		}
	}
	return COperationABI{Operation: NativeCOperationRegister, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "register"), TypeName: "", Params: params, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) streamingCallbackOperations() []NativeCOperation {
	switch b.method.Streaming {
	case StreamingKindClientStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationFinish, NativeCOperationCancel}
	case StreamingKindServerStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationRecv, NativeCOperationDone, NativeCOperationCancel}
	case StreamingKindBidiStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationRecv, NativeCOperationCloseSend, NativeCOperationDone, NativeCOperationCancel}
	default:
		return nil
	}
}

func (b nativeCABIBuilder) callbackTypeName(operation NativeCOperation) string {
	switch b.method.Streaming {
	case StreamingKindUnary:
		return nativeCGOServerCallbackName(b.service, b.method)
	case StreamingKindClientStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerClientStreamStartCallbackName(b.service, b.method)
		case NativeCOperationSend:
			return nativeCGOServerClientStreamSendCallbackName(b.service, b.method)
		case NativeCOperationFinish:
			return nativeCGOServerClientStreamFinishCallbackName(b.service, b.method)
		case NativeCOperationCancel:
			return nativeCGOServerClientStreamCancelCallbackName(b.service, b.method)
		}
	case StreamingKindServerStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerServerStreamStartCallbackName(b.service, b.method)
		case NativeCOperationRecv:
			return nativeCGOServerServerStreamRecvCallbackName(b.service, b.method)
		case NativeCOperationDone:
			return nativeCGOServerServerStreamDoneCallbackName(b.service, b.method)
		case NativeCOperationCancel:
			return nativeCGOServerServerStreamCancelCallbackName(b.service, b.method)
		}
	case StreamingKindBidiStreaming:
		switch operation {
		case NativeCOperationStart:
			return nativeCGOServerBidiStreamStartCallbackName(b.service, b.method)
		case NativeCOperationSend:
			return nativeCGOServerBidiStreamSendCallbackName(b.service, b.method)
		case NativeCOperationRecv:
			return nativeCGOServerBidiStreamRecvCallbackName(b.service, b.method)
		case NativeCOperationCloseSend:
			return nativeCGOServerBidiStreamCloseSendCallbackName(b.service, b.method)
		case NativeCOperationDone:
			return nativeCGOServerBidiStreamDoneCallbackName(b.service, b.method)
		case NativeCOperationCancel:
			return nativeCGOServerBidiStreamCancelCallbackName(b.service, b.method)
		}
	}
	return ""
}

func (b nativeCABIBuilder) inputSlots(fields []FieldPlan) []CABISlot {
	slots := make([]CABISlot, 0, len(fields)*3)
	for _, field := range fields {
		slots = append(slots, nativeCABIFieldSlots(field, false)...)
	}
	return slots
}

func (b nativeCABIBuilder) outputSlots(fields []FieldPlan) []CABISlot {
	slots := make([]CABISlot, 0, len(fields)*3)
	for _, field := range fields {
		slots = append(slots, nativeCABIFieldSlots(field, true)...)
	}
	return slots
}

func nativeCABIFieldSlots(field FieldPlan, output bool) []CABISlot {
	source := nativeCABIFieldRef(field)
	cleanup := CABICleanupNoCleanup
	if output && nativeCABIFieldNeedsRuntimeFree(field) {
		cleanup = CABICleanupFreeWithRuntime
	}
	slot := func(name, ctype string, role CABISlotRole) CABISlot {
		ref := source
		ref.CName = name
		return CABISlot{Source: &ref, Name: name, CType: ctype, Role: role, Cleanup: cleanup}
	}
	ptr := ""
	if output {
		ptr = "*"
	}
	name := func(suffix string) string {
		base := field.GoName + suffix
		if output {
			return "out" + base
		}
		return base
	}
	roleValue := CABISlotRoleValue
	rolePointer := CABISlotRolePointer
	roleLength := CABISlotRoleLength
	roleCount := CABISlotRoleCount
	if output {
		roleValue = CABISlotRoleOutValue
		rolePointer = CABISlotRoleOutPointer
		roleLength = CABISlotRoleOutLength
		roleCount = CABISlotRoleOutCount
	}
	switch field.Native.Shape {
	case NativeABIShapeBoolByte:
		return []CABISlot{slot(name(""), "int8_t"+ptr, roleValue)}
	case NativeABIShapeRepeated, NativeABIShapeBoolByteBufferWrapper:
		return []CABISlot{slot(name("Ptr"), "uintptr_t"+ptr, rolePointer), slot(name("Len"), "int32_t"+ptr, roleCount), slot(name("Ownership"), "int32_t"+ptr, roleValue)}
	case NativeABIShapeScalar, NativeABIShapeMessageBytes:
		switch field.Kind {
		case FieldKindSignedInt32, FieldKindEnum:
			return []CABISlot{slot(name(""), "int32_t"+ptr, roleValue)}
		case FieldKindUnsignedInt32:
			return []CABISlot{slot(name(""), "uint32_t"+ptr, roleValue)}
		case FieldKindSignedInt64:
			return []CABISlot{slot(name(""), "int64_t"+ptr, roleValue)}
		case FieldKindUnsignedInt64:
			return []CABISlot{slot(name(""), "uint64_t"+ptr, roleValue)}
		case FieldKindFloat:
			return []CABISlot{slot(name(""), "float"+ptr, roleValue)}
		case FieldKindDouble:
			return []CABISlot{slot(name(""), "double"+ptr, roleValue)}
		case FieldKindString, FieldKindBytes, FieldKindMessage:
			return []CABISlot{slot(name("Ptr"), "uintptr_t"+ptr, rolePointer), slot(name("Len"), "int32_t"+ptr, roleLength), slot(name("Ownership"), "int32_t"+ptr, roleValue)}
		default:
			return []CABISlot{slot(name(""), "uintptr_t"+ptr, roleValue)}
		}
	default:
		return []CABISlot{slot(name(""), "uintptr_t"+ptr, roleValue)}
	}
}

func nativeCABIFieldRef(field FieldPlan) NativeFieldRef {
	return NativeFieldRef{
		ProtoName: field.Name,
		GoName:    field.GoName,
		GoType:    string(field.Kind),
		Kind:      field.Kind,
		Scalar:    !field.Repeated,
	}
}

func nativeCABIFieldNeedsRuntimeFree(field FieldPlan) bool {
	if field.Native.Shape == NativeABIShapeRepeated || field.Native.Shape == NativeABIShapeBoolByteBufferWrapper {
		return true
	}
	return field.Native.Shape == NativeABIShapeScalar && (field.Kind == FieldKindString || field.Kind == FieldKindBytes)
}

func handleSlot(name string) CABISlot {
	return CABISlot{Name: name, CType: "int32_t", Role: CABISlotRoleHandle, Cleanup: CABICleanupNoCleanup}
}

func outHandleSlot(name string) CABISlot {
	return CABISlot{Name: name, CType: "int32_t*", Role: CABISlotRoleHandle, Cleanup: CABICleanupNoCleanup}
}

func errorIDReturnSlot() CABISlot {
	return CABISlot{Name: "error_id", CType: "int32_t", Role: CABISlotRoleErrorID, Cleanup: CABICleanupNoCleanup}
}

func nativeCABIRegisterParamName(operation NativeCOperation) string {
	switch operation {
	case NativeCOperationStart:
		return "start"
	case NativeCOperationSend:
		return "send"
	case NativeCOperationRecv:
		return "recv"
	case NativeCOperationFinish:
		return "finish"
	case NativeCOperationCloseSend:
		return "closeSend"
	case NativeCOperationDone:
		return "done"
	case NativeCOperationCancel:
		return "cancel"
	default:
		return "callback"
	}
}
