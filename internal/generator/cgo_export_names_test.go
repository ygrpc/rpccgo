package generator

import "testing"

func TestMessageCExportFuncNameUsesCamelCase(t *testing.T) {
	plan := FilePlan{GoPackageName: "testv1"}
	service := ServicePlan{GoName: "Greeter"}
	method := MethodPlan{GoName: "SayHello"}

	if got, want := messageCExportFuncName(plan, service, method, ""), "rpccgoMsgTestv1GreeterSayHello"; got != want {
		t.Fatalf("message unary export name = %q, want %q", got, want)
	}
	if got, want := messageCExportFuncName(plan, service, MethodPlan{GoName: "Chat"}, "close_send"), "rpccgoMsgTestv1GreeterChatCloseSend"; got != want {
		t.Fatalf("message bidi close-send export name = %q, want %q", got, want)
	}
}

func TestMessageCServerRegisterExportNamesUseCamelCase(t *testing.T) {
	plan := FilePlan{GoPackageName: "testv1"}
	service := ServicePlan{GoName: "Greeter"}
	method := MethodPlan{GoName: "SayHello"}

	if got, want := messageCServiceRegisterExportFuncName(plan, service), "rpccgoMsgTestv1GreeterRegister"; got != want {
		t.Fatalf("message service register export name = %q, want %q", got, want)
	}
	if got, want := messageCServiceMethodRegisterExportFuncName(plan, service, method), "rpccgoMsgTestv1GreeterRegisterSayHello"; got != want {
		t.Fatalf("message method register export name = %q, want %q", got, want)
	}
}

func TestNativeCExportNamesUseCamelCase(t *testing.T) {
	plan := FilePlan{GoPackageName: "testv1"}
	service := ServicePlan{GoName: "Greeter"}
	method := MethodPlan{GoName: "Upload"}

	if got, want := nativeCExportFuncName(plan, service, MethodPlan{GoName: "SayHello"}, ""), "rpccgoNativeTestv1GreeterSayHello"; got != want {
		t.Fatalf("native unary export name = %q, want %q", got, want)
	}
	if got, want := nativeCExportFuncName(plan, service, method, "start"), "rpccgoNativeTestv1GreeterUploadStart"; got != want {
		t.Fatalf("native start export name = %q, want %q", got, want)
	}
	if got, want := nativeCServiceRegisterExportFuncName(plan, service), "rpccgoNativeTestv1GreeterRegister"; got != want {
		t.Fatalf("native service register export name = %q, want %q", got, want)
	}
}
