package generator

import (
	"path"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

func renderJNICPPCommonHeaderFile(plugin *protogen.Plugin, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniCPPCommonHeaderFilename(config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P()
	g.P("#pragma once")
	g.P()
	g.P("#include <jni.h>")
	g.P("#include <stdint.h>")
	g.P()
	g.P("#include <string>")
	g.P("#include <vector>")
	g.P()
	g.P(`#include "`, config.RPCCGOHeader, `"`)
	g.P()
	g.P("extern JavaVM* javaVM;")
	g.P()
	g.P("class rpccgoJNIEnvScope {")
	g.P("public:")
	g.P("    explicit rpccgoJNIEnvScope(JNIEnv* current);")
	g.P("    ~rpccgoJNIEnvScope();")
	g.P()
	g.P("    JNIEnv* env;")
	g.P()
	g.P("private:")
	g.P("    bool attached;")
	g.P("};")
	g.P()
	g.P("jbyteArray rpccgoJNIByteArray(JNIEnv* env, const std::vector<uint8_t>& data);")
	g.P("std::vector<uint8_t> rpccgoJNIBytes(JNIEnv* env, jbyteArray value, bool* ok);")
	g.P("jbyteArray rpccgoResult(JNIEnv* env, bool ok, const std::vector<uint8_t>& payload);")
	g.P("jbyteArray rpccgoErrorResult(JNIEnv* env, const std::string& message);")
	g.P("jbyteArray rpccgoErrorIDResult(JNIEnv* env, int32_t errID);")
	g.P("jbyteArray rpccgoSuccessBytes(JNIEnv* env, uintptr_t responsePtr, int32_t responseLen);")
	g.P("jbyteArray rpccgoSuccessHandle(JNIEnv* env, int32_t handle);")
	g.P("jbyteArray rpccgoSuccessUnit(JNIEnv* env);")
	g.P("uintptr_t rpccgoVectorPtr(const std::vector<uint8_t>& data);")
	g.P("std::string rpccgoErrorString(int32_t errID);")
	g.P("int32_t rpccgoStoreErrorString(const std::string& message);")
	g.P("int32_t rpccgoExceptionError(JNIEnv* env, const std::string& message);")
	g.P("int32_t rpccgoRequestByteArray(JNIEnv* env, uintptr_t requestPtr, int32_t requestLen, jbyteArray* out);")
	g.P("int32_t rpccgoKotlinUnitResult(JNIEnv* env, jbyteArray result);")
	g.P("int32_t rpccgoKotlinBytesResult(JNIEnv* env, jbyteArray result, uintptr_t* responsePtr, int32_t* responseLen);")
	g.P("int32_t rpccgoKotlinHandleResult(JNIEnv* env, jbyteArray result, int32_t* stream);")
}

func renderJNICPPCommonFile(plugin *protogen.Plugin, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniCPPCommonFilename(config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P()
	g.P(`#include "`, path.Base(jniCPPCommonHeaderFilename(config)), `"`)
	g.P()
	g.P("#include <utility>")
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
}

func renderJNICPPFile(plugin *protogen.Plugin, file FilePlan, service ServicePlan, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniCPPServiceFilename(file, service, config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P("// Source: ", file.ProtoPath)
	g.P()
	g.P("#include <mutex>")
	g.P()
	g.P(`#include "`, path.Base(jniCPPCommonHeaderFilename(config)), `"`)
	g.P()
	renderJNICPPServerBridgeState(g, service)
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
		g.P()
		renderJNICPPServerRegistration(g, file, service, method, config)
	}
}

func renderJNICPPHelpers(g *protogen.GeneratedFile) {
	g.P("rpccgoJNIEnvScope::rpccgoJNIEnvScope(JNIEnv* current) : env(current), attached(false) {")
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
	g.P("}")
	g.P()
	g.P("rpccgoJNIEnvScope::~rpccgoJNIEnvScope() {")
	g.P("        if (attached && javaVM != nullptr) { javaVM->DetachCurrentThread(); }")
	g.P("}")
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
	g.P("std::string rpccgoErrorString(int32_t errID) {")
	g.P("    std::vector<uint8_t> text = rpccgoErrorText(errID);")
	g.P("    return std::string(text.begin(), text.end());")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoStoreErrorString(const std::string& message) {")
	g.P("    return rpccgoStoreErrorText(const_cast<char*>(message.data()), static_cast<int32_t>(message.size()));")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoExceptionError(JNIEnv* env, const std::string& message) {")
	g.P("    if (env != nullptr && env->ExceptionCheck()) { env->ExceptionClear(); }")
	g.P("    return rpccgoStoreErrorString(message);")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoRequestByteArray(JNIEnv* env, uintptr_t requestPtr, int32_t requestLen, jbyteArray* out) {")
	g.P("    if (out == nullptr) { return rpccgoStoreErrorString(\"rpccgo: JNI request output is nil\"); }")
	g.P("    *out = nullptr;")
	g.P("    if (requestLen < 0) { return rpccgoStoreErrorString(\"rpccgo: JNI request length is negative\"); }")
	g.P("    if (requestPtr == 0 && requestLen != 0) { return rpccgoStoreErrorString(\"rpccgo: JNI request pointer is null\"); }")
	g.P("    jbyteArray request = env->NewByteArray(requestLen);")
	g.P("    if (request == nullptr) { return rpccgoExceptionError(env, \"rpccgo: JNI request allocation failed\"); }")
	g.P("    if (requestLen > 0) {")
	g.P("        env->SetByteArrayRegion(request, 0, requestLen, reinterpret_cast<const jbyte*>(requestPtr));")
	g.P("        if (env->ExceptionCheck()) {")
	g.P("            env->DeleteLocalRef(request);")
	g.P("            return rpccgoExceptionError(env, \"rpccgo: JNI request copy failed\");")
	g.P("        }")
	g.P("    }")
	g.P("    *out = request;")
	g.P("    return 0;")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoReadResultPayload(JNIEnv* env, jbyteArray result, bool* ok, std::vector<uint8_t>* payload) {")
	g.P("    if (ok == nullptr || payload == nullptr) { return rpccgoStoreErrorString(\"rpccgo: JNI result output is nil\"); }")
	g.P("    bool resultOK = false;")
	g.P("    std::vector<uint8_t> data = rpccgoJNIBytes(env, result, &resultOK);")
	g.P("    if (!resultOK) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server returned unreadable result\"); }")
	g.P("    if (data.size() < 5) { return rpccgoStoreErrorString(\"rpccgo: Kotlin server returned malformed result\"); }")
	g.P("    *ok = data[0] != 0;")
	g.P("    int32_t length = (static_cast<int32_t>(data[1]) << 24) | (static_cast<int32_t>(data[2]) << 16) | (static_cast<int32_t>(data[3]) << 8) | static_cast<int32_t>(data[4]);")
	g.P("    if (length < 0 || static_cast<size_t>(length) != data.size() - 5) { return rpccgoStoreErrorString(\"rpccgo: Kotlin server returned invalid result length\"); }")
	g.P("    payload->assign(data.begin() + 5, data.end());")
	g.P("    return 0;")
	g.P("}")
	g.P()
	g.P("thread_local std::vector<uint8_t> rpccgoServerResponse;")
	g.P()
	g.P("int32_t rpccgoKotlinUnitResult(JNIEnv* env, jbyteArray result) {")
	g.P("    bool ok = false;")
	g.P("    std::vector<uint8_t> payload;")
	g.P("    int32_t errID = rpccgoReadResultPayload(env, result, &ok, &payload);")
	g.P("    if (errID != 0) { return errID; }")
	g.P("    if (!ok) { return rpccgoStoreErrorString(std::string(payload.begin(), payload.end())); }")
	g.P("    return 0;")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoKotlinBytesResult(JNIEnv* env, jbyteArray result, uintptr_t* responsePtr, int32_t* responseLen) {")
	g.P("    if (responsePtr == nullptr || responseLen == nullptr) { return rpccgoStoreErrorString(\"rpccgo: JNI response output is nil\"); }")
	g.P("    bool ok = false;")
	g.P("    std::vector<uint8_t> payload;")
	g.P("    int32_t errID = rpccgoReadResultPayload(env, result, &ok, &payload);")
	g.P("    if (errID != 0) { return errID; }")
	g.P("    if (!ok) { return rpccgoStoreErrorString(std::string(payload.begin(), payload.end())); }")
	g.P("    rpccgoServerResponse = std::move(payload);")
	g.P("    *responseLen = static_cast<int32_t>(rpccgoServerResponse.size());")
	g.P("    *responsePtr = rpccgoServerResponse.empty() ? 0 : reinterpret_cast<uintptr_t>(rpccgoServerResponse.data());")
	g.P("    return 0;")
	g.P("}")
	g.P()
	g.P("int32_t rpccgoKotlinHandleResult(JNIEnv* env, jbyteArray result, int32_t* stream) {")
	g.P("    if (stream == nullptr) { return rpccgoStoreErrorString(\"rpccgo: JNI stream output is nil\"); }")
	g.P("    bool ok = false;")
	g.P("    std::vector<uint8_t> payload;")
	g.P("    int32_t errID = rpccgoReadResultPayload(env, result, &ok, &payload);")
	g.P("    if (errID != 0) { return errID; }")
	g.P("    if (!ok) { return rpccgoStoreErrorString(std::string(payload.begin(), payload.end())); }")
	g.P("    if (payload.size() != 4) { return rpccgoStoreErrorString(\"rpccgo: Kotlin server returned invalid stream handle\"); }")
	g.P("    *stream = (static_cast<int32_t>(payload[0]) << 24) | (static_cast<int32_t>(payload[1]) << 16) | (static_cast<int32_t>(payload[2]) << 8) | static_cast<int32_t>(payload[3]);")
	g.P("    return 0;")
	g.P("}")
	g.P()
}

func renderJNICPPServerBridgeState(g *protogen.GeneratedFile, service ServicePlan) {
	prefix := lowerInitial(service.GoName)
	g.P("std::mutex ", prefix, "ServerBridgeMu;")
	g.P("jobject ", prefix, "ServerBridge = nullptr;")
	for _, method := range service.Methods {
		for _, op := range jniServerBridgeOperations(method) {
			g.P("jmethodID ", jniServerBridgeMethodID(service, method, op), " = nullptr;")
		}
	}
	g.P()
	g.P("bool ensure", service.GoName, "ServerBridge(JNIEnv* env, jobject thiz) {")
	g.P("    if (env == nullptr || thiz == nullptr) { return false; }")
	g.P("    std::lock_guard<std::mutex> lock(", prefix, "ServerBridgeMu);")
	g.P("    if (", prefix, "ServerBridge == nullptr) {")
	g.P("        ", prefix, "ServerBridge = env->NewGlobalRef(thiz);")
	g.P("    }")
	g.P("    return ", prefix, "ServerBridge != nullptr;")
	g.P("}")
	g.P()
	g.P("jobject ", prefix, "ServerBridgeObject() {")
	g.P("    std::lock_guard<std::mutex> lock(", prefix, "ServerBridgeMu);")
	g.P("    return ", prefix, "ServerBridge;")
	g.P("}")
	g.P()
}

func renderJNICPPUnary(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method))
	cgoName := messageCExportFuncName(file, service, method, "")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jbyteArray request) {")
	renderJNICPPEnvScope(g, "nullptr")
	renderJNICPPRequestDecode(g, `rpccgoErrorResult(env, "rpccgo: JNI request bytes are null or unreadable")`)
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
	renderJNICPPCallbackState(g, file, service, method)
	g.P()
	renderJNICPPStartWithRequest(g, file, service, method, config)
	g.P()
	renderJNICALLResponseByHandle(g, file, service, method, config, "Recv", "recv")
	g.P()
	renderJNICALLUnitByHandle(g, file, service, method, config, "Cancel", "cancel")
	g.P()
	renderJNICALLStartCallbackWithRequest(g, file, service, method, config)
	g.P()
	renderJNICALLCancelCallback(g, file, service, method, config)
}

func renderJNICPPBidiStreaming(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	renderJNICPPCallbackState(g, file, service, method)
	g.P()
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
	g.P()
	renderJNICALLStartCallbackNoRequest(g, file, service, method, config)
	g.P()
	renderJNICALLCancelCallback(g, file, service, method, config)
}

func renderJNICPPStartNoRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"Start")
	cgoName := messageCExportFuncName(file, service, method, "start")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject) {")
	renderJNICPPEnvScope(g, "nullptr")
	g.P("    int32_t handle = 0;")
	if method.Streaming == StreamingKindBidiStreaming {
		g.P("    int32_t errID = ", cgoName, "(&handle, nullptr, nullptr);")
	} else {
		g.P("    int32_t errID = ", cgoName, "(&handle);")
	}
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessHandle(env, handle);")
	g.P("}")
}

func renderJNICPPStartWithRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"Start")
	cgoName := messageCExportFuncName(file, service, method, "start")
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jbyteArray request) {")
	renderJNICPPEnvScope(g, "nullptr")
	renderJNICPPRequestDecode(g, `rpccgoErrorResult(env, "rpccgo: JNI request bytes are null or unreadable")`)
	g.P("    int32_t handle = 0;")
	g.P("    int32_t errID = ", cgoName, "(rpccgoVectorPtr(requestBytes), static_cast<int32_t>(requestBytes.size()), &handle, nullptr, nullptr);")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessHandle(env, handle);")
	g.P("}")
}

func renderJNICALLWithRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig, kotlinSuffix, cgoOperation string) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+kotlinSuffix)
	cgoName := messageCExportFuncName(file, service, method, cgoOperation)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject, jint handle, jbyteArray request) {")
	renderJNICPPEnvScope(g, "nullptr")
	renderJNICPPRequestDecode(g, `rpccgoErrorResult(env, "rpccgo: JNI request bytes are null or unreadable")`)
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
	renderJNICPPEnvScope(g, "nullptr")
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
	renderJNICPPEnvScope(g, "nullptr")
	g.P("    int32_t errID = ", cgoName, "(static_cast<int32_t>(handle));")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessUnit(env);")
	g.P("}")
}

func renderJNICPPCallbackState(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan) {
	prefix := jniCPPCallbackPrefix(service, method)
	listenerType := jniKotlinListenerType(service, method)
	cancelName := messageCExportFuncName(file, service, method, "cancel")
	g.P("std::mutex ", prefix, "Mu;")
	g.P("jobject ", prefix, "Listener = nullptr;")
	g.P("jmethodID ", prefix, "OnRecv = nullptr;")
	g.P("jmethodID ", prefix, "OnDone = nullptr;")
	g.P("int32_t ", prefix, "Handle = 0;")
	g.P()
	g.P("void clear", listenerType, "Callback(JNIEnv* env) {")
	g.P("    std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("    if (env != nullptr && ", prefix, "Listener != nullptr) {")
	g.P("        env->DeleteGlobalRef(", prefix, "Listener);")
	g.P("    }")
	g.P("    ", prefix, "Listener = nullptr;")
	g.P("    ", prefix, "OnRecv = nullptr;")
	g.P("    ", prefix, "OnDone = nullptr;")
	g.P("    ", prefix, "Handle = 0;")
	g.P("}")
	g.P()
	g.P("bool cancel", listenerType, "Callback(JNIEnv* env) {")
	g.P("    int32_t handle = 0;")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        handle = ", prefix, "Handle;")
	g.P("    }")
	g.P("    if (handle == 0) {")
	g.P("        clear", listenerType, "Callback(env);")
	g.P("        return true;")
	g.P("    }")
	g.P("    int32_t errID = ", cancelName, "(handle);")
	g.P("    return errID == 0;")
	g.P("}")
	g.P()
	g.P("void on", listenerType, "Recv(int32_t, uintptr_t responsePtr, int32_t responseLen) {")
	g.P("    rpccgoJNIEnvScope envScope(nullptr);")
	g.P("    JNIEnv* env = envScope.env;")
	g.P("    if (env == nullptr) {")
	g.P("        if (responsePtr != 0) { rpccgoRelease(responsePtr); }")
	g.P("        return;")
	g.P("    }")
	g.P("    std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("    if (", prefix, "Listener == nullptr || ", prefix, "OnRecv == nullptr) {")
	g.P("        if (responsePtr != 0) { rpccgoRelease(responsePtr); }")
	g.P("        return;")
	g.P("    }")
	g.P("    jbyteArray payload = env->NewByteArray(responseLen);")
	g.P("    if (payload == nullptr) {")
	g.P("        if (responsePtr != 0) { rpccgoRelease(responsePtr); }")
	g.P("        return;")
	g.P("    }")
	g.P("    if (responseLen > 0) {")
	g.P("        env->SetByteArrayRegion(payload, 0, responseLen, reinterpret_cast<const jbyte*>(responsePtr));")
	g.P("    }")
	g.P("    if (responsePtr != 0) { rpccgoRelease(responsePtr); }")
	g.P("    env->CallVoidMethod(", prefix, "Listener, ", prefix, "OnRecv, payload);")
	g.P("    env->DeleteLocalRef(payload);")
	g.P("    if (env->ExceptionCheck()) { env->ExceptionClear(); }")
	g.P("}")
	g.P()
	g.P("void on", listenerType, "Done(int32_t, int32_t errID) {")
	g.P("    rpccgoJNIEnvScope envScope(nullptr);")
	g.P("    JNIEnv* env = envScope.env;")
	g.P("    if (env == nullptr) { return; }")
	g.P("    jobject listener = nullptr;")
	g.P("    jmethodID onDone = nullptr;")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        listener = ", prefix, "Listener;")
	g.P("        onDone = ", prefix, "OnDone;")
	g.P("    }")
	g.P("    if (listener != nullptr && onDone != nullptr) {")
	g.P("        jstring error = nullptr;")
	g.P("        if (errID != 0) {")
	g.P("            std::string text = rpccgoErrorString(errID);")
	g.P("            error = env->NewStringUTF(text.c_str());")
	g.P("        }")
	g.P("        env->CallVoidMethod(listener, onDone, error);")
	g.P("        if (error != nullptr) { env->DeleteLocalRef(error); }")
	g.P("        if (env->ExceptionCheck()) { env->ExceptionClear(); }")
	g.P("    }")
	g.P("    clear", listenerType, "Callback(env);")
	g.P("}")
}

func renderJNICALLStartCallbackWithRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"StartCallback")
	cgoName := messageCExportFuncName(file, service, method, "start")
	listenerType := jniKotlinListenerType(service, method)
	prefix := jniCPPCallbackPrefix(service, method)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jboolean JNICALL ", name, "(JNIEnv* env, jobject, jbyteArray request, jobject listener) {")
	renderJNICPPEnvScope(g, "JNI_FALSE")
	g.P("    if (request == nullptr || listener == nullptr) { return JNI_FALSE; }")
	g.P("    cancel", listenerType, "Callback(env);")
	g.P("    jclass listenerClass = env->GetObjectClass(listener);")
	g.P("    jmethodID onRecv = env->GetMethodID(listenerClass, \"onRecv\", \"([B)V\");")
	g.P("    jmethodID onDone = env->GetMethodID(listenerClass, \"onDone\", \"(Ljava/lang/String;)V\");")
	g.P("    env->DeleteLocalRef(listenerClass);")
	g.P("    if (onRecv == nullptr || onDone == nullptr) { return JNI_FALSE; }")
	renderJNICPPRequestDecode(g, "JNI_FALSE")
	g.P("    jobject globalListener = env->NewGlobalRef(listener);")
	g.P("    if (globalListener == nullptr) { return JNI_FALSE; }")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        ", prefix, "Listener = globalListener;")
	g.P("        ", prefix, "OnRecv = onRecv;")
	g.P("        ", prefix, "OnDone = onDone;")
	g.P("    }")
	g.P("    int32_t handle = 0;")
	g.P("    int32_t errID = ", cgoName, "(rpccgoVectorPtr(requestBytes), static_cast<int32_t>(requestBytes.size()), &handle, on", listenerType, "Recv, on", listenerType, "Done);")
	g.P("    if (errID != 0) {")
	g.P("        clear", listenerType, "Callback(env);")
	g.P("        return JNI_FALSE;")
	g.P("    }")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        ", prefix, "Handle = handle;")
	g.P("    }")
	g.P("    return JNI_TRUE;")
	g.P("}")
}

func renderJNICALLStartCallbackNoRequest(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"StartCallback")
	cgoName := messageCExportFuncName(file, service, method, "start")
	listenerType := jniKotlinListenerType(service, method)
	prefix := jniCPPCallbackPrefix(service, method)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jboolean JNICALL ", name, "(JNIEnv* env, jobject, jobject listener) {")
	renderJNICPPEnvScope(g, "JNI_FALSE")
	g.P("    if (listener == nullptr) { return JNI_FALSE; }")
	g.P("    cancel", listenerType, "Callback(env);")
	g.P("    jclass listenerClass = env->GetObjectClass(listener);")
	g.P("    jmethodID onRecv = env->GetMethodID(listenerClass, \"onRecv\", \"([B)V\");")
	g.P("    jmethodID onDone = env->GetMethodID(listenerClass, \"onDone\", \"(Ljava/lang/String;)V\");")
	g.P("    env->DeleteLocalRef(listenerClass);")
	g.P("    if (onRecv == nullptr || onDone == nullptr) { return JNI_FALSE; }")
	g.P("    jobject globalListener = env->NewGlobalRef(listener);")
	g.P("    if (globalListener == nullptr) { return JNI_FALSE; }")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        ", prefix, "Listener = globalListener;")
	g.P("        ", prefix, "OnRecv = onRecv;")
	g.P("        ", prefix, "OnDone = onDone;")
	g.P("    }")
	g.P("    int32_t handle = 0;")
	g.P("    int32_t errID = ", cgoName, "(&handle, on", listenerType, "Recv, on", listenerType, "Done);")
	g.P("    if (errID != 0) {")
	g.P("        clear", listenerType, "Callback(env);")
	g.P("        return JNI_FALSE;")
	g.P("    }")
	g.P("    {")
	g.P("        std::lock_guard<std::mutex> lock(", prefix, "Mu);")
	g.P("        ", prefix, "Handle = handle;")
	g.P("    }")
	g.P("    return JNI_TRUE;")
	g.P("}")
}

func renderJNICALLCancelCallback(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"CancelCallback")
	listenerType := jniKotlinListenerType(service, method)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jboolean JNICALL ", name, "(JNIEnv* env, jobject) {")
	renderJNICPPEnvScope(g, "JNI_FALSE")
	g.P("    return cancel", listenerType, "Callback(env) ? JNI_TRUE : JNI_FALSE;")
	g.P("}")
}

func renderJNICPPServerRegistration(g *protogen.GeneratedFile, file FilePlan, service ServicePlan, method MethodPlan, config JNIGeneratorConfig) {
	for _, op := range jniServerBridgeOperations(method) {
		renderJNICPPServerCallback(g, service, method, op)
		g.P()
	}
	name := jniExportName(config.JNIClass, jniKotlinNativePrefix(service, method)+"Register")
	registerName := messageCServiceMethodRegisterExportFuncName(file, service, method)
	renderJNICPPExportComment(g, name, method)
	g.P("extern \"C\" JNIEXPORT jbyteArray JNICALL ", name, "(JNIEnv* env, jobject thiz) {")
	renderJNICPPEnvScope(g, "nullptr")
	g.P("    if (!ensure", service.GoName, "ServerBridge(env, thiz)) { return rpccgoErrorResult(env, \"rpccgo: JNI server bridge registration failed\"); }")
	g.P("    jclass bridgeClass = env->GetObjectClass(thiz);")
	g.P("    if (bridgeClass == nullptr) { return rpccgoErrorResult(env, \"rpccgo: JNI server bridge class lookup failed\"); }")
	for _, op := range jniServerBridgeOperations(method) {
		g.P("    ", jniServerBridgeMethodID(service, method, op), " = env->GetMethodID(bridgeClass, \"", jniServerKotlinBridgeMethod(service, method, op), "\", \"", jniServerKotlinBridgeSignature(method, op), "\");")
	}
	g.P("    env->DeleteLocalRef(bridgeClass);")
	g.P("    if (env->ExceptionCheck()) { return rpccgoErrorResult(env, \"rpccgo: JNI server bridge method lookup failed\"); }")
	for _, op := range jniServerBridgeOperations(method) {
		g.P("    if (", jniServerBridgeMethodID(service, method, op), " == nullptr) { return rpccgoErrorResult(env, \"rpccgo: JNI server bridge method is missing\"); }")
	}
	g.P("    int32_t errID = ", registerName, "(", strings.Join(jniServerCallbackArgs(service, method), ", "), ");")
	g.P("    if (errID != 0) { return rpccgoErrorIDResult(env, errID); }")
	g.P("    return rpccgoSuccessUnit(env);")
	g.P("}")
}

func renderJNICPPServerCallback(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, op string) {
	switch op {
	case "Handle":
		g.P("int32_t ", jniServerCallbackName(service, method, op), "(uintptr_t requestPtr, int32_t requestLen, uintptr_t* responsePtr, int32_t* responseLen) {")
		renderJNICPPServerCallbackPreamble(g, service, method, op)
		g.P("    jbyteArray request = nullptr;")
		g.P("    int32_t requestErr = rpccgoRequestByteArray(env, requestPtr, requestLen, &request);")
		g.P("    if (requestErr != 0) { return requestErr; }")
		g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), ", request));")
		g.P("    env->DeleteLocalRef(request);")
		g.P("    if (env->ExceptionCheck()) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server unary handler threw\"); }")
		g.P("    int32_t errID = rpccgoKotlinBytesResult(env, result, responsePtr, responseLen);")
		g.P("    if (result != nullptr) { env->DeleteLocalRef(result); }")
		g.P("    return errID;")
		g.P("}")
	case "Start":
		renderJNICPPServerStartCallback(g, service, method)
	case "Send":
		renderJNICPPServerRequestUnitCallback(g, service, method, op)
	case "Recv":
		renderJNICPPServerResponseCallback(g, service, method, op)
	case "Finish":
		if method.Streaming == StreamingKindClientStreaming {
			renderJNICPPServerResponseCallback(g, service, method, op)
		} else {
			renderJNICPPServerUnitCallback(g, service, method, op)
		}
	case "CloseSend", "Cancel":
		renderJNICPPServerUnitCallback(g, service, method, op)
	}
}

func renderJNICPPServerStartCallback(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	op := "Start"
	if method.Streaming == StreamingKindServerStreaming {
		g.P("int32_t ", jniServerCallbackName(service, method, op), "(uintptr_t requestPtr, int32_t requestLen, int32_t* stream) {")
		renderJNICPPServerCallbackPreamble(g, service, method, op)
		g.P("    jbyteArray request = nullptr;")
		g.P("    int32_t requestErr = rpccgoRequestByteArray(env, requestPtr, requestLen, &request);")
		g.P("    if (requestErr != 0) { return requestErr; }")
		g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), ", request));")
		g.P("    env->DeleteLocalRef(request);")
	} else {
		g.P("int32_t ", jniServerCallbackName(service, method, op), "(int32_t* stream) {")
		renderJNICPPServerCallbackPreamble(g, service, method, op)
		g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), "));")
	}
	g.P("    if (env->ExceptionCheck()) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server stream start threw\"); }")
	g.P("    int32_t errID = rpccgoKotlinHandleResult(env, result, stream);")
	g.P("    if (result != nullptr) { env->DeleteLocalRef(result); }")
	g.P("    return errID;")
	g.P("}")
}

func renderJNICPPServerRequestUnitCallback(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, op string) {
	g.P("int32_t ", jniServerCallbackName(service, method, op), "(int32_t stream, uintptr_t requestPtr, int32_t requestLen) {")
	renderJNICPPServerCallbackPreamble(g, service, method, op)
	g.P("    jbyteArray request = nullptr;")
	g.P("    int32_t requestErr = rpccgoRequestByteArray(env, requestPtr, requestLen, &request);")
	g.P("    if (requestErr != 0) { return requestErr; }")
	g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), ", stream, request));")
	g.P("    env->DeleteLocalRef(request);")
	g.P("    if (env->ExceptionCheck()) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server stream request handler threw\"); }")
	g.P("    int32_t errID = rpccgoKotlinUnitResult(env, result);")
	g.P("    if (result != nullptr) { env->DeleteLocalRef(result); }")
	g.P("    return errID;")
	g.P("}")
}

func renderJNICPPServerResponseCallback(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, op string) {
	g.P("int32_t ", jniServerCallbackName(service, method, op), "(int32_t stream, uintptr_t* responsePtr, int32_t* responseLen) {")
	renderJNICPPServerCallbackPreamble(g, service, method, op)
	g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), ", stream));")
	g.P("    if (env->ExceptionCheck()) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server stream response handler threw\"); }")
	g.P("    int32_t errID = rpccgoKotlinBytesResult(env, result, responsePtr, responseLen);")
	g.P("    if (result != nullptr) { env->DeleteLocalRef(result); }")
	g.P("    return errID;")
	g.P("}")
}

func renderJNICPPServerUnitCallback(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, op string) {
	g.P("int32_t ", jniServerCallbackName(service, method, op), "(int32_t stream) {")
	renderJNICPPServerCallbackPreamble(g, service, method, op)
	g.P("    jbyteArray result = static_cast<jbyteArray>(env->CallObjectMethod(bridge, ", jniServerBridgeMethodID(service, method, op), ", stream));")
	g.P("    if (env->ExceptionCheck()) { return rpccgoExceptionError(env, \"rpccgo: Kotlin server stream control handler threw\"); }")
	g.P("    int32_t errID = rpccgoKotlinUnitResult(env, result);")
	g.P("    if (result != nullptr) { env->DeleteLocalRef(result); }")
	g.P("    return errID;")
	g.P("}")
}

func renderJNICPPServerCallbackPreamble(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, op string) {
	g.P("    rpccgoJNIEnvScope envScope(nullptr);")
	g.P("    JNIEnv* env = envScope.env;")
	g.P("    if (env == nullptr) { return rpccgoStoreErrorString(\"rpccgo: JNI environment is unavailable\"); }")
	g.P("    jobject bridge = ", lowerInitial(service.GoName), "ServerBridgeObject();")
	g.P("    if (bridge == nullptr || ", jniServerBridgeMethodID(service, method, op), " == nullptr) { return rpccgoStoreErrorString(\"rpccgo: Kotlin server bridge is not registered\"); }")
}

func renderJNICPPEnvScope(g *protogen.GeneratedFile, failReturn string) {
	g.P("    rpccgoJNIEnvScope envScope(env);")
	g.P("    env = envScope.env;")
	g.P("    if (env == nullptr) { return ", failReturn, "; }")
}

func renderJNICPPRequestDecode(g *protogen.GeneratedFile, failReturn string) {
	g.P("    bool requestOK = false;")
	g.P("    std::vector<uint8_t> requestBytes = rpccgoJNIBytes(env, request, &requestOK);")
	g.P("    if (!requestOK) { return ", failReturn, "; }")
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

func jniCPPCallbackPrefix(service ServicePlan, method MethodPlan) string {
	return lowerInitial(service.GoName) + method.GoName + "Callback"
}

func jniServerBridgeOperations(method MethodPlan) []string {
	switch method.Streaming {
	case StreamingKindUnary:
		return []string{"Handle"}
	case StreamingKindClientStreaming:
		return []string{"Start", "Send", "Finish", "Cancel"}
	case StreamingKindServerStreaming:
		return []string{"Start", "Recv", "Finish", "Cancel"}
	case StreamingKindBidiStreaming:
		return []string{"Start", "Send", "Recv", "CloseSend", "Finish", "Cancel"}
	default:
		return nil
	}
}

func jniServerBridgeMethodID(service ServicePlan, method MethodPlan, op string) string {
	return lowerInitial(service.GoName) + method.GoName + "Server" + op
}

func jniServerKotlinBridgeMethod(service ServicePlan, method MethodPlan, op string) string {
	if op == "Handle" {
		return jniKotlinNativePrefix(service, method) + "Handle"
	}
	return jniKotlinNativePrefix(service, method) + "Server" + op
}

func jniServerKotlinBridgeSignature(method MethodPlan, op string) string {
	switch op {
	case "Handle":
		return "([B)[B"
	case "Start":
		if method.Streaming == StreamingKindServerStreaming {
			return "([B)[B"
		}
		return "()[B"
	case "Send":
		return "(I[B)[B"
	default:
		return "(I)[B"
	}
}

func jniServerCallbackName(service ServicePlan, method MethodPlan, op string) string {
	return "on" + service.GoName + method.GoName + "Server" + op
}

func jniServerCallbackArgs(service ServicePlan, method MethodPlan) []string {
	ops := jniServerBridgeOperations(method)
	args := make([]string, 0, len(ops))
	for _, op := range ops {
		args = append(args, jniServerCallbackName(service, method, op))
	}
	return args
}
