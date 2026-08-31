package com.penndev.prism.data

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject

class Prefs(context: Context) {
    private val prefs = context.applicationContext.getSharedPreferences("prism", Context.MODE_PRIVATE)

    fun loadSettings(): AppSettings {
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
                    language = system?.optString("language").orEmpty().ifBlank { LANGUAGE_SYSTEM },
                    themeMode = runCatching {
                        ThemeMode.valueOf(system?.optString("themeMode") ?: "System")
                    }.getOrDefault(ThemeMode.System),
                    enableLogRecording = system?.optBoolean("enableLogRecording", true) ?: true,
                ),
            )
        }.getOrDefault(AppSettings())
    }

    fun saveSettings(settings: AppSettings) {
        prefs.edit()
            .putString(
                KEY_SETTINGS,
                JSONObject()
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
                    .toString(),
            )
            .apply()
    }

    fun loadServers(): List<ServerItem> {
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

    fun saveServers(servers: List<ServerItem>) {
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

    fun loadSelectedId(): String? = prefs.getString(KEY_SELECTED, null)

    fun saveSelectedId(id: String?) {
        prefs.edit().putString(KEY_SELECTED, id).apply()
    }

    fun loadRules(): RuleDraft {
        val raw = prefs.getString(KEY_RULES, null) ?: return RuleDraft()
        return runCatching {
            val obj = JSONObject(raw)
            val areas = obj.optJSONArray("selectedAreaIds") ?: obj.optJSONArray("selectedAreas")
            RuleDraft(
                geoMode = runCatching {
                    GeoMode.valueOf(obj.optString("geoMode", "Global"))
                }.getOrDefault(GeoMode.Global),
                selectedAreaIds = buildSet {
                    if (areas != null) {
                        for (i in 0 until areas.length()) {
                            val id = areas.optLong(i, 0L)
                            if (id > 0L) add(id)
                        }
                    }
                },
                dbUrl = obj.optString("dbUrl"),
            )
        }.getOrDefault(RuleDraft())
    }

    fun saveRules(draft: RuleDraft) {
        val areas = JSONArray()
        draft.selectedAreaIds.forEach { areas.put(it) }
        prefs.edit()
            .putString(
                KEY_RULES,
                JSONObject()
                    .put("geoMode", draft.geoMode.name)
                    .put("selectedAreaIds", areas)
                    .put("dbUrl", draft.dbUrl)
                    .toString(),
            )
            .apply()
    }

    private companion object {
        const val KEY_SERVERS = "servers"
        const val KEY_SELECTED = "selected_id"
        const val KEY_SETTINGS = "settings"
        const val KEY_RULES = "rules"
    }
}
