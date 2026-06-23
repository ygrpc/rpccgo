package com.ygrpc.examples.rpccgoandroidforegroundservice

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.Binder
import android.os.IBinder
import android.util.Log
import examples.android.foregroundservice.v1.Tick
import examples.android.foregroundservice.v1.WatchTicksRequest

class StreamForegroundService : Service() {
    companion object {
        const val ACTION_STOP = "com.ygrpc.examples.rpccgoandroidforegroundservice.STOP"
        const val ACTION_LOG = "com.ygrpc.examples.rpccgoandroidforegroundservice.LOG"
        const val EXTRA_LINE = "line"
        private const val CHANNEL_ID = "rpccgo-stream"
        private const val NOTIFICATION_ID = 1
        private const val TAG = "RpccgoForegroundService"

        init {
            System.loadLibrary("rpccgo_android_foreground_service")
            System.loadLibrary("rpccgo_android_foreground_service_jni")
        }
    }

    private var started = false
    private var lastLine = "waiting for ticks"
    private var activitySink: ActivityUiBridge.ActivitySink? = null
    private val binder = LocalBinder()

    inner class LocalBinder : Binder() {
        val service: StreamForegroundService get() = this@StreamForegroundService
    }

    override fun onBind(intent: Intent?): IBinder = binder

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopStream()
            stopSelf()
            return START_NOT_STICKY
        }
        createChannel()
        startForeground(NOTIFICATION_ID, notification(lastLine))
        return START_STICKY
    }

    override fun onDestroy() {
        Log.i(TAG, "service onDestroy")
        stopStream()
        super.onDestroy()
    }

    override fun onTaskRemoved(rootIntent: Intent?) {
        Log.i(TAG, "task removed; service keeps callback stream until Stop")
        super.onTaskRemoved(rootIntent)
    }

    fun startUiUpdatingStream(sink: ActivityUiBridge.ActivitySink) {
        activitySink = sink
        if (started) {
            publish("normal request attached to running stream")
            return
        }
        startStream()
    }

    private fun startStream() {
        if (started) return
        started = ForegroundServiceDemoJni.WatchTicksStartCallback(
            WatchTicksRequest.newBuilder()
                .setCaller("android-foreground-service")
                .setIntervalMillis(1000)
                .build(),
            object : ForegroundServiceDemoJni.ForegroundServiceDemoWatchTicksListener {
                override fun onMessage(responseBytes: ByteArray) {
                    val tick = Tick.parseFrom(responseBytes)
                    val line = "tick seq=${tick.seq} pid=${tick.pid} instance=${tick.instanceAddress}"
                    publish(line)
                    val sink = activitySink
                    if (sink != null) {
                        try {
                            sink.appendFromServiceCallback(line)
                        } catch (err: Throwable) {
                            publish("bad ui callback failed: ${err.message}")
                        }
                    }
                }

                override fun onDone(error: String?) {
                    publish("done error=${error ?: "none"}")
                    started = false
                }
            },
        )
        publish("callback stream started=$started viaServiceRequest=${activitySink != null}")
        if (!started) stopSelf()
    }

    private fun stopStream() {
        if (!started) return
        val canceled = ForegroundServiceDemoJni.WatchTicksCancelCallback()
        publish("cancel callback result=$canceled")
        started = false
        activitySink = null
    }

    private fun publish(line: String) {
        lastLine = line
        Log.i(TAG, line)
        sendBroadcast(Intent(ACTION_LOG).setPackage(packageName).putExtra(EXTRA_LINE, line))
        getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, notification(line))
    }

    private fun notification(text: String): Notification {
        val stopIntent = Intent(this, StreamForegroundService::class.java).setAction(ACTION_STOP)
        val stopPendingIntent = PendingIntent.getService(
            this,
            0,
            stopIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        return Notification.Builder(this, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_sys_upload)
            .setContentTitle("rpccgo foreground service")
            .setContentText(text)
            .setOngoing(true)
            .addAction(android.R.drawable.ic_menu_close_clear_cancel, "Stop", stopPendingIntent)
            .build()
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java)
        if (manager.getNotificationChannel(CHANNEL_ID) == null) {
            manager.createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "rpccgo stream", NotificationManager.IMPORTANCE_LOW),
            )
        }
    }
}
