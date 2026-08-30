package com.penndev.prism

import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.appcompat.app.AppCompatActivity
import androidx.compose.runtime.Composable
import androidx.lifecycle.viewmodel.compose.viewModel
import com.penndev.prism.ui.PrismApp
import com.penndev.prism.ui.PrismViewModel

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            PrismRoot()
        }
    }
}

@Composable
private fun PrismRoot(viewModel: PrismViewModel = viewModel()) {
    PrismApp(viewModel)
}
