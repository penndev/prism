package com.penndev.prism.ui.rules

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.data.DEMO_AREAS
import com.penndev.prism.data.GeoMode
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.SettingSection

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun RulesScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
) {
    val rules = state.rules
    var areaFilter by remember { mutableStateOf("") }
    val pending = stringResource(R.string.business_pending)
    val showAreas = rules.geoMode == GeoMode.Proxy || rules.geoMode == GeoMode.Bypass
    val areas = DEMO_AREAS.filter { it.contains(areaFilter, ignoreCase = true) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        TopAppBar(
            title = { Text(stringResource(R.string.nav_rules)) },
            windowInsets = WindowInsets(0),
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                stringResource(R.string.rules_desc),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            SettingSection(title = stringResource(R.string.rules_geo_title)) {
                Text(
                    stringResource(R.string.rules_geo_desc),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(
                    stringResource(R.string.rules_db_label),
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.padding(top = 12.dp),
                )
                Text(
                    stringResource(R.string.rules_db_placeholder),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 4.dp, bottom = 8.dp),
                )
                OutlinedTextField(
                    value = "",
                    onValueChange = {},
                    enabled = false,
                    label = { Text(stringResource(R.string.rules_db_url)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.padding(top = 8.dp),
                ) {
                    Button(onClick = { viewModel.notifyPending(pending) }) {
                        Text(stringResource(R.string.rules_download))
                    }
                    Button(onClick = { viewModel.notifyPending(pending) }) {
                        Text(stringResource(R.string.rules_upload))
                    }
                }

                val modes = listOf(
                    GeoMode.Global to R.string.rules_mode_global,
                    GeoMode.None to R.string.rules_mode_none,
                    GeoMode.Proxy to R.string.rules_mode_proxy,
                    GeoMode.Bypass to R.string.rules_mode_bypass,
                )
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.padding(top = 12.dp),
                ) {
                    modes.forEach { (mode, label) ->
                        FilterChip(
                            selected = rules.geoMode == mode,
                            onClick = { viewModel.updateRules { it.copy(geoMode = mode) } },
                            label = { Text(stringResource(label)) },
                        )
                    }
                }

                if (showAreas) {
                    OutlinedTextField(
                        value = areaFilter,
                        onValueChange = { areaFilter = it },
                        label = { Text(stringResource(R.string.rules_area_filter)) },
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 8.dp),
                    )
                    Text(
                        stringResource(R.string.rules_area_demo),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                    FlowRow(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.padding(top = 8.dp),
                    ) {
                        areas.forEach { area ->
                            val selected = area in rules.selectedAreas
                            FilterChip(
                                selected = selected,
                                onClick = {
                                    viewModel.updateRules { draft ->
                                        val next = draft.selectedAreas.toMutableSet()
                                        if (selected) next.remove(area) else next.add(area)
                                        draft.copy(selectedAreas = next)
                                    }
                                },
                                label = { Text(area) },
                            )
                        }
                    }
                }
            }

            SettingSection(title = stringResource(R.string.rules_domain_title)) {
                Text(
                    stringResource(R.string.rules_domain_desc),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                OutlinedTextField(
                    value = rules.domains,
                    onValueChange = { value -> viewModel.updateRules { it.copy(domains = value) } },
                    placeholder = { Text(stringResource(R.string.rules_domain_placeholder)) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(160.dp)
                        .padding(top = 8.dp),
                )
            }

            Button(
                onClick = { viewModel.saveRules(pending) },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stringResource(R.string.rules_save))
            }
        }
    }
}
