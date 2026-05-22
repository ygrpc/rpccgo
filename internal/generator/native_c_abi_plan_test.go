package generator

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestBuildMethodNativeCABIPlanUnaryAllFields(t *testing.T) {
	file := nativeCABIAllFieldsFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	service := plan.Services[0]
	method := service.Methods[0]
	abi, err := BuildMethodNativeCABIPlan(plan, service, method)
	if err != nil {
		t.Fatalf("BuildMethodNativeCABIPlan() error = %v", err)
	}

	if got, want := abi.MethodFullName, "test.v1.NativeABI.Check"; got != want {
		t.Fatalf("MethodFullName = %q, want %q", got, want)
	}
	if got, want := nativeCABIPlanOperations(abi), []NativeCOperation{NativeCOperationUnary, NativeCOperationRegister}; !reflect.DeepEqual(got, want) {
		t.Fatalf("operations = %#v, want %#v", got, want)
	}

	unary := nativeCABIPlanOperation(abi, NativeCOperationUnary)
	if unary.Symbol != "rpccgo_native_testv1_NativeABI_Check" {
		t.Fatalf("unary Symbol = %q", unary.Symbol)
	}
	if unary.TypeName != "NativeABICheckCGONativeUnaryCallback" {
		t.Fatalf("unary TypeName = %q", unary.TypeName)
	}
	if unary.Return.Role != CABISlotRoleErrorID || unary.Return.CType != "int32_t" || unary.Return.CGoType != "C.int32_t" {
		t.Fatalf("unary Return = %#v, want int32_t error id", unary.Return)
	}

	want := []CABISlot{
		{Name: "SignedCount", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "UnsignedCount", CType: "uint32_t", CGoType: "C.uint32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "SignedTotal", CType: "int64_t", CGoType: "C.int64_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "UnsignedTotal", CType: "uint64_t", CGoType: "C.uint64_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "Ratio", CType: "float", CGoType: "C.float", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "Enabled", CType: "int8_t", CGoType: "C.int8_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "NamePtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, Cleanup: CABICleanupNoCleanup},
		{Name: "NameLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, Cleanup: CABICleanupNoCleanup},
		{Name: "NameOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "PayloadPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, Cleanup: CABICleanupNoCleanup},
		{Name: "PayloadLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, Cleanup: CABICleanupNoCleanup},
		{Name: "PayloadOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "ChildPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, Cleanup: CABICleanupNoCleanup},
		{Name: "ChildLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, Cleanup: CABICleanupNoCleanup},
		{Name: "ChildOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "ScoresPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, Cleanup: CABICleanupNoCleanup},
		{Name: "ScoresLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleCount, Cleanup: CABICleanupNoCleanup},
		{Name: "ScoresOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "FlagsPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, Cleanup: CABICleanupNoCleanup},
		{Name: "FlagsLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleCount, Cleanup: CABICleanupNoCleanup},
		{Name: "FlagsOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, Cleanup: CABICleanupNoCleanup},
		{Name: "outAccepted", CType: "int8_t*", CGoType: "*C.int8_t", Role: CABISlotRoleOutValue, Cleanup: CABICleanupNoCleanup},
		{Name: "outReplyPayloadPtr", CType: "uintptr_t*", CGoType: "*C.uintptr_t", Role: CABISlotRoleOutPointer, Cleanup: CABICleanupFreeWithRuntime},
		{Name: "outReplyPayloadLen", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutLength, Cleanup: CABICleanupFreeWithRuntime},
		{Name: "outReplyPayloadOwnership", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutValue, Cleanup: CABICleanupFreeWithRuntime},
		{Name: "outReplyFlagsPtr", CType: "uintptr_t*", CGoType: "*C.uintptr_t", Role: CABISlotRoleOutPointer, Cleanup: CABICleanupFreeWithRuntime},
		{Name: "outReplyFlagsLen", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutCount, Cleanup: CABICleanupFreeWithRuntime},
		{Name: "outReplyFlagsOwnership", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutValue, Cleanup: CABICleanupFreeWithRuntime},
	}
	assertCABISlots(t, unary.Params, want)

	if got := unary.Params[1].Source; got == nil || got.Kind != FieldKindUnsignedInt32 || !got.Scalar || got.ProtoName != "unsigned_count" {
		t.Fatalf("unsigned field source = %#v, want unsigned proto scalar metadata", got)
	}

	register := nativeCABIPlanOperation(abi, NativeCOperationRegister)
	if register.Symbol != "rpccgo_native_testv1_NativeABI_Check_register" {
		t.Fatalf("register Symbol = %q", register.Symbol)
	}
	assertCABISlots(t, register.Params, []CABISlot{{Name: "callback", CType: "NativeABICheckCGONativeUnaryCallback", CGoType: "C.NativeABICheckCGONativeUnaryCallback", Role: CABISlotRoleCallback, Cleanup: CABICleanupNoCleanup}})
}

func TestBuildMethodNativeCABIPlanStreamingOperationSets(t *testing.T) {
	file := nativeCABIStreamingFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	service := plan.Services[0]
	tests := []struct {
		method string
		want   []NativeCOperation
	}{
		{method: "Upload", want: []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationFinish, NativeCOperationCancel, NativeCOperationRegister}},
		{method: "List", want: []NativeCOperation{NativeCOperationStart, NativeCOperationRecv, NativeCOperationDone, NativeCOperationCancel, NativeCOperationRegister}},
		{method: "Chat", want: []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationRecv, NativeCOperationCloseSend, NativeCOperationDone, NativeCOperationCancel, NativeCOperationRegister}},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			method := methodByGoName(t, service, tt.method)
			abi, err := BuildMethodNativeCABIPlan(plan, service, method)
			if err != nil {
				t.Fatalf("BuildMethodNativeCABIPlan() error = %v", err)
			}
			if got := nativeCABIPlanOperations(abi); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("operations = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func assertCABISlots(t *testing.T, got, want []CABISlot) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slots len = %d, want %d\ngot: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].CType != want[i].CType || got[i].CGoType != want[i].CGoType || got[i].Role != want[i].Role || got[i].Cleanup != want[i].Cleanup {
			t.Fatalf("slot[%d] = {Name:%q CType:%q CGoType:%q Role:%q Cleanup:%q}, want {Name:%q CType:%q CGoType:%q Role:%q Cleanup:%q}", i, got[i].Name, got[i].CType, got[i].CGoType, got[i].Role, got[i].Cleanup, want[i].Name, want[i].CType, want[i].CGoType, want[i].Role, want[i].Cleanup)
		}
	}
}

func nativeCABIPlanOperations(plan MethodNativeCABIPlan) []NativeCOperation {
	operations := make([]NativeCOperation, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		operations = append(operations, operation.Operation)
	}
	return operations
}

func methodByGoName(t *testing.T, service ServicePlan, goName string) MethodPlan {
	t.Helper()
	for _, method := range service.Methods {
		if method.GoName == goName {
			return method
		}
	}
	t.Fatalf("method %s not found", goName)
	return MethodPlan{}
}

func nativeCABIAllFieldsFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/native_c_abi.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/test/v1;testv1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("ABIRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("signed_count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("unsigned_count", 2, descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("signed_total", 3, descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("unsigned_total", 4, descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("ratio", 5, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("enabled", 6, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("name", 7, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("payload", 8, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("child", 9, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.Child"),
					fieldDescriptor("scores", 10, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
					fieldDescriptor("flags", 11, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
				},
			},
			{
				Name: proto.String("ABIReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("reply_payload", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("reply_flags", 3, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
				},
			},
			childMessageDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("NativeABI"),
			Method: []*descriptorpb.MethodDescriptorProto{
				methodDescriptor("Check", ".test.v1.ABIRequest", ".test.v1.ABIReply", false, false),
			},
		}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
			Path:            []int32{6, 0},
			Span:            []int32{0, 0, 0},
			LeadingComments: proto.String("@rpccgo: native\n"),
		}}},
	}
}

func nativeCABIStreamingFile() *descriptorpb.FileDescriptorProto {
	file := messageCgoTestFile()
	file.Name = proto.String("test/v1/native_c_abi_streaming.proto")
	setNativeCABIStreamingComment(file)
	return file
}

func setNativeCABIStreamingComment(file *descriptorpb.FileDescriptorProto) {
	file.SourceCodeInfo = &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
		Path:            []int32{6, 0},
		Span:            []int32{0, 0, 0},
		LeadingComments: proto.String("@rpccgo: native\n"),
	}}}
}
