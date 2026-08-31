package com.penndev.prism.ui.home

import android.app.Activity
import android.net.VpnService
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.penndev.prism.R
import com.penndev.prism.data.ServerItem
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.PreferenceDivider
import com.penndev.prism.ui.components.groupedItemShape
import com.penndev.prism.ui.theme.LatencyBad
import com.penndev.prism.ui.theme.LatencyGood
import com.penndev.prism.ui.theme.LatencyMedium
import com.penndev.prism.ui.theme.PrismBlue

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    onOpenSubscribe: () -> Unit,
    onAddServer: () -> Unit,
    onEditServer: (ServerItem) -> Unit,
) {
    var selecting by remember { mutableStateOf(false) }
    var checkedIds by remember { mutableStateOf(setOf<String>()) }
    var pendingDelete by remember { mutableStateOf<ServerItem?>(null) }
    var pendingBatchDelete by remember { mutableStateOf(false) }
    var waitingPermission by remember { mutableStateOf(false) }
    val selected = state.selectedServer
    val context = LocalContext.current
    val vpnPermission = rememberLauncherForActivityResult(
        ActivityResultContracts.StartActivityForResult(),
    ) { result ->
        waitingPermission = false
        if (result.resultCode == Activity.RESULT_OK) {
            viewModel.start()
        } else {
            viewModel.onVpnPermissionDenied()
        }
    }

    fun exitSelecting() {
        selecting = false
        checkedIds = emptySet()
    }

    BackHandler(enabled = selecting) { exitSelecting() }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        TopAppBar(
            title = {
                if (selecting) {
                    Text(stringResource(R.string.server_list_selected_count, checkedIds.size))
                } else {
                    Text(stringResource(R.string.app_title))
                }
            },
            navigationIcon = {
                if (selecting) {
                    IconButton(onClick = { exitSelecting() }) {
                        Icon(Icons.Filled.Close, contentDescription = stringResource(R.string.server_list_delete_cancel))
                    }
                }
            },
            actions = {
                if (selecting) {
                    TextButton(
                        onClick = {
                            checkedIds = if (checkedIds.size == state.servers.size) {
                                emptySet()
                            } else {
                                state.servers.map { it.id }.toSet()
                            }
                        },
                    ) {
                        Text(stringResource(R.string.server_list_select_all))
                    }
                    IconButton(
                        onClick = { pendingBatchDelete = true },
                        enabled = checkedIds.isNotEmpty(),
                    ) {
                        Icon(
                            Icons.Filled.Delete,
                            contentDescription = stringResource(R.string.server_list_delete_ok),
                            tint = if (checkedIds.isNotEmpty()) LatencyBad else MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    IconButton(onClick = onAddServer) {
                        Icon(
                            Icons.Filled.Add,
                            contentDescription = stringResource(R.string.server_list_add),
                        )
                    }
                    IconButton(onClick = onOpenSubscribe) {
                        Icon(
                            painter = painterResource(R.drawable.ic_import),
                            contentDescription = stringResource(R.string.server_list_import),
                        )
                    }
                    IconButton(
                        onClick = { viewModel.pingAll() },
                        enabled = !state.pingingAll && state.servers.isNotEmpty(),
                    ) {
                        if (state.pingingAll) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(
                                Icons.Filled.Refresh,
                                contentDescription = stringResource(R.string.server_list_ping_all),
                            )
                        }
                    }
                }
            },
            windowInsets = WindowInsets(0),
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
        )

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(bottom = 24.dp),
        ) {
            item {
                val switchOn = waitingPermission || if (state.vpnBusy) state.vpnDesired else state.running
                val statusTitle = when {
                    state.vpnBusy && state.vpnDesired -> stringResource(R.string.proxy_connecting)
                    state.vpnBusy && !state.vpnDesired -> stringResource(R.string.proxy_stopping)
                    state.running -> stringResource(R.string.proxy_connected)
                    else -> stringResource(R.string.proxy_disconnected)
                }
                val dotColor = when {
                    state.running && !state.vpnBusy -> LatencyGood
                    state.vpnBusy || waitingPermission -> PrismBlue
                    else -> MaterialTheme.colorScheme.outline
                }
                val pulse = rememberInfiniteTransition(label = "vpn-status")
                val dotAlpha by pulse.animateFloat(
                    initialValue = 0.4f,
                    targetValue = 1f,
                    animationSpec = infiniteRepeatable(
                        animation = tween(700),
                        repeatMode = RepeatMode.Reverse,
                    ),
                    label = "vpn-dot",
                )
                Row(
                    modifier = Modifier
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surface)
                        .padding(horizontal = 14.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        modifier = Modifier
                            .size(10.dp)
                            .clip(CircleShape)
                            .background(dotColor.copy(alpha = if (state.vpnBusy || waitingPermission) dotAlpha else 1f)),
                    )
                    Column(Modifier.padding(start = 12.dp).weight(1f)) {
                        Text(
                            text = statusTitle,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.SemiBold,
                        )
                        Text(
                            text = selected?.displayName ?: stringResource(R.string.proxy_no_selected),
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (selected == null) {
                                MaterialTheme.colorScheme.onSurfaceVariant
                            } else {
                                PrismBlue
                            },
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.padding(top = 2.dp),
                        )
                    }
                    if (state.vpnBusy) {
                        CircularProgressIndicator(
                            modifier = Modifier
                                .padding(end = 8.dp)
                                .size(16.dp),
                            strokeWidth = 2.dp,
                            color = PrismBlue,
                        )
                    }
                    Switch(
                        checked = switchOn,
                        onCheckedChange = { wantOn ->
                            if (!wantOn) {
                                waitingPermission = false
                                viewModel.stop()
                            } else if (state.selectedServer == null) {
                                viewModel.start()
                            } else {
                                val prepare = VpnService.prepare(context)
                                if (prepare != null) {
                                    waitingPermission = true
                                    vpnPermission.launch(prepare)
                                } else {
                                    viewModel.start()
                                }
                            }
                        },
                        colors = SwitchDefaults.colors(
                            checkedTrackColor = PrismBlue,
                            checkedBorderColor = PrismBlue,
                        ),
                    )
                }
            }
            item {
                Text(
                    text = stringResource(R.string.server_list_title),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 8.dp, bottom = 6.dp),
                )
            }
            if (state.servers.isEmpty()) {
                item {
                    Column(Modifier.padding(horizontal = 20.dp, vertical = 20.dp)) {
                        Text(
                            text = stringResource(R.string.server_list_empty),
                            style = MaterialTheme.typography.bodyLarge,
                            fontWeight = FontWeight.Medium,
                        )
                        Text(
                            text = stringResource(R.string.server_list_empty_hint),
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 6.dp),
                        )
                    }
                }
            } else {
                itemsIndexed(state.servers, key = { _, server -> server.id }) { index, server ->
                    ServerRow(
                        server = server,
                        current = server.id == state.selectedId,
                        checking = selecting,
                        checked = server.id in checkedIds,
                        shape = groupedItemShape(index, state.servers.size),
                        showDivider = index < state.servers.lastIndex,
                        onClick = {
                            if (selecting) {
                                checkedIds = if (server.id in checkedIds) {
                                    checkedIds - server.id
                                } else {
                                    checkedIds + server.id
                                }
                            } else {
                                viewModel.selectServer(server)
                            }
                        },
                        onLongClick = {
                            selecting = true
                            checkedIds = checkedIds + server.id
                        },
                        onEdit = { onEditServer(server) },
                        onDelete = { pendingDelete = server },
                    )
                }
            }
        }
    }

    pendingDelete?.let { server ->
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text(stringResource(R.string.server_list_delete_title)) },
            text = {
                Text(stringResource(R.string.server_list_delete_content, server.displayName))
            },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.deleteServer(server.id)
                    pendingDelete = null
                }) {
                    Text(stringResource(R.string.server_list_delete_ok), color = LatencyBad)
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingDelete = null }) {
                    Text(stringResource(R.string.server_list_delete_cancel))
                }
            },
        )
    }

    if (pendingBatchDelete) {
        AlertDialog(
            onDismissRequest = { pendingBatchDelete = false },
            title = { Text(stringResource(R.string.server_list_delete_title)) },
            text = {
                Text(stringResource(R.string.server_list_batch_delete_content, checkedIds.size))
            },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.deleteServers(checkedIds)
                    pendingBatchDelete = false
                    exitSelecting()
                }) {
                    Text(stringResource(R.string.server_list_delete_ok), color = LatencyBad)
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingBatchDelete = false }) {
                    Text(stringResource(R.string.server_list_delete_cancel))
                }
            },
        )
    }
}
@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ServerRow(
    server: ServerItem,
    current: Boolean,
    checking: Boolean,
    checked: Boolean,
    shape: RoundedCornerShape,
    showDivider: Boolean,
    onClick: () -> Unit,
    onLongClick: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    var menu by remember { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .padding(horizontal = 16.dp)
            .clip(shape)
            .background(
                if (current && !checking) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.55f)
                else MaterialTheme.colorScheme.surface,
            ),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .combinedClickable(onClick = onClick, onLongClick = onLongClick)
                .padding(end = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (checking) {
                Checkbox(
                    checked = checked,
                    onCheckedChange = { onClick() },
                )
            } else {
                Box(
                    modifier = Modifier
                        .padding(vertical = 12.dp)
                        .width(3.dp)
                        .height(28.dp)
                        .background(if (current) PrismBlue else MaterialTheme.colorScheme.surface),
                )
            }
            Column(
                Modifier
                    .weight(1f)
                    .padding(
                        start = if (checking) 0.dp else 13.dp,
                        top = 12.dp,
                        bottom = 12.dp,
                    ),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = server.displayName,
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = if (current && !checking) FontWeight.SemiBold else FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    LatencyLabel(server.latencyMs)
                }
                Text(
                    text = server.protocol,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(top = 2.dp),
                )
            }
            if (!checking) {
                Box {
                    IconButton(onClick = { menu = true }, modifier = Modifier.size(36.dp)) {
                        Icon(
                            Icons.Filled.MoreVert,
                            contentDescription = stringResource(R.string.server_list_more),
                            modifier = Modifier.size(18.dp),
                        )
                    }
                    DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.server_list_edit_title)) },
                            onClick = {
                                menu = false
                                onEdit()
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.server_list_delete_ok), color = LatencyBad) },
                            onClick = {
                                menu = false
                                onDelete()
                            },
                        )
                    }
                }
            }
        }
        if (showDivider) PreferenceDivider()
    }
}

@Composable
private fun LatencyLabel(latencyMs: Int?) {
    if (latencyMs == null) return
    val (text, color) = when {
        latencyMs < 0 -> stringResource(R.string.server_list_ping_failed) to LatencyBad
        latencyMs < 100 -> stringResource(R.string.server_list_ms, latencyMs) to LatencyGood
        latencyMs < 300 -> stringResource(R.string.server_list_ms, latencyMs) to LatencyMedium
        else -> stringResource(R.string.server_list_ms, latencyMs) to LatencyBad
    }
    Text(
        text = text,
        color = color,
        fontSize = 12.sp,
        fontWeight = FontWeight.Medium,
        modifier = Modifier.padding(start = 8.dp),
    )
}
