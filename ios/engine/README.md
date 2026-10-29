# engine（iOS）

本目录放预编译 `Engine.xcframework`。Go 源码在仓库根目录的 `engine/`，说明见 `engine/README.md`。`engine_ios.go` 与 `pktun.go` 带 `//go:build ios`。Xcode 里 Prism 和 PacketTunnel 只链接该 framework，不把它拷进 App。它是静态库，拷进去后上传会要求一份不存在的 dSYM。不在 Xcode 里编 Go。

`Ping` 和 IP 库接口与 Android 一样是真实现。`Start` / `Stop` 挂 gVisor channel 端点；iOS 没有 TUN fd，包经 `Engine.writePacket` / `handler.writePacket` 进出。系统流量在 `PacketTunnel` 扩展里跑，主 App 用 `NETunnelProviderManager` 启动它。

`Start` 会把 Go 的内存软上限设成 36MB：扩展整个进程限 50MB，超了被系统直接杀掉。`ipregion.db` 用文件模式打开，段表解压到 `NSTemporaryDirectory()` 下的 `ipregion-*` 目录（约 30MB 磁盘），内存只留索引。主 App 和扩展是两个进程，各打开一份，各有自己的临时目录。

Swift 经 `Shared/Engine.swift` 调用隧道相关的 gomobile 符号（`EnginePing`、`EngineStart` 等）；IP 库在 `Shared/Ipregion.swift` 里直接调 `EngineSetIpregionDB` / `EngineIpregionStatus` / `EngineAreaTree` / `EngineLookup`。

## Swift 接口

```swift
let opt = EngineOptions()
opt.mtu = 1500
opt.proxy = "socks5://user:pass@host:port" // socks5 / socks5s / http / https
opt.upstream = systemDns
opt.handler = handler
try Engine.start(opt)
Engine.writePacket(packet)      // packetFlow.readPackets 推入
Engine.stop()

Engine.ping(proxyURL, latencyHost) // ms，失败 -1
EngineGoroutines()                 // 当前 goroutine 数，扩展内存打点用

EngineSetIpregionDB(path, NSTemporaryDirectory(), &err) // 文件模式打开，换库时关旧句柄
EngineIpregionStatus()             // version / areas，未打开库时 nil
EngineAreaTree()                   // JSON，形状同桌面 /rule/api/areas，未打开库时 "[]"
EngineLookup(address)              // 地域 ID 链（叶 → 父），len() / get(i) 取
```

下载 IP 库、选模式和勾选地域都在 Swift 完成。域名列表也在 Swift：`needFake` 判断是否 fake。`Engine.writePacket` 是扩展把 `packetFlow` 读到的包写入栈；`handler.writePacket` 把出站包写回 `packetFlow`。

改过 `Start` / `WritePacket` 之后要重编 xcframework，否则 Xcode 对不上新符号。

## 重编 xcframework

仓库根目录。`go get` 只加依赖，不会安装命令：

```sh
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init   # 只需第一次

# 若 sumdb 超时（常见于 goproxy.cn）：加上 GOSUMDB=off
gomobile bind -target=ios -iosversion=16.0 -o ios/engine/Engine.xcframework github.com/penndev/prism/engine
```
