package com.penndev.prism.vpn

import android.content.Context
import android.content.Intent
import androidx.core.content.ContextCompat
import com.penndev.prism.data.RuleDraft
import com.penndev.prism.data.ServerItem
import com.penndev.prism.data.TrafficUi
import com.penndev.prism.data.formatBytes
import java.util.concurrent.atomic.AtomicLong
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow

object VpnController {
    const val ACTION_START = "com.penndev.prism.vpn.START"
    const val ACTION_STOP = "com.penndev.prism.vpn.STOP"

    // 用 SharedFlow 而不是 StateFlow：Service 的所有启动失败分支和重复停止分支
    // 都是在 _running 已经是 false 的情况下再写一次 false，StateFlow 会去重、
    // 不发射，于是 UI 侧的 vpnBusy 永远复位不了，开关卡死。
    // replay = 1 保留「当前状态」语义，晚订阅的收集者仍能拿到最后一次的值。
    private val _running = MutableSharedFlow<Boolean>(
        replay = 1,
        onBufferOverflow = BufferOverflow.DROP_OLDEST,
    )
    val running: SharedFlow<Boolean> = _running.asSharedFlow()

    private val _status = MutableSharedFlow<String>(extraBufferCapacity = 64)
    val status: SharedFlow<String> = _status.asSharedFlow()

    private val _connections = MutableSharedFlow<String>(extraBufferCapacity = 256)
    val connections: SharedFlow<String> = _connections.asSharedFlow()

    val uploadBytes = AtomicLong(0)
    val downloadBytes = AtomicLong(0)

    @Volatile
    var session: ServerItem? = null
        private set

    @Volatile
    var fakeDomains: Set<String> = emptySet()
        private set

    @Volatile
    var rules: RuleDraft = RuleDraft()
        set(value) {
            field = value
            fakeDomains = value.domains.filter { d ->
                d.isNotEmpty() && android.util.Patterns.DOMAIN_NAME.matcher(d).matches()
            }.toHashSet()
        }

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
        _running.tryEmit(value)
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
}
