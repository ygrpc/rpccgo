package com.ygrpc.examples.rpccgofluttersharedso

import android.Manifest
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.util.Log
import examples.flutter.sharedso.v1.AndroidEchoRequest
import examples.flutter.sharedso.v1.AndroidEchoResponse
import examples.flutter.sharedso.v1.FlutterEchoRequest
import examples.flutter.sharedso.v1.FlutterEchoResponse
import examples.flutter.sharedso.v1.ReadRuntimeStateRequest
import examples.flutter.sharedso.v1.RuntimeStateResponse
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private var events: EventChannel.EventSink? = null
    private var kotlinStream: SharedSoDemoJni.RpccgoCallbackStream? = null
    private var androidStream: SharedSoDemoJni.RpccgoCallbackStream? = null
    private var dartServerStream: SharedSoDemoJni.FlutterDeviceWatchFlutterEchoServerStream? = null
    private val dartServerStreamHandler = Handler(Looper.getMainLooper())
    private var dartServerStreamPoll: Runnable? = null
    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            if (intent.action != SharedSoRuntimeService.ACTION_STATE) return
            events?.success(intent.getStringExtra(SharedSoRuntimeService.EXTRA_LINE) ?: return)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestNotificationPermission()
        requestCameraPermission()
        startRuntimeService()
    }

    override fun onDestroy() {
        val hadKotlinStream = kotlinStream != null
        val hadAndroidStream = androidStream != null
        val hadDartServerStream = dartServerStream != null
        stopKotlinStream()
        stopAndroidStream()
        stopDartServerStream()
        if (hadKotlinStream) Log.i(TAG, "kotlin stream cancelled on activity destroy")
        if (hadAndroidStream) Log.i(TAG, "android stream cancelled on activity destroy")
        if (hadDartServerStream) Log.i(TAG, "dart server stream cancelled on activity destroy")
        events = null
        super.onDestroy()
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL_COMMAND).setMethodCallHandler { call, result ->
            when (call.method) {
                "kotlinRead" -> {
                    sendServiceAction(SharedSoRuntimeService.ACTION_READ)
                    result.success(null)
                }
                "kotlinIncrement" -> {
                    sendServiceAction(SharedSoRuntimeService.ACTION_INCREMENT)
                    result.success(null)
                }
                "kotlinStartStream" -> result.success(startKotlinStream())
                "kotlinStopStream" -> result.success(stopKotlinStream())
                "androidStartStream" -> result.success(startAndroidStream())
                "androidStopStream" -> result.success(stopAndroidStream())
                "flutterDeviceDescribe" -> result.success(describeFlutterDevice())
                "dartServerStartStream" -> {
                    dartServerStreamHandler.post { startDartServerStream() }
                    result.success(true)
                }
                "dartServerStopStream" -> {
                    dartServerStreamHandler.post { stopDartServerStream() }
                    result.success(true)
                }
                else -> result.notImplemented()
            }
        }
        EventChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL_EVENTS).setStreamHandler(
            object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, sink: EventChannel.EventSink) {
                    events = sink
                    registerReceiverCompat()
                }

                override fun onCancel(arguments: Any?) {
                    events = null
                    unregisterReceiver(receiver)
                }
            },
        )
    }

    private fun startRuntimeService() {
        val intent = Intent(this, SharedSoRuntimeService::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun sendServiceAction(action: String) {
        val intent = Intent(this, SharedSoRuntimeService::class.java).setAction(action)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun startKotlinStream(): Boolean {
        if (kotlinStream != null) return true
        val listener = object : SharedSoDemoJni.SharedSoDemoWatchRuntimeStateListener {
            override fun onRecv(responseBytes: ByteArray) {
                val line = try {
                    formatState("kotlin stream", RuntimeStateResponse.parseFrom(responseBytes))
                } catch (error: Exception) {
                    "kotlin stream decode error=${error.message ?: error::class.java.name}"
                }
                sendEvent(line)
            }

            override fun onDone(error: String?) {
                sendEvent("kotlin stream done error=${error ?: "none"}")
                kotlinStream = null
            }
        }
        val result = SharedSoDemoJni.WatchRuntimeStateStartCallback(
            this,
            ReadRuntimeStateRequest.newBuilder()
                .setCaller("kotlin-activity-count-stream")
                .build(),
            listener,
        )
        val stream = result.value
        if (!result.ok || stream == null) {
            sendEvent("kotlin stream start error=${result.error ?: "missing stream"}")
            return false
        }
        kotlinStream = stream
        return true
    }

    private fun stopKotlinStream(): Boolean {
        val stream = kotlinStream
        kotlinStream = null
        return stream?.cancel() ?: true
    }

    private fun startAndroidStream(): Boolean {
        if (androidStream != null) return true
        val listener = object : SharedSoDemoJni.AndroidDeviceWatchAndroidEchoListener {
            override fun onRecv(responseBytes: ByteArray) {
                val line = try {
                    formatAndroidEcho("android stream", AndroidEchoResponse.parseFrom(responseBytes))
                } catch (error: Exception) {
                    "android stream decode error=${error.message ?: error::class.java.name}"
                }
                sendEvent(line)
            }

            override fun onDone(error: String?) {
                sendEvent("android stream done error=${error ?: "none"}")
                androidStream = null
            }
        }
        val result = SharedSoDemoJni.WatchAndroidEchoStartCallback(
            this,
            AndroidEchoRequest.newBuilder()
                .setValue(100)
                .setCaller("android-activity-echo-stream")
                .build(),
            listener,
        )
        val stream = result.value
        if (!result.ok || stream == null) {
            sendEvent("android stream start error=${result.error ?: "missing stream"}")
            return false
        }
        androidStream = stream
        return true
    }

    private fun stopAndroidStream(): Boolean {
        val stream = androidStream
        androidStream = null
        return stream?.cancel() ?: true
    }

    private fun startDartServerStream(): Boolean {
        if (dartServerStream != null) return true
        val result = SharedSoDemoJni.WatchFlutterEchoStart(
            FlutterEchoRequest.newBuilder()
                .setCaller("kotlin-activity-to-dart-server-stream")
                .build(),
        )
        val stream = result.value
        if (!result.ok || stream == null) {
            sendEvent("dart server stream start error=${result.error ?: "missing stream"}")
            return false
        }
        dartServerStream = stream
        scheduleDartServerStreamRecv()
        return true
    }

    private fun stopDartServerStream(): Boolean {
        val stream = dartServerStream
        dartServerStreamPoll?.let { dartServerStreamHandler.removeCallbacks(it) }
        dartServerStreamPoll = null
        dartServerStream = null
        val result = stream?.Cancel() ?: return true
        if (!result.ok) sendEvent("dart server stream stop error=${result.error ?: "cancel failed"}")
        return result.ok
    }

    private fun scheduleDartServerStreamRecv() {
        val poll = object : Runnable {
            override fun run() {
                val stream = dartServerStream ?: return
                val result = stream.Recv()
                val value = result.value
                if (!result.ok || value == null) {
                    sendEvent("dart server stream done error=${result.error ?: "missing response"}")
                    stopDartServerStream()
                    return
                }
                sendEvent(formatFlutterEcho("dart server stream", value))
                dartServerStreamHandler.postDelayed(this, 700)
            }
        }
        dartServerStreamPoll = poll
        dartServerStreamHandler.postDelayed(poll, 100)
    }

    private fun describeFlutterDevice(): String {
        val result = SharedSoDemoJni.DescribeFlutter(
            FlutterEchoRequest.newBuilder()
                .setCaller("kotlin-activity-to-dart-server")
                .build(),
        )
        val value = result.value
        if (!result.ok || value == null) {
            return "error=${result.error ?: "missing response"}"
        }
        return "message=${value.message} caller=${value.caller} served_by=${value.servedBy}"
    }

    private fun sendEvent(line: String) {
        Log.i(TAG, line)
        runOnUiThread {
            events?.success(line)
        }
    }

    private fun formatState(label: String, value: RuntimeStateResponse): String =
        "$label value=${value.value} rev=${value.revision} pid=${value.pid} instance=${value.instanceAddress}"

    private fun formatAndroidEcho(label: String, value: AndroidEchoResponse): String =
        "$label value=${value.value} seq=${value.sequence} caller=${value.caller} served_by=${value.servedBy}"

    private fun formatFlutterEcho(label: String, value: FlutterEchoResponse): String =
        "$label message=${value.message} caller=${value.caller} served_by=${value.servedBy}"

    private fun registerReceiverCompat() {
        val filter = IntentFilter(SharedSoRuntimeService.ACTION_STATE)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(receiver, filter, RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("DEPRECATION")
            registerReceiver(receiver, filter)
        }
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
        }
    }

    private fun requestCameraPermission() {
        if (checkSelfPermission(Manifest.permission.CAMERA) != PackageManager.PERMISSION_GRANTED) {
            requestPermissions(arrayOf(Manifest.permission.CAMERA), 2)
        }
    }

    companion object {
        private const val CHANNEL_COMMAND = "rpccgo.shared.so/command"
        private const val CHANNEL_EVENTS = "rpccgo.shared.so/events"
        private const val TAG = "RpccgoSharedActivity"
    }
}
