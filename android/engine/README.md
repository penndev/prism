# engine（Android）

本目录同时放 **Go 源码** 和预编译 `engine.aar`。Android 工程直接依赖 AAR，不在 Gradle 里编 Go。

## Java 接口

```kotlin
val opt = Options()
opt.fd = fd
opt.mtu = mtu
opt.proxy = "socks5://user:pass@host:port" // socks5 / socks5s / http / https
opt.upstream = systemDns // VPN 启动前的系统 DNS，给未 fake 的查询用
opt.handler = object : Handler {
    override fun protect(fd: Int) = vpn.protect(fd)
    override fun onLog(line: String?) { /* 连接日志 */ }
    override fun needFake(name: String?): Boolean {
        // Java 持有域名表，判断是否 fake
        ...
    }
    override fun useProxy(network: String?, address: String?): Boolean {
        val chain = Engine.lookup(address) // 叶 → 父，再和已保存的地域 ID 比对
        // 在 Java 打连接日志，例如 "proxy tcp 1.2.3.4:443"
        ...
    }
    override fun onProxyRead(n: Long) { /* 代理路径读增量 */ }
    override fun onProxyWrite(n: Long) { /* 代理路径写增量 */ }
}
Engine.start(opt)
Engine.stop()

Engine.ping(proxyURL, latencyHost) // ms，失败 -1
Engine.setIpregionDB(path)          // Java 负责下载/拷贝文件
Engine.dbStatus()
Engine.areaTree()                   // JSON，形状同桌面 /rule/api/areas
Engine.lookup(address)              // 该 IP 的地域链（叶 → 父）
```

下载 IP 库、选模式和勾选地域都在 Java 完成。域名列表也在 Java：`needFake` 判断是否 fake。上游 DNS 通过 `opt.upstream` 传入（VPN 启动前的系统 DNS）。Go 只拦截 UDP/53 交给 fakeip、对 fake IP 做 Lookup 后按域名走代理；未 fake 的 DNS 直连上游（依赖 Java 的 `protect`，不要 `SetHandleConnect`）。

`Start` 成功后 fd 归 Go；失败时 Java 用 `ParcelFileDescriptor.adoptFd(fd).close()`。`Stop()` 会关掉 fd。出站 socket 必须 `protect`。

## 重编 AAR

仓库根目录：

```powershell
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
$env:ANDROID_NDK_HOME = "$env:ANDROID_HOME\ndk\27.2.12479018"
$env:JAVA_HOME = "C:\Users\Penn\.jdks\jbr-17.0.14"
$env:PATH = "$env:JAVA_HOME\bin;$env:USERPROFILE\go\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
$env:JAVA_TOOL_OPTIONS = "-Dfile.encoding=UTF-8"
$env:JDK_JAVAC_OPTIONS = "-encoding UTF-8"
$env:CGO_LDFLAGS = "-Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384"

go get golang.org/x/mobile/bind
gomobile bind -target=android -androidapi 26 -javapkg com.penndev.prism -o android/engine/engine.aar github.com/penndev/prism/android/engine

```

