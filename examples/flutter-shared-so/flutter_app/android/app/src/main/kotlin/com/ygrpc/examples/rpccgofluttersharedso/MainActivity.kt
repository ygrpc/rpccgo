package com.ygrpc.examples.rpccgofluttersharedso

import android.util.Log
import examples.flutter.sharedso.v1.ComposeGreetingRequest
import examples.flutter.sharedso.v1.IncrementRuntimeStateRequest
import examples.flutter.sharedso.v1.ReadRuntimeStateRequest
import examples.flutter.sharedso.v1.RuntimeStateResponse
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.embedding.android.FlutterActivity
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

class MainActivity : FlutterActivity() {
    companion object {
        init {
            System.loadLibrary("rpccgo_flutter_shared")
            System.loadLibrary("rpccgo_flutter_shared_jni")
        }
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            "rpccgo.shared.so/jni",
        ).setMethodCallHandler { call, result ->
            when (call.method) {
                "composeGreeting" -> {
                    val name = call.argument<String>("name").orEmpty()
                    val response = SharedSoDemoJni.ComposeGreeting(
                        ComposeGreetingRequest.newBuilder()
                            .setName(name)
                            .setCaller("kotlin-jni")
                            .build(),
                    )
                    if (response.ok) {
                        val value = response.value
                        result.success(
                            "Kotlin/JNI unary call\n" +
                                "Message: ${value?.message}\n" +
                                "Go handler: ${value?.servedBy}\n" +
                                "Shared library: ${value?.library}",
                        )
                    } else {
                        result.success("jni error: ${response.error}")
                    }
                }

                "readRuntimeState" -> {
                    val response = SharedSoDemoJni.ReadRuntimeState(
                        ReadRuntimeStateRequest.newBuilder()
                            .setCaller("kotlin-jni")
                            .build(),
                    )
                    if (response.ok) {
                        val value = response.value
                        val state = "Value: ${value?.value}\n" +
                            "Revision: ${value?.revision}\n" +
                            "Go instance: ${value?.instanceAddress}\n" +
                            "Process ID: ${value?.pid}\n" +
                            "Caller: ${value?.caller}"
                        Log.i("RpccgoSharedRuntime", "Kotlin/JNI observed $state")
                        result.success(state)
                    } else {
                        result.success("jni runtime read error: ${response.error}")
                    }
                }

                "runStreams" -> {
                    result.success(runJniStreams())
                }

                else -> result.notImplemented()
            }
        }
    }

    override fun onDestroy() {
        SharedSoDemoJni.StreamRuntimeStateCancelCallback()
        super.onDestroy()
    }

    private fun runJniStreams(): String {
        val collect = SharedSoDemoJni.CollectRuntimeStateStart()
        val collectStream = collect.value ?: return "jni client stream start error: ${collect.error}"
        collectStream.Send(
            IncrementRuntimeStateRequest.newBuilder()
                .setDelta(2)
                .setCaller("kotlin-jni-client-stream-a")
                .build(),
        ).let { if (!it.ok) return "jni client stream send error: ${it.error}" }
        collectStream.Send(
            IncrementRuntimeStateRequest.newBuilder()
                .setDelta(3)
                .setCaller("kotlin-jni-client-stream-b")
                .build(),
        ).let { if (!it.ok) return "jni client stream send error: ${it.error}" }
        val collected = collectStream.Finish()
        val collectedValue = collected.value ?: return "jni client stream finish error: ${collected.error}"

        val stream = SharedSoDemoJni.StreamRuntimeStateStart(
            ReadRuntimeStateRequest.newBuilder()
                .setCaller("kotlin-jni-server-stream")
                .build(),
        )
        val serverStream = stream.value ?: return "jni server stream start error: ${stream.error}"
        val serverValues = mutableListOf<Long>()
        repeat(3) {
            val next = serverStream.Recv()
            val value = next.value ?: return "jni server stream recv error: ${next.error}"
            serverValues.add(value.value)
        }

        val callbackValues = CopyOnWriteArrayList<Long>()
        val callbackDone = CountDownLatch(1)
        var callbackError: String? = null
        val callbackStarted = SharedSoDemoJni.StreamRuntimeStateStartCallback(
            ReadRuntimeStateRequest.newBuilder()
                .setCaller("kotlin-jni-callback-server-stream")
                .build(),
            object : SharedSoDemoJni.SharedSoDemoStreamRuntimeStateListener {
                override fun onMessage(responseBytes: ByteArray) {
                    try {
                        callbackValues.add(RuntimeStateResponse.parseFrom(responseBytes).value)
                    } catch (err: Exception) {
                        callbackError = err.message ?: "callback parse failed"
                        SharedSoDemoJni.StreamRuntimeStateCancelCallback()
                    }
                }

                override fun onDone(error: String?) {
                    callbackError = callbackError ?: error
                    callbackDone.countDown()
                }
            },
        )
        if (!callbackStarted) return "jni callback stream start error"
        if (!callbackDone.await(3, TimeUnit.SECONDS)) {
            SharedSoDemoJni.StreamRuntimeStateCancelCallback()
            return "jni callback stream did not complete"
        }
        if (callbackError != null) return "jni callback stream error: $callbackError"

        val chat = SharedSoDemoJni.ChatRuntimeStateStart()
        val bidi = chat.value ?: return "jni bidi stream start error: ${chat.error}"
        val bidiValues = mutableListOf<Long>()
        listOf(
            IncrementRuntimeStateRequest.newBuilder()
                .setDelta(4)
                .setCaller("kotlin-jni-bidi-a")
                .build(),
            IncrementRuntimeStateRequest.newBuilder()
                .setDelta(5)
                .setCaller("kotlin-jni-bidi-b")
                .build(),
        ).forEach { req ->
            bidi.Send(req).let { if (!it.ok) return "jni bidi stream send error: ${it.error}" }
            val next = bidi.Recv()
            val value = next.value ?: return "jni bidi stream recv error: ${next.error}"
            bidiValues.add(value.value)
        }
        bidi.CloseSend().let { if (!it.ok) return "jni bidi stream close-send error: ${it.error}" }
        bidi.Finish().let { if (!it.ok) return "jni bidi stream finish error: ${it.error}" }

        return "Kotlin/JNI: +2+3 -> ${collectedValue.value} (rev ${collectedValue.revision}); " +
            "read ${serverValues.joinToString(", ")}; callback ${callbackValues.joinToString(", ")}; +4+5 -> ${bidiValues.joinToString(" -> ")}; " +
            "final ${bidiValues.lastOrNull() ?: collectedValue.value}"
    }
}
