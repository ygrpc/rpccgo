package com.ygrpc.examples.rpccgofluttersharedso

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.pm.PackageManager
import android.hardware.camera2.CameraAccessException
import android.hardware.camera2.CameraCharacteristics
import android.hardware.camera2.CameraManager
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import examples.flutter.sharedso.v1.IncrementRuntimeStateRequest
import examples.flutter.sharedso.v1.ReadRuntimeStateRequest
import examples.flutter.sharedso.v1.RuntimeStateResponse
import examples.flutter.sharedso.v1.SetTorchRequest
import examples.flutter.sharedso.v1.SetTorchResponse
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit

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

    private var androidDeviceRegistered = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        createChannel()
        startForeground(NOTIFICATION_ID, notification())
        registerAndroidDeviceServer()
        when (intent?.action) {
            ACTION_INCREMENT -> increment()
            ACTION_READ -> read()
            else -> read()
        }
        return START_STICKY
    }

    private fun registerAndroidDeviceServer() {
        if (androidDeviceRegistered) return
        for (result in listOf(
            SharedSoDemoJni.RegisterSetTorch(::setTorch),
            SharedSoDemoJni.RegisterWatchTorch(::watchTorch),
            SharedSoDemoJni.RegisterCollectTorch(::collectTorch),
            SharedSoDemoJni.RegisterChatTorch(::chatTorch),
        )) {
            if (!result.ok) {
                publishLine("android device register error=${result.error ?: "unknown"}")
                return
            }
        }
        androidDeviceRegistered = true
        publishLine("android device server registered")
    }

    private fun setTorch(req: SetTorchRequest): RpccgoResult<SetTorchResponse> {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.M) {
            return RpccgoResult.failure("torch mode requires Android 6.0 or newer")
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M &&
            checkSelfPermission(Manifest.permission.CAMERA) != PackageManager.PERMISSION_GRANTED
        ) {
            return RpccgoResult.failure("camera permission is not granted")
        }
        return try {
            val cameraId = torchCameraId()
                ?: return RpccgoResult.failure("no flash camera available")
            getSystemService(CameraManager::class.java).setTorchMode(cameraId, req.enabled)
            RpccgoResult.success(
                SetTorchResponse.newBuilder()
                    .setEnabled(req.enabled)
                    .setCameraId(cameraId)
                    .setCaller(req.caller.ifBlank { "unknown-caller" })
                    .setStatus(if (req.enabled) "torch-on" else "torch-off")
                    .build(),
            )
        } catch (error: CameraAccessException) {
            RpccgoResult.failure("camera access failed: ${error.message ?: error.reason}")
        } catch (error: SecurityException) {
            RpccgoResult.failure("camera permission denied: ${error.message ?: error::class.java.name}")
        } catch (error: Exception) {
            RpccgoResult.failure("torch operation failed: ${error.message ?: error::class.java.name}")
        }
    }

    private fun watchTorch(req: SetTorchRequest): RpccgoResult<SharedSoDemoJni.AndroidDeviceWatchTorchServerHandler> {
        val base = SetTorchResponse.newBuilder()
            .setEnabled(req.enabled)
            .setCameraId(torchCameraId() ?: "unknown-camera")
            .setCaller(req.caller.ifBlank { "unknown-caller" })
            .setStatus(if (req.enabled) "torch-watch-enabled" else "torch-watch-disabled")
            .build()
        return RpccgoResult.success(object : SharedSoDemoJni.AndroidDeviceWatchTorchServerHandler {
            private var next = 0

            override fun Recv(): RpccgoResult<SetTorchResponse> {
                if (next >= 3) return RpccgoResult.failure("EOF")
                next += 1
                return RpccgoResult.success(base.toBuilder().setStatus("torch-watch-$next").build())
            }

            override fun Finish(): RpccgoResult<Unit> = RpccgoResult.success(Unit)
            override fun Cancel(): RpccgoResult<Unit> = RpccgoResult.success(Unit)
        })
    }

    private fun collectTorch(): RpccgoResult<SharedSoDemoJni.AndroidDeviceCollectTorchServerHandler> =
        RpccgoResult.success(object : SharedSoDemoJni.AndroidDeviceCollectTorchServerHandler {
            private var last: SetTorchResponse? = null

            override fun Send(req: SetTorchRequest): RpccgoResult<Unit> {
                last = SetTorchResponse.newBuilder()
                    .setEnabled(req.enabled)
                    .setCameraId(torchCameraId() ?: "unknown-camera")
                    .setCaller(req.caller.ifBlank { "unknown-caller" })
                    .setStatus(if (req.enabled) "torch-collect-enabled" else "torch-collect-disabled")
                    .build()
                return RpccgoResult.success(Unit)
            }

            override fun Finish(): RpccgoResult<SetTorchResponse> =
                last?.let { RpccgoResult.success(it.toBuilder().setStatus("torch-collect-finished").build()) }
                    ?: RpccgoResult.failure("torch collect received no requests")

            override fun Cancel(): RpccgoResult<Unit> = RpccgoResult.success(Unit)
        })

    private fun chatTorch(): RpccgoResult<SharedSoDemoJni.AndroidDeviceChatTorchServerHandler> =
        RpccgoResult.success(object : SharedSoDemoJni.AndroidDeviceChatTorchServerHandler {
            private val responses = LinkedBlockingQueue<SetTorchResponse>()
            @Volatile private var closed = false

            override fun Send(req: SetTorchRequest): RpccgoResult<Unit> {
                if (closed) return RpccgoResult.failure("torch chat is closed")
                responses.put(
                    SetTorchResponse.newBuilder()
                        .setEnabled(req.enabled)
                        .setCameraId(torchCameraId() ?: "unknown-camera")
                        .setCaller(req.caller.ifBlank { "unknown-caller" })
                        .setStatus("torch-chat")
                        .build(),
                )
                return RpccgoResult.success(Unit)
            }

            override fun Recv(): RpccgoResult<SetTorchResponse> {
                while (true) {
                    try {
                        responses.poll(100, TimeUnit.MILLISECONDS)
                            ?.let { return RpccgoResult.success(it) }
                    } catch (error: InterruptedException) {
                        Thread.currentThread().interrupt()
                        return RpccgoResult.failure("torch chat recv interrupted")
                    }
                    if (closed) return RpccgoResult.failure("EOF")
                }
            }

            override fun CloseSend(): RpccgoResult<Unit> {
                closed = true
                return RpccgoResult.success(Unit)
            }
            override fun Finish(): RpccgoResult<Unit> {
                closed = true
                return RpccgoResult.success(Unit)
            }
            override fun Cancel(): RpccgoResult<Unit> {
                closed = true
                return RpccgoResult.success(Unit)
            }
        })

    private fun torchCameraId(): String? {
        val cameraManager = getSystemService(CameraManager::class.java)
        var fallback: String? = null
        for (cameraId in cameraManager.cameraIdList) {
            val characteristics = cameraManager.getCameraCharacteristics(cameraId)
            if (characteristics.get(CameraCharacteristics.FLASH_INFO_AVAILABLE) != true) continue
            if (fallback == null) fallback = cameraId
            if (characteristics.get(CameraCharacteristics.LENS_FACING) == CameraCharacteristics.LENS_FACING_BACK) {
                return cameraId
            }
        }
        return fallback
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
        publishLine(line)
    }

    private fun publishLine(line: String) {
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
