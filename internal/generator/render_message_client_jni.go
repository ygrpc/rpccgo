package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderMessageClientJNIFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, file GeneratedArtifactPlan) error {
	cgoImportPath := protogen.GoImportPath(cgoGoImportPath(plan))
	g := plugin.NewGeneratedFile(file.Filename, cgoImportPath)
	servicePackage := cgoServicePackageQualifier(g, plan.GoImportPath, lowerInitial(service.GoName)+"ServiceID")

	g.P("//go:build android && cgo")
	g.P()
	renderGeneratedHeader(g)
	g.P("// Source: ", plan.ProtoPath)
	g.P()
	g.P("package main")
	g.P()
	renderMessageClientJNIPreamble(g)
	g.P(`import "C"`)
	g.P()
	g.P("import (")
	g.P(`bytes "bytes"`)
	g.P(`context "context"`)
	g.P(`fmt "fmt"`)
	g.P(`protobuf "google.golang.org/protobuf/proto"`)
	g.P(`rpcruntime "`, rpcruntimeImportPath, `"`)
	g.P(`unsafe "unsafe"`)
	g.P(")")
	g.P()
	g.P("// ", messageStageMarker(service, file))
	g.P()

	renderJNIResultHelpers(g)
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderJNIUnaryMethod(g, service, method, servicePackage, plan.JNIClass)
		case StreamingKindClientStreaming:
			renderJNIClientStreamingMethods(g, service, method, servicePackage, plan.JNIClass)
		case StreamingKindServerStreaming:
			renderJNIServerStreamingMethods(g, service, method, servicePackage, plan.JNIClass)
		case StreamingKindBidiStreaming:
			renderJNIBidiStreamingMethods(g, service, method, servicePackage, plan.JNIClass)
		}
	}
	return nil
}

func renderMessageClientJNIPreamble(g *protogen.GeneratedFile) {
	g.P("/*")
	g.P("#include <jni.h>")
	g.P("#include <stdlib.h>")
	g.P()
	g.P("static int rpccgoJNIByteArrayIsNull(jbyteArray value) {")
	g.P("\treturn value == NULL;")
	g.P("}")
	g.P()
	g.P("static int rpccgoJNIByteArrayLength(JNIEnv* env, jbyteArray value) {")
	g.P("\tif (value == NULL) { return 0; }")
	g.P("\treturn (int)(*env)->GetArrayLength(env, value);")
	g.P("}")
	g.P()
	g.P("static void rpccgoJNIGetByteArrayRegion(JNIEnv* env, jbyteArray value, int length, void* dst) {")
	g.P("\tif (value == NULL || length == 0 || dst == NULL) { return; }")
	g.P("\t(*env)->GetByteArrayRegion(env, value, 0, length, (jbyte*)dst);")
	g.P("}")
	g.P()
	g.P("static jbyteArray rpccgoJNICALLocByteArray(JNIEnv* env, int length) {")
	g.P("\tif (length < 0) { return (jbyteArray)0; }")
	g.P("\treturn (*env)->NewByteArray(env, (jsize)length);")
	g.P("}")
	g.P()
	g.P("static void rpccgoJNISetByteArrayRegion(JNIEnv* env, jbyteArray value, int length, void* src) {")
	g.P("\tif (value == NULL || length == 0 || src == NULL) { return; }")
	g.P("\t(*env)->SetByteArrayRegion(env, value, 0, length, (jbyte*)src);")
	g.P("}")
	g.P("*/")
}

func renderJNIResultHelpers(g *protogen.GeneratedFile) {
	g.P("func rpccgoJNIBytes(env *C.JNIEnv, value C.jbyteArray) ([]byte, error) {")
	g.P("if int(C.rpccgoJNIByteArrayIsNull(value)) != 0 {")
	g.P(`return nil, fmt.Errorf("rpccgo: JNI request bytes are null")`)
	g.P("}")
	g.P("length := int(C.rpccgoJNIByteArrayLength(env, value))")
	g.P("if length < 0 {")
	g.P(`return nil, fmt.Errorf("rpccgo: JNI request length is negative")`)
	g.P("}")
	g.P("if length == 0 {")
	g.P("return nil, nil")
	g.P("}")
	g.P("data := make([]byte, length)")
	g.P("C.rpccgoJNIGetByteArrayRegion(env, value, C.int(length), unsafe.Pointer(&data[0]))")
	g.P("return data, nil")
	g.P("}")
	g.P()
	g.P("func rpccgoJNIByteArray(env *C.JNIEnv, data []byte) C.jbyteArray {")
	g.P("array := C.rpccgoJNICALLocByteArray(env, C.int(len(data)))")
	g.P("if int(C.rpccgoJNIByteArrayIsNull(array)) != 0 {")
	g.P("return 0")
	g.P("}")
	g.P("if len(data) == 0 {")
	g.P("return array")
	g.P("}")
	g.P("ptr := C.CBytes(data)")
	g.P("defer C.free(ptr)")
	g.P("C.rpccgoJNISetByteArrayRegion(env, array, C.int(len(data)), ptr)")
	g.P("return array")
	g.P("}")
	g.P()
	g.P("func rpccgoJNISuccess(payload []byte) []byte {")
	g.P("var out bytes.Buffer")
	g.P("out.WriteByte(1)")
	g.P("out.Write(rpccgoJNIInt32Payload(int32(len(payload))))")
	g.P("out.Write(payload)")
	g.P("return out.Bytes()")
	g.P("}")
	g.P()
	g.P("func rpccgoJNIInt32Payload(value int32) []byte {")
	g.P("var payload [4]byte")
	g.P("payload[0] = byte(value >> 24)")
	g.P("payload[1] = byte(value >> 16)")
	g.P("payload[2] = byte(value >> 8)")
	g.P("payload[3] = byte(value)")
	g.P("return payload[:]")
	g.P("}")
	g.P()
	g.P("func rpccgoJNIError(err error) []byte {")
	g.P("if err == nil {")
	g.P(`err = fmt.Errorf("rpccgo: unknown JNI error")`)
	g.P("}")
	g.P("payload := []byte(err.Error())")
	g.P("var out bytes.Buffer")
	g.P("out.WriteByte(0)")
	g.P("out.Write(rpccgoJNIInt32Payload(int32(len(payload))))")
	g.P("out.Write(payload)")
	g.P("return out.Bytes()")
	g.P("}")
	g.P()
	g.P("func rpccgoJNIResult(env *C.JNIEnv, payload []byte, err error) C.jbyteArray {")
	g.P("if err != nil {")
	g.P("return rpccgoJNIByteArray(env, rpccgoJNIError(err))")
	g.P("}")
	g.P("return rpccgoJNIByteArray(env, rpccgoJNISuccess(payload))")
	g.P("}")
	g.P()
}

func renderJNIUnaryMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method))
	renderCGOExportDoc(g, name, "invokes "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, request C.jbyteArray) C.jbyteArray {")
	g.P("ctx := context.Background()")
	renderJNIDecodeRequest(g, method)
	g.P("resp, err := ", servicePackage, "Invoke", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("if err != nil { return rpccgoJNIResult(env, nil, err) }")
	g.P("payload, err = protobuf.Marshal(resp)")
	g.P("return rpccgoJNIResult(env, payload, err)")
	g.P("}")
	g.P()
}

func renderJNIClientStreamingMethods(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	renderJNIStartNoRequest(g, service, method, servicePackage, jniClass)
	renderJNISend(g, service, method, servicePackage, jniClass)
	renderJNIFinishWithResponse(g, service, method, servicePackage, jniClass)
	renderJNICancel(g, service, method, servicePackage, jniClass)
}

func renderJNIServerStreamingMethods(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	renderJNIStartWithRequest(g, service, method, servicePackage, jniClass)
	renderJNIRecv(g, service, method, servicePackage, jniClass)
	renderJNIFinishVoid(g, service, method, servicePackage, jniClass)
	renderJNICancel(g, service, method, servicePackage, jniClass)
}

func renderJNIBidiStreamingMethods(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	renderJNIStartNoRequest(g, service, method, servicePackage, jniClass)
	renderJNISend(g, service, method, servicePackage, jniClass)
	renderJNIRecv(g, service, method, servicePackage, jniClass)
	renderJNICloseSend(g, service, method, servicePackage, jniClass)
	renderJNIFinishVoid(g, service, method, servicePackage, jniClass)
	renderJNICancel(g, service, method, servicePackage, jniClass)
}

func renderJNIStartNoRequest(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Start")
	renderCGOExportDoc(g, name, "starts "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("handle, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx)")
	g.P("return rpccgoJNIResult(env, rpccgoJNIInt32Payload(int32(handle)), err)")
	g.P("}")
	g.P()
}

func renderJNIStartWithRequest(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Start")
	renderCGOExportDoc(g, name, "starts "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, request C.jbyteArray) C.jbyteArray {")
	g.P("ctx := context.Background()")
	renderJNIDecodeRequest(g, method)
	g.P("handle, err := ", servicePackage, "Start", service.GoName, "Message", method.GoName, "(ctx, req)")
	g.P("return rpccgoJNIResult(env, rpccgoJNIInt32Payload(int32(handle)), err)")
	g.P("}")
	g.P()
}

func renderJNISend(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Send")
	renderCGOExportDoc(g, name, "sends to "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t, request C.jbyteArray) C.jbyteArray {")
	g.P("ctx := context.Background()")
	renderJNIDecodeRequest(g, method)
	g.P("err := ", servicePackage, "Send", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)), req)")
	g.P("return rpccgoJNIResult(env, nil, err)")
	g.P("}")
	g.P()
}

func renderJNIRecv(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Recv")
	renderCGOExportDoc(g, name, "receives from "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("resp, err := ", servicePackage, "Recv", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)))")
	g.P("if err != nil { return rpccgoJNIResult(env, nil, err) }")
	g.P("payload, err := protobuf.Marshal(resp)")
	g.P("return rpccgoJNIResult(env, payload, err)")
	g.P("}")
	g.P()
}

func renderJNIFinishWithResponse(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Finish")
	renderCGOExportDoc(g, name, "finishes "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("resp, err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)))")
	g.P("if err != nil { return rpccgoJNIResult(env, nil, err) }")
	g.P("payload, err := protobuf.Marshal(resp)")
	g.P("return rpccgoJNIResult(env, payload, err)")
	g.P("}")
	g.P()
}

func renderJNIFinishVoid(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Finish")
	renderCGOExportDoc(g, name, "finishes "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("err := ", servicePackage, "Finish", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)))")
	g.P("return rpccgoJNIResult(env, nil, err)")
	g.P("}")
	g.P()
}

func renderJNICloseSend(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"CloseSend")
	renderCGOExportDoc(g, name, "closes send for "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("err := ", servicePackage, "CloseSend", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)))")
	g.P("return rpccgoJNIResult(env, nil, err)")
	g.P("}")
	g.P()
}

func renderJNICancel(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, servicePackage, jniClass string) {
	name := jniExportName(jniClass, jniKotlinNativePrefix(service, method)+"Cancel")
	renderCGOExportDoc(g, name, "cancels "+method.FullName+" through the JNI message client bridge.")
	g.P("//export ", name)
	g.P("func ", name, "(env *C.JNIEnv, _ C.jobject, handle C.int32_t) C.jbyteArray {")
	g.P("ctx := context.Background()")
	g.P("err := ", servicePackage, "Cancel", service.GoName, "Message", method.GoName, "(ctx, rpcruntime.StreamHandle(int32(handle)))")
	g.P("return rpccgoJNIResult(env, nil, err)")
	g.P("}")
	g.P()
}

func renderJNIDecodeRequest(g *protogen.GeneratedFile, method MethodPlan) {
	g.P("payload, err := rpccgoJNIBytes(env, request)")
	g.P("if err != nil { return rpccgoJNIResult(env, nil, err) }")
	g.P("req := &", g.QualifiedGoIdent(protogen.GoIdent{GoName: method.Request.GoName, GoImportPath: protogen.GoImportPath(method.Request.GoImportPath)}), "{}")
	g.P("if err := protobuf.Unmarshal(payload, req); err != nil {")
	g.P(`return rpccgoJNIResult(env, nil, fmt.Errorf("rpccgo: JNI request decode failed: %w", err))`)
	g.P("}")
}

func jniExportName(jniClass, method string) string {
	return "Java_" + strings.ReplaceAll(jniClass, ".", "_") + "_" + method
}

func jniKotlinNativePrefix(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName
}

func jniClassPackageAndSimpleName(jniClass string) (string, string) {
	lastDot := strings.LastIndex(jniClass, ".")
	if lastDot < 0 {
		return "", jniClass
	}
	return jniClass[:lastDot], jniClass[lastDot+1:]
}

func rpccgoKotlinMessageType(message MethodIOPlan) string {
	pkg, name := rpccgoKotlinMessagePackageAndName(message)
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

func rpccgoKotlinMessagePackageAndName(message MethodIOPlan) (string, string) {
	lastDot := strings.LastIndex(message.FullName, ".")
	if lastDot < 0 {
		return "", message.GoName
	}
	return message.FullName[:lastDot], message.GoName
}
