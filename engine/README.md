# engine

Android、iOS、桌面共用的连接处理。gomobile 只绑这个包：Android 用 `//go:build android`，iOS 用 `//go:build ios`。

| 文件 | 谁能编到 |
| --- | --- |
| `handle.go` `ping.go` `ipregion.go` | 三端 |
| `engine_android.go` | Android。TUN fd，出站 socket 要 `protect` |
| `engine_ios.go` `pktun.go` | iOS。没有 fd，包经 `WritePacket` 进出 |

`Relay` 处理一条连接，顺序固定：

1. UDP/53 交给 fakeip。假 IP 的查询回 `198.18.0.0/15`，其余转到 `upstream`。
2. 目标是假 IP 时换回域名，走代理。
3. 其它连接问 `Hook.UseProxy`：真走代理，假直连。直连不计入代理流量。

`Hook` 有 `OnStatus`、`OnConnection`、`OnDNS`、`UseProxy`、`OnProxyRead`、`OnProxyWrite`。日志只传字段，排版由客户端决定。Android / iOS 的 `Handler` 已经包含这些方法，另外的 `Protect` 和 `WritePacket` 只属于各自的隧道。桌面在 `desktop/internal/proxy/proxy_hook.go` 里实现 `Hook`，地域判断留在那里。移动端用这个包的 `SetIpregionDB` / `IpregionStatus` / `AreaTree` / `Lookup` 查 IP 库，地域判断在原生 `UseProxy` 里做。

`Relay` 失败时关连接并返回错误，成功路径不关。桌面本地代理端口在调用处 `defer conn.Close()`。非 DNS 的 UDP 连接 60 秒没有收发就关掉。

重编 AAR 见 [android/engine/README.md](../android/engine/README.md)。重编 xcframework 见 [ios/engine/README.md](../ios/engine/README.md)。
