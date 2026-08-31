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
import com.penndev.prism.vpn.VpnController
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

private const val LOG_LIMIT = 1000

data class PrismUiState(
    val settings: AppSettings = AppSettings(),
    val servers: List<ServerItem> = emptyList(),
    val selectedId: String? = null,
    val running: Boolean = false,
    val vpnBusy: Boolean = false,
    val vpnDesired: Boolean = false,
    val pingingAll: Boolean = false,
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
        _state.value = PrismUiState(
            settings = settings,
            servers = servers,
            selectedId = prefs.loadSelectedId(),
            rules = prefs.loadRules(),
        )
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
        _state.update { it.copy(selectedId = server?.id) }
        prefs.saveSelectedId(server?.id)
        if (server == null) {
            if (_state.value.running) stop()
            appendStatus(str(R.string.status_node_cleared))
        } else {
            appendStatus(str(R.string.status_node_selected, server.displayName))
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

    fun pingAll() {
        val host = _state.value.settings.latencyTest.host.trim()
        if (host.isEmpty()) {
            snack(str(R.string.settings_latency_test_host_required))
            return
        }
        if (_state.value.servers.isEmpty() || _state.value.pingingAll) return
        viewModelScope.launch {
            _state.update { it.copy(pingingAll = true) }
            coroutineScope {
                _state.value.servers.map { server ->
                    async {
                        val latency = withContext(Dispatchers.IO) {
                            runCatching {
                                Engine.ping(server.toProxyURL(), host).toInt()
                            }.getOrDefault(-1)
                        }
                        _state.update { state ->
                            state.copy(
                                servers = state.servers.map {
                                    if (it.id == server.id) it.copy(latencyMs = latency) else it
                                },
                            )
                        }
                    }
                }.awaitAll()
            }
            val current = _state.value.servers
            val sorted = if (_state.value.settings.latencyTest.sortAfterPing) {
                current.sortedWith(latencyComparator())
            } else {
                current
            }
            prefs.saveServers(sorted)
            _state.update {
                it.copy(servers = sorted, pingingAll = false, snackbar = str(R.string.server_list_ping_all_done))
            }
        }
    }

    fun parseSubscription(type: String, source: String, fromFile: Boolean = false) {
        val trimmed = source.trim()
        if (trimmed.isEmpty()) {
            snack(str(R.string.subscribe_error_url))
            return
        }
        viewModelScope.launch {
            _state.update { it.copy(importing = true) }
            runCatching {
                withContext(Dispatchers.IO) {
                    val isUrl = !fromFile && (
                        trimmed.startsWith("http://", ignoreCase = true) ||
                            trimmed.startsWith("https://", ignoreCase = true)
                    )
                    if (isUrl) fetchSubscription(type, trimmed)
                    else SubscriptionParser.parseContent(type, trimmed)
                }
            }.onSuccess { servers ->
                _state.update {
                    it.copy(
                        importing = false,
                        importPreview = servers,
                        importSource = str(
                            if (fromFile) R.string.subscribe_source_file else R.string.subscribe_source_url,
                        ),
                        snackbar = str(R.string.subscribe_parsed, servers.size),
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

    fun confirmImport() {
        val preview = _state.value.importPreview
        if (preview.isEmpty()) {
            snack(str(R.string.subscribe_error_no_preview))
            return
        }
        val selectedStill = preview.firstOrNull { it.id == _state.value.selectedId }
        val shouldStop = selectedStill == null && _state.value.running
        prefs.saveServers(preview)
        prefs.saveSelectedId(selectedStill?.id)
        if (shouldStop) stop()
        _state.update {
            it.copy(
                servers = preview,
                selectedId = selectedStill?.id,
                importPreview = emptyList(),
                importSource = null,
                snackbar = str(R.string.subscribe_imported, preview.size),
            )
        }
        appendStatus(str(R.string.status_imported, preview.size))
    }

    fun exportServers(): String = SubscriptionParser.exportJson(_state.value.servers)

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
        val payload = ServerItem(
            id = editingId ?: UUID.randomUUID().toString(),
            host = trimmedHost,
            remark = remark.trim(),
            protocol = protocol,
            username = username.trim(),
            password = password,
        )
        val current = _state.value.servers.toMutableList()
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
        _state.update { it.copy(dbStatus = text, dbReady = ready, geoAreas = loadAreaTree()) }
    }

    private fun fetchSubscription(type: String, url: String): List<ServerItem> {
        val body = try {
            httpGet(url)
        } catch (error: SubscriptionException) {
            throw error
        } catch (_: Exception) {
            throw SubscriptionException(R.string.subscribe_error_fetch)
        }
        return SubscriptionParser.parseContent(type, body)
    }

    private fun httpGet(url: String): String {
        val connection = (URI(url).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 15_000
            instanceFollowRedirects = true
            setRequestProperty("User-Agent", "Prism-Android/0.1")
        }
        try {
            val code = connection.responseCode
            val stream = if (code in 200..299) connection.inputStream else connection.errorStream
            val body = stream?.bufferedReader()?.readText().orEmpty()
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
