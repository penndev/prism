package com.penndev.prism

import android.app.Application
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import com.penndev.prism.data.LANGUAGE_SYSTEM
import com.penndev.prism.data.Prefs
import com.penndev.prism.data.ipregionFile
import com.penndev.prism.data.openIpregionDb
import com.penndev.prism.vpn.VpnController

class PrismApplication : Application() {
    val prefs by lazy { Prefs(this) }

    override fun onCreate() {
        super.onCreate()
        applyAppLanguage(prefs.loadSettings().system.language)
        VpnController.rules = prefs.loadRules()
        openIpregionDb(ipregionFile(this))
    }
}

fun applyAppLanguage(tag: String) {
    val locales = if (tag.isBlank() || tag == LANGUAGE_SYSTEM) {
        LocaleListCompat.getEmptyLocaleList()
    } else {
        LocaleListCompat.forLanguageTags(tag)
    }
    if (AppCompatDelegate.getApplicationLocales().toLanguageTags() != locales.toLanguageTags()) {
        AppCompatDelegate.setApplicationLocales(locales)
    }
}
