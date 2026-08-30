package com.penndev.prism.data

import android.content.Context
import com.penndev.prism.vpn.VpnController
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import kotlin.random.Random

class LocalPrismRepository(context: Context) : PrismRepository {
    private val context = context.applicationContext
    private val prefs = this.context.getSharedPreferences("prism", Context.MODE_PRIVATE)

    private var settings: AppSettings = loadSettingsLocked()
    private var servers: List<ServerItem> = loadServersLocked()
    private var selectedId: String? = prefs.getString(KEY_SELECTED, null)
    private var rules: RuleDraft = loadRulesLocked()

    override fun loadSettings(): AppSettings = settings

    override fun saveSettings(settings: AppSettings) {
        this.settings = settings
        prefs.edit()
            .putString(KEY_SETTINGS, encodeSettings(settings))
            .apply()
    }

    override fun loadServers(): List<ServerItem> = servers

    override fun saveServers(servers: List<ServerItem>) {
        this.servers = servers
        val arr = JSONArray()
        servers.forEach { server ->
            arr.put(
                JSONObject()
                    .put("id", server.id)
                    .put("host", server.host)
                    .put("remark", server.remark)
                    .put("username", server.username)
                    .put("password", server.password)
                    .put("protocol", server.protocol)
                    .put("latencyMs", server.latencyMs ?: JSONObject.NULL),
            )
        }
        prefs.edit().putString(KEY_SERVERS, arr.toString()).apply()
    }

    override fun loadSelectedServerId(): String? = selectedId

    override fun saveSelectedServer(server: ServerItem?) {
        selectedId = server?.id
        prefs.edit().putString(KEY_SELECTED, selectedId).apply()
    }

    override fun setRemote(server: ServerItem?) = Unit

    override fun startProxy(server: ServerItem) {
        VpnController.start(context)
    }

    override fun stopProxy() {
        VpnController.stop(context)
    }

    override suspend fun pingServer(server: ServerItem, latencyHost: String): Int {
        delay(400 + Random.nextLong(400))
        return if (Random.nextInt(10) == 0) -1 else Random.nextInt(40, 420)
    }

    override fun trafficSnapshot(): Pair<Long, Long> {
        return VpnController.uploadBytes.get() to VpnController.downloadBytes.get()
    }

    override fun saveRules(draft: RuleDraft) {
        rules = draft
        prefs.edit()
            .putString(KEY_RULES, encodeRules(draft))
            .apply()
    }

    override fun loadRules(): RuleDraft = rules

    override suspend fun fetchSubscription(type: String, url: String): List<ServerItem> {
        val body = withContext(Dispatchers.IO) {
            try {
                httpGet(url)
            } catch (error: SubscriptionException) {
                throw error
            } catch (_: Exception) {
                throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_fetch)
            }
        }
        return SubscriptionParser.parseContent(type, body)
    }

    override fun parseSubscriptionContent(type: String, text: String): List<ServerItem> {
        return SubscriptionParser.parseContent(type, text)
    }

    override fun parseServerFile(text: String): List<ServerItem> {
        return SubscriptionParser.parseJsonFile(text)
    }

    override fun exportServers(servers: List<ServerItem>): String {
        return SubscriptionParser.exportJson(servers)
    }

    private fun httpGet(url: String): String {
        val connection = (URI(url).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 15_000
            readTimeout = 15_000
            instanceFollowRedirects = true
            setRequestProperty("User-Agent", "Prism/0.1")
        }
        try {
            val code = connection.responseCode
            val stream = if (code in 200..299) connection.inputStream else connection.errorStream
            val body = stream?.bufferedReader()?.readText().orEmpty()
            if (code !in 200..299) {
                throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_http, code)
            }
            if (body.isBlank()) {
                throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_empty)
            }
            return body
        } finally {
            connection.disconnect()
        }
    }

    private fun loadServersLocked(): List<ServerItem> {
        val raw = prefs.getString(KEY_SERVERS, null) ?: return emptyList()
        return runCatching {
            val arr = JSONArray(raw)
            buildList {
                for (i in 0 until arr.length()) {
                    val obj = arr.optJSONObject(i) ?: continue
                    val host = obj.optString("host")
                    val protocol = SubscriptionParser.normalizeProtocol(obj.optString("protocol", "Socks5"))
                    val username = obj.optString("username")
                    val password = obj.optString("password")
                    val storedId = obj.optString("id")
                    add(
                        ServerItem(
                            id = storedId.ifBlank {
                                SubscriptionParser.identity(host, protocol, username, password)
                            },
                            host = host,
                            remark = obj.optString("remark"),
                            username = username,
                            password = password,
                            protocol = protocol,
                            latencyMs = if (obj.isNull("latencyMs")) null else obj.optInt("latencyMs"),
                        ),
                    )
                }
            }
        }.getOrDefault(emptyList())
    }

    private fun loadSettingsLocked(): AppSettings {
        val raw = prefs.getString(KEY_SETTINGS, null) ?: return AppSettings()
        return runCatching {
            val obj = JSONObject(raw)
            val latency = obj.optJSONObject("latencyTest")
            val system = obj.optJSONObject("system")
            AppSettings(
                latencyTest = LatencyTestSettings(
                    host = latency?.optString("host") ?: "google.com",
                    sortAfterPing = latency?.optBoolean("sortAfterPing", true) ?: true,
                ),
                system = SystemSettings(
                    language = system?.optString("language") ?: "zh-CN",
                    themeMode = runCatching {
                        ThemeMode.valueOf(system?.optString("themeMode") ?: "System")
                    }.getOrDefault(ThemeMode.System),
                    enableLogRecording = system?.optBoolean("enableLogRecording", true) ?: true,
                ),
            )
        }.getOrDefault(AppSettings())
    }

    private fun encodeSettings(settings: AppSettings): String {
        return JSONObject()
            .put(
                "latencyTest",
                JSONObject()
                    .put("host", settings.latencyTest.host)
                    .put("sortAfterPing", settings.latencyTest.sortAfterPing),
            )
            .put(
                "system",
                JSONObject()
                    .put("language", settings.system.language)
                    .put("themeMode", settings.system.themeMode.name)
                    .put("enableLogRecording", settings.system.enableLogRecording),
            )
            .toString()
    }

    private fun loadRulesLocked(): RuleDraft {
        val raw = prefs.getString(KEY_RULES, null) ?: return RuleDraft()
        return runCatching {
            val obj = JSONObject(raw)
            val areas = obj.optJSONArray("selectedAreas")
            RuleDraft(
                geoMode = runCatching {
                    GeoMode.valueOf(obj.optString("geoMode", "Global"))
                }.getOrDefault(GeoMode.Global),
                selectedAreas = buildSet {
                    if (areas != null) {
                        for (i in 0 until areas.length()) add(areas.optString(i))
                    }
                },
                domains = obj.optString("domains"),
            )
        }.getOrDefault(RuleDraft())
    }

    private fun encodeRules(draft: RuleDraft): String {
        val areas = JSONArray()
        draft.selectedAreas.forEach { areas.put(it) }
        return JSONObject()
            .put("geoMode", draft.geoMode.name)
            .put("selectedAreas", areas)
            .put("domains", draft.domains)
            .toString()
    }

    companion object {
        private const val KEY_SERVERS = "servers"
        private const val KEY_SELECTED = "selected_id"
        private const val KEY_SETTINGS = "settings"
        private const val KEY_RULES = "rules"
    }
}
