package com.penndev.prism

import android.app.Application
import com.penndev.prism.data.LocalPrismRepository
import com.penndev.prism.data.PrismRepository

class PrismApplication : Application() {
    val repository: PrismRepository by lazy { LocalPrismRepository(this) }
}
