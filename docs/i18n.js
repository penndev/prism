const messages = {
  "zh-CN": {
    title: "Prism · 下载与使用说明",
    description:
      "Prism 桌面代理客户端下载站。支持 SOCKS5 / HTTP，手动模式与 TUN 模式。",
    navDownload: "下载",
    navGuide: "使用说明",
    lead: "桌面代理客户端。把远程 SOCKS5 / HTTP 节点接到本机，按需走手动代理或 TUN 全局接管。",
    downloadTitle: "下载",
    winDesc: "amd64 安装包",
    macDesc: "Apple Silicon (arm64)",
    androidDesc: "APK（准备中）",
    iosDesc: "安装包（准备中）",
    goReleases: "前往 Releases",
    downloadHint:
      '安装包来自 <a href="https://github.com/penndev/socks5/releases">GitHub Releases</a>。Android / iOS 包就绪后会出现在对应卡片，现在先保留同一链接。',
    guideTitle: "使用说明",
    installTitle: "1. 安装",
    installWinSteps:
      "<li>在上方下载区点 Windows 卡片，获取 <code>Prism-windows-amd64-installer.exe</code>。</li><li>运行安装向导，装完后从开始菜单打开 Prism。</li>",
    installWinSign:
      "安装包目前未做代码签名。若弹出 SmartScreen，选「更多信息」→「仍要运行」。",
    installIosSteps:
      "<li>在上方下载区点 iOS 卡片。包就绪前会先跳到 GitHub Releases。</li><li>按下载页提供的方式安装（TestFlight / 描述文件等以实际说明为准）。</li><li>若提示未受信任的开发者：设置 → 通用 → VPN 与设备管理 → 信任该证书。</li>",
    installIosSign:
      "iOS 包尚未上架 App Store 时，需要按企业签或侧载说明安装，系统可能会拦截未签名应用。",
    installAndroidSteps:
      "<li>在上方下载区点 Android 卡片，获取 APK。包就绪前会先跳到 GitHub Releases。</li><li>允许「安装未知来源」后打开 APK 安装。</li>",
    installMacSteps:
      "<li>在上方下载区点 macOS 卡片，打开 <code>Prism-darwin-arm64.dmg</code>。</li><li>把 Prism 拖进「应用程序」，再从启动台打开。</li>",
    installMacSign:
      "若提示无法验证开发者：系统设置 → 隐私与安全性 → 仍要打开。",
    nodeTitle: "2. 添加节点",
    nodeIntro: "主界面下方「添加节点」，填这些即可：",
    nodeHost:
      "<strong>地址</strong>：<code>host:port</code>，例如 <code>1.2.3.4:1080</code>",
    nodeProto:
      "<strong>协议</strong>：<code>socks5</code> / <code>socks5s</code>（TLS）/ <code>http</code> / <code>https</code>",
    nodeAuth: "<strong>用户名 / 密码</strong>：节点需要认证再填",
    nodeRemark: "<strong>备注</strong>：可选，方便辨认",
    nodeSub:
      "批量导入：点节点列表右上角编辑按钮，浏览器会打开订阅页。支持 Prism 订阅、Shadowrocket 订阅，以及桌面导出的 JSON。解析预览确认后才会覆盖当前列表。",
    connectTitle: "3. 连接",
    connect1: "在节点列表里点选一个节点。",
    connect2: "上方代理面板会出现当前节点。",
    connect3: "选模式：",
    modeManual:
      "<strong>手动模式</strong>：只在本机开一个本地代理端口，浏览器或其他软件自己填代理。",
    modeTun:
      "<strong>TUN 模式</strong>：创建虚拟网卡接管系统流量。必须先选好节点；Windows 需要能加载 wintun，失败时用管理员身份再开一次。",
    localTitle: "4. 本地代理（手动模式）",
    localIntro: "右侧设置 → 本地代理：",
    localIp:
      "<strong>IP</strong>：<code>127.0.0.1</code> 仅本机；<code>0.0.0.0</code> 允许局域网设备连进来。",
    localPort:
      "<strong>端口</strong>：默认 <code>1080</code>。被占用就换一个。",
    localAuth:
      "<strong>用户名 / 密码</strong>：可选。填了之后，连这个本地端口也要带同样的账号。",
    localBrowser: "浏览器或系统代理一般填：",
    localSample: "协议 SOCKS5\n主机 127.0.0.1\n端口 1080",
    ruleTitle: "5. 代理规则",
    ruleOpen:
      "规则决定哪些流量走远程节点、哪些直连。选好节点后，点代理面板右上角漏斗图标打开规则页（地址形如 <code>http://127.0.0.1:1080/rule/</code>）。改完自动保存。",
    ruleOrder:
      "实际判定分两层，<strong>先看域名，再看地域</strong>：",
    ruleOrder1:
      "<strong>TUN 模式</strong>会拦截系统 DNS（UDP/53）。域名写在列表里时，会被分配 fake IP，之后这条连接<strong>一定走代理</strong>，不再套用地域。",
    ruleOrder2:
      "不在域名列表里的请求，按目标 IP 查地域库，再按下面四种模式决定代理还是直连。",
    ruleManual:
      "手动模式没有接管系统 DNS。如果软件先在本地把域名解析成 IP，再连本地代理端口，域名规则可能用不上，这时主要看地域。",
    ruleGeoTitle: "地域规则",
    ruleGeo: "四种模式互斥，同一时间只能选一个：",
    ruleGlobal:
      "<strong>全局模式</strong>：不按 IP 地区筛选，除已命中域名规则的之外，其余连接都走代理。",
    ruleNone:
      "<strong>本地模式</strong>：不按 IP 地区筛选，除已命中域名规则的之外，其余连接都直连。",
    ruleProxy:
      "<strong>代理选定区域</strong>：只代理勾选的国家/地区，没勾选的直连。要先加载 IP 库。",
    ruleBypass:
      "<strong>绕过选定区域</strong>：勾选的地区直连，其余走代理。同样要 IP 库。",
    ruleDb:
      'IP 库文件是 <code>ipregion.db</code>，可在规则页下载或上传。库文件说明见 <a href="https://github.com/penndev/gopkg/tree/main/ipregion" target="_blank" rel="noopener">penndev/gopkg ipregion</a>。',
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
    ruleDomainExample:
      '完整示例见 <a href="./domain.txt">domain.txt</a> ，可直接复制到规则页。节选如下：',
    otherTitle: "6. 其它设置",
    otherPing: "<strong>测速域名</strong>：测全部节点延迟时用。不填则无法测速。",
    otherSort: "<strong>测速后自动排序</strong>：测完按延迟从低到高排。",
    otherTheme:
      "<strong>语言 / 主题</strong>：中文、English；浅色、深色、跟随系统。",
    otherBoot: "<strong>开机启动</strong>：登录系统后自动打开 Prism。",
    otherLog:
      "<strong>日志开关</strong>：打开后窗口底部出现状态日志、连接日志和流量。",
    faqTitle: "7. 常见问题",
    faqRuleQ: "规则页或订阅页打不开",
    faqRuleA: "先在设置里填一个有效的本地端口，并保证 Prism 正在运行。",
    faqTunQ: "TUN 开不起来",
    faqTunA:
      "确认已选节点。Windows 用管理员启动；仍失败就检查是否拦截了虚拟网卡。",
    faqSaveQ: "改了设置好像没保存",
    faqSaveA: "设置是改完自动写入的。关掉再开应能看到上次的值。",
    faqPingQ: "测速全失败",
    faqPingA:
      "设置里的测速域名要能从该节点访问，例如 <code>google.com</code> 或 <code>host:port</code>。",
    footerSource: "源代码",
    footerReleases: "全部版本",
    releaseLoading: "正在读取 GitHub Release…",
    releaseVersion: "当前版本 {tag} · {date}",
    releaseVersionOnly: "当前版本 {tag}",
    releaseNoAsset: "已有版本 {tag}，但还没有可识别的安装包。",
    releaseFail:
      '暂时读不到正式包，请到 <a href="https://github.com/penndev/socks5/releases">GitHub Releases</a> 查看。',
    downloadFile: "下载 {name}",
  },
  en: {
    title: "Prism · Download & Guide",
    description:
      "Download Prism for desktop. SOCKS5 / HTTP, with manual proxy or TUN.",
    navDownload: "Download",
    navGuide: "Guide",
    lead: "A desktop proxy client. Attach a remote SOCKS5 / HTTP node and use it as a local proxy or as a TUN that takes over system traffic.",
    downloadTitle: "Download",
    winDesc: "amd64 installer",
    macDesc: "Apple Silicon (arm64)",
    androidDesc: "APK (coming soon)",
    iosDesc: "Package (coming soon)",
    goReleases: "Open Releases",
    downloadHint:
      'Builds come from <a href="https://github.com/penndev/socks5/releases">GitHub Releases</a>. Android / iOS buttons stay on that page until those packages are published.',
    guideTitle: "How to use",
    installTitle: "1. Install",
    installWinSteps:
      "<li>Use the Windows card above to get <code>Prism-windows-amd64-installer.exe</code>.</li><li>Run the wizard, then open Prism from the Start menu.</li>",
    installWinSign:
      "Builds are not code-signed yet. If SmartScreen appears, choose More info → Run anyway.",
    installIosSteps:
      "<li>Use the iOS card above. Until a package is published, it opens GitHub Releases.</li><li>Install with whatever method the download page describes (TestFlight, a profile, etc.).</li><li>If iOS says the developer is untrusted: Settings → General → VPN & Device Management → Trust.</li>",
    installIosSign:
      "Until the app is on the App Store, you may need enterprise signing or sideloading. Unsigned apps can be blocked by the system.",
    installAndroidSteps:
      "<li>Use the Android card above to get the APK. Until a package is published, it opens GitHub Releases.</li><li>Allow installs from unknown sources, then open the APK.</li>",
    installMacSteps:
      "<li>Use the macOS card above and open <code>Prism-darwin-arm64.dmg</code>.</li><li>Drag Prism into Applications, then open it from Launchpad.</li>",
    installMacSign:
      "If macOS says the developer cannot be verified: System Settings → Privacy & Security → Open Anyway.",
    nodeTitle: "2. Add a node",
    nodeIntro: "On the main window, click Add node and fill in:",
    nodeHost:
      "<strong>Address</strong>: <code>host:port</code>, e.g. <code>1.2.3.4:1080</code>",
    nodeProto:
      "<strong>Protocol</strong>: <code>socks5</code> / <code>socks5s</code> (TLS) / <code>http</code> / <code>https</code>",
    nodeAuth: "<strong>Username / password</strong>: only if the node requires auth",
    nodeRemark: "<strong>Remark</strong>: optional label",
    nodeSub:
      "Bulk import: click the edit button on the node list. A browser page opens. It accepts Prism subscriptions, Shadowrocket subscriptions, and JSON exported from the desktop app. The current list is replaced only after you confirm the preview.",
    connectTitle: "3. Connect",
    connect1: "Select a node in the list.",
    connect2: "The proxy panel shows the current node.",
    connect3: "Pick a mode:",
    modeManual:
      "<strong>Manual</strong>: Prism only opens a local listen port. Point your browser or other apps at it.",
    modeTun:
      "<strong>TUN</strong>: creates a virtual adapter and takes over system traffic. A node must be selected first. On Windows, wintun must load; if it fails, try running as administrator.",
    localTitle: "4. Local proxy (manual mode)",
    localIntro: "Right-hand Settings → Local proxy:",
    localIp:
      "<strong>IP</strong>: <code>127.0.0.1</code> for this machine only; <code>0.0.0.0</code> also accepts LAN clients.",
    localPort:
      "<strong>Port</strong>: default <code>1080</code>. Change it if the port is taken.",
    localAuth:
      "<strong>Username / password</strong>: optional. If set, clients of this local port must use the same credentials.",
    localBrowser: "Typical browser / system proxy values:",
    localSample: "Protocol SOCKS5\nHost 127.0.0.1\nPort 1080",
    ruleTitle: "5. Routing rules",
    ruleOpen:
      "Rules decide what goes through the remote node and what stays direct. After a node is selected, the funnel icon on the proxy panel opens the rules page (URL like <code>http://127.0.0.1:1080/rule/</code>). Changes save automatically.",
    ruleOrder:
      "Evaluation has two layers: <strong>domain first, then geo</strong>.",
    ruleOrder1:
      "<strong>TUN mode</strong> intercepts system DNS (UDP/53). Names in the domain list get a fake IP, and that connection <strong>always uses the proxy</strong> — geo is skipped.",
    ruleOrder2:
      "Everything else is looked up in the IP database, then the four geo modes below decide proxy vs direct.",
    ruleManual:
      "Manual mode does not take over system DNS. If an app resolves the name locally and then connects to the local proxy by IP, domain rules may not apply; geo rules do.",
    ruleGeoTitle: "Geo rules",
    ruleGeo: "The four modes are exclusive — pick one:",
    ruleGlobal:
      "<strong>Global mode</strong>: no region filter. Traffic that did not match a domain rule also goes through the proxy.",
    ruleNone:
      "<strong>Local mode</strong>: no region filter. Traffic that did not match a domain rule goes direct.",
    ruleProxy:
      "<strong>Proxy selected regions</strong>: only checked countries/regions use the proxy; the rest go direct. Needs the IP database.",
    ruleBypass:
      "<strong>Bypass selected regions</strong>: checked regions go direct; the rest use the proxy. Also needs the IP database.",
    ruleDb:
      'The database file is <code>ipregion.db</code>. Download or upload it on the rules page. See <a href="https://github.com/penndev/gopkg/tree/main/ipregion" target="_blank" rel="noopener">penndev/gopkg ipregion</a>.',
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
    ruleDomainExample:
      'A full sample is <a href="./domain.txt">domain.txt</a>. You can paste it into the rules page. Excerpt:',
    otherTitle: "6. Other settings",
    otherPing:
      "<strong>Latency test host</strong>: used when pinging all nodes. Leave it empty and ping is disabled.",
    otherSort:
      "<strong>Sort after ping</strong>: reorder the list from low to high latency.",
    otherTheme:
      "<strong>Language / theme</strong>: Chinese or English; light, dark, or follow system.",
    otherBoot: "<strong>Launch at startup</strong>: open Prism after login.",
    otherLog:
      "<strong>Logging</strong>: shows status logs, connection logs, and traffic at the bottom of the window.",
    faqTitle: "7. FAQ",
    faqRuleQ: "Rules or subscription page will not open",
    faqRuleA:
      "Set a valid local port in Settings and keep Prism running.",
    faqTunQ: "TUN will not start",
    faqTunA:
      "Select a node first. On Windows, run as administrator. If it still fails, check whether a virtual adapter is being blocked.",
    faqSaveQ: "Settings look unsaved",
    faqSaveA:
      "Changes are written automatically. Quit and reopen to confirm they persist.",
    faqPingQ: "Every ping fails",
    faqPingA:
      "The latency test host must be reachable through that node, e.g. <code>google.com</code> or <code>host:port</code>.",
    footerSource: "Source",
    footerReleases: "All releases",
    releaseLoading: "Loading GitHub Release…",
    releaseVersion: "Version {tag} · {date}",
    releaseVersionOnly: "Version {tag}",
    releaseNoAsset: "Release {tag} exists, but no matching installer was found.",
    releaseFail:
      'Could not load a release. See <a href="https://github.com/penndev/socks5/releases">GitHub Releases</a>.',
    downloadFile: "Download {name}",
  },
};
