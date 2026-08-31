package com.penndev.prism

import android.app.Application
import com.penndev.prism.data.Prefs
import com.penndev.prism.data.ipregionFile
import com.penndev.prism.data.openIpregionDb
import com.penndev.prism.vpn.VpnController

class PrismApplication : Application() {
    val prefs by lazy { Prefs(this) }

    override fun onCreate() {
        super.onCreate()
        VpnController.rules = prefs.loadRules()
        openIpregionDb(ipregionFile(this))
    }
}
