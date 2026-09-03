package com.penndev.prism.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat
import com.penndev.prism.MainActivity
import com.penndev.prism.R
import com.penndev.prism.data.GeoMode
import com.penndev.prism.engine.Engine
import com.penndev.prism.engine.Handler
import com.penndev.prism.engine.Options
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

class PrismVpnService : VpnService() {
    private val running = AtomicBoolean(false)

    // Engine.start / Engine.stop 都是阻塞调用（stop 要等 gVisor 的 dispatch goroutine 退出），
    // establish() 同样可能耗时，放在主线程会 ANR。
    // 单线程执行器顺带保证启动和停止不会交错。
    private val worker = Executors.newSingleThreadExecutor()

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            VpnController.ACTION_STOP -> worker.execute { stopTunnel() }
            else -> {
                // startForegroundService 之后必须尽快进前台，不能排队等工作线程
                startAsForeground()
                worker.execute { startTunnel() }
            }
        }
        return START_NOT_STICKY
    }

    override fun onDestroy() {
        worker.execute { stopTunnel() }
        worker.shutdown()
        // 给收尾留点时间，但不能无限占着主线程
        runCatching { worker.awaitTermination(SHUTDOWN_WAIT_MS, TimeUnit.MILLISECONDS) }
        super.onDestroy()
    }

    override fun onRevoke() {
        worker.execute { stopTunnel() }
        super.onRevoke()
    }

    private fun startTunnel() {
        if (running.get()) {
            // 已经在跑了，也要回一次状态，否则 UI 的 vpnBusy 复位不了
            VpnController.markRunning(true)
            return
        }

        val server = VpnController.session
        if (server == null) {
            VpnController.emitStatus(getString(R.string.proxy_need_node))
            VpnController.markRunning(false)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }

        val upstream = systemDnsIpv4()

        val builder = Builder()
            .setSession(getString(R.string.app_title))
            .setMtu(MTU)
            .addAddress(TUN_IPV4, 32)
            .addRoute("0.0.0.0", 0)
            .addDnsServer(TUN_DNS)
            .setBlocking(true)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            builder.setMetered(false)
        }
        val pfd = builder.establish()
        if (pfd == null) {
            VpnController.emitStatus(getString(R.string.vpn_establish_failed))
            VpnController.markRunning(false)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }

        // detachFd 之后这个 fd 归 Go 侧所有，由 Engine.stop() 负责关闭；
        // pfd 本身已经不持有它了，不用也不能再 close。
        val fd = pfd.detachFd()
        VpnController.uploadBytes.set(0)
        VpnController.downloadBytes.set(0)
        try {
            val opt = Options()
            opt.setFD(fd)
            opt.setMTU(MTU)
            opt.proxy = server.toProxyURL()
            opt.upstream = upstream
            opt.handler = VpnHandler()
            Engine.start(opt)
        } catch (error: Exception) {
            runCatching { ParcelFileDescriptor.adoptFd(fd).close() }
            VpnController.emitStatus("engine start: ${error.message}")
            VpnController.markRunning(false)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }

        running.set(true)
        VpnController.markRunning(true)
        VpnController.emitStatus(getString(R.string.vpn_started))
    }

    private fun stopTunnel() {
        if (!running.getAndSet(false)) {
            VpnController.markRunning(false)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }
        runCatching { Engine.stop() }
        VpnController.markRunning(false)
        VpnController.emitStatus(getString(R.string.status_stopped))
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun startAsForeground() {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    CHANNEL_ID,
                    getString(R.string.vpn_channel_name),
                    NotificationManager.IMPORTANCE_LOW,
                ),
            )
        }
        val notification = buildNotification()
        if (Build.VERSION.SDK_INT >= 34) {
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE,
            )
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun buildNotification(): Notification {
        val launch = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(getString(R.string.app_title))
            .setContentText(getString(R.string.vpn_notification_text))
            .setContentIntent(launch)
            .setOngoing(true)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .build()
    }

    private fun shouldProxy(address: String): Boolean {
        val rules = VpnController.rules
        return when (rules.geoMode) {
            GeoMode.Global -> true
            GeoMode.None -> false
            GeoMode.Proxy -> inSelectedAreas(address, rules.selectedAreaIds)
            GeoMode.Bypass -> !inSelectedAreas(address, rules.selectedAreaIds)
        }
    }

    private fun inSelectedAreas(address: String, ids: Set<Long>): Boolean {
        if (ids.isEmpty()) return false
        val list = runCatching { Engine.lookup(address) }.getOrNull() ?: return false
        val n = list.len()
        for (i in 0 until n) {
            val area = list.get(i) ?: continue
            if (area.id in ids) return true
        }
        return false
    }

    private fun systemDnsIpv4(): String {
        val cm = getSystemService(ConnectivityManager::class.java) ?: return ""
        val servers = cm.getLinkProperties(cm.activeNetwork)?.dnsServers ?: return ""
        for (server in servers) {
            if (server.isLoopbackAddress || server.isAnyLocalAddress) continue
            val host = server.hostAddress ?: continue
            if (host.contains(':')) continue
            return host
        }
        return ""
    }

    private inner class VpnHandler : Handler {
        override fun protect(fd: Int): Boolean = this@PrismVpnService.protect(fd)

        override fun onLog(line: String?) {
            val text = line.orEmpty()
            if (text.isNotEmpty()) VpnController.emitConnection(text)
        }

        override fun needFake(name: String?): Boolean {
            val host = name.orEmpty().trim('.').lowercase()
            if (host.isEmpty()) return false
            val domains = VpnController.fakeDomains
            if (domains.isEmpty()) return false
            var n = host
            while (n.isNotEmpty()) {
                if (n in domains) return true
                val i = n.indexOf('.')
                if (i < 0) return false
                n = n.substring(i + 1)
            }
            return false
        }

        override fun useProxy(network: String?, address: String?): Boolean {
            val dest = address.orEmpty()
            val use = shouldProxy(dest)
            val tag = if (use) "proxy" else "direct"
            val net = network.orEmpty()
            val line = if (net.isEmpty()) "$tag $dest" else "$tag $net $dest"
            if (dest.isNotEmpty()) VpnController.emitConnection(line)
            return use
        }

        override fun onProxyRead(n: Long) {
            if (n > 0) VpnController.uploadBytes.addAndGet(n)
        }

        override fun onProxyWrite(n: Long) {
            if (n > 0) VpnController.downloadBytes.addAndGet(n)
        }
    }

    companion object {
        private const val CHANNEL_ID = "prism_vpn"
        private const val NOTIFICATION_ID = 1
        private const val MTU = 1500
        private const val TUN_IPV4 = "172.19.0.1"

        // 只是告诉系统「隧道内的 DNS 服务器是这个地址」。
        // 实际上所有目标端口 53 的 UDP 都在 Go 侧被 fakeip 无条件劫持，不看目标 IP，
        // 真正的上游是 VPN 启动前抓到的系统 DNS（见 systemDnsIpv4）。
        private const val TUN_DNS = "114.114.114.114"

        private const val SHUTDOWN_WAIT_MS = 2000L
    }
}
