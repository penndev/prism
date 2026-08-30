# mstack（Android）

本目录同时放 **Go 源码** 和预编译 `mstack.aar`。Android 工程直接依赖 AAR，不在 Gradle 里编 Go。

## 目录

| 文件 | 说明 |
|---|---|
| `mstack.go` | 导出 `Start` / `Stop`、`Protector`、`Logger` |
| `fdtun.go` | 用 gVisor `fdbased` 把 TUN fd 做成 LinkEndpoint |
| `mstack.aar` | gomobile 编好的库，App 直接引用 |
| `mstack-sources.jar` | gomobile 附带的 Java 源码，可忽略 |

还依赖仓库里的 `stack`、`transport`（不拷到这里）。

## 做什么

把系统 VPN 的 **TUN fd** 交给 Go。gVisor 在进程内读/写这个 fd（`fdbased`，无以太网头），TCP/UDP 用 `transport.Local()` 直连外网。

```
App → TUN fd → gVisor → transport.Local() → 真实 socket（VpnService.protect）
                 ↑
            回包直接写回同一 fd
```

包不过 JNI。这是 Android 专用路径（Linux fd），iOS 以后再单独 bind。

## 工程里怎么用

`app/build.gradle.kts`：

```kotlin
implementation(files("${rootProject.projectDir}/mstack/mstack.aar"))
```

Java 包名：`com.penndev.prism.mstack`

```kotlin
val fd = pfd.detachFd()
Mstack.start(fd, mtu, protector, logger)
// ...
Mstack.stop()
```

| Go | Java |
|---|---|
| `Start(fd, mtu, Protector, Logger)` | `Mstack.start(int, int, Protector, Logger)` |
| `Stop()` | `Mstack.stop()` |
| `Protector.Protect(fd)` | `boolean protect(int fd)`，接到 `VpnService.protect` |
| `Logger.OnConnect(network, address)` | `void onConnect(String, String)`，请求日志 |

`Start` 成功后 fd 归 Go；失败时 Java 要用 `ParcelFileDescriptor.adoptFd(fd).close()` 自己关掉。`Stop()` 会关掉 fd。

直连 socket 必须 `protect`，否则流量打回 TUN 形成环路。

## 重编 AAR

在仓库根目录执行。需要：Go、NDK、JDK、`gomobile`。只 bind 本目录，不要 bind `stack`（有 `func` 字段，gomobile 过不了）。

Windows 示例：

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
gomobile bind -target=android -androidapi 26 -javapkg com.penndev.prism -o android/mstack/mstack.aar github.com/penndev/prism/android/mstack
git checkout -- go.mod go.sum
```

`CGO_LDFLAGS` 把 `libgojni.so` 链成 16 KB 页对齐（Android 15+）。`go get golang.org/x/mobile/bind` 可能把 `go.mod` 升到 1.26，绑完后要还原。
