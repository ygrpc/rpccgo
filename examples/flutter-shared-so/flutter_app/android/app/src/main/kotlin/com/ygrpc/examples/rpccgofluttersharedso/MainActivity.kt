package com.ygrpc.examples.rpccgofluttersharedso

import android.Manifest
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private var events: EventChannel.EventSink? = null
    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            if (intent.action != SharedSoRuntimeService.ACTION_STATE) return
            events?.success(intent.getStringExtra(SharedSoRuntimeService.EXTRA_LINE) ?: return)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestNotificationPermission()
        startRuntimeService()
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

    companion object {
        private const val CHANNEL_COMMAND = "rpccgo.shared.so/command"
        private const val CHANNEL_EVENTS = "rpccgo.shared.so/events"
    }
}
