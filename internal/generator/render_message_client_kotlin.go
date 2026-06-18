package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderJNIKotlinFile(plugin *protogen.Plugin, plan FilePlan, service ServicePlan, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniKotlinFilename(config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P("// Source: ", plan.ProtoPath)
	g.P()
	pkg, className := jniClassPackageAndSimpleName(config.JNIClass)
	g.P("package ", pkg)
	g.P()
	g.P("import java.nio.ByteBuffer")
	g.P("import java.nio.ByteOrder")
	g.P()
	g.P("data class RpccgoResult<T>(val value: T?, val error: String?) {")
	g.P("    val ok: Boolean get() = error == null")
	g.P("    companion object {")
	g.P("        fun <T> success(value: T): RpccgoResult<T> = RpccgoResult(value, null)")
	g.P("        fun <T> failure(error: String): RpccgoResult<T> = RpccgoResult(null, error)")
	g.P("    }")
	g.P("}")
	g.P()
	g.P("object ", className, " {")
	for _, method := range service.Methods {
		renderKotlinNativeDeclarations(g, service, method)
	}
	g.P()
	for _, method := range service.Methods {
		switch method.Streaming {
		case StreamingKindUnary:
			renderKotlinUnaryMethod(g, service, method)
		case StreamingKindClientStreaming:
			renderKotlinClientStreamingMethod(g, service, method, className)
		case StreamingKindServerStreaming:
			renderKotlinServerStreamingMethod(g, service, method, className)
		case StreamingKindBidiStreaming:
			renderKotlinBidiStreamingMethod(g, service, method, className)
		}
	}
	g.P("    private fun decodeResultPayload(bytes: ByteArray?): RpccgoResult<ByteArray> {")
	g.P(`        if (bytes == null) return RpccgoResult.failure("rpccgo: JNI returned null")`)
	g.P(`        if (bytes.size < 5) return RpccgoResult.failure("rpccgo: JNI returned malformed result")`)
	g.P("        val ok = bytes[0].toInt() != 0")
	g.P("        val length = ByteBuffer.wrap(bytes, 1, 4).order(ByteOrder.BIG_ENDIAN).int")
	g.P(`        if (length < 0 || length != bytes.size - 5) return RpccgoResult.failure("rpccgo: JNI returned invalid result length")`)
	g.P("        val payload = bytes.copyOfRange(5, bytes.size)")
	g.P("        if (!ok) return RpccgoResult.failure(payload.toString(Charsets.UTF_8))")
	g.P("        return RpccgoResult.success(payload)")
	g.P("    }")
	g.P()
	g.P("    private fun <T> decodeResult(bytes: ByteArray?, parser: (ByteArray) -> T): RpccgoResult<T> {")
	g.P("        val payload = decodeResultPayload(bytes)")
	g.P(`        if (!payload.ok) return RpccgoResult.failure(payload.error ?: "rpccgo: JNI call failed")`)
	g.P("        return try {")
	g.P("            RpccgoResult.success(parser(payload.value ?: ByteArray(0)))")
	g.P("        } catch (e: Exception) {")
	g.P(`            RpccgoResult.failure("rpccgo: JNI payload decode failed: ${e.message ?: e::class.java.name}")`)
	g.P("        }")
	g.P("    }")
	g.P()
	g.P("    private fun decodeUnitResult(bytes: ByteArray?): RpccgoResult<Unit> =")
	g.P("        decodeResult(bytes) { Unit }")
	g.P()
	g.P("    private fun decodeHandleResult(bytes: ByteArray?): RpccgoResult<Int> {")
	g.P("        val payload = decodeResultPayload(bytes)")
	g.P(`        if (!payload.ok) return RpccgoResult.failure(payload.error ?: "rpccgo: stream start failed")`)
	g.P("        val value = payload.value ?: ByteArray(0)")
	g.P(`        if (value.size != 4) return RpccgoResult.failure("rpccgo: JNI returned invalid stream handle")`)
	g.P("        return RpccgoResult.success(ByteBuffer.wrap(value).order(ByteOrder.BIG_ENDIAN).int)")
	g.P("    }")
	g.P("}")
	g.P()
}

func renderKotlinNativeDeclarations(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	prefix := lowerInitial(service.GoName) + method.GoName
	switch method.Streaming {
	case StreamingKindUnary:
		g.P("    private external fun ", prefix, "(request: ByteArray): ByteArray?")
	case StreamingKindClientStreaming:
		g.P("    private external fun ", prefix, "Start(): ByteArray?")
		g.P("    private external fun ", prefix, "Send(handle: Int, request: ByteArray): ByteArray?")
		g.P("    private external fun ", prefix, "Finish(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Cancel(handle: Int): ByteArray?")
	case StreamingKindServerStreaming:
		g.P("    private external fun ", prefix, "Start(request: ByteArray): ByteArray?")
		g.P("    private external fun ", prefix, "Recv(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Finish(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Cancel(handle: Int): ByteArray?")
	case StreamingKindBidiStreaming:
		g.P("    private external fun ", prefix, "Start(): ByteArray?")
		g.P("    private external fun ", prefix, "Send(handle: Int, request: ByteArray): ByteArray?")
		g.P("    private external fun ", prefix, "Recv(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "CloseSend(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Finish(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Cancel(handle: Int): ByteArray?")
	}
}

func renderKotlinUnaryMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := rpccgoKotlinMessageType(method.Request)
	respType := rpccgoKotlinMessageType(method.Response)
	nativeName := lowerInitial(service.GoName) + method.GoName
	g.P("    fun ", method.GoName, "(req: ", reqType, "): RpccgoResult<", respType, "> =")
	g.P("        decodeResult(", nativeName, "(req.toByteArray())) { ", respType, ".parseFrom(it) }")
	g.P()
}

func renderKotlinClientStreamingMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, className string) {
	reqType := rpccgoKotlinMessageType(method.Request)
	respType := rpccgoKotlinMessageType(method.Response)
	streamType := service.GoName + method.GoName + "ClientStream"
	nativeName := lowerInitial(service.GoName) + method.GoName
	g.P("    fun ", method.GoName, "Start(): RpccgoResult<", streamType, "> {")
	g.P("        val handle = decodeHandleResult(", nativeName, "Start())")
	g.P("        if (!handle.ok) return RpccgoResult.failure(handle.error ?: \"rpccgo: stream start failed\")")
	g.P("        return RpccgoResult.success(", streamType, "(handle.value ?: 0))")
	g.P("    }")
	g.P()
	g.P("    class ", streamType, " internal constructor(private val handle: Int) {")
	g.P("        fun Send(req: ", reqType, "): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Send(handle, req.toByteArray()))")
	g.P("        fun Finish(): RpccgoResult<", respType, "> =")
	g.P("            decodeResult(", className, ".", nativeName, "Finish(handle)) { ", respType, ".parseFrom(it) }")
	g.P("        fun Cancel(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Cancel(handle))")
	g.P("    }")
	g.P()
}

func renderKotlinServerStreamingMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, className string) {
	reqType := rpccgoKotlinMessageType(method.Request)
	respType := rpccgoKotlinMessageType(method.Response)
	streamType := service.GoName + method.GoName + "ServerStream"
	nativeName := lowerInitial(service.GoName) + method.GoName
	g.P("    fun ", method.GoName, "Start(req: ", reqType, "): RpccgoResult<", streamType, "> {")
	g.P("        val handle = decodeHandleResult(", nativeName, "Start(req.toByteArray()))")
	g.P("        if (!handle.ok) return RpccgoResult.failure(handle.error ?: \"rpccgo: stream start failed\")")
	g.P("        return RpccgoResult.success(", streamType, "(handle.value ?: 0))")
	g.P("    }")
	g.P()
	g.P("    class ", streamType, " internal constructor(private val handle: Int) {")
	g.P("        fun Recv(): RpccgoResult<", respType, "> =")
	g.P("            decodeResult(", className, ".", nativeName, "Recv(handle)) { ", respType, ".parseFrom(it) }")
	g.P("        fun Finish(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Finish(handle))")
	g.P("        fun Cancel(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Cancel(handle))")
	g.P("    }")
	g.P()
}

func renderKotlinBidiStreamingMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, className string) {
	reqType := rpccgoKotlinMessageType(method.Request)
	respType := rpccgoKotlinMessageType(method.Response)
	streamType := service.GoName + method.GoName + "BidiStream"
	nativeName := lowerInitial(service.GoName) + method.GoName
	g.P("    fun ", method.GoName, "Start(): RpccgoResult<", streamType, "> {")
	g.P("        val handle = decodeHandleResult(", nativeName, "Start())")
	g.P("        if (!handle.ok) return RpccgoResult.failure(handle.error ?: \"rpccgo: stream start failed\")")
	g.P("        return RpccgoResult.success(", streamType, "(handle.value ?: 0))")
	g.P("    }")
	g.P()
	g.P("    class ", streamType, " internal constructor(private val handle: Int) {")
	g.P("        fun Send(req: ", reqType, "): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Send(handle, req.toByteArray()))")
	g.P("        fun Recv(): RpccgoResult<", respType, "> =")
	g.P("            decodeResult(", className, ".", nativeName, "Recv(handle)) { ", respType, ".parseFrom(it) }")
	g.P("        fun CloseSend(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "CloseSend(handle))")
	g.P("        fun Finish(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Finish(handle))")
	g.P("        fun Cancel(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Cancel(handle))")
	g.P("    }")
	g.P()
}
