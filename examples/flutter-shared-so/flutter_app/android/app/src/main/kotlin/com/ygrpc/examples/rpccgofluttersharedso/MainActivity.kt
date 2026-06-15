package com.ygrpc.examples.rpccgofluttersharedso

import android.util.Log
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.embedding.android.FlutterActivity

class MainActivity : FlutterActivity() {
    companion object {
        init {
            System.loadLibrary("rpccgo_flutter_shared")
        }
    }

    private external fun nativeComposeGreeting(name: String): String
    private external fun nativeReadRuntimeState(): String

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            "rpccgo.shared.so/jni",
        ).setMethodCallHandler { call, result ->
            when (call.method) {
                "composeGreeting" -> {
                    val name = call.argument<String>("name").orEmpty()
                    result.success(nativeComposeGreeting(name))
                }

                "readRuntimeState" -> {
                    val state = nativeReadRuntimeState()
                    Log.i("RpccgoSharedRuntime", "Kotlin/JNI observed $state")
                    result.success(state)
                }

                else -> result.notImplemented()
            }
        }
    }
}
