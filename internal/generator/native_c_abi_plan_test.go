package generator

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestNativeCOperationABIUnaryAllFields(t *testing.T) {
	file := nativeCABIAllFieldsFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	service := plan.Services[0]
	method := service.Methods[0]
	unary, err := NativeCOperationABI(plan, service, method, NativeCOperationUnary)
	if err != nil {
		t.Fatalf("NativeCOperationABI() error = %v", err)
	}
	gotOperations, err := NativeCOperationsForMethod(method)
	if err != nil {
		t.Fatalf("NativeCOperationsForMethod() error = %v", err)
	}
	if want := []NativeCOperation{NativeCOperationUnary}; !reflect.DeepEqual(gotOperations, want) {
		t.Fatalf("operations = %#v, want %#v", gotOperations, want)
	}

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
		{Name: "SignedCount", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "SignedCount"},
		{Name: "UnsignedCount", CType: "uint32_t", CGoType: "C.uint32_t", Role: CABISlotRoleValue, FieldGoName: "UnsignedCount"},
		{Name: "SignedTotal", CType: "int64_t", CGoType: "C.int64_t", Role: CABISlotRoleValue, FieldGoName: "SignedTotal"},
		{Name: "UnsignedTotal", CType: "uint64_t", CGoType: "C.uint64_t", Role: CABISlotRoleValue, FieldGoName: "UnsignedTotal"},
		{Name: "Ratio", CType: "float", CGoType: "C.float", Role: CABISlotRoleValue, FieldGoName: "Ratio"},
		{Name: "Enabled", CType: "int8_t", CGoType: "C.int8_t", Role: CABISlotRoleValue, FieldGoName: "Enabled"},
		{Name: "NamePtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, FieldGoName: "Name"},
		{Name: "NameLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, FieldGoName: "Name"},
		{Name: "NameOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "Name"},
		{Name: "PayloadPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, FieldGoName: "Payload"},
		{Name: "PayloadLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, FieldGoName: "Payload"},
		{Name: "PayloadOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "Payload"},
		{Name: "ChildPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, FieldGoName: "Child"},
		{Name: "ChildLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleLength, FieldGoName: "Child"},
		{Name: "ChildOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "Child"},
		{Name: "ScoresPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, FieldGoName: "Scores"},
		{Name: "ScoresLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleCount, FieldGoName: "Scores"},
		{Name: "ScoresOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "Scores"},
		{Name: "FlagsPtr", CType: "uintptr_t", CGoType: "C.uintptr_t", Role: CABISlotRolePointer, FieldGoName: "Flags"},
		{Name: "FlagsLen", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleCount, FieldGoName: "Flags"},
		{Name: "FlagsOwnership", CType: "int32_t", CGoType: "C.int32_t", Role: CABISlotRoleValue, FieldGoName: "Flags"},
		{Name: "outAccepted", CType: "int8_t*", CGoType: "*C.int8_t", Role: CABISlotRoleOutValue, FieldGoName: "Accepted"},
		{Name: "outReplyPayloadPtr", CType: "uintptr_t*", CGoType: "*C.uintptr_t", Role: CABISlotRoleOutPointer, FieldGoName: "ReplyPayload"},
		{Name: "outReplyPayloadLen", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutLength, FieldGoName: "ReplyPayload"},
		{Name: "outReplyPayloadOwnership", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutValue, FieldGoName: "ReplyPayload"},
		{Name: "outReplyFlagsPtr", CType: "uintptr_t*", CGoType: "*C.uintptr_t", Role: CABISlotRoleOutPointer, FieldGoName: "ReplyFlags"},
		{Name: "outReplyFlagsLen", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutCount, FieldGoName: "ReplyFlags"},
		{Name: "outReplyFlagsOwnership", CType: "int32_t*", CGoType: "*C.int32_t", Role: CABISlotRoleOutValue, FieldGoName: "ReplyFlags"},
	}
	assertCABISlots(t, unary.Params, want)
}

func TestNativeCOperationsForMethodStreamingOperationSets(t *testing.T) {
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
		{method: "Upload", want: []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationFinish, NativeCOperationCancel}},
		{method: "List", want: []NativeCOperation{NativeCOperationStart, NativeCOperationRecv, NativeCOperationFinish, NativeCOperationCancel}},
		{method: "Chat", want: []NativeCOperation{NativeCOperationStart, NativeCOperationSend, NativeCOperationRecv, NativeCOperationCloseSend, NativeCOperationFinish, NativeCOperationCancel}},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			method := methodByGoName(t, service, tt.method)
			got, err := NativeCOperationsForMethod(method)
			if err != nil {
				t.Fatalf("NativeCOperationsForMethod() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("operations = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNativeCRegisterABIDefinesFlatServiceRegistration(t *testing.T) {
	file := nativeCABIStreamingFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	service := plan.Services[0]
	abi, err := NativeCRegisterABI(plan, service)
	if err != nil {
		t.Fatalf("NativeCRegisterABI() error = %v", err)
	}

	if got, want := abi.Symbol, "rpccgo_native_testv1_Greeter_register"; got != want {
		t.Fatalf("register Symbol = %q, want %q", got, want)
	}
	if got, want := len(abi.Params), 15; got != want {
		t.Fatalf("register params len = %d, want %d", got, want)
	}
	if got, want := abi.Params[0].Name, "unaryCallback"; got != want {
		t.Fatalf("register params[0].Name = %q, want %q", got, want)
	}
	if got, want := abi.Params[7].Name, "listFinish"; got != want {
		t.Fatalf("register params[6].Name = %q, want %q", got, want)
	}
	if got, want := abi.Params[13].Name, "chatFinish"; got != want {
		t.Fatalf("register params[12].Name = %q, want %q", got, want)
	}
}

func TestNativeCOperationsForMethodRejectsUnknownStreamingKind(t *testing.T) {
	_, err := NativeCOperationsForMethod(MethodPlan{FullName: "test.v1.Bad.Unknown", Streaming: StreamingKind(99)})
	if err == nil {
		t.Fatal("NativeCOperationsForMethod() error = nil, want unsupported streaming kind")
	}
}

func TestNativeCOperationABIRejectsInvalidOperation(t *testing.T) {
	file := nativeCABIAllFieldsFile()
	plugin := newTestPlugin(t, "paths=source_relative", file)
	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	service := plan.Services[0]
	method := service.Methods[0]
	_, err = NativeCOperationABI(plan, service, method, NativeCOperationSend)
	if err == nil {
		t.Fatal("NativeCOperationABI() error = nil, want invalid operation")
	}
}

func TestNativeCRegisterABIReportsMethodLoweringFailure(t *testing.T) {
	plan := FilePlan{GoPackageName: "testv1"}
	service := ServicePlan{
		FullName: "test.v1.Bad",
		GoName:   "Bad",
		Methods: []MethodPlan{{
			FullName:  "test.v1.Bad.Unknown",
			GoName:    "Unknown",
			Streaming: StreamingKind(99),
		}},
	}
	_, err := NativeCRegisterABI(plan, service)
	if err == nil {
		t.Fatal("NativeCRegisterABI() error = nil, want method lowering failure")
	}
}

func assertCABISlots(t *testing.T, got, want []CABISlot) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slots len = %d, want %d\ngot: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].CType != want[i].CType || got[i].CGoType != want[i].CGoType || got[i].Role != want[i].Role || got[i].FieldGoName != want[i].FieldGoName {
			t.Fatalf("slot[%d] = {Name:%q CType:%q CGoType:%q Role:%q FieldGoName:%q}, want {Name:%q CType:%q CGoType:%q Role:%q FieldGoName:%q}", i, got[i].Name, got[i].CType, got[i].CGoType, got[i].Role, got[i].FieldGoName, want[i].Name, want[i].CType, want[i].CGoType, want[i].Role, want[i].FieldGoName)
		}
	}
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
