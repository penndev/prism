package com.penndev.prism.data

/**
 * 桌面端业务能力在安卓侧的对接口。
 * 节点导入 / 列表 / 设置已本地实现；VPN 通过 mstack AAR 走系统 VpnService。
 */
interface PrismRepository {
    fun loadSettings(): AppSettings
    fun saveSettings(settings: AppSettings)

    fun loadServers(): List<ServerItem>
    fun saveServers(servers: List<ServerItem>)
    fun loadSelectedServerId(): String?
    fun saveSelectedServer(server: ServerItem?)

    fun startProxy(server: ServerItem)
    fun stopProxy()
    fun setRemote(server: ServerItem?)

    suspend fun pingServer(server: ServerItem, latencyHost: String): Int
    fun trafficSnapshot(): Pair<Long, Long>

    fun saveRules(draft: RuleDraft)
    fun loadRules(): RuleDraft

    suspend fun fetchSubscription(type: String, url: String): List<ServerItem>
    fun parseSubscriptionContent(type: String, text: String): List<ServerItem>
    fun parseServerFile(text: String): List<ServerItem>
    fun exportServers(servers: List<ServerItem>): String
}
