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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
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
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
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
    val selected: Boolean,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GeoRulesScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    onBack: () -> Unit,
) {
    val rules = state.rules
    var areaFilter by rememberSaveable { mutableStateOf("") }
    // expandedIds 依赖 geoAreas，跨进程恢复没有意义，所以仍用 remember
    var expandedIds by remember(state.geoAreas) {
        mutableStateOf(topLevelIdsWithSelected(state.geoAreas, rules.selectedAreaIds))
    }
    val showAreas = rules.geoMode == GeoMode.Proxy || rules.geoMode == GeoMode.Bypass
    val hasDb = state.geoAreas.isNotEmpty()
    var editingDb by rememberSaveable { mutableStateOf(false) }
    // 输入时只改本地草稿，真正要用的时候才落库：updateRules 每次都会把整份
    // 地域 ID 和域名列表序列化后写 SharedPreferences，扛不住逐字符触发。
    var dbUrlDraft by rememberSaveable(rules.dbUrl) { mutableStateOf(rules.dbUrl) }
    val showDbEditor = !state.dbReady || editingDb
    LaunchedEffect(state.dbBusy) {
        if (!state.dbBusy && state.dbReady) editingDb = false
    }
    val pickDb = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) viewModel.importDb(uri)
    }
    val uriHandler = LocalUriHandler.current
    val dbPage = stringResource(R.string.rules_db_page)
    val rows = remember(state.geoAreas, areaFilter, expandedIds, rules.selectedAreaIds) {
        flattenAreas(state.geoAreas, expandedIds, areaFilter, rules.selectedAreaIds)
    }
    val modes = listOf(
        Triple(GeoMode.Global, R.string.rules_mode_global, R.string.rules_mode_global_desc),
        Triple(GeoMode.None, R.string.rules_mode_none, R.string.rules_mode_none_desc),
        Triple(GeoMode.Proxy, R.string.rules_mode_proxy, R.string.rules_mode_proxy_desc),
        Triple(GeoMode.Bypass, R.string.rules_mode_bypass, R.string.rules_mode_bypass_desc),
    )

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(stringResource(R.string.rules_geo_title)) },
            windowInsets = WindowInsets(0),
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
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
                        Text(
                            stringResource(R.string.rules_db_hint),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { uriHandler.openUri(dbPage) }
                                .padding(horizontal = 16.dp, vertical = 4.dp),
                        )
                        OutlinedTextField(
                            value = dbUrlDraft,
                            onValueChange = { dbUrlDraft = it },
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
                                onClick = {
                                    viewModel.updateRules { it.copy(dbUrl = dbUrlDraft) }
                                    viewModel.downloadDb()
                                },
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
                            onToggleExpand = {
                                expandedIds = if (row.area.id in expandedIds) {
                                    expandedIds - row.area.id
                                } else {
                                    expandedIds + row.area.id
                                }
                            },
                            onToggleSelect = {
                                viewModel.updateRules { draft ->
                                    draft.copy(
                                        selectedAreaIds = toggleAreaSelection(
                                            state.geoAreas,
                                            draft.selectedAreaIds,
                                            row.area,
                                            row.selected,
                                        ),
                                    )
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
        Checkbox(checked = row.selected, onCheckedChange = { onToggleSelect() })
        Text(
            row.area.name,
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.weight(1f),
        )
    }
}

private fun selfAndDescendantIds(node: AreaUi): Set<Long> = buildSet {
    fun walk(n: AreaUi) {
        add(n.id)
        n.children.forEach(::walk)
    }
    walk(node)
}

// 父地区勾上时子地区只是显示成勾选，id 并不在集合里。
// 去勾子节点要把祖先拆开，把旁支加回去，才能做到「选整个亚洲但排除日本」。
private fun toggleAreaSelection(
    tree: List<AreaUi>,
    selected: Set<Long>,
    node: AreaUi,
    currentlyChecked: Boolean,
): Set<Long> {
    val next = selected.toMutableSet()
    if (!currentlyChecked) {
        next.add(node.id)
        return next
    }
    if (node.id in next) {
        next.removeAll(selfAndDescendantIds(node))
        return next
    }
    val path = pathFromRoot(tree, node.id) ?: run {
        next.removeAll(selfAndDescendantIds(node))
        return next
    }
    for (i in path.indices.reversed()) {
        val cur = path[i]
        if (cur.id !in next) continue
        next.remove(cur.id)
        val skip = if (i + 1 < path.size) path[i + 1].id else node.id
        cur.children.forEach { child ->
            if (child.id != skip) next.add(child.id)
        }
    }
    next.removeAll(selfAndDescendantIds(node))
    return next
}

private fun pathFromRoot(nodes: List<AreaUi>, id: Long): List<AreaUi>? {
    fun walk(n: AreaUi, acc: List<AreaUi>): List<AreaUi>? {
        val next = acc + n
        if (n.id == id) return next
        for (c in n.children) {
            walk(c, next)?.let { return it }
        }
        return null
    }
    for (n in nodes) {
        walk(n, emptyList())?.let { return it }
    }
    return null
}

private fun topLevelIdsWithSelected(nodes: List<AreaUi>, selected: Set<Long>): Set<Long> {
    if (selected.isEmpty()) return emptySet()
    fun containsSelected(node: AreaUi): Boolean =
        node.id in selected || node.children.any(::containsSelected)
    return nodes.mapNotNull { node -> node.id.takeIf { containsSelected(node) } }.toSet()
}

private fun flattenAreas(
    nodes: List<AreaUi>,
    expanded: Set<Long>,
    filter: String,
    selectedIds: Set<Long>,
): List<AreaRow> {
    val query = filter.trim()
    val out = mutableListOf<AreaRow>()

    // 自底向上算一遍并缓存。原来 walk 对每个节点都现算 matches，
    // 而 matches 会递归整棵子树，于是每个节点被它的每一个祖先重扫一次，
    // 退化成 O(n²)——过滤框每敲一个键都在主线程跑一次。
    val matched = HashMap<Long, Boolean>()
    fun computeMatches(node: AreaUi): Boolean {
        val self = query.isEmpty() ||
            node.name.contains(query, ignoreCase = true) ||
            node.id.toString().contains(query)
        // 不能短路：子节点的结果也要落进缓存
        var any = false
        node.children.forEach { if (computeMatches(it)) any = true }
        val result = self || any
        matched[node.id] = result
        return result
    }
    nodes.forEach { computeMatches(it) }

    fun walk(node: AreaUi, depth: Int, ancestorSelected: Boolean) {
        if (matched[node.id] != true) return
        val expandable = node.children.isNotEmpty()
        val open = expandable && (query.isNotEmpty() || node.id in expanded)
        val selected = ancestorSelected || node.id in selectedIds
        out += AreaRow(node, depth, expandable, open, selected)
        if (open) {
            node.children.forEach { walk(it, depth + 1, selected) }
        }
    }
    nodes.forEach { walk(it, 0, false) }
    return out
}
