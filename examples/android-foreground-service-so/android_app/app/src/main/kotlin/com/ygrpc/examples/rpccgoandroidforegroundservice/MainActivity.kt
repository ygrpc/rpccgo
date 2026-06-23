package com.ygrpc.examples.rpccgoandroidforegroundservice

import android.Manifest
import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.view.ViewGroup
import android.widget.Button
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView

class MainActivity : Activity() {
    companion object {
        private const val TAG = "RpccgoForegroundService"
    }

    private val logView by lazy { TextView(this) }
    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            appendLog(intent.getStringExtra(StreamForegroundService.EXTRA_LINE).orEmpty())
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestNotificationPermission()

        val layout = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(32, 300, 32, 32)
        }
        layout.addView(button("Start foreground service") {
            startForegroundService(Intent(this, StreamForegroundService::class.java))
        })
        layout.addView(button("Finish activity") {
            Log.i(TAG, "activity finish requested")
            finish()
        })
        layout.addView(button("Stop foreground service") {
            startService(
                Intent(
                    this,
                    StreamForegroundService::class.java
                ).setAction(StreamForegroundService.ACTION_STOP)
            )
        })
        layout.addView(ScrollView(this).apply {
            addView(logView)
        }, LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f))
        setContentView(layout)
    }

    override fun onStart() {
        super.onStart()
        Log.i(TAG, "activity onStart")
        val filter = IntentFilter(StreamForegroundService.ACTION_LOG)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(receiver, filter, RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("DEPRECATION")
            registerReceiver(receiver, filter)
        }
    }

    override fun onStop() {
        Log.i(TAG, "activity onStop")
        unregisterReceiver(receiver)
        super.onStop()
    }

    override fun onDestroy() {
        Log.i(TAG, "activity onDestroy")
        super.onDestroy()
    }

    private fun button(text: String, onClick: () -> Unit): Button =
        Button(this).apply {
            this.text = text
            setOnClickListener { onClick() }
        }

    private fun appendLog(line: String) {
        if (line.isBlank()) return
        logView.append(line + "\n")
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
        }
    }
}
