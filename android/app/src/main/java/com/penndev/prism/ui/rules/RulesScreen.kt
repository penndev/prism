package com.penndev.prism.ui.rules

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.data.AreaUi
import com.penndev.prism.data.GeoMode
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.PreferenceDivider
import com.penndev.prism.ui.components.PreferenceGroup

private data class AreaRow(
    val area: AreaUi,
    val depth: Int,
    val expandable: Boolean,
    val expanded: Boolean,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RulesScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
) {
    val rules = state.rules
    var areaFilter by remember { mutableStateOf("") }
    var expandedIds by remember { mutableStateOf(setOf<Long>()) }
    val showAreas = rules.geoMode == GeoMode.Proxy || rules.geoMode == GeoMode.Bypass
    val hasDb = state.geoAreas.isNotEmpty()
    var editingDb by remember { mutableStateOf(false) }
    val showDbEditor = !state.dbReady || editingDb
    LaunchedEffect(state.dbBusy) {
        if (!state.dbBusy && state.dbReady) editingDb = false
    }
    val pickDb = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) viewModel.importDb(uri)
    }
    val rows = remember(state.geoAreas, areaFilter, expandedIds) {
        flattenAreas(state.geoAreas, expandedIds, areaFilter)
    }
    val modes = listOf(
        Triple(GeoMode.Global, R.string.rules_mode_global, R.string.rules_mode_global_desc),
        Triple(GeoMode.None, R.string.rules_mode_none, R.string.rules_mode_none_desc),
        Triple(GeoMode.Proxy, R.string.rules_mode_proxy, R.string.rules_mode_proxy_desc),
        Triple(GeoMode.Bypass, R.string.rules_mode_bypass, R.string.rules_mode_bypass_desc),
    )

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(stringResource(R.string.nav_rules)) },
            windowInsets = WindowInsets(0),
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
        )
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    stringResource(R.string.rules_desc),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 16.dp),
                )
            }
            item {
                PreferenceGroup(
                    title = stringResource(R.string.rules_db_label),
                    actions = {
                        if (state.dbReady) {
                            TextButton(
                                onClick = { editingDb = !editingDb },
                                enabled = !state.dbBusy,
                            ) {
                                Text(
                                    stringResource(
                                        if (editingDb) R.string.rules_db_cancel else R.string.rules_db_replace,
                                    ),
                                )
                            }
                        }
                    },
                ) {
                    Text(
                        state.dbStatus.ifBlank { stringResource(R.string.rules_db_missing) },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
                    )
                    if (state.dbBusy) {
                        LinearProgressIndicator(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 4.dp),
                        )
                    }
                    if (showDbEditor) {
                        OutlinedTextField(
                            value = rules.dbUrl,
                            onValueChange = { value -> viewModel.updateRules { it.copy(dbUrl = value) } },
                            enabled = !state.dbBusy,
                            label = { Text(stringResource(R.string.rules_db_url_label)) },
                            placeholder = { Text(stringResource(R.string.rules_db_url)) },
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp),
                        )
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
                        ) {
                            Button(
                                onClick = { viewModel.downloadDb() },
                                enabled = !state.dbBusy,
                            ) {
                                Text(stringResource(R.string.rules_download))
                            }
                            Button(
                                onClick = { pickDb.launch("*/*") },
                                enabled = !state.dbBusy,
                            ) {
                                Text(stringResource(R.string.rules_upload))
                            }
                        }
                    }
                }
            }
            item {
                PreferenceGroup(title = stringResource(R.string.rules_geo_title)) {
                    Text(
                        stringResource(R.string.rules_geo_desc),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                    modes.forEachIndexed { index, (mode, title, desc) ->
                        val enabled = mode == GeoMode.Global || mode == GeoMode.None || hasDb
                        if (index > 0) PreferenceDivider()
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable(enabled = enabled) {
                                    viewModel.updateRules { it.copy(geoMode = mode) }
                                }
                                .padding(horizontal = 8.dp, vertical = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            RadioButton(
                                selected = rules.geoMode == mode,
                                onClick = { viewModel.updateRules { it.copy(geoMode = mode) } },
                                enabled = enabled,
                            )
                            Column(Modifier.weight(1f)) {
                                Text(
                                    stringResource(title),
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = if (enabled) {
                                        MaterialTheme.colorScheme.onSurface
                                    } else {
                                        MaterialTheme.colorScheme.onSurface.copy(alpha = 0.38f)
                                    },
                                )
                                Text(
                                    stringResource(desc),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                }
            }
            if (showAreas) {
                item {
                    Column(Modifier.padding(horizontal = 16.dp)) {
                        OutlinedTextField(
                            value = areaFilter,
                            onValueChange = { areaFilter = it },
                            label = { Text(stringResource(R.string.rules_area_filter)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Text(
                            stringResource(R.string.rules_selected_count, rules.selectedAreaIds.size),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 8.dp),
                        )
                    }
                }
                if (!hasDb) {
                    item {
                        Text(
                            stringResource(R.string.rules_area_empty),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(horizontal = 16.dp),
                        )
                    }
                } else {
                    items(rows, key = { it.area.id }) { row ->
                        AreaTreeRow(
                            row = row,
                            selected = row.area.id in rules.selectedAreaIds,
                            onToggleExpand = {
                                expandedIds = if (row.area.id in expandedIds) {
                                    expandedIds - row.area.id
                                } else {
                                    expandedIds + row.area.id
                                }
                            },
                            onToggleSelect = {
                                viewModel.updateRules { draft ->
                                    val next = draft.selectedAreaIds.toMutableSet()
                                    if (row.area.id in next) next.remove(row.area.id) else next.add(row.area.id)
                                    draft.copy(selectedAreaIds = next)
                                }
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun AreaTreeRow(
    row: AreaRow,
    selected: Boolean,
    onToggleExpand: () -> Unit,
    onToggleSelect: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onToggleSelect)
            .padding(end = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Spacer(Modifier.width((12 + row.depth * 16).dp))
        if (row.expandable) {
            IconButton(onClick = onToggleExpand, modifier = Modifier.size(36.dp)) {
                Icon(
                    imageVector = if (row.expanded) {
                        Icons.Filled.KeyboardArrowDown
                    } else {
                        Icons.AutoMirrored.Filled.KeyboardArrowRight
                    },
                    contentDescription = null,
                )
            }
        } else {
            Spacer(Modifier.width(36.dp))
        }
        Checkbox(checked = selected, onCheckedChange = { onToggleSelect() })
        Text(
            row.area.name,
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.weight(1f),
        )
    }
}

private fun flattenAreas(
    nodes: List<AreaUi>,
    expanded: Set<Long>,
    filter: String,
): List<AreaRow> {
    val query = filter.trim()
    val out = mutableListOf<AreaRow>()
    fun matches(node: AreaUi): Boolean {
        if (query.isEmpty()) return true
        if (node.name.contains(query, ignoreCase = true) || node.id.toString().contains(query)) {
            return true
        }
        return node.children.any(::matches)
    }
    fun walk(node: AreaUi, depth: Int) {
        if (!matches(node)) return
        val expandable = node.children.isNotEmpty()
        val open = expandable && (query.isNotEmpty() || node.id in expanded)
        out += AreaRow(node, depth, expandable, open)
        if (open) {
            node.children.forEach { walk(it, depth + 1) }
        }
    }
    nodes.forEach { walk(it, 0) }
    return out
}
