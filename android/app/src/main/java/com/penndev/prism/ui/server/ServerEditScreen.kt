package com.penndev.prism.ui.server

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.penndev.prism.R
import com.penndev.prism.data.PROXY_SCHEMES
import com.penndev.prism.data.ServerItem
import com.penndev.prism.ui.PrismViewModel
import com.penndev.prism.ui.components.DropdownField

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerEditScreen(
    viewModel: PrismViewModel,
    server: ServerItem?,
    onBack: () -> Unit,
) {
    var host by remember { mutableStateOf(server?.host.orEmpty()) }
    var remark by remember { mutableStateOf(server?.remark.orEmpty()) }
    var protocol by remember { mutableStateOf(server?.protocol ?: "Socks5") }
    var username by remember { mutableStateOf(server?.username.orEmpty()) }
    var password by remember { mutableStateOf(server?.password.orEmpty()) }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        contentWindowInsets = WindowInsets(0),
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        if (server == null) stringResource(R.string.server_list_add_title)
                        else stringResource(R.string.server_list_edit_title),
                    )
                },
                windowInsets = WindowInsets(0),
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
        ) {
            OutlinedTextField(
                value = host,
                onValueChange = { host = it },
                label = { Text(stringResource(R.string.server_list_host)) },
                placeholder = { Text(stringResource(R.string.server_list_host_placeholder)) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = remark,
                onValueChange = { remark = it },
                label = { Text(stringResource(R.string.server_list_remark)) },
                placeholder = { Text(stringResource(R.string.server_list_remark_placeholder)) },
                singleLine = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp),
            )
            DropdownField(
                label = stringResource(R.string.server_list_protocol),
                value = protocol,
                options = PROXY_SCHEMES,
                modifier = Modifier.padding(top = 8.dp),
                onSelect = { protocol = it },
            )
            OutlinedTextField(
                value = username,
                onValueChange = { username = it },
                label = { Text(stringResource(R.string.settings_username)) },
                placeholder = { Text(stringResource(R.string.settings_username_placeholder)) },
                singleLine = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp),
            )
            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text(stringResource(R.string.settings_password)) },
                placeholder = { Text(stringResource(R.string.settings_password_placeholder)) },
                visualTransformation = PasswordVisualTransformation(),
                singleLine = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp),
            )
            Button(
                onClick = {
                    val ok = viewModel.addOrUpdateServer(
                        editingId = server?.id,
                        host = host,
                        remark = remark,
                        protocol = protocol,
                        username = username,
                        password = password,
                    )
                    if (ok) onBack()
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 20.dp),
            ) {
                Text(stringResource(R.string.server_list_save))
            }
        }
    }
}
