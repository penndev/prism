package com.penndev.prism.data

import android.content.Context
import com.penndev.prism.engine.Engine
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import org.json.JSONArray

const val IPREGION_DB = "ipregion.db"

@Volatile
private var ipregionPath = ""

data class IpregionStatus(
    val exists: Boolean = false,
    val size: Long = 0,
    val version: String = "",
    val areas: Int = 0,
)

fun ipregionFile(context: Context): File =
    File(context.applicationContext.filesDir, IPREGION_DB)

// engine 以文件模式打开：段表解压到临时目录（约 30MB 磁盘），内存只留索引。
// java.io.tmpdir 由系统设成应用的 cacheDir；Go 默认的 /data/local/tmp 应用没有写权限。
fun openIpregionDb(file: File) {
    if (!file.exists()) return
    runCatching { Engine.setIpregionDB(file.absolutePath, System.getProperty("java.io.tmpdir")) }
        .onSuccess { ipregionPath = file.absolutePath }
}

fun ipregionStatus(): IpregionStatus {
    val st = Engine.ipregionStatus() ?: return IpregionStatus()
    val file = File(ipregionPath)
    val exists = file.exists()
    return IpregionStatus(
        exists = exists,
        size = if (exists) file.length() else 0,
        version = st.version,
        areas = st.areas.toInt(),
    )
}

fun loadAreaTree(): List<AreaUi> {
    fun build(arr: JSONArray): List<AreaUi> = List(arr.length()) { i ->
        val obj = arr.getJSONObject(i)
        AreaUi(
            id = obj.optLong("id"),
            parentId = obj.optLong("parent_id"),
            name = obj.optString("name"),
            children = build(obj.optJSONArray("children") ?: JSONArray()),
        )
    }
    return runCatching { build(JSONArray(Engine.areaTree())) }.getOrDefault(emptyList())
}

fun ipregionLookupIds(address: String): List<Long> {
    val list = Engine.lookup(address) ?: return emptyList()
    return List(list.len().toInt()) { list.get(it.toLong()) }
}

fun installIpregionDb(src: File, dest: File) {
    if (src.canonicalPath != dest.canonicalPath) {
        src.copyTo(dest, overwrite = true)
    }
    Engine.setIpregionDB(dest.absolutePath, System.getProperty("java.io.tmpdir"))
    ipregionPath = dest.absolutePath
}

fun downloadIpregionDb(url: String, dest: File, tmp: File) {
    tmp.delete()
    httpDownload(url, tmp)
    installIpregionDb(tmp, dest)
    tmp.delete()
}

private fun httpDownload(url: String, dest: File) {
    val uri = URI(url)
    val scheme = uri.scheme?.lowercase().orEmpty()
    if (scheme != "http" && scheme != "https") error("url must be http or https")
    val connection = (uri.toURL().openConnection() as HttpURLConnection).apply {
        connectTimeout = 15_000
        readTimeout = 5 * 60_000
        instanceFollowRedirects = true
        setRequestProperty("User-Agent", "Prism-Android/0.1")
    }
    try {
        val code = connection.responseCode
        if (code !in 200..299) error("HTTP $code")
        dest.outputStream().use { output ->
            connection.inputStream.use { input ->
                val buf = ByteArray(8 * 1024)
                var total = 0L
                while (true) {
                    val n = input.read(buf)
                    if (n < 0) break
                    total += n
                    if (total > 64L * 1024 * 1024) error("file too large")
                    output.write(buf, 0, n)
                }
            }
        }
    } finally {
        connection.disconnect()
    }
}
