package com.penndev.prism.ui.logs

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.selection.SelectionContainer
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
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextOverflow
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
                            when (tab) {
                                0 -> viewModel.clearStatusLogs()
                                1 -> viewModel.clearConnectionLogs()
                                else -> viewModel.clearDnsLogs()
                            }
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
                Tab(
                    selected = tab == 2,
                    onClick = { tab = 2 },
                    text = { Text(stringResource(R.string.log_dns_title)) },
                )
            }

            val lines = when (tab) {
                0 -> state.statusLogs
                1 -> state.connectionLogs
                else -> state.dnsLogs
            }
            val empty = when (tab) {
                0 -> stringResource(R.string.log_status_empty)
                1 -> stringResource(R.string.log_connection_empty)
                else -> stringResource(R.string.log_dns_empty)
            }
            val logModifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(MaterialTheme.colorScheme.surface)
            key(tab) {
                val vertical = rememberScrollState()
                val horizontal = rememberScrollState()
                Box(modifier = logModifier) {
                    Column(
                        modifier = Modifier
                            .horizontalScroll(horizontal)
                            .verticalScroll(vertical)
                            .padding(14.dp),
                    ) {
                        if (lines.isEmpty()) {
                            Text(
                                empty,
                                fontFamily = FontFamily.Monospace,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        } else {
                            lines.forEach { line ->
                                SelectionContainer {
                                    Text(
                                        line,
                                        fontFamily = FontFamily.Monospace,
                                        style = MaterialTheme.typography.bodySmall,
                                        softWrap = false,
                                        maxLines = 1,
                                        overflow = TextOverflow.Visible,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
