package generator

import (
	"fmt"
	"strings"
)

// NativeCOperation identifies one C ABI operation generated for a native method.
type NativeCOperation string

// Native C operations supported by cgo native client and server artifacts.
const (
	NativeCOperationUnary     NativeCOperation = "unary"
	NativeCOperationStart     NativeCOperation = "start"
	NativeCOperationSend      NativeCOperation = "send"
	NativeCOperationRecv      NativeCOperation = "recv"
	NativeCOperationFinish    NativeCOperation = "finish"
	NativeCOperationCloseSend NativeCOperation = "close_send"
	NativeCOperationCancel    NativeCOperation = "cancel"
	NativeCOperationRegister  NativeCOperation = "register"
)

// COperationABI describes the C symbol, callback type, parameters, and return slot for one operation.
type COperationABI struct {
	Operation NativeCOperation
	Symbol    string
	TypeName  string
	Params    []CABISlot
	Return    CABISlot
}

type nativeCServiceABI struct {
	Methods  map[string]map[NativeCOperation]COperationABI
	Register COperationABI
}

// CABISlot describes one lowered C ABI parameter or return slot.
type CABISlot struct {
	Name        string
	CType       string
	CGoType     string
	Role        CABISlotRole
	FieldGoName string
}

// CABISlotRole identifies how a lowered C ABI slot participates in an operation.
type CABISlotRole string

// C ABI slot roles used by native C lowering.
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

// NativeCRegisterABI builds the service-level C callback registration ABI for a native server.
func NativeCRegisterABI(plan FilePlan, service ServicePlan) (COperationABI, error) {
	var registerParams []CABISlot
	for _, method := range service.Methods {
		operations, err := NativeCOperationsForMethod(method)
		if err != nil {
			return COperationABI{}, fmt.Errorf("service %s native C register ABI: %w", service.FullName, err)
		}
		for _, operation := range operations {
			abi, err := NativeCOperationABI(plan, service, method, operation)
			if err != nil {
				return COperationABI{}, fmt.Errorf("service %s native C register ABI: %w", service.FullName, err)
			}
			if abi.TypeName == "" {
				return COperationABI{}, fmt.Errorf("service %s native C register ABI: method %s operation %s callback type is empty", service.FullName, methodPlanName(method), operation)
			}
			registerParams = append(registerParams, callbackSlot(
				lowerInitial(method.GoName)+upperInitial(nativeCABIRegisterParamName(operation)),
				abi.TypeName,
			))
		}
	}
	return COperationABI{
		Operation: NativeCOperationRegister,
		Symbol:    nativeCServiceRegisterExportFuncName(plan, service),
		Params:    registerParams,
		Return:    errorIDReturnSlot(),
	}, nil
}

func nativeCServiceABIs(plan FilePlan, service ServicePlan) (nativeCServiceABI, error) {
	methods := make(map[string]map[NativeCOperation]COperationABI, len(service.Methods))
	for _, method := range service.Methods {
		abi, err := nativeCOperationABIsByOperation(plan, service, method)
		if err != nil {
			return nativeCServiceABI{}, err
		}
		methods[method.FullName] = abi
	}
	registerABI, err := NativeCRegisterABI(plan, service)
	if err != nil {
		return nativeCServiceABI{}, err
	}
	return nativeCServiceABI{Methods: methods, Register: registerABI}, nil
}

// NativeCOperationsForMethod returns the native C operations required by a method's streaming kind.
func NativeCOperationsForMethod(method MethodPlan) ([]NativeCOperation, error) {
	switch method.Streaming {
	case StreamingKindUnary:
		return []NativeCOperation{NativeCOperationUnary}, nil
	case StreamingKindClientStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationFinish, NativeCOperationCancel}, nil
	case StreamingKindServerStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationRecv, NativeCOperationFinish, NativeCOperationCancel}, nil
	case StreamingKindBidiStreaming:
		return []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationRecv, NativeCOperationCloseSend, NativeCOperationFinish, NativeCOperationCancel}, nil
	default:
		return nil, fmt.Errorf("method %s: unsupported native C ABI streaming kind %q", methodPlanName(method), method.Streaming)
	}
}

// NativeCOperationABI lowers one method operation into its C ABI shape.
func NativeCOperationABI(plan FilePlan, service ServicePlan, method MethodPlan, operation NativeCOperation) (COperationABI, error) {
	operations, err := NativeCOperationsForMethod(method)
	if err != nil {
		return COperationABI{}, err
	}
	if !nativeCOperationAllowed(operations, operation) {
		return COperationABI{}, fmt.Errorf("method %s: native C operation %q is invalid for streaming kind %q", methodPlanName(method), operation, method.Streaming)
	}

	builder := nativeCABIBuilder{file: plan, service: service, method: method}
	switch operation {
	case NativeCOperationUnary:
		return builder.unary(), nil
	case NativeCOperationStart:
		if method.Streaming == StreamingKindServerStreaming {
			return builder.serverStreamStart(), nil
		}
		return builder.startOutHandle(), nil
	case NativeCOperationSend:
		return builder.send(), nil
	case NativeCOperationRecv:
		return builder.recv(), nil
	case NativeCOperationFinish:
		if method.Streaming == StreamingKindClientStreaming {
			return builder.finish(), nil
		}
		return builder.finishTerminal(), nil
	case NativeCOperationCloseSend:
		return builder.closeSend(), nil
	case NativeCOperationCancel:
		return builder.cancel(), nil
	default:
		return COperationABI{}, fmt.Errorf("method %s: unknown native C operation %q", methodPlanName(method), operation)
	}
}

func nativeCOperationABIsByOperation(plan FilePlan, service ServicePlan, method MethodPlan) (map[NativeCOperation]COperationABI, error) {
	operations, err := NativeCOperationsForMethod(method)
	if err != nil {
		return nil, err
	}
	byOperation := make(map[NativeCOperation]COperationABI, len(operations))
	for _, operation := range operations {
		abi, err := NativeCOperationABI(plan, service, method, operation)
		if err != nil {
			return nil, err
		}
		byOperation[operation] = abi
	}
	return byOperation, nil
}

func nativeCOperationAllowed(operations []NativeCOperation, operation NativeCOperation) bool {
	for _, current := range operations {
		if current == operation {
			return true
		}
	}
	return false
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

func (b nativeCABIBuilder) finishTerminal() COperationABI {
	return COperationABI{Operation: NativeCOperationFinish, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "finish"), TypeName: b.callbackTypeName(NativeCOperationFinish), Params: []CABISlot{handleSlot("stream")}, Return: errorIDReturnSlot()}
}

func (b nativeCABIBuilder) cancel() COperationABI {
	return COperationABI{Operation: NativeCOperationCancel, Symbol: nativeCExportFuncName(b.file, b.service, b.method, "cancel"), TypeName: b.callbackTypeName(NativeCOperationCancel), Params: []CABISlot{handleSlot("stream")}, Return: errorIDReturnSlot()}
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
		case NativeCOperationFinish:
			return nativeCGOServerServerStreamFinishCallbackName(b.service, b.method)
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
		case NativeCOperationFinish:
			return nativeCGOServerBidiStreamFinishCallbackName(b.service, b.method)
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
	slot := func(name, ctype string, role CABISlotRole) CABISlot {
		return CABISlot{Name: name, CType: ctype, CGoType: nativeCGoType(ctype), Role: role, FieldGoName: field.GoName}
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

func handleSlot(name string) CABISlot {
	return CABISlot{Name: name, CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleHandle}
}

func outHandleSlot(name string) CABISlot {
	return CABISlot{Name: name, CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleHandle}
}

func errorIDReturnSlot() CABISlot {
	return CABISlot{Name: "error_id", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleErrorID}
}

func callbackSlot(name, typeName string) CABISlot {
	return CABISlot{Name: name, CType: typeName, CGoType: "C." + typeName, Role: CABISlotRoleCallback}
}

func nativeCGoType(ctype string) string {
	base, pointer := strings.CutSuffix(ctype, "*")
	goType := "C." + base
	if pointer {
		return "*" + goType
	}
	return goType
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
	case NativeCOperationCancel:
		return "cancel"
	default:
		return "callback"
	}
}

func nativeCServiceRegisterExportFuncName(plan FilePlan, service ServicePlan) string {
	return cgoServiceExportName("native", plan, service, "register")
}

func upperInitial(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
