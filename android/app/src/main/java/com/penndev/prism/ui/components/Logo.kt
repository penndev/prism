package com.penndev.prism.ui.components

import androidx.compose.foundation.Image
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import com.penndev.prism.R

@Composable
fun PrismLogo(modifier: Modifier = Modifier) {
    val dark = MaterialTheme.colorScheme.background.luminance() < 0.5f
    Image(
        painter = painterResource(if (dark) R.drawable.logo_dark else R.drawable.logo),
        contentDescription = stringResource(R.string.app_title),
        modifier = modifier,
    )
}
