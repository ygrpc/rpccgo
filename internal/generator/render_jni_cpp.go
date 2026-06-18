package generator

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderJNICPPFile(plugin *protogen.Plugin, file FilePlan, service ServicePlan, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniCPPFilename(file, service, config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P("// Source: ", file.ProtoPath)
	g.P()
	g.P("#include <jni.h>")
	g.P("#include <stdint.h>")
	g.P("#include <stdlib.h>")
	g.P()
	g.P("#include <string>")
	g.P("#include <vector>")
	g.P()
	g.P(`#include "`, config.RPCCGOHeader, `"`)
	g.P()
	g.P("JavaVM* javaVM = nullptr;")
	g.P()
	g.P("JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM* vm, void*) {")
	g.P("    javaVM = vm;")
	g.P("    // Android supports JNI 1.6; the JavaVM is also used to resolve JNIEnv on stream operation threads.")
	g.P("    return JNI_VERSION_1_6;")
	g.P("}")
	g.P()
	g.P("JNIEXPORT void JNICALL JNI_OnUnload(JavaVM*, void*) {")
	g.P("    javaVM = nullptr;")
	g.P("}")
	g.P()
	renderJNICPPHelpers(g)
	for i, method := range service.Methods {
		if i > 0 {
			g.P()
		}
		switch method.Streaming {
		case StreamingKindUnary:
			renderJNICPPUnary(g, file, service, method, config)
		case StreamingKindClientStreaming:
			renderJNICPPClientStreaming(g, file, service, method, config)
		case StreamingKindServerStreaming:
			renderJNICPPServerStreaming(g, file, service, method, config)
		case StreamingKindBidiStreaming:
			renderJNICPPBidiStreaming(g, file, service, method, config)
		}
	}
}

func renderJNICPPHelpers(g *protogen.GeneratedFile) {
	g.P("namespace {")
	g.P()
	g.P("class rpccgoJNIEnvScope {")
	g.P("public:")
	g.P("    explicit rpccgoJNIEnvScope(JNIEnv* current) : env(current), attached(false) {")
	g.P("        if (env != nullptr || javaVM == nullptr) { return; }")
	g.P("        void* rawEnv = nullptr;")
	g.P("        jint status = javaVM->GetEnv(&rawEnv, JNI_VERSION_1_6);")
	g.P("        if (status == JNI_OK) {")
	g.P("            env = static_cast<JNIEnv*>(rawEnv);")
	g.P("            return;")
	g.P("        }")
	g.P("        JNIEnv* attachedEnv = nullptr;")
	g.P("        if (status == JNI_EDETACHED && javaVM->AttachCurrentThread(&attachedEnv, nullptr) == JNI_OK) {")
	g.P("            env = attachedEnv;")
	g.P("            attached = true;")
	g.P("        }")
	g.P("    }")
	g.P("    ~rpccgoJNIEnvScope() {")
	g.P("        if (attached && javaVM != nullptr) { javaVM->DetachCurrentThread(); }")
	g.P("    }")
	g.P("    JNIEnv* env;")
	g.P()
	g.P("private:")
	g.P("    bool attached;")
	g.P("};")
	g.P()
	g.P("jbyteArray rpccgoJNIByteArray(JNIEnv* env, const std::vector<uint8_t>& data) {")
	g.P("    if (env == nullptr) { return nullptr; }")
	g.P("    jbyteArray array = env->NewByteArray(static_cast<jsize>(data.size()));")
	g.P("    if (array == nullptr) { return nullptr; }")
	g.P("    if (!data.empty()) {")
	g.P("        env->SetByteArrayRegion(array, 0, static_cast<jsize>(data.size()), reinterpret_cast<const jbyte*>(data.data()));")
	g.P("    }")
	g.P("    return array;")
	g.P("}")
	g.P()
	g.P("std::vector<uint8_t> rpccgoJNIBytes(JNIEnv* env, jbyteArray value, bool* ok) {")
	g.P("    if (ok != nullptr) { *ok = false; }")
	g.P("    if (env == nullptr || value == nullptr) { return {}; }")
	g.P("    jsize length = env->GetArrayLength(value);")
	g.P("    if (length < 0) { return {}; }")
	g.P("    std::vector<uint8_t> data(static_cast<size_t>(length));")
	g.P("    if (length != 0) {")
	g.P("        env->GetByteArrayRegion(value, 0, length, reinterpret_cast<jbyte*>(data.data()));")
	g.P("    }")
	g.P("    if (ok != nullptr) { *ok = !env->ExceptionCheck(); }")
	g.P("    return data;")
	g.P("}")
	g.P()
	g.P("void rpccgoWriteInt32(std::vector<uint8_t>* out, int32_t value) {")
	g.P("    out->push_back(static_cast<uint8_t>((value >> 24) & 0xff));")
	g.P("    out->push_back(static_cast<uint8_t>((value >> 16) & 0xff));")
	g.P("    out->push_back(static_cast<uint8_t>((value >> 8) & 0xff));")
	g.P("    out->push_back(static_cast<uint8_t>(value & 0xff));")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoResult(JNIEnv* env, bool ok, const std::vector<uint8_t>& payload) {")
	g.P("    std::vector<uint8_t> out;")
	g.P("    out.reserve(payload.size() + 5);")
	g.P("    out.push_back(ok ? 1 : 0);")
	g.P("    rpccgoWriteInt32(&out, static_cast<int32_t>(payload.size()));")
	g.P("    out.insert(out.end(), payload.begin(), payload.end());")
	g.P("    return rpccgoJNIByteArray(env, out);")
	g.P("}")
	g.P()
	g.P("std::vector<uint8_t> rpccgoErrorText(int32_t errID) {")
	g.P("    uintptr_t textPtr = 0;")
	g.P("    int32_t textLen = 0;")
	g.P("    int32_t status = rpccgoTakeErrorText(errID, &textPtr, &textLen);")
	g.P("    if (status != 0 || textLen < 0 || textPtr == 0) {")
	g.P(`        std::string fallback = "rpccgo: unknown error id " + std::to_string(errID);`)
	g.P("        return std::vector<uint8_t>(fallback.begin(), fallback.end());")
	g.P("    }")
	g.P("    const uint8_t* data = reinterpret_cast<const uint8_t*>(textPtr);")
	g.P("    std::vector<uint8_t> text(data, data + textLen);")
	g.P("    rpccgoRelease(textPtr);")
	g.P("    return text;")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoErrorResult(JNIEnv* env, const std::string& message) {")
	g.P("    return rpccgoResult(env, false, std::vector<uint8_t>(message.begin(), message.end()));")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoErrorIDResult(JNIEnv* env, int32_t errID) {")
	g.P("    return rpccgoResult(env, false, rpccgoErrorText(errID));")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoSuccessBytes(JNIEnv* env, uintptr_t responsePtr, int32_t responseLen) {")
	g.P("    if (responseLen < 0) { return rpccgoErrorResult(env, \"rpccgo: response length is negative\"); }")
	g.P("    if (responsePtr == 0 && responseLen != 0) { return rpccgoErrorResult(env, \"rpccgo: response pointer is null\"); }")
	g.P("    const uint8_t* data = reinterpret_cast<const uint8_t*>(responsePtr);")
	g.P("    std::vector<uint8_t> payload;")
	g.P("    if (responseLen != 0) { payload.assign(data, data + responseLen); }")
	g.P("    if (responsePtr != 0) { rpccgoRelease(responsePtr); }")
	g.P("    return rpccgoResult(env, true, payload);")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoSuccessHandle(JNIEnv* env, int32_t handle) {")
	g.P("    std::vector<uint8_t> payload;")
	g.P("    rpccgoWriteInt32(&payload, handle);")
	g.P("    return rpccgoResult(env, true, payload);")
	g.P("}")
	g.P()
	g.P("jbyteArray rpccgoSuccessUnit(JNIEnv* env) {")
	g.P("    return rpccgoResult(env, true, {});")
	g.P("}")
	g.P()
	g.P("uintptr_t rpccgoVectorPtr(const std::vector<uint8_t>& data) {")
	g.P("    if (data.empty()) { return 0; }")
	g.P("    return reinterpret_cast<uintptr_t>(data.data());")
	g.P("}")
	g.P()
	g.P("}  // namespace")
	g.P()
}

func renderJNICPPUnary(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method))
	cgoName := messageCExportFuncName(file, service, method, "")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jbyteArray request) {")
	renderJNICPPEnvScope(g)
	renderJNICPPRequestDecode(g)
	g.P("    uintptr_t responsePtr = 0;")
	g.P("    int32_t responseLen = 0;")
	g.P("    int32_t errID = ", cgoName, "(rpccgoVectorPtr(requestBytes), static_cast<int32_t>(requestBytes.size()), &responsePtr, &responseLen);")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessBytes(env, responsePtr, responseLen);")
	g.P("}")
}

func renderJNICPPClientStreaming(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	renderJNICPPStartNoRequest(g, file, service, method, config)
	g.P()
	renderJNICALLWithRequest(g, file, service, method, config, "Send", "send")
	g.P()
	renderJNICALLResponseByHandle(g, file, service, method, config, "Finish", "finish")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Cancel", "cancel")
}

func renderJNICPPServerStreaming(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	renderJNICPPStartWithRequest(g, file, service, method, config)
	g.P()
	renderJNICALLResponseByHandle(g, file, service, method, config, "Recv", "recv")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Finish", "finish")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Cancel", "cancel")
}

func renderJNICPPBidiStreaming(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	renderJNICPPStartNoRequest(g, file, service, method, config)
	g.P()
	renderJNICALLWithRequest(g, file, service, method, config, "Send", "send")
	g.P()
	renderJNICALLResponseByHandle(g, file, service, method, config, "Recv", "recv")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "CloseSend", "close_send")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Finish", "finish")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Cancel", "cancel")
}

func renderJNICPPStartNoRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"Start")
	cgoName := messageCExportFuncName(file, service, method, "start")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject) {")
	renderJNICPPEnvScope(g)
	g.P("    int32_t handle = 0;")
	g.P("    int32_t errID = ", cgoName, "(&handle);")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessHandle(env, handle);")
	g.P("}")
}

func renderJNICPPStartWithRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"Start")
	cgoName := messageCExportFuncName(file, service, method, "start")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jbyteArray request) {")
	renderJNICPPEnvScope(g)
	renderJNICPPRequestDecode(g)
	g.P("    int32_t handle = 0;")
	g.P("    int32_t errID = ", cgoName, "(rpccgoVectorPtr(requestBytes), static_cast<int32_t>(requestBytes.size()), &handle);")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessHandle(env, handle);")
	g.P("}")
}

func renderJNICALLWithRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig, kotlinSuffix, cgoOperation string) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+kotlinSuffix)
	cgoName := messageCExportFuncName(file, service, method, cgoOperation)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jint handle, jbyteArray request) {")
	renderJNICPPEnvScope(g)
	renderJNICPPRequestDecode(g)
	g.P("    int32_t errID = ", cgoName, "(static_cast<int32_t>(handle), rpccgoVectorPtr(requestBytes), static_cast<int32_t>(requestBytes.size()));")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessUnit(env);")
	g.P("}")
}

func renderJNICALLResponseByHandle(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig, kotlinSuffix, cgoOperation string) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+kotlinSuffix)
	cgoName := messageCExportFuncName(file, service, method, cgoOperation)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jint handle) {")
	renderJNICPPEnvScope(g)
	g.P("    uintptr_t responsePtr = 0;")
	g.P("    int32_t responseLen = 0;")
	g.P("    int32_t errID = ", cgoName, "(static_cast<int32_t>(handle), &responsePtr, &responseLen);")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessBytes(env, responsePtr, responseLen);")
	g.P("}")
}

func renderJNICALLUnitByHandle(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig, kotlinSuffix, cgoOperation string) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+kotlinSuffix)
	cgoName := messageCExportFuncName(file, service, method, cgoOperation)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jint handle) {")
	renderJNICPPEnvScope(g)
	g.P("    int32_t errID = ", cgoName, "(static_cast<int32_t>(handle));")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessUnit(env);")
	g.P("}")
}

func renderJNICPPEnvScope(g *protogen.GeneratedFile) {
	g.P("    rpccgoJNIEnvScope envScope(env);")
	g.P("    env = envScope.env;")
	g.P("    if (env == nullptr) { return nullptr; }")
}

func renderJNICPPRequestDecode(g *protogen.GeneratedFile) {
	g.P("    bool requestOK = false;")
	g.P("    std::vector<uint8_t> requestBytes = rpccgoJNIBytes(env, request, &requestOK);")
	g.P("    if (!requestOK) { return rpccgoErrorResult(env, \"rpccgo: JNI request bytes are null or unreadable\"); }")
}

func renderJNICPPExportComment(g *protogen.GeneratedFile, name string, method MethodPlan) {
	g.P("// ", name, " invokes ", method.FullName, " through the Android C++ JNI adapter.")
}

func jniExportName(jniClass, method string) string {
	return "Java_" + strings.ReplaceAll(jniClass, ".", "_") + "_" + method
}

func jniKotlinNativePrefix(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName
}
