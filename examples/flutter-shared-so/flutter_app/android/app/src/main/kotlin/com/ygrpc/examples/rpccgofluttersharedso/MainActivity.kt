package com.ygrpc.examples.rpccgofluttersharedso

import android.util.Log
import examples.flutter.sharedso.v1.ComposeGreetingRequest
import examples.flutter.sharedso.v1.ReadRuntimeStateRequest
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.embedding.android.FlutterActivity

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
                        result.success("${value?.message} | served_by=${value?.servedBy} | library=${value?.library}")
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
                        val state = "instance_address=${value?.instanceAddress} | pid=${value?.pid} | value=${value?.value} | revision=${value?.revision} | caller=${value?.caller}"
                        Log.i("RpccgoSharedRuntime", "Kotlin/JNI observed $state")
                        result.success(state)
                    } else {
                        result.success("jni runtime read error: ${response.error}")
                    }
                }

                else -> result.notImplemented()
            }
        }
    }
}
