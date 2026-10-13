package com.penndev.prism.ui.subscribe

import android.widget.ImageButton
import com.journeyapps.barcodescanner.CaptureActivity
import com.journeyapps.barcodescanner.DecoratedBarcodeView
import com.penndev.prism.R

class QrScanActivity : CaptureActivity() {
    override fun initializeContent(): DecoratedBarcodeView {
        setContentView(R.layout.activity_qr_scan)
        findViewById<ImageButton>(R.id.qr_scan_back).setOnClickListener { finish() }
        return findViewById(R.id.zxing_barcode_scanner)
    }
}
