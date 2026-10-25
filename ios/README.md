# Prism iOS

SwiftUI 客户端，页面和能力对齐 `android/`：节点、订阅导入、规则、日志、设置。

核心代理在 `engine/`。主 App 负责节点、规则和 `Ping`。系统流量走嵌入的 `PacketTunnel`：`NETunnelProviderManager` 拉起扩展，扩展里 `Start` 挂 gVisor，包经 `writePacket` / `handler.writePacket` 进出。配置、规则和 IP 库在 App Group `group.com.penndev.prism`。

## 打开工程

```sh
open ios/Prism.xcodeproj
```

编 **Prism** 会带上 PacketTunnel。真机需要在 Signing 里给 Prism 和 PacketTunnel 选同一个 Team，并打开 Network Extension（packet-tunnel-provider）和 App Group `group.com.penndev.prism`。个人免费账号没有 Network Extension，装不上扩展。

## 编 engine

见 [engine/README.md](engine/README.md)。
