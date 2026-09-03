package com.penndev.prism.data

import java.net.URI
import java.util.Base64
import org.json.JSONArray
import org.json.JSONObject

class SubscriptionException(
    val messageRes: Int,
    vararg val formatArgs: Any,
) : Exception()

object SubscriptionParser {
    fun parseContent(type: String, raw: String): List<ServerItem> {
        val text = raw.trim()
        if (text.isEmpty()) {
            throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_empty)
        }
        if (text.startsWith("[")) {
            return serversFromJson(text)
        }
        return when (type.lowercase()) {
            "prism" -> parsePrism(text)
            "shadowrocket" -> parseShadowrocket(text)
            else -> throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_type)
        }
    }

    fun exportJson(servers: List<ServerItem>): String {
        val arr = JSONArray()
        servers.forEach { server ->
            arr.put(
                JSONObject()
                    .put("host", server.host)
                    .put("remark", server.remark)
                    .put("username", server.username)
                    .put("password", server.password)
                    .put("protocol", server.protocol),
            )
        }
        return arr.toString(2)
    }

    fun normalizeProtocol(raw: String): String {
        val key = raw.trim().lowercase()
        return if (key in PROXY_SCHEMES) key else "socks5"
    }

    fun identity(host: String, protocol: String, username: String, password: String): String {
        val scheme = normalizeProtocol(protocol).lowercase()
        return "$scheme://$username:$password@$host"
    }

    private fun parsePrism(text: String): List<ServerItem> {
        val json = if (text.startsWith("[")) text else decodeMaybeBase64(text)
        return serversFromJson(json)
    }

    private fun parseShadowrocket(text: String): List<ServerItem> {
        val payload = decodeMaybeBase64(text)
        val seen = mutableSetOf<String>()
        val servers = payload
            .replace("\r\n", "\n")
            .split('\n')
            .mapNotNull { parseShadowrocketLine(it) }
            .filter { seen.add(it.id) }
        if (servers.isEmpty()) {
            throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_no_nodes)
        }
        return servers
    }

    private fun parseShadowrocketLine(raw: String): ServerItem? {
        val uri = parseNodeUri(raw.trim()) ?: return null
        val scheme = uri.scheme?.lowercase().orEmpty()
        val protocol = when (scheme) {
            "https" -> "https"
            "socks5s" -> "socks5s"
            // Shadowrocket 的 socks5:// 是明文 SOCKS5；带 TLS 的才是 socks5s://
            "socks5", "socks" -> "socks5"
            else -> return null
        }
        val hostname = uri.host ?: return null
        if (uri.port <= 0) return null
        val host = if (hostname.contains(':')) "[$hostname]:${uri.port}" else "$hostname:${uri.port}"
        if (!HOST_PATTERN.matches(host)) return null
        val userInfo = uri.userInfo.orEmpty()
        val username = userInfo.substringBefore(':', missingDelimiterValue = userInfo)
        val password = if (userInfo.contains(':')) userInfo.substringAfter(':') else ""
        val remark = uri.fragment?.trim().orEmpty().ifBlank { host }
        return ServerItem(
            id = identity(host, protocol, username, password),
            host = host,
            remark = remark,
            username = username,
            password = password,
            protocol = protocol,
        )
    }

    private fun parseNodeUri(line: String): URI? {
        if (line.isEmpty() || !line.contains("://")) return null
        val scheme = line.substringBefore("://")
        val payload = line.substringAfter("://")
        if (scheme.isBlank() || payload.isBlank()) return runCatching { URI(line) }.getOrNull()
        val decoded = runCatching { decodeBase64Bytes(payload) }.getOrNull()
        val rebuilt = if (decoded != null) {
            "$scheme://${String(decoded, Charsets.UTF_8).trim()}"
        } else {
            line
        }
        return runCatching { URI(rebuilt) }.getOrNull()
    }

    private fun serversFromJson(json: String): List<ServerItem> {
        val arr = try {
            JSONArray(json)
        } catch (_: Exception) {
            throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_json)
        }
        val servers = buildList {
            val seen = mutableSetOf<String>()
            for (i in 0 until arr.length()) {
                val obj = arr.optJSONObject(i) ?: continue
                val host = obj.optString("host").trim()
                if (host.isEmpty()) continue
                val protocol = normalizeProtocol(obj.optString("protocol", "socks5"))
                val username = obj.optString("username")
                val password = obj.optString("password")
                val id = identity(host, protocol, username, password)
                if (!seen.add(id)) continue
                add(
                    ServerItem(
                        id = id,
                        host = host,
                        remark = obj.optString("remark"),
                        username = username,
                        password = password,
                        protocol = protocol,
                    ),
                )
            }
        }
        if (servers.isEmpty()) {
            throw SubscriptionException(com.penndev.prism.R.string.subscribe_error_no_nodes)
        }
        return servers
    }

    private fun decodeMaybeBase64(text: String): String {
        return runCatching {
            String(decodeBase64Bytes(text), Charsets.UTF_8)
        }.getOrElse { text }
    }

    private fun decodeBase64Bytes(text: String): ByteArray {
        val compact = text.replace("\\s".toRegex(), "")
        val padded = compact + "=".repeat((4 - compact.length % 4) % 4)
        return try {
            Base64.getDecoder().decode(padded)
        } catch (_: Exception) {
            Base64.getUrlDecoder().decode(padded)
        }
    }
}
