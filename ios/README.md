# Prism iOS

SwiftUI 客户端，页面和能力对齐 `android/`：节点、订阅导入、规则、日志、设置。

核心代理在 `engine/`，主 App 通过 `Engine.xcframework` 调用 Go。`Ping` 和 IP 库是真实现；`Start` / `Stop` 仍是进程内模拟，**不申请系统 VPN 权限**。

`PacketTunnel/` 和 entitlements 先留着，主 App 暂不嵌入扩展。接 TUN 时再打开 Network Extension / App Groups。

## 打开工程

```sh
open ios/Prism.xcodeproj
```

只编 **Prism** target。真机跑需要在 Signing 里给 Prism 选一个 Team（个人免费账号即可，不必开 Network Extension）。

## 编 engine

见 [engine/README.md](engine/README.md)。
