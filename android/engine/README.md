# engine（Android）

本目录同时放 **Go 源码** 和预编译 `engine.aar`。Android 工程直接依赖 AAR，不在 Gradle 里编 Go。包名是 `com.penndev.prism.engine`。

Go 把 gVisor 挂到 VPN 的 TUN fd 上，转发 TCP/UDP。UDP/53 一律在本地交给 fakeip：`needFake` 为真时回假 IP（`198.18.0.0/15`），之后连到这个假 IP 会换回域名再走代理；为假时把查询转到 `upstream`。其余连接由 `useProxy` 决定走代理还是直连。地域库、模式和域名表都在 Java，Go 不保存这些配置。

`start` 成功后 fd 归 Go；失败时 Java 用 `ParcelFileDescriptor.adoptFd(fd).close()`。`stop` 会关掉 fd，并且会阻塞到 gVisor 的收包协程退出，不要放在主线程。出站 socket 必须 `protect`，否则会绕回隧道。未 fake 的 DNS 直连 `upstream`，不要对 fakeip 调用 `SetHandleConnect`。

## Engine

| 方法 | 作用 |
| --- | --- |
| `start(opt)` | 用 `Options` 拉起隧道。已启动、缺少 `handler`、代理 URL 无法解析时返回错误。`mtu <= 0` 时用 1500。 |
| `stop()` | 停掉协议栈并关闭 TUN fd。 |
| `ping(proxyURL, latencyHost)` | 经代理测 HTTP 首字节时间，单位毫秒。`latencyHost` 是主机或 `主机:端口`（缺端口当 80）。失败或主机为空返回 `-1`。不经过 TUN。 |
| `setIpregionDB(path)` | 打开 Java 已经下载或拷好的 `ipregion.db`。换库时会关掉旧句柄。 |
| `dbStatus()` | 当前库的文件和元数据：是否存在、路径、大小、版本、备注、地域数、IPv4/IPv6 条目数。 |
| `areaTree()` | 整棵地域树的 JSON，形状与桌面 `/rule/api/areas` 相同。未打开库时返回 `[]`。 |
| `lookup(address)` | 查 `主机` 或 `主机:端口` 的地域链，叶节点在前、父节点在后。传入的是域名时会先解析成 IP。无库或查不到时返回空列表。 |

## Options

| 字段 | 作用 |
| --- | --- |
| `fd` | TUN 的文件描述符。 |
| `mtu` | 隧道 MTU。 |
| `proxy` | 上游代理 URL：`socks5`、`socks5s`、`http`、`https`，形如 `socks5://user:pass@host:port`。 |
| `upstream` | VPN 启动前抓到的系统 DNS（IPv4）。给未 fake 的查询用；空则用 `8.8.8.8:53`。 |
| `handler` | Java 实现的回调，不能为空。 |

## Handler

Java 实现，Go 在转发时回调。

| 方法 | 作用 |
| --- | --- |
| `protect(fd)` | 包一层 `VpnService.protect`，让这条出站 socket 不进隧道。返回 `false` 时这次拨号失败。 |
| `needFake(name)` | 每个 DNS 名问一次。返回 `true` 则回假 IP，随后该连接按域名走代理，不再问 `useProxy`。返回 `false` 则把查询转到 `upstream`。域名表在 Java。 |
| `useProxy(network, address)` | 非假 IP 的连接问一次。返回 `true` 走代理，`false` 直连。Java 用 `lookup` 拿地域链（叶 → 父）和已保存的地域 ID 比对，并在这里打连接日志，例如 `proxy tcp 1.2.3.4:443`。 |
| `onLog(line)` | 假 IP 改写、直连或代理失败、relay panic。 |
| `onProxyRead(n)` / `onProxyWrite(n)` | 代理路径上的读、写字节增量。直连不计入。 |

## 返回类型

| 类型 | 作用 |
| --- | --- |
| `Area` | 地域节点：`id`、`parentId`（顶级为 0）、`name`。 |
| `AreaList` | `lookup` 的结果。用 `len()` 和 `get(i)` 取，没有直接暴露切片。 |
| `DbStatus` | `dbStatus` 的结果，字段见上表。 |

## 重编 AAR

在仓库根目录执行。NDK 版本和 `CGO_LDFLAGS` 的 16KB 页大小要一起留着，否则 Android 15 的设备上库加载会失败。

`JAVA_HOME` 用当前环境里已有的 JDK 17。没设置时再找本机 Android Studio 自带的 JBR，不写死某台电脑上的用户目录。

```powershell
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
$env:ANDROID_NDK_HOME = "$env:ANDROID_HOME\ndk\27.2.12479018"

if (-not $env:JAVA_HOME -or -not (Test-Path "$env:JAVA_HOME\bin\java.exe")) {
    $studioJbr = @(
        "$env:ProgramFiles\Android\Android Studio\jbr",
        "$env:LOCALAPPDATA\Programs\Android\Android Studio\jbr"
    ) | Where-Object { Test-Path "$_\bin\java.exe" } | Select-Object -First 1
    if (-not $studioJbr) {
        throw "找不到 JDK 17。请安装 Android Studio，或先把 JAVA_HOME 指到 JDK 根目录。"
    }
    $env:JAVA_HOME = $studioJbr
}

$env:PATH = "$env:JAVA_HOME\bin;$env:USERPROFILE\go\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
$env:JAVA_TOOL_OPTIONS = "-Dfile.encoding=UTF-8"
$env:JDK_JAVAC_OPTIONS = "-encoding UTF-8"
$env:CGO_LDFLAGS = "-Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384"

go get golang.org/x/mobile/bind
gomobile bind -target=android -androidapi 26 -javapkg com.penndev.prism -o android/engine/engine.aar github.com/penndev/prism/android/engine
```
