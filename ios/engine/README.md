# engine（iOS）

本目录同时放 **Go 源码** 和预编译 `Engine.xcframework`。Xcode 里 Prism target 链接并 embed 该 framework，不在 Xcode 里编 Go。

`Ping` / `SetIpregionDB` / `AreaTree` / `Lookup` 和 Android 一样是真实现。`Start` / `Stop` 挂 gVisor channel 端点；iOS 没有 TUN fd，包经 `Engine.writePacket` / `handler.writePacket` 进出。主 App 暂不嵌入 PacketTunnel，进程内 Start 时 handler 的 `writePacket` 是空实现。

Swift 经 `Shared/Engine.swift` 调用 gomobile 符号（`EnginePing`、`EngineStart` 等）。

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
try Engine.setIpregionDB(path)     // Swift 负责下载/拷贝文件
Engine.dbStatus()
Engine.areaTree()                  // JSON，形状同桌面 /rule/api/areas
Engine.lookup(address)             // 该 IP 的地域链（叶 → 父）
```

下载 IP 库、选模式和勾选地域都在 Swift 完成。域名列表也在 Swift：`needFake` 判断是否 fake。`Engine.writePacket` 是 Swift 写入栈；`handler.writePacket` 把出站包写回 `packetFlow`。主 App 进程内 Start 给空实现即可。

改过 `Start` / `WritePacket` 之后要重编 xcframework，否则 Xcode 对不上新符号。

## 重编 xcframework

仓库根目录。`go get` 只加依赖，不会安装命令：

```sh
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init   # 只需第一次

# 若 sumdb 超时（常见于 goproxy.cn）：加上 GOSUMDB=off
gomobile bind -target=ios -iosversion=16.0 -o ios/engine/Engine.xcframework github.com/penndev/prism/ios/engine
```

