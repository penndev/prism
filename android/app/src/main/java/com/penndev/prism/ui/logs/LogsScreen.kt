package com.penndev.prism.ui.logs

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.PreferenceDivider
import com.penndev.prism.ui.components.PreferenceGroup
import com.penndev.prism.ui.components.PreferenceRow

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LogsScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
) {
    var tab by remember { mutableIntStateOf(0) }
    val enabled = state.settings.system.enableLogRecording

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        TopAppBar(
            title = { Text(stringResource(R.string.nav_logs)) },
            windowInsets = WindowInsets(0),
            actions = {
                if (enabled) {
                    TextButton(
                        onClick = {
                            if (tab == 0) viewModel.clearStatusLogs() else viewModel.clearConnectionLogs()
                        },
                    ) {
                        Text(stringResource(R.string.log_clear))
                    }
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
        )

        PreferenceGroup(title = stringResource(R.string.log_traffic_title)) {
            PreferenceRow(
                title = stringResource(R.string.log_traffic_down),
                value = "${state.traffic.downSpeed} · ${state.traffic.downTotal}",
                showChevron = false,
            )
            PreferenceDivider()
            PreferenceRow(
                title = stringResource(R.string.log_traffic_up),
                value = "${state.traffic.upSpeed} · ${state.traffic.upTotal}",
                showChevron = false,
            )
        }

        if (!enabled) {
            Text(
                text = stringResource(R.string.log_disabled_hint),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(24.dp),
            )
        } else {
            TabRow(
                selectedTabIndex = tab,
                containerColor = MaterialTheme.colorScheme.background,
                modifier = Modifier.padding(top = 12.dp),
            ) {
                Tab(
                    selected = tab == 0,
                    onClick = { tab = 0 },
                    text = { Text(stringResource(R.string.log_status_title)) },
                )
                Tab(
                    selected = tab == 1,
                    onClick = { tab = 1 },
                    text = { Text(stringResource(R.string.log_connection_title)) },
                )
            }

            val lines = if (tab == 0) state.statusLogs else state.connectionLogs
            val empty = if (tab == 0) {
                stringResource(R.string.log_status_empty)
            } else {
                stringResource(R.string.log_connection_empty)
            }
            Text(
                text = lines.joinToString("\n").ifBlank { empty },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(16.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(MaterialTheme.colorScheme.surface)
                    .padding(14.dp)
                    .verticalScroll(rememberScrollState()),
                fontFamily = FontFamily.Monospace,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}
