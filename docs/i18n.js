const messages = {
  "zh-CN": {
    title: "Prism · 下载与使用说明",
    description:
      "Prism 代理客户端下载站。桌面端支持 SOCKS5 / HTTP，手动模式与 TUN；移动端走本地 VPN。",
    navDownload: "下载",
    navProxy: "代理",
    navRules: "规则",
    navOther: "其它",
    lead: "代理客户端。把远程 SOCKS5 / HTTP 节点接到本机：桌面端可选手动代理或 TUN，移动端通过本地 VPN 接管流量。",
    downloadTitle: "下载与安装",
    platformWin: "Windows",
    platformMac: "macOS",
    platformAndroid: "Android",
    platformIos: "iOS",
    winDesc: "64 位安装包",
    macDesc: "Apple 芯片（arm64）安装包",
    androidDesc: "安装包（准备中）",
    iosDesc: "安装包（准备中）",
    goReleases: "前往发行页",
    downloadHint:
      '安装包来自 <a href="https://github.com/penndev/prism/releases">GitHub 发行页</a>。用「较新 / 较旧」切换历史版本，下载按钮会跟着当前版本更新。macOS 为 <code>.pkg</code>。Android / iOS 安装包尚未发布时，按钮会打开同一页面。',
    installToggle: "安装说明",
    installWinSteps:
      "<li>点上方按钮下载 <code>Prism-windows-amd64-installer.exe</code>。</li><li>运行安装向导，装完后从开始菜单打开 Prism。</li>",
    installWinSign:
      "安装包目前未做代码签名。若弹出 Windows 安全提示，选「更多信息」→「仍要运行」。",
    installMacSteps:
      "<li>点上方按钮下载并打开 <code>Prism-darwin-arm64.pkg</code>。</li><li>按安装向导装到「应用程序」，再从启动台打开。</li><li>较早版本可能是 <code>.dmg</code> 或 <code>.zip</code>，把 Prism 拖进「应用程序」即可。</li>",
    installMacSign:
      "若提示无法验证开发者：系统设置 → 隐私与安全性 → 仍要打开。",
    installAndroidSteps:
      "<li>点上方按钮下载安装包。包就绪前会先跳到 GitHub 发行页。</li><li>允许「安装未知来源」后打开文件安装。</li>",
    installIosSteps:
      "<li>点上方按钮下载。包就绪前会先跳到 GitHub 发行页。</li><li>按下载页提供的方式安装（测试分发 / 描述文件等以实际说明为准）。</li><li>若提示未受信任的开发者：设置 → 通用 → VPN 与设备管理 → 信任该证书。</li>",
    installIosSign:
      "尚未上架应用商店时，需要按企业签或侧载说明安装，系统可能会拦截未签名应用。",
    proxyTitle: "代理说明",
    proxyIntro:
      "先选一个节点，再在代理面板里选模式。桌面端是手动模式或 TUN；移动端没有这个开关，一律走系统 VPN，效果接近 TUN。",
    proxyStep1: "在节点列表里点选一个节点。",
    proxyStep2: "代理面板会出现当前节点。",
    proxyStep3: "桌面端选手动或 TUN；移动端点启动并允许 VPN。",
    proxyManualTitle: "手动模式",
    proxyManualLead:
      "Prism 只在本机开一个本地代理端口，<strong>不会接管系统流量</strong>。浏览器、终端或其他软件要自己填这个代理，没填的程序仍然直连。适合只让部分软件走代理。",
    localIntro: "右侧设置 → 本地代理：",
    localIp:
      "<strong>IP</strong>：<code>127.0.0.1</code> 仅本机；<code>0.0.0.0</code> 允许局域网设备连进来。",
    localPort:
      "<strong>端口</strong>：默认 <code>1080</code>。被占用就换一个。",
    localAuth:
      "<strong>用户名 / 密码</strong>：可选。填了之后，连这个本地端口也要带同样的账号。",
    localBrowser: "浏览器或系统代理一般填：协议 SOCKS5，主机 127.0.0.1，端口 1080。",
    proxyManualDns:
      "手动模式<strong>不拦截系统 DNS</strong>。如果软件先在本地把域名解析成 IP，再连本地代理端口，域名规则可能用不上，这时主要看 IP 地域规则。",
    proxyTunTitle: "TUN 模式",
    proxyTunLead:
      "Prism 会创建一块虚拟网卡，<strong>接管系统流量</strong>。大多数软件不用自己填代理。必须先选好节点才能开。",
    proxyTunNeed:
      "<strong>先选节点</strong>：没选节点时打不开 TUN。关掉 TUN 或换成手动后，系统路由会收回。",
    proxyTunWin:
      "<strong>Windows</strong>：需要能加载 wintun。失败时用管理员身份再开一次，并检查是否拦截了虚拟网卡。",
    proxyTunDns:
      "<strong>DNS</strong>：TUN 会拦截系统 DNS（UDP/53）。写在域名列表里的名字会被分配假 IP，之后这条连接一定走代理，不再套用地域。",
    proxyTunMobile:
      "<strong>移动端</strong>：没有手动 / TUN 切换，点启动后走系统 VPN，权限允许后才接管流量。停用时在应用里点停止，或在系统 VPN 设置里断开。",
    ruleTitle: "代理规则",
    ruleOpen:
      "规则决定哪些流量走远程节点、哪些直连。桌面端：选好节点后点代理面板右上角漏斗打开规则页（地址形如 <code>http://127.0.0.1:1080/rule/</code>）。移动端：底部「规则」页。改完自动保存。",
    ruleOrder:
      "判定分两层，<strong>先看域名，再看 IP 地域</strong>。",
    ruleDomainTitle: "域名规则",
    ruleDomain:
      "只填<strong>需要代理</strong>的域名，不需要代理的不必写。格式：",
    ruleDomainLine: "一行一个域名，空行忽略。",
    ruleDomainComment:
      "以 <code>!</code> 开头的是分组注释，不会当成域名。",
    ruleDomainMatch:
      "后缀匹配：写 <code>google.com</code> 时，<code>www.google.com</code>、<code>mail.google.com</code> 都会命中。",
    ruleDomainNoUrl:
      "不要写 <code>http://</code>、端口或路径，只写域名本身。",
    ruleDomainTun:
      "TUN / 移动端 VPN 下，命中域名的连接<strong>一定走代理</strong>，不再套用地域。",
    ruleDomainHint: "点击链接下载：",
    ruleGeoTitle: "IP 地域规则",
    ruleGeo:
      "没命中域名规则的请求，按目标 IP 查地域库。四种模式互斥，同一时间只能选一个：",
    ruleGlobal:
      "<strong>全局模式</strong>：不按 IP 地区筛选，除已命中域名规则的之外，其余连接都走代理。",
    ruleNone:
      "<strong>本地模式</strong>：不按 IP 地区筛选，除已命中域名规则的之外，其余连接都直连。",
    ruleProxy:
      "<strong>代理选定区域</strong>：只代理勾选的国家/地区，没勾选的直连。要先加载 IP 库。",
    ruleBypass:
      "<strong>绕过选定区域</strong>：勾选的地区直连，其余走代理。同样要 IP 库。",
    ruleDb: "地域规则需要先加载 IP 库，可在应用规则页下载或上传。",
    ruleDbHint: "点击链接下载：",
    ruleManual:
      "手动模式没有接管系统 DNS。软件若已解析成 IP 再出站，主要看这里的地域规则和 IP 库是否已加载。",
    otherTitle: "其它说明",
    nodeTitle: "添加节点",
    nodeIntro: "桌面：主界面「添加节点」。移动端：节点页右上角添加或导入。填这些即可：",
    nodeHost:
      "<strong>地址</strong>：<code>host:port</code>，例如 <code>1.2.3.4:1080</code>",
    nodeProto:
      "<strong>协议</strong>：<code>socks5</code> / <code>socks5s</code>（TLS）/ <code>http</code> / <code>https</code>",
    nodeAuth: "<strong>用户名 / 密码</strong>：节点需要认证再填",
    nodeRemark: "<strong>备注</strong>：可选，方便辨认",
    nodeSub:
      "批量导入：桌面点节点列表右上角编辑，浏览器打开订阅页；移动端进导入页。支持 Prism 订阅、Shadowrocket 订阅，以及桌面导出的 JSON。解析预览确认后才会覆盖当前列表。",
    otherSettingsTitle: "设置",
    otherPing: "<strong>测速域名</strong>：测全部节点延迟时用。不填则无法测速。",
    otherSort: "<strong>测速后自动排序</strong>：测完按延迟从低到高排。",
    otherTheme:
      "<strong>语言 / 主题</strong>：中文、English；浅色、深色、跟随系统。",
    otherBoot: "<strong>开机启动</strong>（桌面）：登录系统后自动打开 Prism。",
    otherLog:
      "<strong>日志开关</strong>：桌面在窗口底部看状态、连接和流量；移动端在「日志」页。",
    faqTitle: "常见问题",
    faqRuleQ: "规则页或订阅页打不开",
    faqRuleA: "先在设置里填一个有效的本地端口，并保证 Prism 正在运行。",
    faqTunQ: "TUN 开不起来",
    faqTunA:
      "确认已选节点。Windows 用管理员启动；仍失败就检查是否拦截了虚拟网卡。",
    faqVpnQ: "移动端点了启动但连不上",
    faqVpnA:
      "先确认已选节点，并在系统弹窗里允许 VPN。若曾拒绝过，到系统设置里重新授予。",
    faqSaveQ: "改了设置好像没保存",
    faqSaveA: "设置是改完自动写入的。关掉再开应能看到上次的值。",
    faqPingQ: "测速全失败",
    faqPingA:
      "设置里的测速域名要能从该节点访问，例如 <code>google.com</code> 或 <code>host:port</code>。",
    footerSource: "源代码",
    footerReleases: "全部版本",
    releaseLoading: "正在读取发行信息…",
    releaseNewer: "较新版本",
    releaseOlder: "较旧版本",
    releasePage: "{n} / {total}",
    releaseVersion: "当前版本 {tag} · {date} · {page}",
    releaseVersionOnly: "当前版本 {tag} · {page}",
    releaseNoAsset: "已有版本 {tag}，但还没有可识别的安装包。",
    releaseFail:
      '暂时读不到正式包，请到 <a href="https://github.com/penndev/prism/releases">GitHub 发行页</a> 查看。',
    downloadFile: "下载 {name}",
  },
  en: {
    title: "Prism · Download & Guide",
    description:
      "Download Prism. Desktop: SOCKS5 / HTTP with manual proxy or TUN. Mobile: local VPN.",
    navDownload: "Download",
    navProxy: "Proxy",
    navRules: "Rules",
    navOther: "Other",
    lead: "A proxy client. Attach a remote SOCKS5 / HTTP node: on desktop use a local proxy or TUN; on mobile, a system VPN takes over traffic.",
    downloadTitle: "Download & install",
    platformWin: "Windows",
    platformMac: "macOS",
    platformAndroid: "Android",
    platformIos: "iOS",
    winDesc: "64-bit installer",
    macDesc: "Apple silicon (arm64) installer",
    androidDesc: "Package (coming soon)",
    iosDesc: "Package (coming soon)",
    goReleases: "Open releases",
    downloadHint:
      'Builds come from <a href="https://github.com/penndev/prism/releases">GitHub Releases</a>. Use Newer / Older to page through tags; buttons follow the selected release. macOS is a <code>.pkg</code>. Android / iOS buttons stay on that page until those packages are published.',
    installToggle: "Installation",
    installWinSteps:
      "<li>Use the button above to get <code>Prism-windows-amd64-installer.exe</code>.</li><li>Run the wizard, then open Prism from the Start menu.</li>",
    installWinSign:
      "Builds are not code-signed yet. If SmartScreen appears, choose More info → Run anyway.",
    installMacSteps:
      "<li>Use the button above to open <code>Prism-darwin-arm64.pkg</code>.</li><li>Follow the installer into Applications, then open Prism from Launchpad.</li><li>Older releases may still be a <code>.dmg</code> or <code>.zip</code> — drag Prism into Applications.</li>",
    installMacSign:
      "If macOS says the developer cannot be verified: System Settings → Privacy & Security → Open Anyway.",
    installAndroidSteps:
      "<li>Use the button above to get the APK. Until a package is published, it opens GitHub Releases.</li><li>Allow installs from unknown sources, then open the APK.</li>",
    installIosSteps:
      "<li>Use the button above. Until a package is published, it opens GitHub Releases.</li><li>Install with whatever method the download page describes (TestFlight, a profile, etc.).</li><li>If iOS says the developer is untrusted: Settings → General → VPN & Device Management → Trust.</li>",
    installIosSign:
      "Until the app is on the App Store, you may need enterprise signing or sideloading. Unsigned apps can be blocked by the system.",
    proxyTitle: "Proxy modes",
    proxyIntro:
      "Select a node, then pick a mode on the proxy panel. Desktop has Manual and TUN. Mobile has no switch — it always uses a system VPN, close to TUN.",
    proxyStep1: "Select a node in the list.",
    proxyStep2: "The proxy panel shows the current node.",
    proxyStep3: "On desktop pick Manual or TUN. On mobile tap Start and allow VPN.",
    proxyManualTitle: "Manual mode",
    proxyManualLead:
      "Prism only opens a local listen port. It <strong>does not take over system traffic</strong>. Browsers, terminals, and other apps must be pointed at that proxy. Apps you do not configure stay direct. Use this when only some programs should use the proxy.",
    localIntro: "Right-hand Settings → Local proxy:",
    localIp:
      "<strong>IP</strong>: <code>127.0.0.1</code> for this machine only; <code>0.0.0.0</code> also accepts LAN clients.",
    localPort:
      "<strong>Port</strong>: default <code>1080</code>. Change it if the port is taken.",
    localAuth:
      "<strong>Username / password</strong>: optional. If set, clients of this local port must use the same credentials.",
    localBrowser:
      "Typical browser / system proxy values: protocol SOCKS5, host 127.0.0.1, port 1080.",
    proxyManualDns:
      "Manual mode <strong>does not intercept system DNS</strong>. If an app resolves the name locally and then connects to the local proxy by IP, domain rules may not apply; geo rules do.",
    proxyTunTitle: "TUN mode",
    proxyTunLead:
      "Prism creates a virtual adapter and <strong>takes over system traffic</strong>. Most apps do not need their own proxy settings. A node must be selected first.",
    proxyTunNeed:
      "<strong>Select a node first</strong>. TUN will not start without one. Stopping TUN or switching back to Manual restores system routing.",
    proxyTunWin:
      "<strong>Windows</strong>: wintun must load. If it fails, run as administrator and check whether a virtual adapter is being blocked.",
    proxyTunDns:
      "<strong>DNS</strong>: TUN intercepts system DNS (UDP/53). Names in the domain list get a fake IP, and that connection always uses the proxy — geo is skipped.",
    proxyTunMobile:
      "<strong>Mobile</strong>: no Manual / TUN switch. Start uses a system VPN; traffic is taken over only after you allow it. Stop it in the app, or disconnect from system VPN settings.",
    ruleTitle: "Routing rules",
    ruleOpen:
      "Rules decide what goes through the remote node and what stays direct. Desktop: after a node is selected, the funnel on the proxy panel opens the rules page (URL like <code>http://127.0.0.1:1080/rule/</code>). Mobile: the Rules tab. Changes save automatically.",
    ruleOrder:
      "Evaluation has two layers: <strong>domain first, then IP geo</strong>.",
    ruleDomainTitle: "Domain rules",
    ruleDomain:
      "List only domains that <strong>should</strong> use the proxy. Leave out names that should stay direct. Format:",
    ruleDomainLine: "One domain per line; blank lines are ignored.",
    ruleDomainComment:
      "Lines starting with <code>!</code> are section comments, not domains.",
    ruleDomainMatch:
      "Suffix match: <code>google.com</code> also matches <code>www.google.com</code> and <code>mail.google.com</code>.",
    ruleDomainNoUrl:
      "Do not write <code>http://</code>, a port, or a path — the hostname only.",
    ruleDomainTun:
      "In TUN or the mobile VPN, a domain hit <strong>always uses the proxy</strong> — geo is skipped.",
    ruleDomainHint: "Click the link to download: ",
    ruleGeoTitle: "IP geo rules",
    ruleGeo:
      "Requests that did not match a domain are looked up in the IP database. The four modes are exclusive — pick one:",
    ruleGlobal:
      "<strong>Global mode</strong>: no region filter. Traffic that did not match a domain rule also goes through the proxy.",
    ruleNone:
      "<strong>Local mode</strong>: no region filter. Traffic that did not match a domain rule goes direct.",
    ruleProxy:
      "<strong>Proxy selected regions</strong>: only checked countries/regions use the proxy; the rest go direct. Needs the IP database.",
    ruleBypass:
      "<strong>Bypass selected regions</strong>: checked regions go direct; the rest use the proxy. Also needs the IP database.",
    ruleDb: "Geo rules need the IP database. Download or upload it on the rules page.",
    ruleDbHint: "Click the link to download: ",
    ruleManual:
      "Manual mode does not take over system DNS. If an app already resolved the name to an IP, geo rules and the IP database matter more.",
    otherTitle: "Other",
    nodeTitle: "Add a node",
    nodeIntro:
      "Desktop: Add node on the main window. Mobile: add or import from the Nodes tab. Fill in:",
    nodeHost:
      "<strong>Address</strong>: <code>host:port</code>, e.g. <code>1.2.3.4:1080</code>",
    nodeProto:
      "<strong>Protocol</strong>: <code>socks5</code> / <code>socks5s</code> (TLS) / <code>http</code> / <code>https</code>",
    nodeAuth: "<strong>Username / password</strong>: only if the node requires auth",
    nodeRemark: "<strong>Remark</strong>: optional label",
    nodeSub:
      "Bulk import: on desktop, the edit button opens a browser page; on mobile, open Import. It accepts Prism subscriptions, Shadowrocket subscriptions, and JSON exported from the desktop app. The current list is replaced only after you confirm the preview.",
    otherSettingsTitle: "Settings",
    otherPing:
      "<strong>Latency test host</strong>: used when pinging all nodes. Leave it empty and ping is disabled.",
    otherSort:
      "<strong>Sort after ping</strong>: reorder the list from low to high latency.",
    otherTheme:
      "<strong>Language / theme</strong>: Chinese or English; light, dark, or follow system.",
    otherBoot: "<strong>Launch at startup</strong> (desktop): open Prism after login.",
    otherLog:
      "<strong>Logging</strong>: desktop shows status, connections, and traffic at the bottom of the window; mobile uses the Logs tab.",
    faqTitle: "FAQ",
    faqRuleQ: "Rules or subscription page will not open",
    faqRuleA:
      "Set a valid local port in Settings and keep Prism running.",
    faqTunQ: "TUN will not start",
    faqTunA:
      "Select a node first. On Windows, run as administrator. If it still fails, check whether a virtual adapter is being blocked.",
    faqVpnQ: "Mobile Start does nothing",
    faqVpnA:
      "Select a node first, and allow the VPN prompt. If you denied it earlier, grant it again in system settings.",
    faqSaveQ: "Settings look unsaved",
    faqSaveA:
      "Changes are written automatically. Quit and reopen to confirm they persist.",
    faqPingQ: "Every ping fails",
    faqPingA:
      "The latency test host must be reachable through that node, e.g. <code>google.com</code> or <code>host:port</code>.",
    footerSource: "Source",
    footerReleases: "All releases",
    releaseLoading: "Loading GitHub Release…",
    releaseNewer: "Newer",
    releaseOlder: "Older",
    releasePage: "{n} / {total}",
    releaseVersion: "Version {tag} · {date} · {page}",
    releaseVersionOnly: "Version {tag} · {page}",
    releaseNoAsset: "Release {tag} exists, but no matching installer was found.",
    releaseFail:
      'Could not load a release. See <a href="https://github.com/penndev/prism/releases">GitHub Releases</a>.',
    downloadFile: "Download {name}",
  },
};
