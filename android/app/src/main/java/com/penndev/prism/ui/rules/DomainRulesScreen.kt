package com.penndev.prism.ui.rules

import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.penndev.prism.R
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DomainRulesScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    // 只在返回时才保存，用 remember 的话旋转屏幕会把整段编辑内容丢掉
    var text by rememberSaveable {
        mutableStateOf(state.rules.domains.joinToString("\n"))
    }
    var sourceUrl by rememberSaveable { mutableStateOf("") }
    var fetching by remember { mutableStateOf(false) }
    var linkDialog by rememberSaveable { mutableStateOf(false) }

    fun saveAndBack() {
        viewModel.updateRules { it.copy(domains = parseDomainList(text)) }
        onBack()
    }

    fun mergeContent(raw: String) {
        val body = raw.trim()
        if (body.isEmpty()) return
        text = if (text.trim().isEmpty()) body else text.trimEnd() + "\n" + body
    }

    fun fetchLink() {
        val url = sourceUrl.trim()
        if (url.isEmpty()) {
            viewModel.snack(R.string.subscribe_error_url)
            return
        }
        fetching = true
        scope.launch {
            val body = withContext(Dispatchers.IO) {
                runCatching {
                    val uri = URI(url)
                    val scheme = uri.scheme?.lowercase().orEmpty()
                    if (scheme != "http" && scheme != "https") error("bad url")
                    val connection = (uri.toURL().openConnection() as HttpURLConnection).apply {
                        connectTimeout = 15_000
                        readTimeout = 15_000
                        instanceFollowRedirects = true
                        setRequestProperty("User-Agent", "Prism-Android/0.1")
                    }
                    try {
                        val code = connection.responseCode
                        val stream = if (code in 200..299) connection.inputStream else connection.errorStream
                        val raw = stream?.bufferedReader()?.use { it.readText() }.orEmpty().trim()
                        if (code !in 200..299 || raw.isEmpty() || raw.length > 2 * 1024 * 1024) {
                            error("fetch failed")
                        }
                        raw
                    } finally {
                        connection.disconnect()
                    }
                }.getOrNull()
            }
            fetching = false
            if (body.isNullOrBlank()) {
                viewModel.snack(R.string.rules_domain_fetch_error)
            } else {
                mergeContent(body)
                linkDialog = false
            }
        }
    }

    val openFile = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        val raw = runCatching {
            context.contentResolver.openInputStream(uri)?.bufferedReader()?.readText()
        }.getOrNull()?.trim().orEmpty()
        if (raw.isEmpty()) {
            viewModel.snack(R.string.subscribe_error_file)
        } else {
            mergeContent(raw)
        }
    }

    BackHandler { saveAndBack() }

    Column(
        Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
    ) {
        TopAppBar(
            title = { Text(stringResource(R.string.rules_domain_title)) },
            windowInsets = WindowInsets(0),
            navigationIcon = {
                IconButton(onClick = { saveAndBack() }) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.background,
            ),
            actions = {
                IconButton(onClick = { linkDialog = true }, enabled = !fetching) {
                    Icon(
                        painter = painterResource(R.drawable.ic_import),
                        contentDescription = stringResource(R.string.rules_domain_import_url),
                    )
                }
            },
        )
        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState()),
        ) {
            Text(
                stringResource(R.string.rules_domain_hint),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 8.dp, bottom = 6.dp),
            )
            BasicTextField(
                value = text,
                onValueChange = { text = it },
                textStyle = MaterialTheme.typography.bodyLarge.copy(
                    color = MaterialTheme.colorScheme.onSurface,
                ),
                modifier = Modifier
                    .padding(horizontal = 16.dp)
                    .fillMaxWidth()
                    .heightIn(min = 220.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(MaterialTheme.colorScheme.surface)
                    .padding(16.dp),
                decorationBox = { inner ->
                    Box(Modifier.fillMaxWidth()) {
                        if (text.isEmpty()) {
                            Text(
                                stringResource(R.string.rules_domain_placeholder),
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        inner()
                    }
                },
            )
            Spacer(Modifier.height(16.dp))
        }
        Text(
            stringResource(R.string.rules_domain_import_file),
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier
                .padding(horizontal = 16.dp, vertical = 12.dp)
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .background(MaterialTheme.colorScheme.surface)
                .clickable { openFile.launch(arrayOf("text/plain", "text/*", "*/*")) }
                .padding(horizontal = 16.dp, vertical = 14.dp),
        )
    }

    if (linkDialog) {
        Dialog(onDismissRequest = { if (!fetching) linkDialog = false }) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(12.dp))
                    .background(MaterialTheme.colorScheme.surface)
                    .padding(16.dp),
            ) {
                Text(
                    stringResource(R.string.rules_domain_import_url),
                    style = MaterialTheme.typography.titleSmall,
                )
                BasicTextField(
                    value = sourceUrl,
                    onValueChange = { sourceUrl = it },
                    singleLine = true,
                    textStyle = MaterialTheme.typography.bodyLarge.copy(
                        color = MaterialTheme.colorScheme.onSurface,
                    ),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 12.dp),
                    decorationBox = { inner ->
                        Box(Modifier.fillMaxWidth()) {
                            if (sourceUrl.isEmpty()) {
                                Text(
                                    stringResource(R.string.rules_domain_url_placeholder),
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
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 8.dp),
                    horizontalArrangement = Arrangement.End,
                ) {
                    TextButton(
                        onClick = { linkDialog = false },
                        enabled = !fetching,
                    ) {
                        Text(stringResource(R.string.server_list_delete_cancel))
                    }
                    Button(
                        onClick = { fetchLink() },
                        enabled = !fetching,
                    ) {
                        if (fetching) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(16.dp),
                                strokeWidth = 2.dp,
                                color = MaterialTheme.colorScheme.onPrimary,
                            )
                        } else {
                            Text(stringResource(R.string.rules_domain_fetch))
                        }
                    }
                }
            }
        }
    }
}

private fun parseDomainList(raw: String): List<String> {
    // 按行保留，注释行也存着方便回看；匹配时 fakeDomains 会再过滤。
    val seen = LinkedHashSet<String>()
    raw.split('\n', '\r').forEach { item ->
        val d = item.trim().trim('.').lowercase()
        if (d.isNotEmpty()) seen.add(d)
    }
    return seen.toList()
}
