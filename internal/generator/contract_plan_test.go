package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestBuildContractPlanBuildsNativeAndMessageFields(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", contractPlanTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	method := plan.Services[0].Methods[0]
	if method.Contract.Message.RequestType != method.Request || method.Contract.Message.ResponseType != method.Response {
		t.Fatalf("Message contract = %#v, want method request/response identity", method.Contract.Message)
	}
	if !method.Contract.RenderInputs.NeedsCodec {
		t.Fatalf("RenderInputs.NeedsCodec = false, want true")
	}
	if len(method.Contract.Native.RequestFields) != 8 {
		t.Fatalf("request native fields = %d, want 8", len(method.Contract.Native.RequestFields))
	}
	assertNativeField(t, method.Contract.Native.RequestFields[0], FieldPlan{
		Name:     "signed_count",
		GoName:   "SignedCount",
		FullName: "test.v1.ContractRequest.signed_count",
		Kind:     FieldKindSignedInt32,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindSignedNumeric,
			Shape: NativeABIShapeScalar,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[1], FieldPlan{
		Name:     "signed_total",
		GoName:   "SignedTotal",
		FullName: "test.v1.ContractRequest.signed_total",
		Kind:     FieldKindSignedInt64,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindSignedNumeric,
			Shape: NativeABIShapeScalar,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[2], FieldPlan{
		Name:     "ratio",
		GoName:   "Ratio",
		FullName: "test.v1.ContractRequest.ratio",
		Kind:     FieldKindFloat,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindFloat,
			Shape: NativeABIShapeScalar,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[3], FieldPlan{
		Name:     "enabled",
		GoName:   "Enabled",
		FullName: "test.v1.ContractRequest.enabled",
		Kind:     FieldKindBool,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindBool,
			Shape: NativeABIShapeBoolByte,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[4], FieldPlan{
		Name:     "tags",
		GoName:   "Tags",
		FullName: "test.v1.ContractRequest.tags",
		Kind:     FieldKindSignedInt32,
		Repeated: true,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindSignedNumeric,
			Shape: NativeABIShapeRepeated,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[5], FieldPlan{
		Name:     "payload",
		GoName:   "Payload",
		FullName: "test.v1.ContractRequest.payload",
		Kind:     FieldKindBytes,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindBytes,
			Shape: NativeABIShapeScalar,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[6], FieldPlan{
		Name:     "child",
		GoName:   "Child",
		FullName: "test.v1.ContractRequest.child",
		Kind:     FieldKindMessage,
		Message:  true,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindMessageBytes,
			Shape: NativeABIShapeMessageBytes,
		},
	})
	assertNativeField(t, method.Contract.Native.RequestFields[7], FieldPlan{
		Name:     "state",
		GoName:   "State",
		FullName: "test.v1.ContractRequest.state",
		Kind:     FieldKindEnum,
		Enum:     true,
		EnumType: MethodIOPlan{
			GoName:       "State",
			GoImportPath: "example.com/test/v1",
			FullName:     "test.v1.State",
		},
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindEnum,
			Shape: NativeABIShapeScalar,
		},
	})
	assertNativeField(t, method.Contract.Native.ResponseFields[0], FieldPlan{
		Name:     "accepted",
		GoName:   "Accepted",
		FullName: "test.v1.ContractReply.accepted",
		Kind:     FieldKindBool,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindBool,
			Shape: NativeABIShapeBoolByte,
		},
	})
}

func TestNativeFieldPlanMarksRepeatedBoolAsByteBufferWrapper(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedBoolContractTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}

	field := plan.Services[0].Methods[0].Contract.Native.RequestFields[0]
	assertNativeField(t, field, FieldPlan{
		Name:     "flags",
		GoName:   "Flags",
		FullName: "test.v1.BoolRequest.flags",
		Kind:     FieldKindBool,
		Repeated: true,
		Native: NativeFieldPlan{
			Kind:  NativeFieldKindBool,
			Shape: NativeABIShapeBoolByteBufferWrapper,
		},
	})
}

func TestBuildContractPlanRejectsRepeatedMessage(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedMessageContractTestFile())

	_, err := BuildDescriptorPlan(plugin.Files[0])
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want repeated message error")
	}
	got := err.Error()
	forbidden := []string{"test.v1.Contracts.Check", "test.v1.BadRequest.children", "repeated message fields are not supported"}
	for _, want := range forbidden {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want substring %q", got, want)
		}
	}
}

func TestBuildContractPlanRejectsRepeatedStringNativeABI(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedStringContractTestFile())

	_, err := BuildDescriptorPlan(plugin.Files[0])
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want repeated string unsupported error")
	}
	got := err.Error()
	for _, want := range []string{"repeated string", "not supported"} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want substring %q", got, want)
		}
	}
}

func TestBuildContractPlanRejectsRepeatedBytesNativeABI(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", repeatedBytesContractTestFile())

	_, err := BuildDescriptorPlan(plugin.Files[0])
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want repeated bytes unsupported error")
	}
	got := err.Error()
	for _, want := range []string{"repeated bytes", "not supported"} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want substring %q", got, want)
		}
	}
}

func TestBuildContractPlanRejectsMapField(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", mapContractTestFile())

	_, err := BuildDescriptorPlan(plugin.Files[0])
	if err == nil {
		t.Fatal("BuildDescriptorPlan() error = nil, want map field error")
	}
	got := err.Error()
	for _, want := range []string{"test.v1.Contracts.Check", "test.v1.BadRequest.labels", "map fields are not supported"} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildDescriptorPlan() error = %q, want substring %q", got, want)
		}
	}
}

func TestBuildContractPlanAllowsUnsignedProtoFields(t *testing.T) {
	plugin := newTestPlugin(t, "paths=source_relative", unsignedContractTestFile())

	plan, err := BuildDescriptorPlan(plugin.Files[0])
	if err != nil {
		t.Fatalf("BuildDescriptorPlan() error = %v", err)
	}
	fields := plan.Services[0].Methods[0].Contract.Native.RequestFields
	if fields[0].Kind != FieldKindUnsignedInt32 || fields[1].Kind != FieldKindUnsignedInt64 {
		t.Fatalf("unsigned field kinds = (%q, %q), want (%q, %q)", fields[0].Kind, fields[1].Kind, FieldKindUnsignedInt32, FieldKindUnsignedInt64)
	}
	if fields[0].Native.Shape != NativeABIShapeScalar || fields[1].Native.Shape != NativeABIShapeScalar {
		t.Fatalf("unsigned field native shapes = (%q, %q), want scalar", fields[0].Native.Shape, fields[1].Native.Shape)
	}
}

func assertNativeField(t *testing.T, got FieldPlan, want FieldPlan) {
	t.Helper()

	if got.Name != want.Name || got.GoName != want.GoName || got.FullName != want.FullName {
		t.Fatalf("field identity = (%q, %q, %q), want (%q, %q, %q)",
			got.Name, got.GoName, got.FullName, want.Name, want.GoName, want.FullName)
	}
	if got.Kind != want.Kind || got.Repeated != want.Repeated || got.Enum != want.Enum || got.Message != want.Message {
		t.Fatalf("%s metadata = (%q, repeated=%v, enum=%v, message=%v), want (%q, repeated=%v, enum=%v, message=%v)",
			got.Name, got.Kind, got.Repeated, got.Enum, got.Message, want.Kind, want.Repeated, want.Enum, want.Message)
	}
	if got.EnumType != want.EnumType {
		t.Fatalf("%s EnumType = %#v, want %#v", got.Name, got.EnumType, want.EnumType)
	}
	if got.Native != want.Native {
		t.Fatalf("%s Native = %#v, want %#v", got.Name, got.Native, want.Native)
	}
}

func contractPlanTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/contracts.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			contractRequestDescriptor(),
			contractReplyDescriptor(),
			childMessageDescriptor(),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			stateEnumDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			contractServiceDescriptor(".test.v1.ContractRequest", ".test.v1.ContractReply"),
		},
	}
}

func repeatedBoolContractTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/bools.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("BoolRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("flags", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
				},
			},
			{Name: proto.String("BoolReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			contractServiceDescriptor(".test.v1.BoolRequest", ".test.v1.BoolReply"),
		},
	}
}

func repeatedMessageContractTestFile() *descriptorpb.FileDescriptorProto {
	return badFieldContractTestFile(fieldDescriptor("children", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ".test.v1.Child"))
}

func repeatedStringContractTestFile() *descriptorpb.FileDescriptorProto {
	return badFieldContractTestFile(fieldDescriptor("tags", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""))
}

func repeatedBytesContractTestFile() *descriptorpb.FileDescriptorProto {
	return badFieldContractTestFile(fieldDescriptor("payloads", 1, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""))
}

func mapContractTestFile() *descriptorpb.FileDescriptorProto {
	file := badFieldContractTestFile(fieldDescriptor("labels", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ".test.v1.BadRequest.LabelsEntry"))
	file.MessageType[0].NestedType = []*descriptorpb.DescriptorProto{
		{
			Name: proto.String("LabelsEntry"),
			Field: []*descriptorpb.FieldDescriptorProto{
				fieldDescriptor("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				fieldDescriptor("value", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			},
			Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
		},
	}
	return file
}

func unsignedContractTestFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/unsigned_contracts.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("UnsignedRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					fieldDescriptor("count", 1, descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
					fieldDescriptor("total", 2, descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
				},
			},
			{Name: proto.String("UnsignedReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			contractServiceDescriptor(".test.v1.UnsignedRequest", ".test.v1.UnsignedReply"),
		},
	}
}

func badFieldContractTestFile(field *descriptorpb.FieldDescriptorProto) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/v1/bad_contracts.proto"),
		Package: proto.String("test.v1"),
		Syntax:  proto.String("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:  proto.String("BadRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{field},
			},
			{Name: proto.String("BadReply")},
			childMessageDescriptor(),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			contractServiceDescriptor(".test.v1.BadRequest", ".test.v1.BadReply"),
		},
	}
}

func contractRequestDescriptor() *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String("ContractRequest"),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("signed_count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("signed_total", 2, descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("ratio", 3, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("enabled", 4, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("tags", 5, descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, ""),
			fieldDescriptor("payload", 6, descriptorpb.FieldDescriptorProto_TYPE_BYTES, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
			fieldDescriptor("child", 7, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.Child"),
			fieldDescriptor("state", 8, descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ".test.v1.State"),
		},
	}
}

func contractReplyDescriptor() *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String("ContractReply"),
		Field: []*descriptorpb.FieldDescriptorProto{
			fieldDescriptor("accepted", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, ""),
		},
	}
}

func childMessageDescriptor() *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String("Child")}
}

func stateEnumDescriptor() *descriptorpb.EnumDescriptorProto {
	return &descriptorpb.EnumDescriptorProto{
		Name: proto.String("State"),
		Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: proto.String("STATE_UNKNOWN"), Number: proto.Int32(0)},
			{Name: proto.String("STATE_READY"), Number: proto.Int32(1)},
		},
	}
}

func contractServiceDescriptor(input, output string) *descriptorpb.ServiceDescriptorProto {
	return &descriptorpb.ServiceDescriptorProto{
		Name: proto.String("Contracts"),
		Method: []*descriptorpb.MethodDescriptorProto{
			methodDescriptor("Check", input, output, false, false),
		},
	}
}

func fieldDescriptor(name string, number int32, fieldType descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
		Type:   fieldType.Enum(),
		Label:  label.Enum(),
	}
	if typeName != "" {
		field.TypeName = proto.String(typeName)
	}
	return field
}
