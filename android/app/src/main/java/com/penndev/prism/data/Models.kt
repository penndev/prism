package com.penndev.prism.data

data class ServerItem(
    val id: String,
    val host: String,
    val remark: String = "",
    val username: String = "",
    val password: String = "",
    val protocol: String = "socks5",
    val latencyMs: Int? = null,
) {
    val displayName: String get() = remark.ifBlank { host }

    fun toProxyURL(): String {
        val scheme = protocol.lowercase()
        val userinfo = if (username.isEmpty() && password.isEmpty()) {
            ""
        } else {
            "${encodeUserinfo(username)}:${encodeUserinfo(password)}@"
        }
        return "$scheme://$userinfo$host"
    }
}

enum class ThemeMode { System, Light, Dark }

enum class GeoMode { Global, None, Proxy, Bypass }

data class LatencyTestSettings(
    val host: String = "google.com",
    val sortAfterPing: Boolean = true,
)

const val LANGUAGE_SYSTEM = "system"

data class SystemSettings(
    val language: String = LANGUAGE_SYSTEM,
    val themeMode: ThemeMode = ThemeMode.System,
    val enableLogRecording: Boolean = true,
)

data class AppSettings(
    val latencyTest: LatencyTestSettings = LatencyTestSettings(),
    val system: SystemSettings = SystemSettings(),
)

data class AreaUi(
    val id: Long,
    val parentId: Long,
    val name: String,
    val children: List<AreaUi> = emptyList(),
)

data class RuleDraft(
    val geoMode: GeoMode = GeoMode.Global,
    val selectedAreaIds: Set<Long> = emptySet(),
    val dbUrl: String = "",
)

data class TrafficUi(
    val downSpeed: String = "0 B/s",
    val upSpeed: String = "0 B/s",
    val downTotal: String = "0 B",
    val upTotal: String = "0 B",
)

fun formatBytes(bytes: Long): String {
    if (bytes < 1024) return "$bytes B"
    val units = arrayOf("KB", "MB", "GB", "TB")
    var value = bytes.toDouble() / 1024
    var index = 0
    while (value >= 1024 && index < units.lastIndex) {
        value /= 1024
        index++
    }
    return "%.1f %s".format(value, units[index])
}

val PROXY_SCHEMES = listOf("socks5", "socks5s", "http", "https")

val HOST_PATTERN = Regex("""^(\[[^\]]+]|[^:\[\]]+):\d{1,5}$""")

private fun encodeUserinfo(value: String): String {
    return java.net.URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
}

