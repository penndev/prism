package com.penndev.prism.ui.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.data.ThemeMode
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.PreferenceDivider
import com.penndev.prism.ui.components.PreferenceGroup
import com.penndev.prism.ui.components.PreferenceRow
import com.penndev.prism.ui.components.PreferenceSwitch

private enum class SettingsDialog { None, Language, Theme, LatencyHost }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
) {
    val settings = state.settings
    var dialog by remember { mutableStateOf(SettingsDialog.None) }
    val languageLabel = if (settings.system.language == "zh-CN") {
        stringResource(R.string.lang_zh)
    } else {
        stringResource(R.string.lang_en)
    }
    val themeLabel = when (settings.system.themeMode) {
        ThemeMode.System -> stringResource(R.string.settings_theme_system)
        ThemeMode.Light -> stringResource(R.string.settings_theme_light)
        ThemeMode.Dark -> stringResource(R.string.settings_theme_dark)
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        TopAppBar(
            title = { Text(stringResource(R.string.nav_settings)) },
            windowInsets = WindowInsets(0),
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState()),
        ) {
            PreferenceGroup(title = stringResource(R.string.settings_latency_test)) {
                PreferenceRow(
                    title = stringResource(R.string.settings_latency_test_host),
                    value = settings.latencyTest.host.ifBlank { "—" },
                    onClick = { dialog = SettingsDialog.LatencyHost },
                )
                PreferenceDivider()
                PreferenceSwitch(
                    title = stringResource(R.string.settings_sort_by_latency),
                    description = stringResource(R.string.settings_sort_by_latency_desc),
                    checked = settings.latencyTest.sortAfterPing,
                    onCheckedChange = { checked ->
                        viewModel.updateLatencySettings { it.copy(sortAfterPing = checked) }
                    },
                )
            }
            PreferenceGroup(title = stringResource(R.string.settings_system)) {
                PreferenceRow(
                    title = stringResource(R.string.settings_system_language),
                    value = languageLabel,
                    onClick = { dialog = SettingsDialog.Language },
                )
                PreferenceDivider()
                PreferenceRow(
                    title = stringResource(R.string.settings_theme),
                    value = themeLabel,
                    onClick = { dialog = SettingsDialog.Theme },
                )
                PreferenceDivider()
                PreferenceSwitch(
                    title = stringResource(R.string.settings_enable_log),
                    description = stringResource(R.string.settings_enable_log_desc),
                    checked = settings.system.enableLogRecording,
                    onCheckedChange = { checked ->
                        viewModel.updateSystemSettings { it.copy(enableLogRecording = checked) }
                    },
                )
            }
        }
    }

    when (dialog) {
        SettingsDialog.None -> Unit
        SettingsDialog.Language -> OptionDialog(
            title = stringResource(R.string.settings_select_language),
            options = listOf(
                "zh-CN" to stringResource(R.string.lang_zh),
                "en" to stringResource(R.string.lang_en),
            ),
            selected = settings.system.language,
            onDismiss = { dialog = SettingsDialog.None },
            onSelect = { lang ->
                viewModel.updateSystemSettings { it.copy(language = lang) }
                dialog = SettingsDialog.None
            },
        )
        SettingsDialog.Theme -> OptionDialog(
            title = stringResource(R.string.settings_theme),
            options = listOf(
                ThemeMode.System.name to stringResource(R.string.settings_theme_system),
                ThemeMode.Light.name to stringResource(R.string.settings_theme_light),
                ThemeMode.Dark.name to stringResource(R.string.settings_theme_dark),
            ),
            selected = settings.system.themeMode.name,
            onDismiss = { dialog = SettingsDialog.None },
            onSelect = { name ->
                viewModel.updateSystemSettings { it.copy(themeMode = ThemeMode.valueOf(name)) }
                dialog = SettingsDialog.None
            },
        )
        SettingsDialog.LatencyHost -> {
            var draft by remember { mutableStateOf(settings.latencyTest.host) }
            AlertDialog(
                onDismissRequest = { dialog = SettingsDialog.None },
                title = { Text(stringResource(R.string.settings_latency_test_host)) },
                text = {
                    OutlinedTextField(
                        value = draft,
                        onValueChange = { draft = it },
                        placeholder = { Text(stringResource(R.string.settings_latency_test_host_placeholder)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                },
                confirmButton = {
                    TextButton(onClick = {
                        viewModel.updateLatencySettings { it.copy(host = draft.trim()) }
                        dialog = SettingsDialog.None
                    }) {
                        Text(stringResource(R.string.settings_confirm))
                    }
                },
                dismissButton = {
                    TextButton(onClick = { dialog = SettingsDialog.None }) {
                        Text(stringResource(R.string.server_list_delete_cancel))
                    }
                },
            )
        }
    }
}

@Composable
private fun OptionDialog(
    title: String,
    options: List<Pair<String, String>>,
    selected: String,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column {
                options.forEach { (key, label) ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onSelect(key) }
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(selected = key == selected, onClick = { onSelect(key) })
                        Text(label)
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.server_list_delete_cancel))
            }
        },
    )
}
