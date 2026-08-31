package com.penndev.prism.ui.subscribe

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.DropdownField
import com.penndev.prism.ui.components.PreferenceGroup

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SubscribeScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    var subscribeType by remember { mutableStateOf("") }
    var subscribeUrl by remember { mutableStateOf("") }
    var confirmImport by remember { mutableStateOf(false) }

    val openFile = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        val text = runCatching {
            context.contentResolver.openInputStream(uri)?.bufferedReader()?.readText()
        }.getOrNull()
        if (text.isNullOrBlank()) {
            viewModel.snack(R.string.subscribe_error_file)
        } else {
            viewModel.parseSubscription(subscribeType, text, fromFile = true)
        }
    }
    val createFile = rememberLauncherForActivityResult(
        ActivityResultContracts.CreateDocument("application/json"),
    ) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openOutputStream(uri)?.use { out ->
                out.write(viewModel.exportServers().toByteArray(Charsets.UTF_8))
            }
            viewModel.snack(R.string.subscribe_exported)
        }.onFailure {
            viewModel.snack(R.string.subscribe_error_file)
        }
    }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        contentWindowInsets = WindowInsets(0),
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.subscribe_title)) },
                windowInsets = WindowInsets(0),
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item { Spacer(Modifier.height(4.dp)) }
            item {
                PreferenceGroup(
                    title = stringResource(R.string.subscribe_import_title),
                    contentPadding = PaddingValues(16.dp),
                ) {
                    DropdownField(
                        label = stringResource(R.string.subscribe_type),
                        value = subscribeType,
                        options = listOf("Prism", "Shadowrocket"),
                        onSelect = { subscribeType = it },
                    )
                    if (subscribeType.isNotEmpty()) {
                        OutlinedTextField(
                            value = subscribeUrl,
                            onValueChange = { subscribeUrl = it },
                            label = { Text(stringResource(R.string.subscribe_url_label)) },
                            placeholder = { Text(stringResource(R.string.subscribe_url_placeholder)) },
                            singleLine = true,
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 8.dp),
                        )
                        OutlinedButton(
                            onClick = { openFile.launch(arrayOf("application/json", "text/*", "*/*")) },
                            modifier = Modifier.padding(top = 8.dp),
                        ) {
                            Text(stringResource(R.string.subscribe_import_file))
                        }
                        Button(
                            onClick = { viewModel.parseSubscription(subscribeType, subscribeUrl) },
                            enabled = !state.importing,
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 8.dp),
                        ) {
                            if (state.importing) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(18.dp),
                                    strokeWidth = 2.dp,
                                    color = MaterialTheme.colorScheme.onPrimary,
                                )
                            } else {
                                Text(stringResource(R.string.subscribe_parse))
                            }
                        }
                    }
                }
            }
            if (state.importPreview.isNotEmpty()) {
                item {
                    PreferenceGroup(
                        title = stringResource(R.string.subscribe_preview_title),
                        contentPadding = PaddingValues(16.dp),
                    ) {
                        Text(
                            stringResource(
                                R.string.subscribe_preview_meta,
                                state.importSource ?: stringResource(R.string.subscribe_source_url),
                                state.importPreview.size,
                            ),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        Button(
                            onClick = { confirmImport = true },
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 10.dp),
                        ) {
                            Text(stringResource(R.string.subscribe_confirm))
                        }
                    }
                }
                items(state.importPreview, key = { it.id }) { server ->
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(MaterialTheme.colorScheme.surface)
                            .padding(horizontal = 16.dp, vertical = 10.dp),
                    ) {
                        Text(server.displayName, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Text(
                            "${server.protocol}  ${server.host}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
            item {
                PreferenceGroup(
                    title = stringResource(R.string.subscribe_list_title),
                    contentPadding = PaddingValues(16.dp),
                ) {
                    Text(
                        stringResource(R.string.subscribe_current_count, state.servers.size),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedButton(
                        onClick = {
                            if (state.servers.isEmpty()) {
                                viewModel.snack(R.string.subscribe_error_export_empty)
                            } else {
                                createFile.launch("prism-servers.json")
                            }
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 10.dp),
                    ) {
                        Text(stringResource(R.string.subscribe_export))
                    }
                }
            }
            item { Spacer(Modifier.height(16.dp)) }
        }
    }

    if (confirmImport) {
        AlertDialog(
            onDismissRequest = { confirmImport = false },
            title = { Text(stringResource(R.string.subscribe_confirm_title)) },
            text = {
                Text(stringResource(R.string.subscribe_confirm_content, state.importPreview.size))
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        confirmImport = false
                        viewModel.confirmImport()
                        onBack()
                    },
                ) {
                    Text(stringResource(R.string.subscribe_confirm_ok))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmImport = false }) {
                    Text(stringResource(R.string.server_list_delete_cancel))
                }
            },
        )
    }
}
