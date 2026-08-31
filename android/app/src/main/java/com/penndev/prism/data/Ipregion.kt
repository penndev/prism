package com.penndev.prism.data

import android.content.Context
import com.penndev.prism.engine.Engine
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import org.json.JSONArray
import org.json.JSONObject

const val IPREGION_DB = "ipregion.db"

fun ipregionFile(context: Context): File =
    File(context.applicationContext.filesDir, IPREGION_DB)

fun openIpregionDb(file: File) {
    if (file.exists()) {
        runCatching { Engine.setIpregionDB(file.absolutePath) }
    }
}

fun loadAreaTree(): List<AreaUi> {
    val raw = runCatching { Engine.areaTree() }.getOrNull().orEmpty()
    if (raw.isBlank() || raw == "[]") return emptyList()
    return runCatching { parseAreaTree(JSONArray(raw)) }.getOrDefault(emptyList())
}

fun installIpregionDb(src: File, dest: File) {
    Engine.setIpregionDB(src.absolutePath)
    if (src.canonicalPath != dest.canonicalPath) {
        src.copyTo(dest, overwrite = true)
        Engine.setIpregionDB(dest.absolutePath)
    }
}

fun downloadIpregionDb(url: String, dest: File, tmp: File) {
    tmp.delete()
    httpDownload(url, tmp)
    installIpregionDb(tmp, dest)
    tmp.delete()
}

private fun parseAreaTree(arr: JSONArray): List<AreaUi> = buildList {
    for (i in 0 until arr.length()) {
        val obj = arr.optJSONObject(i) ?: continue
        add(parseArea(obj))
    }
}

private fun parseArea(obj: JSONObject): AreaUi {
    val kids = obj.optJSONArray("children")
    val children = if (kids == null) emptyList() else parseAreaTree(kids)
    return AreaUi(
        id = obj.optLong("id"),
        parentId = obj.optLong("parent_id"),
        name = obj.optString("name"),
        children = children,
    )
}

private fun httpDownload(url: String, dest: File) {
    val connection = (URI(url).toURL().openConnection() as HttpURLConnection).apply {
        connectTimeout = 15_000
        readTimeout = 5 * 60_000
        instanceFollowRedirects = true
        setRequestProperty("User-Agent", "Prism-Android/0.1")
    }
    try {
        val code = connection.responseCode
        if (code !in 200..299) error("HTTP $code")
        dest.outputStream().use { output ->
            connection.inputStream.use { input -> input.copyTo(output) }
        }
    } finally {
        connection.disconnect()
    }
}
