package com.penndev.prism.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat
import com.penndev.prism.MainActivity
import com.penndev.prism.R
import com.penndev.prism.mstack.Logger
import com.penndev.prism.mstack.Mstack
import com.penndev.prism.mstack.Protector
import java.util.concurrent.atomic.AtomicBoolean

class PrismVpnService : VpnService() {
    private val running = AtomicBoolean(false)

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            VpnController.ACTION_STOP -> stopTunnel()
            else -> startTunnel()
        }
        return START_NOT_STICKY
    }

    override fun onDestroy() {
        stopTunnel()
        super.onDestroy()
    }

    override fun onRevoke() {
        stopTunnel()
        super.onRevoke()
    }

    private fun startTunnel() {
        if (running.get()) return
        startAsForeground()

        val builder = Builder()
            .setSession(getString(R.string.app_title))
            .setMtu(MTU)
            .addAddress(TUN_IPV4, 32)
            .addRoute("0.0.0.0", 0)
            .addDnsServer("8.8.8.8")
            .addDnsServer("1.1.1.1")
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

        val fd = pfd.detachFd()
        pfd.close()
        try {
            Mstack.start(fd, MTU, VpnProtector(), VpnLogger())
        } catch (error: Exception) {
            runCatching { ParcelFileDescriptor.adoptFd(fd).close() }
            VpnController.emitStatus("mstack start: ${error.message}")
            VpnController.markRunning(false)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return
        }

        running.set(true)
        VpnController.uploadBytes.set(0)
        VpnController.downloadBytes.set(0)
        VpnController.markRunning(true)
        VpnController.emitStatus(getString(R.string.vpn_started))
    }

    private fun stopTunnel() {
        if (!running.getAndSet(false)) {
            VpnController.markRunning(false)
            stopSelf()
            return
        }
        runCatching { Mstack.stop() }
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

    private fun protectSocket(fd: Int): Boolean = protect(fd)

    private inner class VpnProtector : Protector {
        override fun protect(fd: Int): Boolean = protectSocket(fd)
    }

    private inner class VpnLogger : Logger {
        override fun onConnect(network: String?, address: String?) {
            val net = network.orEmpty()
            val addr = address.orEmpty()
            if (net.isEmpty() && addr.isEmpty()) return
            VpnController.emitConnection("$net $addr")
        }
    }

    companion object {
        private const val CHANNEL_ID = "prism_vpn"
        private const val NOTIFICATION_ID = 1
        private const val MTU = 1500
        private const val TUN_IPV4 = "172.19.0.1"
    }
}
