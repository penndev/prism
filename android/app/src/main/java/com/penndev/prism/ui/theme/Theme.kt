package com.penndev.prism.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import com.penndev.prism.data.ThemeMode

private val LightColors = lightColorScheme(
    primary = PrismBlue,
    onPrimary = Color.White,
    primaryContainer = Color(0xFFE6F4FF),
    onPrimaryContainer = Color(0xFF003A8C),
    secondary = PrismBlue,
    background = Color(0xFFF5F5F5),
    surface = Color.White,
    surfaceContainer = Color.White,
    surfaceVariant = Color(0xFFF5F5F5),
    outline = Color(0xFFD9D9D9),
    onSurface = Color(0xFF1F1F1F),
    onSurfaceVariant = Color(0xFF8C8C8C),
)

private val DarkColors = darkColorScheme(
    primary = PrismBlueDark,
    onPrimary = Color.White,
    primaryContainer = Color(0xFF163A66),
    onPrimaryContainer = Color(0xFFD6E8FF),
    secondary = PrismBlueDark,
    background = Color(0xFF141414),
    surface = Color(0xFF1F1F1F),
    surfaceContainer = Color(0xFF1F1F1F),
    surfaceVariant = Color(0xFF2A2A2A),
    outline = Color(0xFF434343),
    onSurface = Color(0xFFFAFAFA),
    onSurfaceVariant = Color(0xFFA6A6A6),
)

@Composable
fun PrismTheme(
    themeMode: ThemeMode,
    content: @Composable () -> Unit,
) {
    val dark = when (themeMode) {
        ThemeMode.Light -> false
        ThemeMode.Dark -> true
        ThemeMode.System -> isSystemInDarkTheme()
    }
    MaterialTheme(
        colorScheme = if (dark) DarkColors else LightColors,
        typography = Typography,
        content = content,
    )
}
