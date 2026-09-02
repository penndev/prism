package com.penndev.prism.ui.rules

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.activity.compose.BackHandler
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.PrismViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DomainRulesScreen(
    state: PrismUiState,
    viewModel: PrismViewModel,
    onBack: () -> Unit,
) {
    var text by remember {
        mutableStateOf(state.rules.domains.joinToString("\n"))
    }

    fun saveAndBack() {
        viewModel.updateRules { it.copy(domains = parseDomainList(text)) }
        onBack()
    }

    BackHandler { saveAndBack() }

    Column(Modifier.fillMaxSize()) {
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
        )
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp),
        ) {
            Text(
                stringResource(R.string.rules_domain_hint),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(12.dp))
            OutlinedTextField(
                value = text,
                onValueChange = { text = it },
                modifier = Modifier.fillMaxWidth(),
                minLines = 10,
                placeholder = { Text(stringResource(R.string.rules_domain_placeholder)) },
            )
        }
    }
}

private fun parseDomainList(raw: String): List<String> {
    val seen = LinkedHashSet<String>()
    raw.split('\n', '\r', ',', ' ', '\t').forEach { item ->
        val d = item.trim().trim('.').lowercase()
        if (d.isNotEmpty()) seen.add(d)
    }
    return seen.toList()
}
