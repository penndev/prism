package com.penndev.prism.ui.subscribe

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import com.penndev.prism.R
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.groupedItemShape

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SubscribeScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    editing: Boolean,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val existing = if (editing) state.subscription else null
    var subscribeType by rememberSaveable { mutableStateOf(existing?.type.orEmpty()) }
    var subscribeUrl by rememberSaveable { mutableStateOf(existing?.url.orEmpty()) }
    var subscribeRemark by rememberSaveable { mutableStateOf(existing?.remark.orEmpty()) }
    var pendingDelete by rememberSaveable { mutableStateOf(false) }
    var typeDialog by rememberSaveable { mutableStateOf(false) }
    val typeOptions = listOf("Prism", "Shadowrocket")

    val listState = rememberLazyListState()
    val focusManager = LocalFocusManager.current
    val keyboard = LocalSoftwareKeyboardController.current
    LaunchedEffect(state.importPreview) {
        if (state.importPreview.isEmpty()) return@LaunchedEffect
        focusManager.clearFocus(true)
        keyboard?.hide()
        listState.animateScrollToItem(2 + state.importPreview.size)
    }

    val scanQr = rememberLauncherForActivityResult(ScanContract()) { result ->
        val text = result.contents?.trim().orEmpty()
        if (text.isNotEmpty()) {
            subscribeUrl = text
            viewModel.snack(R.string.subscribe_scan_saved)
        }
    }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        contentWindowInsets = WindowInsets(0),
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.subscribe_title)) },
                windowInsets = WindowInsets(0),
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.background,
                ),
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
                actions = {
                    IconButton(
                        onClick = {
                            scanQr.launch(
                                ScanOptions().apply {
                                    setDesiredBarcodeFormats(ScanOptions.QR_CODE)
                                    setPrompt(context.getString(R.string.subscribe_scan_prompt))
                                    setBeepEnabled(false)
                                    setOrientationLocked(false)
                                    setCaptureActivity(QrScanActivity::class.java)
                                },
                            )
                        },
                    ) {
                        Icon(
                            painter = painterResource(R.drawable.ic_qr_code),
                            contentDescription = stringResource(R.string.subscribe_scan),
                        )
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            state = listState,
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            contentPadding = PaddingValues(bottom = 24.dp),
        ) {
            item {
                Column(
                    Modifier
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surface),
                ) {
                    ChoiceRow(
                        label = stringResource(R.string.subscribe_type),
                        value = subscribeType,
                        placeholder = stringResource(R.string.subscribe_type_placeholder),
                        onClick = { typeDialog = true },
                    )
                    HorizontalDivider(
                        modifier = Modifier.padding(start = 16.dp),
                        color = MaterialTheme.colorScheme.outline.copy(alpha = 0.35f),
                    )
                    UnderlineField(
                        label = stringResource(R.string.subscribe_url_label),
                        value = subscribeUrl,
                        placeholder = stringResource(R.string.subscribe_url_placeholder),
                        onValueChange = { subscribeUrl = it },
                    )
                    UnderlineField(
                        label = stringResource(R.string.subscribe_remark),
                        value = subscribeRemark,
                        placeholder = stringResource(R.string.server_list_remark_placeholder),
                        onValueChange = { subscribeRemark = it },
                        showDivider = false,
                    )
                }
                OutlinedButton(
                    onClick = { viewModel.parseSubscription(subscribeType, subscribeUrl) },
                    enabled = !state.importing,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp)
                        .padding(bottom = 8.dp),
                ) {
                    if (state.importing) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                        )
                    } else {
                        Text(stringResource(R.string.subscribe_parse))
                    }
                }
            }
            if (state.importPreview.isNotEmpty()) {
                item {
                    Text(
                        stringResource(
                            R.string.subscribe_preview_meta,
                            state.importSource ?: stringResource(R.string.subscribe_source_url),
                            state.importPreview.size,
                        ),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 8.dp, bottom = 6.dp),
                    )
                }
                itemsIndexed(state.importPreview, key = { _, server -> server.id }) { index, server ->
                    val last = index == state.importPreview.lastIndex
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp)
                            .clip(groupedItemShape(index, state.importPreview.size))
                            .background(MaterialTheme.colorScheme.surface),
                    ) {
                        Column(Modifier.padding(horizontal = 16.dp, vertical = 10.dp)) {
                            Text(server.displayName, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            Text(
                                "${server.protocol}  ${server.host}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        if (!last) {
                            HorizontalDivider(
                                modifier = Modifier.padding(start = 16.dp),
                                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.35f),
                            )
                        }
                    }
                }
            }
            if (editing || state.importPreview.isNotEmpty()) {
                item {
                    Column(Modifier.padding(horizontal = 16.dp).padding(top = 16.dp)) {
                        Button(
                            onClick = {
                                if (state.importPreview.isNotEmpty()) {
                                    viewModel.confirmImport(subscribeType, subscribeUrl, subscribeRemark)
                                } else {
                                    viewModel.updateSubscriptionInfo(subscribeType, subscribeUrl, subscribeRemark)
                                }
                                onBack()
                            },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(
                                stringResource(
                                    if (state.importPreview.isNotEmpty()) {
                                        R.string.subscribe_confirm
                                    } else {
                                        R.string.server_list_save
                                    },
                                ),
                            )
                        }
                        if (editing) {
                            Button(
                                onClick = { pendingDelete = true },
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = MaterialTheme.colorScheme.error,
                                    contentColor = MaterialTheme.colorScheme.onError,
                                ),
                            ) {
                                Text(stringResource(R.string.subscribe_delete))
                            }
                        }
                    }
                }
            }
        }
    }

    if (typeDialog) {
        Dialog(onDismissRequest = { typeDialog = false }) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(MaterialTheme.colorScheme.surface),
            ) {
                Text(
                    stringResource(R.string.subscribe_type),
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 14.dp),
                )
                HorizontalDivider(color = MaterialTheme.colorScheme.outline.copy(alpha = 0.35f))
                typeOptions.forEach { option ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                subscribeType = option
                                typeDialog = false
                            }
                            .padding(horizontal = 16.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        RadioButton(
                            selected = option == subscribeType,
                            onClick = {
                                subscribeType = option
                                typeDialog = false
                            },
                        )
                        Text(option, style = MaterialTheme.typography.bodyLarge)
                    }
                }
                TextButton(
                    onClick = { typeDialog = false },
                    modifier = Modifier
                        .align(Alignment.End)
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                ) {
                    Text(stringResource(R.string.server_list_delete_cancel))
                }
            }
        }
    }

    if (pendingDelete) {
        AlertDialog(
            onDismissRequest = { pendingDelete = false },
            title = { Text(stringResource(R.string.subscribe_delete_title)) },
            text = { Text(stringResource(R.string.subscribe_delete_content)) },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.deleteSubscription()
                    pendingDelete = false
                    onBack()
                }) {
                    Text(
                        stringResource(R.string.server_list_delete_ok),
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingDelete = false }) {
                    Text(stringResource(R.string.server_list_delete_cancel))
                }
            },
        )
    }
}

@Composable
private fun UnderlineField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String = "",
    showDivider: Boolean = true,
) {
    Column(Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                label,
                modifier = Modifier.width(80.dp),
                style = MaterialTheme.typography.bodyLarge,
            )
            BasicTextField(
                value = value,
                onValueChange = onValueChange,
                singleLine = true,
                textStyle = MaterialTheme.typography.bodyLarge.copy(
                    color = MaterialTheme.colorScheme.onSurface,
                ),
                modifier = Modifier.weight(1f),
                decorationBox = { inner ->
                    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
                        if (value.isEmpty() && placeholder.isNotEmpty()) {
                            Text(
                                placeholder,
                                modifier = Modifier.fillMaxWidth(),
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        inner()
                    }
                },
            )
        }
        if (showDivider) {
            HorizontalDivider(
                modifier = Modifier.padding(start = 16.dp),
                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.35f),
            )
        }
    }
}

@Composable
private fun ChoiceRow(
    label: String,
    value: String,
    placeholder: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            label,
            modifier = Modifier.width(80.dp),
            style = MaterialTheme.typography.bodyLarge,
        )
        Text(
            if (value.isBlank()) placeholder else value,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.bodyLarge,
            color = if (value.isBlank()) {
                MaterialTheme.colorScheme.onSurfaceVariant
            } else {
                MaterialTheme.colorScheme.onSurface
            },
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        Icon(
            Icons.AutoMirrored.Outlined.KeyboardArrowRight,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.outline,
        )
    }
}
