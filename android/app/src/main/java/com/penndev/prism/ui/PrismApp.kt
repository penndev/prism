package com.penndev.prism.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.automirrored.outlined.List
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.penndev.prism.R
import com.penndev.prism.ui.components.Hairline
import com.penndev.prism.ui.home.HomeScreen
import com.penndev.prism.ui.logs.LogsScreen
import com.penndev.prism.ui.rules.DomainRulesScreen
import com.penndev.prism.ui.rules.GeoRulesScreen
import com.penndev.prism.ui.rules.RulesScreen
import com.penndev.prism.ui.server.ServerEditScreen
import com.penndev.prism.ui.settings.SettingsScreen
import com.penndev.prism.ui.subscribe.SubscribeScreen
import com.penndev.prism.ui.theme.PrismTheme

private object Routes {
    const val Home = "home"
    const val Settings = "settings"
    const val Logs = "logs"
    const val Rules = "rules"
    const val RulesDomain = "rules/domain"
    const val RulesGeo = "rules/geo"
    const val Subscribe = "subscribe"
    const val ServerEdit = "server/edit"
    const val ServerEditId = "server/edit/{id}"
}

@Composable
fun PrismApp(viewModel: PrismViewModel) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    PrismTheme(themeMode = state.settings.system.themeMode) {
        val navController = rememberNavController()
        val snackbarHostState = remember { SnackbarHostState() }
        val backStack by navController.currentBackStackEntryAsState()
        val currentRoute = backStack?.destination?.route
        val tabRoutes = setOf(Routes.Home, Routes.Rules, Routes.Logs, Routes.Settings)
        val showBottomBar = currentRoute in tabRoutes

        LaunchedEffect(state.snackbar) {
            val message = state.snackbar ?: return@LaunchedEffect
            snackbarHostState.showSnackbar(message)
            viewModel.consumeSnackbar()
        }

        Scaffold(
            containerColor = MaterialTheme.colorScheme.background,
            snackbarHost = { SnackbarHost(snackbarHostState) },
            bottomBar = {
                if (showBottomBar) {
                    val tabs = listOf(
                        TabItem(Routes.Home, stringResource(R.string.nav_home), Icons.Outlined.Home, Icons.Filled.Home),
                        TabItem(Routes.Rules, stringResource(R.string.nav_rules), iconRes = R.drawable.ic_nav_rules),
                        TabItem(Routes.Logs, stringResource(R.string.nav_logs), Icons.AutoMirrored.Outlined.List, Icons.AutoMirrored.Filled.List),
                        TabItem(Routes.Settings, stringResource(R.string.nav_settings), Icons.Outlined.Settings, Icons.Filled.Settings),
                    )
                    Column {
                        Hairline()
                        NavigationBar(
                            containerColor = MaterialTheme.colorScheme.surface,
                            tonalElevation = 0.dp,
                        ) {
                            val itemColors = NavigationBarItemDefaults.colors(
                                selectedIconColor = MaterialTheme.colorScheme.primary,
                                selectedTextColor = MaterialTheme.colorScheme.primary,
                                unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                                unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                                indicatorColor = MaterialTheme.colorScheme.surface,
                            )
                            tabs.forEach { tab ->
                                val selected = currentRoute == tab.route
                                NavigationBarItem(
                                    selected = selected,
                                    onClick = { navController.navigateTab(tab.route) },
                                    icon = {
                                        if (tab.iconRes != 0) {
                                            Icon(
                                                painter = painterResource(tab.iconRes),
                                                contentDescription = tab.label,
                                                modifier = Modifier.size(24.dp),
                                            )
                                        } else {
                                            Icon(
                                                imageVector = if (selected) tab.selectedIcon!! else tab.unselectedIcon!!,
                                                contentDescription = tab.label,
                                            )
                                        }
                                    },
                                    label = { Text(tab.label) },
                                    alwaysShowLabel = true,
                                    colors = itemColors,
                                )
                            }
                        }
                    }
                }
            },
        ) { padding ->
            NavHost(
                navController = navController,
                startDestination = Routes.Home,
                modifier = Modifier.padding(padding),
            ) {
                composable(Routes.Home) {
                    HomeScreen(
                        state = state,
                        viewModel = viewModel,
                        onOpenSubscribe = { navController.navigate(Routes.Subscribe) },
                        onAddServer = { navController.navigate(Routes.ServerEdit) },
                        onEditServer = { navController.navigate("${Routes.ServerEdit}/${it.id}") },
                    )
                }
                composable(Routes.Settings) {
                    SettingsScreen(state = state, viewModel = viewModel)
                }
                composable(Routes.Logs) {
                    LogsScreen(state = state, viewModel = viewModel)
                }
                composable(Routes.Rules) {
                    RulesScreen(
                        state = state,
                        onOpenDomain = { navController.navigate(Routes.RulesDomain) },
                        onOpenGeo = { navController.navigate(Routes.RulesGeo) },
                    )
                }
                composable(Routes.RulesDomain) {
                    DomainRulesScreen(
                        state = state,
                        viewModel = viewModel,
                        onBack = { navController.popBackStack() },
                    )
                }
                composable(Routes.RulesGeo) {
                    GeoRulesScreen(
                        state = state,
                        viewModel = viewModel,
                        onBack = { navController.popBackStack() },
                    )
                }
                composable(Routes.Subscribe) {
                    SubscribeScreen(
                        state = state,
                        viewModel = viewModel,
                        onBack = { navController.popBackStack() },
                    )
                }
                composable(Routes.ServerEdit) {
                    ServerEditScreen(
                        viewModel = viewModel,
                        server = null,
                        onBack = { navController.popBackStack() },
                    )
                }
                composable(
                    route = Routes.ServerEditId,
                    arguments = listOf(navArgument("id") { type = NavType.StringType }),
                ) { entry ->
                    val id = entry.arguments?.getString("id")
                    ServerEditScreen(
                        viewModel = viewModel,
                        server = state.servers.firstOrNull { it.id == id },
                        onBack = { navController.popBackStack() },
                    )
                }
            }
        }
    }
}

private data class TabItem(
    val route: String,
    val label: String,
    val unselectedIcon: ImageVector? = null,
    val selectedIcon: ImageVector? = null,
    val iconRes: Int = 0,
)

private fun androidx.navigation.NavHostController.navigateTab(route: String) {
    navigate(route) {
        popUpTo(graph.findStartDestination().id) { saveState = true }
        launchSingleTop = true
        restoreState = true
    }
}
