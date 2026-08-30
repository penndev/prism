package com.penndev.prism.data

data class ServerItem(
    val id: String,
    val host: String,
    val remark: String = "",
    val username: String = "",
    val password: String = "",
    val protocol: String = "Socks5",
    val latencyMs: Int? = null,
) {
    val displayName: String get() = remark.ifBlank { host }
}

enum class ThemeMode { System, Light, Dark }

enum class GeoMode { Global, None, Proxy, Bypass }

data class LatencyTestSettings(
    val host: String = "google.com",
    val sortAfterPing: Boolean = true,
)

data class SystemSettings(
    val language: String = "zh-CN",
    val themeMode: ThemeMode = ThemeMode.System,
    val enableLogRecording: Boolean = true,
)

data class AppSettings(
    val latencyTest: LatencyTestSettings = LatencyTestSettings(),
    val system: SystemSettings = SystemSettings(),
)

data class RuleDraft(
    val geoMode: GeoMode = GeoMode.Global,
    val selectedAreas: Set<String> = emptySet(),
    val domains: String = "",
)

data class TrafficUi(
    val downSpeed: String = "0 B/s",
    val upSpeed: String = "0 B/s",
    val downTotal: String = "0 B",
    val upTotal: String = "0 B",
)

val PROXY_SCHEMES = listOf("Socks5", "Socks5OverTLS", "Http", "HttpOverTLS")

val HOST_PATTERN = Regex("""^(\[[^\]]+]|[^:\[\]]+):\d{1,5}$""")

val DEMO_AREAS = listOf("CN", "US", "JP", "HK", "TW", "SG", "KR", "GB", "DE", "AU")
