package com.penndev.prism.ui.rules

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import com.penndev.prism.R
import com.penndev.prism.data.GeoMode
import com.penndev.prism.ui.PrismUiState
import com.penndev.prism.ui.components.PreferenceDivider
import com.penndev.prism.ui.components.PreferenceGroup
import com.penndev.prism.ui.components.PreferenceRow

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RulesScreen(
    state: PrismUiState,
    onOpenDomain: () -> Unit,
    onOpenGeo: () -> Unit,
) {
    val rules = state.rules
    val domainValue = if (rules.domains.isEmpty()) {
        stringResource(R.string.rules_domain_empty)
    } else {
        stringResource(R.string.rules_domain_count, rules.domains.size)
    }
    val geoValue = when (rules.geoMode) {
        GeoMode.Global -> stringResource(R.string.rules_mode_global)
        GeoMode.None -> stringResource(R.string.rules_mode_none)
        GeoMode.Proxy -> stringResource(R.string.rules_mode_proxy)
        GeoMode.Bypass -> stringResource(R.string.rules_mode_bypass)
    }

    Column(Modifier.fillMaxSize()) {
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
                .verticalScroll(rememberScrollState()),
        ) {
            PreferenceGroup(title = stringResource(R.string.rules_section)) {
                PreferenceRow(
                    title = stringResource(R.string.rules_entry_domain),
                    description = stringResource(R.string.rules_entry_domain_desc),
                    value = domainValue,
                    onClick = onOpenDomain,
                )
                PreferenceDivider()
                PreferenceRow(
                    title = stringResource(R.string.rules_entry_geo),
                    description = stringResource(R.string.rules_entry_geo_desc),
                    value = geoValue,
                    onClick = onOpenGeo,
                )
            }
        }
    }
}
