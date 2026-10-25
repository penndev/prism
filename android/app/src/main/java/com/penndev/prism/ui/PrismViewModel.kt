package com.penndev.prism.ui

import android.app.Application
import android.net.Uri
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.penndev.prism.PrismApplication
import com.penndev.prism.applyAppLanguage
import com.penndev.prism.R
import com.penndev.prism.data.AppSettings
import com.penndev.prism.data.AreaUi
import com.penndev.prism.data.GeoMode
import com.penndev.prism.data.HOST_PATTERN
import com.penndev.prism.data.LatencyTestSettings
import com.penndev.prism.data.Prefs
import com.penndev.prism.data.RuleDraft
import com.penndev.prism.data.ServerItem
import com.penndev.prism.data.Subscription
import com.penndev.prism.data.SubscriptionException
import com.penndev.prism.data.SubscriptionParser
import com.penndev.prism.data.SystemSettings
import com.penndev.prism.data.TrafficUi
import com.penndev.prism.data.downloadIpregionDb
import com.penndev.prism.data.formatBytes
import com.penndev.prism.data.installIpregionDb
import com.penndev.prism.data.ipregionFile
import com.penndev.prism.data.loadAreaTree
import com.penndev.prism.engine.Engine
import com.penndev.prism.engine.PingCallback
import com.penndev.prism.vpn.VpnController
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import java.util.UUID
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

private const val LOG_LIMIT = 1000
internal const val PING_MANUAL = "manual"

data class PrismUiState(
    val settings: AppSettings = AppSettings(),
    val servers: List<ServerItem> = emptyList(),
    val subscription: Subscription? = null,
    val selectedId: String? = null,
    val running: Boolean = false,
    val vpnBusy: Boolean = false,
    val vpnDesired: Boolean = false,
    val pingingGroup: String? = null,
    val importing: Boolean = false,
    val importPreview: List<ServerItem> = emptyList(),
    val importSource: String? = null,
    val statusLogs: List<String> = emptyList(),
    val connectionLogs: List<String> = emptyList(),
    val traffic: TrafficUi = TrafficUi(),
    val rules: RuleDraft = RuleDraft(),
    val geoAreas: List<AreaUi> = emptyList(),
    val dbStatus: String = "",
    val dbReady: Boolean = false,
    val dbBusy: Boolean = false,
    val snackbar: String? = null,
) {
    val selectedServer: ServerItem?
        get() = servers.firstOrNull { it.id == selectedId }
}

class PrismViewModel(application: Application) : AndroidViewModel(application) {
    private val prefs: Prefs = (application as PrismApplication).prefs
    private val app get() = getApplication<Application>()

    private val _state = MutableStateFlow(PrismUiState())
    val state: StateFlow<PrismUiState> = _state.asStateFlow()

    init {
        val settings = prefs.loadSettings()
        val servers = prefs.loadServers()
        val rules = prefs.loadRules()
        _state.value = PrismUiState(
            settings = settings,
            servers = servers,
            subscription = prefs.loadSubscription(),
            selectedId = prefs.loadSelectedId(),
            rules = rules,
        )
        // Service 读的是 VpnController.rules，只在 updateRules 里写。
        // 冷启动后不进规则页就直接连，保存过的地域规则和 fakeip 域名会静默失效。
        VpnController.rules = rules
        appendStatus(str(R.string.status_ready, servers.size))
        observeVpn()
        viewModelScope.launch(Dispatchers.IO) { refreshEngineUi() }
    }

    fun consumeSnackbar() {
        _state.update { it.copy(snackbar = null) }
    }

    fun snack(message: String) {
        _state.update { it.copy(snackbar = message) }
    }

    fun snack(resId: Int, vararg args: Any) {
        snack(str(resId, *args))
    }

    fun selectServer(server: ServerItem?) {
        val previous = _state.value.selectedId
        _state.update { it.copy(selectedId = server?.id) }
        prefs.saveSelectedId(server?.id)
        if (server == null) {
            if (_state.value.running) stop()
            appendStatus(str(R.string.status_node_cleared))
            return
        }
        appendStatus(str(R.string.status_node_selected, server.displayName))
        if (server.id != previous && _state.value.running) {
            VpnController.start(app, server)
            appendStatus(str(R.string.status_vpn_starting, server.displayName))
        }
    }

    fun start() {
        if (_state.value.vpnBusy) return
        val server = _state.value.selectedServer
        if (server == null) {
            snack(str(R.string.proxy_need_node))
            return
        }
        _state.update { it.copy(vpnBusy = true, vpnDesired = true) }
        VpnController.start(app, server)
        appendStatus(str(R.string.status_vpn_starting, server.displayName))
    }

    fun stop() {
        _state.update { it.copy(vpnBusy = true, vpnDesired = false) }
        VpnController.stop(app)
    }

    fun onVpnPermissionDenied() {
        _state.update {
            it.copy(
                vpnBusy = false,
                vpnDesired = false,
                snackbar = str(R.string.vpn_permission_denied),
            )
        }
    }

    fun pingGroup(subscriptionId: String?) {
        val targets = _state.value.servers.filter { it.subscriptionId == subscriptionId }
        pingTargets(if (subscriptionId == null) PING_MANUAL else subscriptionId, targets, subscriptionId)
    }

    fun pingServer(id: String) {
        val server = _state.value.servers.firstOrNull { it.id == id } ?: return
        pingTargets("server:$id", listOf(server), server.subscriptionId)
    }

    private fun pingTargets(key: String, targets: List<ServerItem>, subscriptionId: String?) {
        val host = _state.value.settings.latencyTest.host.trim()
        if (host.isEmpty()) {
            snack(str(R.string.settings_latency_test_host_required))
            return
        }
        if (targets.isEmpty() || _state.value.pingingGroup != null) return
        val sortAfter = _state.value.settings.latencyTest.sortAfterPing
        _state.update { it.copy(pingingGroup = key) }
        val finished = AtomicInteger(0)
        val total = targets.size
        for (server in targets) {
            Engine.pingAsync(server.toProxyURL(), host, PingCallback { ms ->
                _state.update { state ->
                    state.copy(
                        servers = state.servers.map {
                            if (it.id == server.id) it.copy(latencyMs = ms.toInt()) else it
                        },
                    )
                }
                if (finished.incrementAndGet() == total) {
                    val latest = _state.value.servers
                    val next = if (sortAfter) {
                        val cmp = latencyComparator()
                        val manual = latest.filter { it.subscriptionId == null }
                        val subNodes = latest.filter { it.subscriptionId != null }
                        if (subscriptionId == null) manual.sortedWith(cmp) + subNodes else manual + subNodes.sortedWith(cmp)
                    } else {
                        latest
                    }
                    prefs.saveServers(next)
                    _state.update { it.copy(servers = next, pingingGroup = null) }
                }
            })
        }
    }

    fun parseSubscription(type: String, source: String) {
        val trimmed = source.trim()
        if (trimmed.isEmpty()) {
            snack(str(R.string.subscribe_error_url))
            return
        }
        viewModelScope.launch {
            _state.update { it.copy(importing = true) }
            runCatching {
                withContext(Dispatchers.IO) {
                    val isUrl = trimmed.startsWith("http://", ignoreCase = true) ||
                        trimmed.startsWith("https://", ignoreCase = true)
                    if (isUrl) fetchSubscription(type, trimmed)
                    else SubscriptionParser.parseContent(type, trimmed)
                }
            }.onSuccess { servers ->
                _state.update {
                    it.copy(
                        importing = false,
                        importPreview = servers,
                        importSource = str(R.string.subscribe_source_url),
                    )
                }
            }.onFailure { error ->
                _state.update {
                    it.copy(
                        importing = false,
                        importPreview = emptyList(),
                        importSource = null,
                        snackbar = errorMessage(error),
                    )
                }
            }
        }
    }

    fun updateSubscriptionInfo(type: String, url: String, remark: String) {
        val current = _state.value.subscription ?: return
        val next = current.copy(type = type.trim(), url = url.trim(), remark = remark.trim())
        prefs.saveSubscription(next)
        _state.update { it.copy(subscription = next, snackbar = str(R.string.subscribe_saved)) }
    }

    fun confirmImport(type: String, url: String, remark: String) {
        val preview = _state.value.importPreview
        if (preview.isEmpty()) {
            snack(str(R.string.subscribe_error_no_preview))
            return
        }
        saveSubscriptionNodes(
            Subscription(type = type.trim(), url = url.trim(), remark = remark.trim()),
            preview,
        )
    }

    fun refreshSubscription() {
        val sub = _state.value.subscription ?: return
        val source = sub.url.trim()
        if (source.isEmpty() || _state.value.importing) return
        viewModelScope.launch {
            _state.update { it.copy(importing = true) }
            runCatching {
                withContext(Dispatchers.IO) {
                    val isUrl = source.startsWith("http://", ignoreCase = true) ||
                        source.startsWith("https://", ignoreCase = true)
                    if (isUrl) fetchSubscription(sub.type, source)
                    else SubscriptionParser.parseContent(sub.type, source)
                }
            }.onSuccess { servers ->
                saveSubscriptionNodes(sub, servers)
            }.onFailure { error ->
                _state.update { it.copy(importing = false, snackbar = errorMessage(error)) }
            }
        }
    }

    fun deleteSubscription() {
        val id = _state.value.subscription?.id ?: return
        val selectedInSub = _state.value.selectedServer?.subscriptionId == id
        val remain = _state.value.servers.filter { it.subscriptionId != id }
        prefs.clearSubscription()
        prefs.saveServers(remain)
        _state.update {
            it.copy(
                servers = remain,
                subscription = null,
                importPreview = emptyList(),
                importSource = null,
            )
        }
        if (selectedInSub) selectServer(null)
    }

    private fun saveSubscriptionNodes(subscription: Subscription, preview: List<ServerItem>) {
        val tagged = preview.map { it.copy(subscriptionId = subscription.id) }
        val manual = _state.value.servers.filter { it.subscriptionId == null }
        val merged = manual + tagged
        val selectedStill = merged.firstOrNull { it.id == _state.value.selectedId }
        val shouldStop = selectedStill == null && _state.value.running
        prefs.saveSubscription(subscription)
        prefs.saveServers(merged)
        prefs.saveSelectedId(selectedStill?.id)
        if (shouldStop) stop()
        _state.update {
            it.copy(
                servers = merged,
                subscription = subscription,
                selectedId = selectedStill?.id,
                importing = false,
                importPreview = emptyList(),
                importSource = null,
                snackbar = str(R.string.subscribe_imported, preview.size),
            )
        }
        appendStatus(str(R.string.status_imported, preview.size))
    }

    fun importManualNodes(raw: String, onAdded: () -> Unit) {
        val text = raw.trim()
        if (text.isEmpty()) {
            snack(str(R.string.subscribe_error_empty))
            return
        }
        viewModelScope.launch {
            runCatching {
                withContext(Dispatchers.IO) {
                    if (text.startsWith("{")) listOf(SubscriptionParser.parseOne(text))
                    else SubscriptionParser.parseContent("prism", text)
                }
            }.onSuccess { incoming ->
                val current = _state.value.servers.toMutableList()
                var added = 0
                incoming.forEach { node ->
                    val id = SubscriptionParser.identity(node.host, node.protocol, node.username, node.password)
                    val exists = current.any {
                        SubscriptionParser.identity(it.host, it.protocol, it.username, it.password) == id
                    }
                    if (exists) return@forEach
                    current += node.copy(id = id, subscriptionId = null)
                    added++
                }
                if (added == 0) {
                    snack(str(R.string.server_list_duplicate))
                    return@onSuccess
                }
                prefs.saveServers(current)
                _state.update {
                    it.copy(servers = current, snackbar = str(R.string.server_list_imported, added))
                }
                onAdded()
            }.onFailure { error ->
                snack(errorMessage(error))
            }
        }
    }

    fun addOrUpdateServer(
        editingId: String?,
        host: String,
        remark: String,
        protocol: String,
        username: String,
        password: String,
    ): Boolean {
        val trimmedHost = host.trim()
        if (trimmedHost.isEmpty()) {
            snack(str(R.string.server_list_validate_host))
            return false
        }
        if (!HOST_PATTERN.matches(trimmedHost)) {
            snack(str(R.string.server_list_validate_host_format))
            return false
        }
        if (protocol.isBlank()) {
            snack(str(R.string.server_list_validate_protocol))
            return false
        }
        val current = _state.value.servers.toMutableList()
        val payload = ServerItem(
            id = editingId ?: UUID.randomUUID().toString(),
            host = trimmedHost,
            remark = remark.trim(),
            protocol = protocol,
            username = username.trim(),
            password = password,
            subscriptionId = editingId?.let { id ->
                current.firstOrNull { it.id == id }?.subscriptionId
            },
        )
        val nextId = SubscriptionParser.identity(
            payload.host, payload.protocol, payload.username, payload.password,
        )
        if (current.any {
                it.id != payload.id &&
                    SubscriptionParser.identity(it.host, it.protocol, it.username, it.password) == nextId
            }
        ) {
            snack(str(R.string.server_list_duplicate))
            return false
        }
        val idx = current.indexOfFirst { it.id == payload.id }
        val message: String
        if (idx >= 0) {
            current[idx] = payload.copy(latencyMs = current[idx].latencyMs)
            message = str(R.string.server_list_update_success)
        } else {
            current += payload
            message = str(R.string.server_list_add_success)
        }
        prefs.saveServers(current)
        _state.update { it.copy(servers = current, snackbar = message) }
        if (editingId != null && _state.value.selectedId == editingId) {
            selectServer(current.first { it.id == payload.id })
        }
        return true
    }

    fun deleteServer(id: String) {
        val remain = _state.value.servers.filterNot { it.id == id }
        prefs.saveServers(remain)
        if (_state.value.selectedId == id) selectServer(null)
        _state.update { it.copy(servers = remain, snackbar = str(R.string.server_list_delete_success)) }
    }

    fun deleteServers(ids: Set<String>) {
        val remain = _state.value.servers.filterNot { it.id in ids }
        prefs.saveServers(remain)
        if (_state.value.selectedId in ids) selectServer(null)
        _state.update { it.copy(servers = remain, snackbar = str(R.string.server_list_delete_success)) }
    }

    fun updateLatencySettings(transform: (LatencyTestSettings) -> LatencyTestSettings) {
        updateSettings { it.copy(latencyTest = transform(it.latencyTest)) }
    }

    fun updateSystemSettings(transform: (SystemSettings) -> SystemSettings) {
        val previous = _state.value.settings.system
        updateSettings { it.copy(system = transform(it.system)) }
        val next = _state.value.settings.system
        if (previous.language != next.language) applyAppLanguage(next.language)
    }

    fun clearStatusLogs() {
        _state.update { it.copy(statusLogs = emptyList()) }
    }

    fun clearConnectionLogs() {
        _state.update { it.copy(connectionLogs = emptyList()) }
    }

    fun updateRules(transform: (RuleDraft) -> RuleDraft) {
        _state.update { it.copy(rules = transform(it.rules)) }
        val rules = _state.value.rules
        prefs.saveRules(rules)
        VpnController.rules = rules
    }

    fun downloadDb() {
        val url = _state.value.rules.dbUrl.trim()
        if (url.isEmpty()) {
            snack(str(R.string.rules_db_url_required))
            return
        }
        runDbJob(R.string.rules_download_ok) {
            val dest = ipregionFile(app)
            val tmp = File(app.cacheDir, "ipregion-download.tmp")
            downloadIpregionDb(url, dest, tmp)
        }
    }

    fun importDb(uri: Uri) {
        runDbJob(R.string.rules_upload_ok) {
            val tmp = File(app.cacheDir, "ipregion-upload.tmp")
            app.contentResolver.openInputStream(uri)?.use { input ->
                tmp.outputStream().use { output -> input.copyTo(output) }
            } ?: error(str(R.string.rules_upload_empty))
            installIpregionDb(tmp, ipregionFile(app))
            tmp.delete()
        }
    }

    private fun observeVpn() {
        viewModelScope.launch {
            VpnController.running.collect { running ->
                _state.update {
                    it.copy(running = running, vpnBusy = false, vpnDesired = running)
                }
            }
        }
        viewModelScope.launch {
            VpnController.status.collect { line -> appendStatus(line) }
        }
        viewModelScope.launch {
            VpnController.connections.collect { line ->
                if (_state.value.settings.system.enableLogRecording) {
                    _state.update { it.copy(connectionLogs = (it.connectionLogs + line).takeLast(LOG_LIMIT)) }
                }
            }
        }
        viewModelScope.launch {
            while (true) {
                delay(1_000)
                if (_state.value.running) {
                    _state.update { it.copy(traffic = VpnController.trafficUi()) }
                }
            }
        }
    }

    private fun runDbJob(okRes: Int, block: () -> Unit) {
        if (_state.value.dbBusy) return
        viewModelScope.launch {
            _state.update { it.copy(dbBusy = true) }
            val error = withContext(Dispatchers.IO) {
                runCatching { block() }.exceptionOrNull()
            }
            if (error == null) {
                updateRules { it.copy(geoMode = GeoMode.Global, selectedAreaIds = emptySet()) }
            }
            withContext(Dispatchers.IO) { refreshEngineUi() }
            _state.update {
                it.copy(
                    dbBusy = false,
                    snackbar = error?.message?.takeIf { msg -> msg.isNotBlank() } ?: str(okRes),
                )
            }
        }
    }

    private fun updateSettings(transform: (AppSettings) -> AppSettings) {
        val next = transform(_state.value.settings)
        prefs.saveSettings(next)
        _state.update { it.copy(settings = next) }
    }

    private fun appendStatus(line: String) {
        _state.update { it.copy(statusLogs = (it.statusLogs + line).takeLast(LOG_LIMIT)) }
    }

    private fun refreshEngineUi() {
        val st = runCatching { Engine.dbStatus() }.getOrNull()
        val ready = st != null && st.exists
        val text = if (!ready) {
            str(R.string.rules_db_missing)
        } else {
            val ver = st.version.orEmpty().ifBlank { "-" }
            str(R.string.rules_db_status, ver, formatBytes(st.size), st.areas)
        }
        val areas = if (ready) loadAreaTree() else emptyList()
        _state.update { it.copy(dbStatus = text, dbReady = ready, geoAreas = areas) }
    }

    private fun fetchSubscription(type: String, url: String): List<ServerItem> {
        val userAgent = when {
            type.equals("shadowrocket", ignoreCase = true) -> "shadowrocket/*"
            type.equals("prism", ignoreCase = true) -> {
                val version = runCatching {
                    @Suppress("DEPRECATION")
                    app.packageManager.getPackageInfo(app.packageName, 0).versionName
                }.getOrNull().orEmpty().trim()
                if (version.isEmpty()) "prism/*" else "prism/$version"
            }
            else -> null
        }
        val body = try {
            httpGet(url, userAgent)
        } catch (error: SubscriptionException) {
            throw error
        } catch (_: Exception) {
            throw SubscriptionException(R.string.subscribe_error_fetch)
        }
        return SubscriptionParser.parseContent(type, body)
    }

    private fun httpGet(url: String, userAgent: String?): String {
        val uri = URI(url)
        val scheme = uri.scheme?.lowercase().orEmpty()
        if (scheme != "http" && scheme != "https") {
            throw SubscriptionException(R.string.subscribe_error_fetch)
        }
        val connection = (uri.toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 15_000
            instanceFollowRedirects = true
            if (!userAgent.isNullOrBlank()) {
                setRequestProperty("User-Agent", userAgent)
            }
        }
        try {
            val code = connection.responseCode
            val stream = if (code in 200..299) connection.inputStream else connection.errorStream
            val body = stream?.bufferedReader()?.use { it.readText() }.orEmpty()
            if (body.length > 2 * 1024 * 1024) {
                throw SubscriptionException(R.string.subscribe_error_fetch)
            }
            if (code !in 200..299) {
                throw SubscriptionException(R.string.subscribe_error_http, code)
            }
            if (body.isBlank()) {
                throw SubscriptionException(R.string.subscribe_error_empty)
            }
            return body
        } finally {
            connection.disconnect()
        }
    }

    private fun str(resId: Int, vararg args: Any): String = app.getString(resId, *args)

    private fun errorMessage(error: Throwable): String {
        return if (error is SubscriptionException) {
            str(error.messageRes, *error.formatArgs)
        } else {
            error.message?.takeIf { it.isNotBlank() } ?: str(R.string.subscribe_error_unknown)
        }
    }

    private fun latencyComparator(): Comparator<ServerItem> = Comparator { a, b ->
        fun rank(item: ServerItem): Pair<Int, Int> {
            val latency = item.latencyMs
            return when {
                latency == null -> 2 to Int.MAX_VALUE
                latency < 0 -> 1 to Int.MAX_VALUE
                else -> 0 to latency
            }
        }
        val ra = rank(a)
        val rb = rank(b)
        if (ra.first != rb.first) ra.first - rb.first else ra.second - rb.second
    }
}
