package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BuildContractPlan derives native and message method contracts from protobuf descriptors.
func BuildContractPlan(service *protogen.Service, method *protogen.Method, methodPlan MethodPlan) (MethodContractPlan, error) {
	if service == nil {
		return MethodContractPlan{}, fmt.Errorf("protogen service is nil")
	}
	if method == nil {
		return MethodContractPlan{}, fmt.Errorf("protogen method is nil")
	}

	requestFields, err := buildFieldPlans(method.Input)
	if err != nil {
		return MethodContractPlan{}, fmt.Errorf("service %s method %s: %w", service.Desc.FullName(), method.Desc.FullName(), err)
	}
	responseFields, err := buildFieldPlans(method.Output)
	if err != nil {
		return MethodContractPlan{}, fmt.Errorf("service %s method %s: %w", service.Desc.FullName(), method.Desc.FullName(), err)
	}

	return MethodContractPlan{
		Native: NativeContractPlan{
			RequestFields:  requestFields,
			ResponseFields: responseFields,
		},
		Message: MessageContractPlan{
			RequestType:  methodPlan.Request,
			ResponseType: methodPlan.Response,
		},
	}, nil
}

func buildFieldPlans(message *protogen.Message) ([]FieldPlan, error) {
	if message == nil {
		return nil, fmt.Errorf("protogen message is nil")
	}

	fields := make([]FieldPlan, 0, len(message.Fields))
	for _, field := range message.Fields {
		fieldPlan, err := buildFieldPlan(field)
		if err != nil {
			return nil, err
		}
		fields = append(fields, fieldPlan)
	}
	return fields, nil
}

func buildFieldPlan(field *protogen.Field) (FieldPlan, error) {
	if field == nil {
		return FieldPlan{}, fmt.Errorf("protogen field is nil")
	}
	if field.Desc.IsMap() {
		return FieldPlan{}, fmt.Errorf("field %s: map fields are not supported in native ABI", field.Desc.FullName())
	}

	kind, err := fieldKind(field.Desc.Kind())
	if err != nil {
		return FieldPlan{}, fmt.Errorf("field %s: %w", field.Desc.FullName(), err)
	}

	plan := FieldPlan{
		Name:     string(field.Desc.Name()),
		GoName:   field.GoName,
		FullName: string(field.Desc.FullName()),
		Kind:     kind,
		Repeated: field.Desc.IsList(),
		Enum:     field.Desc.Kind() == protoreflect.EnumKind,
		Message:  field.Desc.Kind() == protoreflect.MessageKind || field.Desc.Kind() == protoreflect.GroupKind,
	}
	if field.Enum != nil {
		plan.EnumType = MethodIOPlan{
			GoName:       field.Enum.GoIdent.GoName,
			GoImportPath: string(field.Enum.GoIdent.GoImportPath),
			FullName:     string(field.Enum.Desc.FullName()),
		}
	}

	native, err := nativeFieldPlan(plan)
	if err != nil {
		return FieldPlan{}, fmt.Errorf("field %s: %w", field.Desc.FullName(), err)
	}
	plan.Native = native
	return plan, nil
}

func fieldKind(kind protoreflect.Kind) (FieldKind, error) {
	switch kind {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return FieldKindSignedInt32, nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return FieldKindSignedInt64, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return FieldKindUnsignedInt32, nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return FieldKindUnsignedInt64, nil
	case protoreflect.FloatKind:
		return FieldKindFloat, nil
	case protoreflect.DoubleKind:
		return FieldKindDouble, nil
	case protoreflect.BoolKind:
		return FieldKindBool, nil
	case protoreflect.StringKind:
		return FieldKindString, nil
	case protoreflect.BytesKind:
		return FieldKindBytes, nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return FieldKindMessage, nil
	case protoreflect.EnumKind:
		return FieldKindEnum, nil
	default:
		return "", fmt.Errorf("unsupported native field kind %d", int(kind))
	}
}

func nativeFieldPlan(field FieldPlan) (NativeFieldPlan, error) {
	switch field.Kind {
	case FieldKindSignedInt32, FieldKindSignedInt64, FieldKindUnsignedInt32, FieldKindUnsignedInt64:
		return NativeFieldPlan{Kind: NativeFieldKindSignedNumeric, Shape: repeatedShape(field.Repeated)}, nil
	case FieldKindFloat, FieldKindDouble:
		return NativeFieldPlan{Kind: NativeFieldKindFloat, Shape: repeatedShape(field.Repeated)}, nil
	case FieldKindBool:
		if field.Repeated {
			return NativeFieldPlan{Kind: NativeFieldKindBool, Shape: NativeABIShapeBoolByteBufferWrapper}, nil
		}
		return NativeFieldPlan{Kind: NativeFieldKindBool, Shape: NativeABIShapeBoolByte}, nil
	case FieldKindString:
		if field.Repeated {
			return NativeFieldPlan{}, fmt.Errorf("repeated string fields are not supported in native ABI")
		}
		return NativeFieldPlan{Kind: NativeFieldKindString, Shape: repeatedShape(field.Repeated)}, nil
	case FieldKindBytes:
		if field.Repeated {
			return NativeFieldPlan{}, fmt.Errorf("repeated bytes fields are not supported in native ABI")
		}
		return NativeFieldPlan{Kind: NativeFieldKindBytes, Shape: repeatedShape(field.Repeated)}, nil
	case FieldKindMessage:
		if field.Repeated {
			return NativeFieldPlan{}, fmt.Errorf("repeated message fields are not supported in native ABI")
		}
		return NativeFieldPlan{Kind: NativeFieldKindMessageBytes, Shape: NativeABIShapeMessageBytes}, nil
	case FieldKindEnum:
		return NativeFieldPlan{Kind: NativeFieldKindEnum, Shape: repeatedShape(field.Repeated)}, nil
	default:
		return NativeFieldPlan{}, fmt.Errorf("unsupported native field kind %q", field.Kind)
	}
}

func repeatedShape(repeated bool) NativeABIShape {
	if repeated {
		return NativeABIShapeRepeated
	}
	return NativeABIShapeScalar
}
