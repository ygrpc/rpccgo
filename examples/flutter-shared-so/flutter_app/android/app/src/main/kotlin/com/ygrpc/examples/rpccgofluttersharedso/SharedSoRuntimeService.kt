package com.ygrpc.examples.rpccgofluttersharedso

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import examples.flutter.sharedso.v1.IncrementRuntimeStateRequest
import examples.flutter.sharedso.v1.ReadRuntimeStateRequest
import examples.flutter.sharedso.v1.RuntimeStateResponse

class SharedSoRuntimeService : Service() {
    companion object {
        const val ACTION_READ = "com.ygrpc.examples.rpccgofluttersharedso.READ"
        const val ACTION_INCREMENT = "com.ygrpc.examples.rpccgofluttersharedso.INCREMENT"
        const val ACTION_STATE = "com.ygrpc.examples.rpccgofluttersharedso.STATE"
        const val EXTRA_LINE = "line"
        private const val CHANNEL_ID = "rpccgo-shared-runtime"
        private const val NOTIFICATION_ID = 1
        private const val TAG = "RpccgoSharedRuntime"

        init {
            System.loadLibrary("rpccgo_flutter_shared")
            System.loadLibrary("rpccgo_flutter_shared_jni")
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        createChannel()
        startForeground(NOTIFICATION_ID, notification())
        when (intent?.action) {
            ACTION_INCREMENT -> increment()
            ACTION_READ -> read()
            else -> read()
        }
        return START_STICKY
    }

    private fun read() {
        val result = SharedSoDemoJni.ReadRuntimeState(
            ReadRuntimeStateRequest.newBuilder()
                .setCaller("kotlin-service-read")
                .build(),
        )
        publish("kotlin read", result.value, result.error)
    }

    private fun increment() {
        val result = SharedSoDemoJni.IncrementRuntimeState(
            IncrementRuntimeStateRequest.newBuilder()
                .setDelta(1)
                .setCaller("kotlin-service-increment")
                .build(),
        )
        publish("kotlin increment", result.value, result.error)
    }

    private fun publish(label: String, value: RuntimeStateResponse?, error: String?) {
        val line = if (error != null || value == null) {
            "$label error=${error ?: "missing response"}"
        } else {
            "$label value=${value.value} rev=${value.revision} pid=${value.pid} instance=${value.instanceAddress}"
        }
        Log.i(TAG, line)
        sendBroadcast(Intent(ACTION_STATE).setPackage(packageName).putExtra(EXTRA_LINE, line))
        getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, notification())
    }

    private fun notification(): Notification {
        val activity = PendingIntent.getActivity(
            this,
            0,
            packageManager.getLaunchIntentForPackage(packageName),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }
        return builder
            .setSmallIcon(android.R.drawable.stat_sys_upload)
            .setContentTitle("rpccgo shared .so")
            .setContentText("Runtime service is running")
            .setContentIntent(activity)
            .setOngoing(true)
            .build()
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java)
        if (manager.getNotificationChannel(CHANNEL_ID) == null) {
            manager.createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "rpccgo shared runtime", NotificationManager.IMPORTANCE_LOW),
            )
        }
    }
}
