package com.penndev.prism.ui

import android.app.Application
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.os.LocaleListCompat
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.penndev.prism.PrismApplication
import com.penndev.prism.R
import com.penndev.prism.data.AppSettings
import com.penndev.prism.data.HOST_PATTERN
import com.penndev.prism.data.LatencyTestSettings
import com.penndev.prism.data.PrismRepository
import com.penndev.prism.data.RuleDraft
import com.penndev.prism.data.ServerItem
import com.penndev.prism.data.SubscriptionException
import com.penndev.prism.data.SystemSettings
import com.penndev.prism.data.TrafficUi
import com.penndev.prism.vpn.VpnController
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

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
    val snackbar: String? = null,
) {
    val selectedServer: ServerItem?
        get() = servers.firstOrNull { it.id == selectedId }
}

class PrismViewModel(application: Application) : AndroidViewModel(application) {
    private val repository: PrismRepository = (application as PrismApplication).repository

    private val _state = MutableStateFlow(PrismUiState())
    val state: StateFlow<PrismUiState> = _state.asStateFlow()

    init {
        val settings = repository.loadSettings()
        val servers = repository.loadServers()
        _state.value = PrismUiState(
            settings = settings,
            servers = servers,
            selectedId = repository.loadSelectedServerId(),
            rules = repository.loadRules(),
        )
        applyLanguage(settings.system.language)
        appendStatus(appString(R.string.status_ready, servers.size))
        observeVpn()
    }

    fun consumeSnackbar() {
        _state.update { it.copy(snackbar = null) }
    }

    fun selectServer(server: ServerItem?) {
        _state.update { it.copy(selectedId = server?.id) }
        repository.saveSelectedServer(server)
        repository.setRemote(server)
        if (server == null) {
            if (_state.value.running) stopVpn()
            appendStatus(appString(R.string.status_node_cleared))
        } else {
            appendStatus(appString(R.string.status_node_selected, server.displayName))
        }
    }

    fun toggleRunning(needNodeMessage: String) {
        if (_state.value.vpnBusy) return
        if (_state.value.running || _state.value.vpnDesired) {
            stopVpn()
            return
        }
        if (_state.value.selectedServer == null) {
            _state.update { it.copy(snackbar = needNodeMessage) }
            return
        }
        beginStart()
        startVpn()
    }

    fun beginStart(): Boolean {
        if (_state.value.vpnBusy) return false
        if (_state.value.selectedServer == null) return false
        _state.update { it.copy(vpnBusy = true, vpnDesired = true) }
        return true
    }

    fun startVpn() {
        val server = _state.value.selectedServer ?: return
        if (!_state.value.vpnBusy) {
            _state.update { it.copy(vpnBusy = true, vpnDesired = true) }
        }
        repository.startProxy(server)
        appendStatus(appString(R.string.status_vpn_starting, server.displayName))
    }

    fun stopVpn() {
        _state.update { it.copy(vpnBusy = true, vpnDesired = false) }
        repository.stopProxy()
    }

    fun onVpnPermissionDenied() {
        _state.update {
            it.copy(
                vpnBusy = false,
                vpnDesired = false,
                snackbar = appString(R.string.vpn_permission_denied),
            )
        }
    }

    fun pingAll(hostRequiredMessage: String, doneMessage: String) {
        val host = _state.value.settings.latencyTest.host.trim()
        if (host.isEmpty()) {
            _state.update { it.copy(snackbar = hostRequiredMessage) }
            return
        }
        if (_state.value.servers.isEmpty() || _state.value.pingingAll) return
        viewModelScope.launch {
            _state.update { it.copy(pingingAll = true) }
            val pinged = _state.value.servers.map { server ->
                server.copy(latencyMs = repository.pingServer(server, host))
            }
            val sorted = if (_state.value.settings.latencyTest.sortAfterPing) {
                pinged.sortedWith(latencyComparator())
            } else {
                pinged
            }
            repository.saveServers(sorted)
            _state.update {
                it.copy(servers = sorted, pingingAll = false, snackbar = doneMessage)
            }
        }
    }

    fun parseSubscription(type: String, source: String, fromFile: Boolean = false) {
        val trimmed = source.trim()
        if (trimmed.isEmpty()) {
            _state.update { it.copy(snackbar = appString(R.string.subscribe_error_url)) }
            return
        }
        viewModelScope.launch {
            _state.update { it.copy(importing = true) }
            runCatching {
                withContext(Dispatchers.IO) {
                    when {
                        fromFile -> repository.parseServerFile(trimmed)
                        trimmed.startsWith("http://", ignoreCase = true) ||
                            trimmed.startsWith("https://", ignoreCase = true) ->
                            repository.fetchSubscription(type, trimmed)
                        else -> repository.parseSubscriptionContent(type, trimmed)
                    }
                }
            }.onSuccess { servers ->
                _state.update {
                    it.copy(
                        importing = false,
                        importPreview = servers,
                        importSource = if (fromFile) {
                            appString(R.string.subscribe_source_file)
                        } else {
                            appString(R.string.subscribe_source_url)
                        },
                        snackbar = appString(R.string.subscribe_parsed, servers.size),
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

    fun confirmImport(successMessage: String) {
        val preview = _state.value.importPreview
        if (preview.isEmpty()) {
            _state.update { it.copy(snackbar = appString(R.string.subscribe_error_no_preview)) }
            return
        }
        val selectedStill = preview.firstOrNull { it.id == _state.value.selectedId }
        val shouldStop = selectedStill == null && _state.value.running
        repository.saveServers(preview)
        repository.saveSelectedServer(selectedStill)
        if (shouldStop) stopVpn()
        _state.update {
            it.copy(
                servers = preview,
                selectedId = selectedStill?.id,
                importPreview = emptyList(),
                importSource = null,
                snackbar = successMessage,
            )
        }
        appendStatus(appString(R.string.status_imported, preview.size))
    }

    fun clearImportPreview() {
        _state.update { it.copy(importPreview = emptyList(), importSource = null) }
    }

    fun exportServers(): String = repository.exportServers(_state.value.servers)

    fun addOrUpdateServer(
        editingId: String?,
        host: String,
        remark: String,
        protocol: String,
        username: String,
        password: String,
        hostRequired: String,
        hostFormat: String,
        protocolRequired: String,
        addSuccess: String,
        updateSuccess: String,
    ): Boolean {
        val trimmedHost = host.trim()
        if (trimmedHost.isEmpty()) {
            _state.update { it.copy(snackbar = hostRequired) }
            return false
        }
        if (!HOST_PATTERN.matches(trimmedHost)) {
            _state.update { it.copy(snackbar = hostFormat) }
            return false
        }
        if (protocol.isBlank()) {
            _state.update { it.copy(snackbar = protocolRequired) }
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
            message = updateSuccess
        } else {
            current += payload
            message = addSuccess
        }
        repository.saveServers(current)
        _state.update { it.copy(servers = current, snackbar = message) }
        if (editingId != null && _state.value.selectedId == editingId) {
            selectServer(current.first { it.id == payload.id })
        }
        return true
    }

    fun deleteServer(id: String, successMessage: String) {
        val remain = _state.value.servers.filterNot { it.id == id }
        repository.saveServers(remain)
        if (_state.value.selectedId == id) {
            selectServer(null)
        }
        _state.update { it.copy(servers = remain, snackbar = successMessage) }
    }

    fun updateLatencySettings(transform: (LatencyTestSettings) -> LatencyTestSettings) {
        updateSettings { it.copy(latencyTest = transform(it.latencyTest)) }
    }

    fun updateSystemSettings(transform: (SystemSettings) -> SystemSettings) {
        val previous = _state.value.settings.system
        updateSettings { it.copy(system = transform(it.system)) }
        val next = _state.value.settings.system
        if (previous.language != next.language) {
            applyLanguage(next.language)
        }
    }

    fun clearStatusLogs() {
        _state.update { it.copy(statusLogs = emptyList()) }
    }

    fun clearConnectionLogs() {
        _state.update { it.copy(connectionLogs = emptyList()) }
    }

    fun appendConnection(line: String) {
        _state.update { it.copy(connectionLogs = (it.connectionLogs + line).takeLast(1000)) }
    }

    fun updateRules(transform: (RuleDraft) -> RuleDraft) {
        _state.update { it.copy(rules = transform(it.rules)) }
    }

    fun saveRules(successMessage: String) {
        repository.saveRules(_state.value.rules)
        _state.update { it.copy(snackbar = successMessage) }
    }

    fun notifyPending(message: String) {
        _state.update { it.copy(snackbar = message) }
    }

    fun deleteServers(ids: Set<String>, successMessage: String) {
        val remain = _state.value.servers.filterNot { it.id in ids }
        repository.saveServers(remain)
        if (_state.value.selectedId in ids) {
            selectServer(null)
        }
        _state.update { it.copy(servers = remain, snackbar = successMessage) }
    }

    private fun observeVpn() {
        viewModelScope.launch {
            VpnController.running.collect { running ->
                _state.update { it.copy(running = running) }
            }
        }
        viewModelScope.launch {
            VpnController.settled.collect { running ->
                _state.update {
                    it.copy(running = running, vpnBusy = false, vpnDesired = running)
                }
            }
        }
        viewModelScope.launch {
            VpnController.status.collect { line ->
                appendStatus(line)
            }
        }
        viewModelScope.launch {
            VpnController.connections.collect { line ->
                if (_state.value.settings.system.enableLogRecording) {
                    appendConnection(line)
                }
            }
        }
        viewModelScope.launch {
            while (true) {
                kotlinx.coroutines.delay(1_000)
                if (_state.value.running) {
                    _state.update { it.copy(traffic = VpnController.trafficUi()) }
                }
            }
        }
    }

    private fun updateSettings(transform: (AppSettings) -> AppSettings) {
        val next = transform(_state.value.settings)
        repository.saveSettings(next)
        _state.update { it.copy(settings = next) }
    }

    private fun appendStatus(line: String) {
        _state.update { it.copy(statusLogs = it.statusLogs + line) }
    }

    private fun applyLanguage(tag: String) {
        val locales = LocaleListCompat.forLanguageTags(tag)
        if (AppCompatDelegate.getApplicationLocales().toLanguageTags() != locales.toLanguageTags()) {
            AppCompatDelegate.setApplicationLocales(locales)
        }
    }

    private fun appString(resId: Int, vararg args: Any): String {
        return getApplication<Application>().getString(resId, *args)
    }

    private fun errorMessage(error: Throwable): String {
        return if (error is SubscriptionException) {
            appString(error.messageRes, *error.formatArgs)
        } else {
            error.message?.takeIf { it.isNotBlank() }
                ?: appString(R.string.subscribe_error_unknown)
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
