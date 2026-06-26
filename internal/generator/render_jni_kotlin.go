package generator

import "google.golang.org/protobuf/compiler/protogen"

func renderJNIKotlinFile(plugin *protogen.Plugin, plan FilePlan, services []ServicePlan, config JNIGeneratorConfig) {
	g := plugin.NewGeneratedFile(jniKotlinFilename(config), "")
	renderGeneratedHeaderForTool(g, "protoc-gen-rpc-cgo-jni")
	g.P("// Source: ", plan.ProtoPath)
	g.P()
	pkg, className := jniClassPackageAndSimpleName(config.JNIClass)
	g.P("package ", pkg)
	g.P()
	if jniServicesHaveRecvStreamingMethod(services) {
		g.P("import androidx.annotation.Keep")
	}
	g.P("import com.google.protobuf.MessageLite")
	g.P("import java.nio.ByteBuffer")
	g.P("import java.nio.ByteOrder")
	if jniServicesHaveStreamingMethod(services) {
		g.P("import java.util.concurrent.ConcurrentHashMap")
		g.P("import java.util.concurrent.atomic.AtomicInteger")
	}
	if jniServicesHaveRecvStreamingMethod(services) {
		g.P("import java.util.concurrent.atomic.AtomicBoolean")
	}
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
	if jniServicesHaveStreamingMethod(services) {
		g.P("    private val rpccgoServerStreamIDs = AtomicInteger(1)")
		g.P()
	}
	if jniServicesHaveRecvStreamingMethod(services) {
		renderKotlinCallbackStreamSupport(g)
	}
	for _, service := range services {
		for _, method := range service.Methods {
			renderKotlinCallbackListener(g, service, method)
			renderKotlinServerTypes(g, service, method)
			renderKotlinNativeDeclarations(g, service, method)
			renderKotlinServerNativeDeclarations(g, service, method)
		}
	}
	g.P()
	for _, service := range services {
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
			renderKotlinServerMethod(g, service, method)
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
	g.P()
	g.P("    private fun encodeResultPayload(ok: Boolean, payload: ByteArray): ByteArray {")
	g.P("        val out = ByteArray(payload.size + 5)")
	g.P("        out[0] = if (ok) 1 else 0")
	g.P("        ByteBuffer.wrap(out, 1, 4).order(ByteOrder.BIG_ENDIAN).putInt(payload.size)")
	g.P("        payload.copyInto(out, 5)")
	g.P("        return out")
	g.P("    }")
	g.P()
	g.P("    private fun encodeErrorResult(error: String): ByteArray =")
	g.P("        encodeResultPayload(false, error.toByteArray(Charsets.UTF_8))")
	g.P()
	g.P("    private fun encodeUnitResult(result: RpccgoResult<Unit>): ByteArray {")
	g.P(`        if (!result.ok) return encodeErrorResult(result.error ?: "rpccgo: Kotlin server returned failure")`)
	g.P("        return encodeResultPayload(true, ByteArray(0))")
	g.P("    }")
	g.P()
	g.P("    private fun encodeMessageResult(result: RpccgoResult<out MessageLite>): ByteArray {")
	g.P(`        if (!result.ok) return encodeErrorResult(result.error ?: "rpccgo: Kotlin server returned failure")`)
	g.P(`        val value = result.value ?: return encodeErrorResult("rpccgo: Kotlin server returned null response")`)
	g.P("        return encodeResultPayload(true, value.toByteArray())")
	g.P("    }")
	if jniServicesHaveStreamingMethod(services) {
		g.P()
		g.P("    private fun encodeHandleResult(handle: Int): ByteArray {")
		g.P("        val payload = ByteArray(4)")
		g.P("        ByteBuffer.wrap(payload).order(ByteOrder.BIG_ENDIAN).putInt(handle)")
		g.P("        return encodeResultPayload(true, payload)")
		g.P("    }")
		g.P()
		g.P("    private fun nextServerStreamHandle(): Int {")
		g.P("        val next = rpccgoServerStreamIDs.getAndIncrement()")
		g.P("        return if (next == 0) rpccgoServerStreamIDs.getAndIncrement() else next")
		g.P("    }")
	}
	g.P("}")
	g.P()
}

func renderKotlinCallbackStreamSupport(g *protogen.GeneratedFile) {
	g.P("    /** Handle for a generated JNI callback stream. */")
	g.P("    class RpccgoCallbackStream internal constructor(")
	g.P("        private val cancelCallback: () -> Boolean,")
	g.P("        private val unregisterLifecycle: () -> Unit = {},")
	g.P("    ) : AutoCloseable {")
	g.P("        private val active = AtomicBoolean(true)")
	g.P()
	g.P("        /** Cancels the native stream and unregisters any generated lifecycle callback. */")
	g.P("        fun cancel(): Boolean {")
	g.P("            if (!active.compareAndSet(true, false)) return true")
	g.P("            unregisterLifecycle()")
	g.P("            return cancelCallback()")
	g.P("        }")
	g.P()
	g.P("        internal fun complete() {")
	g.P("            if (active.compareAndSet(true, false)) unregisterLifecycle()")
	g.P("        }")
	g.P()
	g.P("        override fun close() {")
	g.P("            cancel()")
	g.P("        }")
	g.P("    }")
	g.P()
	g.P("    private fun activityOwnedCallbackStream(owner: android.app.Activity, cancel: () -> Boolean): RpccgoCallbackStream {")
	g.P("        var stream: RpccgoCallbackStream? = null")
	g.P("        val callbacks = object : android.app.Application.ActivityLifecycleCallbacks {")
	g.P("            override fun onActivityCreated(activity: android.app.Activity, savedInstanceState: android.os.Bundle?) {}")
	g.P("            override fun onActivityStarted(activity: android.app.Activity) {}")
	g.P("            override fun onActivityResumed(activity: android.app.Activity) {}")
	g.P("            override fun onActivityPaused(activity: android.app.Activity) {}")
	g.P("            override fun onActivityStopped(activity: android.app.Activity) {}")
	g.P("            override fun onActivitySaveInstanceState(activity: android.app.Activity, outState: android.os.Bundle) {}")
	g.P("            override fun onActivityDestroyed(activity: android.app.Activity) {")
	g.P("                if (activity === owner) stream?.cancel()")
	g.P("            }")
	g.P("        }")
	g.P("        owner.application.registerActivityLifecycleCallbacks(callbacks)")
	g.P("        stream = RpccgoCallbackStream(cancel) { owner.application.unregisterActivityLifecycleCallbacks(callbacks) }")
	g.P("        return stream")
	g.P("    }")
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
		g.P("    private external fun ", prefix, "Cancel(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "StartCallback(request: ByteArray, listener: ", jniKotlinListenerType(service, method), "): Boolean")
		g.P("    private external fun ", prefix, "CancelCallback(): Boolean")
	case StreamingKindBidiStreaming:
		g.P("    private external fun ", prefix, "Start(): ByteArray?")
		g.P("    private external fun ", prefix, "Send(handle: Int, request: ByteArray): ByteArray?")
		g.P("    private external fun ", prefix, "Recv(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "CloseSend(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Finish(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "Cancel(handle: Int): ByteArray?")
		g.P("    private external fun ", prefix, "StartCallback(listener: ", jniKotlinListenerType(service, method), "): Boolean")
		g.P("    private external fun ", prefix, "CancelCallback(): Boolean")
	}
}

func renderKotlinServerNativeDeclarations(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	prefix := lowerInitial(service.GoName) + method.GoName
	g.P("    private external fun ", prefix, "Register(): ByteArray?")
}

func renderKotlinCallbackListener(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	if method.Streaming != StreamingKindServerStreaming && method.Streaming != StreamingKindBidiStreaming {
		return
	}
	g.P("    @Keep")
	g.P("    interface ", jniKotlinListenerType(service, method), " {")
	g.P("        @Keep")
	g.P("        fun onRecv(responseBytes: ByteArray)")
	g.P("        @Keep")
	g.P("        fun onDone(error: String?)")
	g.P("    }")
	g.P()
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
	g.P("    fun ", method.GoName, "StartCallback(req: ", reqType, ", listener: ", jniKotlinListenerType(service, method), "): Boolean =")
	g.P("        ", nativeName, "StartCallback(req.toByteArray(), listener)")
	renderKotlinOwnerCallbackStartMethod(g, service, method, "req: "+reqType+", ", method.GoName+"StartCallback(req, ownerListener)")
	g.P("    fun ", method.GoName, "CancelCallback(): Boolean = ", nativeName, "CancelCallback()")
	g.P()
	g.P("    class ", streamType, " internal constructor(private val handle: Int) {")
	g.P("        private val receiving = AtomicBoolean(false)")
	g.P("        private fun recvUnchecked(): RpccgoResult<", respType, "> =")
	g.P("            decodeResult(", className, ".", nativeName, "Recv(handle)) { ", respType, ".parseFrom(it) }")
	g.P("        /** Receives one response. Do not call while RecvEach is running on this stream. */")
	g.P("        fun Recv(): RpccgoResult<", respType, "> {")
	g.P("            if (!receiving.compareAndSet(false, true)) return RpccgoResult.failure(\"rpccgo: stream already has an active receiver\")")
	g.P("            return try {")
	g.P("                recvUnchecked()")
	g.P("            } finally {")
	g.P("                receiving.set(false)")
	g.P("            }")
	g.P("        }")
	g.P("        fun Cancel(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Cancel(handle))")
	renderKotlinReceiveEachMethod(g, respType)
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
	g.P("    fun ", method.GoName, "StartCallback(listener: ", jniKotlinListenerType(service, method), "): Boolean =")
	g.P("        ", nativeName, "StartCallback(listener)")
	renderKotlinOwnerCallbackStartMethod(g, service, method, "", method.GoName+"StartCallback(ownerListener)")
	g.P("    fun ", method.GoName, "CancelCallback(): Boolean = ", nativeName, "CancelCallback()")
	g.P()
	g.P("    class ", streamType, " internal constructor(private val handle: Int) {")
	g.P("        fun Send(req: ", reqType, "): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Send(handle, req.toByteArray()))")
	g.P("        private val receiving = AtomicBoolean(false)")
	g.P("        private fun recvUnchecked(): RpccgoResult<", respType, "> =")
	g.P("            decodeResult(", className, ".", nativeName, "Recv(handle)) { ", respType, ".parseFrom(it) }")
	g.P("        /** Receives one response. Do not call while RecvEach is running on this stream. */")
	g.P("        fun Recv(): RpccgoResult<", respType, "> {")
	g.P("            if (!receiving.compareAndSet(false, true)) return RpccgoResult.failure(\"rpccgo: stream already has an active receiver\")")
	g.P("            return try {")
	g.P("                recvUnchecked()")
	g.P("            } finally {")
	g.P("                receiving.set(false)")
	g.P("            }")
	g.P("        }")
	g.P("        fun CloseSend(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "CloseSend(handle))")
	g.P("        fun Finish(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Finish(handle))")
	g.P("        fun Cancel(): RpccgoResult<Unit> =")
	g.P("            decodeUnitResult(", className, ".", nativeName, "Cancel(handle))")
	renderKotlinReceiveEachMethod(g, respType)
	g.P("    }")
	g.P()
}

func renderKotlinServerTypes(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	switch method.Streaming {
	case StreamingKindUnary:
		return
	case StreamingKindClientStreaming:
		g.P("    interface ", jniKotlinServerHandlerType(service, method), " {")
		g.P("        fun Send(req: ", rpccgoKotlinMessageType(method.Request), "): RpccgoResult<Unit>")
		g.P("        fun Finish(): RpccgoResult<", rpccgoKotlinMessageType(method.Response), ">")
		g.P("        fun Cancel(): RpccgoResult<Unit>")
		g.P("    }")
		g.P()
	case StreamingKindServerStreaming:
		g.P("    interface ", jniKotlinServerHandlerType(service, method), " {")
		g.P("        fun Recv(): RpccgoResult<", rpccgoKotlinMessageType(method.Response), ">")
		g.P("        fun Finish(): RpccgoResult<Unit>")
		g.P("        fun Cancel(): RpccgoResult<Unit>")
		g.P("    }")
		g.P()
	case StreamingKindBidiStreaming:
		g.P("    interface ", jniKotlinServerHandlerType(service, method), " {")
		g.P("        fun Send(req: ", rpccgoKotlinMessageType(method.Request), "): RpccgoResult<Unit>")
		g.P("        fun Recv(): RpccgoResult<", rpccgoKotlinMessageType(method.Response), ">")
		g.P("        fun CloseSend(): RpccgoResult<Unit>")
		g.P("        fun Finish(): RpccgoResult<Unit>")
		g.P("        fun Cancel(): RpccgoResult<Unit>")
		g.P("    }")
		g.P()
	}
}

func renderKotlinServerMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	switch method.Streaming {
	case StreamingKindUnary:
		renderKotlinUnaryServerMethod(g, service, method)
	case StreamingKindClientStreaming:
		renderKotlinClientStreamingServerMethod(g, service, method)
	case StreamingKindServerStreaming:
		renderKotlinServerStreamingServerMethod(g, service, method)
	case StreamingKindBidiStreaming:
		renderKotlinBidiStreamingServerMethod(g, service, method)
	}
}

func renderKotlinUnaryServerMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	reqType := rpccgoKotlinMessageType(method.Request)
	respType := rpccgoKotlinMessageType(method.Response)
	prefix := lowerInitial(service.GoName) + method.GoName
	g.P("    private var ", prefix, "ServerHandler: ((", reqType, ") -> RpccgoResult<", respType, ">)? = null")
	g.P()
	g.P("    fun Register", method.GoName, "(handler: (", reqType, ") -> RpccgoResult<", respType, ">): RpccgoResult<Unit> {")
	g.P("        ", prefix, "ServerHandler = handler")
	g.P("        val result = decodeUnitResult(", prefix, "Register())")
	g.P("        if (!result.ok) ", prefix, "ServerHandler = null")
	g.P("        return result")
	g.P("    }")
	g.P()
	g.P("    @Keep")
	g.P("    private fun ", prefix, "Handle(requestBytes: ByteArray): ByteArray = try {")
	g.P(`        val handler = `, prefix, `ServerHandler ?: return encodeErrorResult("rpccgo: Kotlin server handler is not registered")`)
	g.P("        encodeMessageResult(handler(", reqType, ".parseFrom(requestBytes)))")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server handler failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinClientStreamingServerMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	respType := rpccgoKotlinMessageType(method.Response)
	prefix := lowerInitial(service.GoName) + method.GoName
	handlerType := jniKotlinServerHandlerType(service, method)
	g.P("    private var ", prefix, "ServerStart: (() -> RpccgoResult<", handlerType, ">)? = null")
	g.P("    private val ", prefix, "ServerStreams = ConcurrentHashMap<Int, ", handlerType, ">()")
	g.P()
	g.P("    fun Register", method.GoName, "(start: () -> RpccgoResult<", handlerType, ">): RpccgoResult<Unit> {")
	g.P("        ", prefix, "ServerStart = start")
	g.P("        val result = decodeUnitResult(", prefix, "Register())")
	g.P("        if (!result.ok) ", prefix, "ServerStart = null")
	g.P("        return result")
	g.P("    }")
	g.P()
	renderKotlinServerStartNoRequest(g, prefix)
	renderKotlinServerSend(g, method, prefix)
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerFinish(handle: Int): ByteArray = try {")
	g.P(`        val stream = `, prefix, `ServerStreams.remove(handle) ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeMessageResult(stream.Finish())")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream finish failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
	renderKotlinServerCancel(g, prefix, false, respType)
}

func renderKotlinServerStreamingServerMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	prefix := lowerInitial(service.GoName) + method.GoName
	handlerType := jniKotlinServerHandlerType(service, method)
	reqType := rpccgoKotlinMessageType(method.Request)
	g.P("    private var ", prefix, "ServerStart: ((", reqType, ") -> RpccgoResult<", handlerType, ">)? = null")
	g.P("    private val ", prefix, "ServerStreams = ConcurrentHashMap<Int, ", handlerType, ">()")
	g.P()
	g.P("    fun Register", method.GoName, "(start: (", reqType, ") -> RpccgoResult<", handlerType, ">): RpccgoResult<Unit> {")
	g.P("        ", prefix, "ServerStart = start")
	g.P("        val result = decodeUnitResult(", prefix, "Register())")
	g.P("        if (!result.ok) ", prefix, "ServerStart = null")
	g.P("        return result")
	g.P("    }")
	g.P()
	renderKotlinServerStartWithRequest(g, method, prefix)
	renderKotlinServerRecv(g, prefix)
	renderKotlinServerFinish(g, prefix)
	renderKotlinServerCancel(g, prefix, true, "")
}

func renderKotlinBidiStreamingServerMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan) {
	prefix := lowerInitial(service.GoName) + method.GoName
	handlerType := jniKotlinServerHandlerType(service, method)
	g.P("    private var ", prefix, "ServerStart: (() -> RpccgoResult<", handlerType, ">)? = null")
	g.P("    private val ", prefix, "ServerStreams = ConcurrentHashMap<Int, ", handlerType, ">()")
	g.P()
	g.P("    fun Register", method.GoName, "(start: () -> RpccgoResult<", handlerType, ">): RpccgoResult<Unit> {")
	g.P("        ", prefix, "ServerStart = start")
	g.P("        val result = decodeUnitResult(", prefix, "Register())")
	g.P("        if (!result.ok) ", prefix, "ServerStart = null")
	g.P("        return result")
	g.P("    }")
	g.P()
	renderKotlinServerStartNoRequest(g, prefix)
	renderKotlinServerSend(g, method, prefix)
	renderKotlinServerRecv(g, prefix)
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerCloseSend(handle: Int): ByteArray = try {")
	g.P(`        val stream = `, prefix, `ServerStreams[handle] ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeUnitResult(stream.CloseSend())")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream close-send failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
	renderKotlinServerFinish(g, prefix)
	renderKotlinServerCancel(g, prefix, true, "")
}

func renderKotlinServerStartNoRequest(g *protogen.GeneratedFile, prefix string) {
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerStart(): ByteArray = try {")
	g.P(`        val start = `, prefix, `ServerStart ?: return encodeErrorResult("rpccgo: Kotlin server handler is not registered")`)
	g.P("        val result = start()")
	g.P(`        if (!result.ok) return encodeErrorResult(result.error ?: "rpccgo: Kotlin server stream start failed")`)
	g.P(`        val stream = result.value ?: return encodeErrorResult("rpccgo: Kotlin server stream start returned null")`)
	g.P("        val handle = nextServerStreamHandle()")
	g.P("        ", prefix, "ServerStreams[handle] = stream")
	g.P("        encodeHandleResult(handle)")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream start failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinServerStartWithRequest(g *protogen.GeneratedFile, method MethodPlan, prefix string) {
	reqType := rpccgoKotlinMessageType(method.Request)
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerStart(requestBytes: ByteArray): ByteArray = try {")
	g.P(`        val start = `, prefix, `ServerStart ?: return encodeErrorResult("rpccgo: Kotlin server handler is not registered")`)
	g.P("        val result = start(", reqType, ".parseFrom(requestBytes))")
	g.P(`        if (!result.ok) return encodeErrorResult(result.error ?: "rpccgo: Kotlin server stream start failed")`)
	g.P(`        val stream = result.value ?: return encodeErrorResult("rpccgo: Kotlin server stream start returned null")`)
	g.P("        val handle = nextServerStreamHandle()")
	g.P("        ", prefix, "ServerStreams[handle] = stream")
	g.P("        encodeHandleResult(handle)")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream start failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinServerSend(g *protogen.GeneratedFile, method MethodPlan, prefix string) {
	reqType := rpccgoKotlinMessageType(method.Request)
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerSend(handle: Int, requestBytes: ByteArray): ByteArray = try {")
	g.P(`        val stream = `, prefix, `ServerStreams[handle] ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeUnitResult(stream.Send(", reqType, ".parseFrom(requestBytes)))")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream send failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinServerRecv(g *protogen.GeneratedFile, prefix string) {
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerRecv(handle: Int): ByteArray = try {")
	g.P(`        val stream = `, prefix, `ServerStreams[handle] ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeMessageResult(stream.Recv())")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream recv failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinServerFinish(g *protogen.GeneratedFile, prefix string) {
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerFinish(handle: Int): ByteArray = try {")
	g.P(`        val stream = `, prefix, `ServerStreams.remove(handle) ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeUnitResult(stream.Finish())")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream finish failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinServerCancel(g *protogen.GeneratedFile, prefix string, remove bool, _ string) {
	lookup := prefix + "ServerStreams[handle]"
	if remove {
		lookup = prefix + "ServerStreams.remove(handle)"
	}
	g.P("    @Keep")
	g.P("    private fun ", prefix, "ServerCancel(handle: Int): ByteArray = try {")
	g.P(`        val stream = `, lookup, ` ?: return encodeErrorResult("rpccgo: Kotlin server stream handle is invalid")`)
	g.P("        encodeUnitResult(stream.Cancel())")
	g.P("    } catch (e: Exception) {")
	g.P(`        encodeErrorResult("rpccgo: Kotlin server stream cancel failed: ${e.message ?: e::class.java.name}")`)
	g.P("    }")
	g.P()
}

func renderKotlinOwnerCallbackStartMethod(g *protogen.GeneratedFile, service ServicePlan, method MethodPlan, params, startCall string) {
	listenerType := jniKotlinListenerType(service, method)
	g.P("    fun ", method.GoName, "StartCallback(owner: android.app.Activity, ", params, "listener: ", listenerType, "): RpccgoResult<RpccgoCallbackStream> {")
	g.P("        if (owner.isDestroyed) return RpccgoResult.failure(\"rpccgo: callback stream owner is destroyed\")")
	g.P("        val callbackOpen = AtomicBoolean(true)")
	g.P("        var stream: RpccgoCallbackStream? = null")
	g.P("        val ownerListener = object : ", listenerType, " {")
	g.P("            override fun onRecv(responseBytes: ByteArray) {")
	g.P("                if (callbackOpen.get()) listener.onRecv(responseBytes)")
	g.P("            }")
	g.P()
	g.P("            override fun onDone(error: String?) {")
	g.P("                if (!callbackOpen.compareAndSet(true, false)) return")
	g.P("                stream?.complete()")
	g.P("                listener.onDone(error)")
	g.P("            }")
	g.P("        }")
	g.P("        stream = activityOwnedCallbackStream(owner) {")
	g.P("            callbackOpen.set(false)")
	g.P("            ", method.GoName, "CancelCallback()")
	g.P("        }")
	g.P("        val activeStream = stream ?: return RpccgoResult.failure(\"rpccgo: callback stream owner registration failed\")")
	g.P("        if (!", startCall, ") {")
	g.P("            activeStream.cancel()")
	g.P("            return RpccgoResult.failure(\"rpccgo: callback stream start failed\")")
	g.P("        }")
	g.P("        return RpccgoResult.success(activeStream)")
	g.P("    }")
}

func serviceHasRecvStreamingMethod(service ServicePlan) bool {
	for _, method := range service.Methods {
		if method.Streaming == StreamingKindServerStreaming || method.Streaming == StreamingKindBidiStreaming {
			return true
		}
	}
	return false
}

func jniServicesHaveRecvStreamingMethod(services []ServicePlan) bool {
	for _, service := range services {
		if serviceHasRecvStreamingMethod(service) {
			return true
		}
	}
	return false
}

func jniServicesHaveStreamingMethod(services []ServicePlan) bool {
	for _, service := range services {
		if serviceHasStreamingMethod(service) {
			return true
		}
	}
	return false
}

func renderKotlinReceiveEachMethod(g *protogen.GeneratedFile, respType string) {
	g.P("        /** Starts a background Recv loop. Do not mix with manual Recv calls on this stream. */")
	g.P("        fun RecvEach(onRecv: (", respType, ") -> Unit, onError: (String) -> Unit = {}): RpccgoResult<Thread> {")
	g.P("            if (!receiving.compareAndSet(false, true)) return RpccgoResult.failure(\"rpccgo: stream already has an active receiver\")")
	g.P("            val worker = Thread {")
	g.P("                try {")
	g.P("                    while (true) {")
	g.P("                        val next = recvUnchecked()")
	g.P("                        if (!next.ok) {")
	g.P("                            onError(next.error ?: \"rpccgo: stream recv failed\")")
	g.P("                            return@Thread")
	g.P("                        }")
	g.P("                        val value = next.value ?: return@Thread")
	g.P("                        onRecv(value)")
	g.P("                    }")
	g.P("                } finally {")
	g.P("                    receiving.set(false)")
	g.P("                    Cancel()")
	g.P("                }")
	g.P("            }")
	g.P("            worker.start()")
	g.P("            return RpccgoResult.success(worker)")
	g.P("        }")
}

func jniKotlinListenerType(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "Listener"
}

func jniKotlinServerHandlerType(service ServicePlan, method MethodPlan) string {
	return service.GoName + method.GoName + "ServerHandler"
}
