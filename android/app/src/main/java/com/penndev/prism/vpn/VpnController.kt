package com.penndev.prism.vpn

import android.content.Context
import android.content.Intent
import androidx.core.content.ContextCompat
import com.penndev.prism.data.RuleDraft
import com.penndev.prism.data.ServerItem
import com.penndev.prism.data.TrafficUi
import java.util.concurrent.atomic.AtomicLong
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow

object VpnController {
    const val ACTION_START = "com.penndev.prism.vpn.START"
    const val ACTION_STOP = "com.penndev.prism.vpn.STOP"

    private val _running = MutableStateFlow(false)
    val running: StateFlow<Boolean> = _running.asStateFlow()

    private val _status = MutableSharedFlow<String>(extraBufferCapacity = 64)
    val status: SharedFlow<String> = _status.asSharedFlow()

    private val _connections = MutableSharedFlow<String>(extraBufferCapacity = 256)
    val connections: SharedFlow<String> = _connections.asSharedFlow()

    private val _settled = MutableSharedFlow<Boolean>(extraBufferCapacity = 8)
    val settled: SharedFlow<Boolean> = _settled.asSharedFlow()

    val uploadBytes = AtomicLong(0)
    val downloadBytes = AtomicLong(0)

    @Volatile
    var session: ServerItem? = null
        private set

    @Volatile
    var rules: RuleDraft = RuleDraft()

    @Volatile
    private var lastUp = 0L

    @Volatile
    private var lastDown = 0L

    fun start(context: Context, server: ServerItem) {
        session = server
        val intent = Intent(context, PrismVpnService::class.java).setAction(ACTION_START)
        ContextCompat.startForegroundService(context, intent)
    }

    fun stop(context: Context) {
        context.startService(Intent(context, PrismVpnService::class.java).setAction(ACTION_STOP))
    }

    fun markRunning(value: Boolean) {
        _running.value = value
        _settled.tryEmit(value)
        if (!value) {
            lastUp = 0
            lastDown = 0
        }
    }

    fun emitStatus(line: String) {
        _status.tryEmit(line)
    }

    fun emitConnection(line: String) {
        _connections.tryEmit(line)
    }

    fun trafficUi(): TrafficUi {
        val up = uploadBytes.get()
        val down = downloadBytes.get()
        val upSpeed = (up - lastUp).coerceAtLeast(0)
        val downSpeed = (down - lastDown).coerceAtLeast(0)
        lastUp = up
        lastDown = down
        return TrafficUi(
            downSpeed = formatRate(downSpeed),
            upSpeed = formatRate(upSpeed),
            downTotal = formatBytes(down),
            upTotal = formatBytes(up),
        )
    }

    private fun formatRate(bytes: Long): String = "${formatBytes(bytes)}/s"

    private fun formatBytes(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val units = arrayOf("KB", "MB", "GB", "TB")
        var value = bytes.toDouble() / 1024
        var index = 0
        while (value >= 1024 && index < units.lastIndex) {
            value /= 1024
            index++
        }
        return "%.1f %s".format(value, units[index])
    }
}
