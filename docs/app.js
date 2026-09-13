const REPO = "penndev/prism";
const RELEASES = `https://github.com/${REPO}/releases`;

let locale = "zh-CN";
let releases = [];
let releaseIndex = 0;
let releaseFailed = false;

function text(key) {
  const pack = messages[locale] || messages["zh-CN"];
  return pack[key] ?? messages["zh-CN"][key] ?? key;
}

function applyLocale(next) {
  if (typeof messages === "undefined") {
    document.documentElement.classList.remove("i18n-pending");
    return;
  }
  locale = messages[next] ? next : "zh-CN";
  document.documentElement.lang = locale;
  document.title = text("title");
  const desc = document.querySelector('meta[name="description"]');
  if (desc) desc.setAttribute("content", text("description"));

  document.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = text(el.getAttribute("data-i18n"));
  });
  document.querySelectorAll("[data-html]").forEach((el) => {
    el.innerHTML = text(el.getAttribute("data-html"));
  });
  document.querySelectorAll("[data-lang]").forEach((btn) => {
    const on = btn.getAttribute("data-lang") === locale;
    btn.classList.toggle("is-active", on);
    btn.setAttribute("aria-pressed", on ? "true" : "false");
  });
  paintRelease();
  document.documentElement.classList.remove("i18n-pending");
}

function pickAsset(assets, test) {
  return (assets || []).find((a) => test(a.name || ""));
}

function pickWindows(assets) {
  return (
    pickAsset(assets, (n) => /windows/i.test(n) && n.endsWith("-installer.exe")) ||
    pickAsset(assets, (n) => /installer/i.test(n) && n.endsWith(".exe")) ||
    pickAsset(assets, (n) => /windows/i.test(n) && n.endsWith(".exe"))
  );
}

function pickMac(assets) {
  return (
    pickAsset(assets, (n) => /darwin/i.test(n) && n.endsWith(".pkg")) ||
    pickAsset(assets, (n) => /darwin/i.test(n) && n.endsWith(".dmg")) ||
    pickAsset(assets, (n) => /darwin/i.test(n) && n.endsWith(".zip"))
  );
}

function pickAndroid(assets) {
  return pickAsset(assets, (n) => n.toLowerCase().endsWith(".apk"));
}

function pickIos(assets) {
  return pickAsset(assets, (n) => n.toLowerCase().endsWith(".ipa"));
}

function paintBtn(el, asset) {
  if (!el) return;
  if (asset) {
    el.href = asset.browser_download_url;
    el.textContent = text("downloadFile").replace("{name}", asset.name);
  } else {
    el.href = RELEASES;
    el.textContent = text("goReleases");
  }
}

function paintRelease() {
  const meta = document.getElementById("release-meta");
  const newer = document.getElementById("rel-newer");
  const older = document.getElementById("rel-older");
  const win = document.getElementById("dl-windows");
  const mac = document.getElementById("dl-darwin");
  const android = document.getElementById("dl-android");
  const ios = document.getElementById("dl-ios");
  if (!meta) return;

  if (newer) newer.disabled = releaseFailed || releaseIndex <= 0;
  if (older) older.disabled = releaseFailed || releaseIndex >= releases.length - 1;

  if (releaseFailed) {
    meta.innerHTML = text("releaseFail");
    paintBtn(win);
    paintBtn(mac);
    paintBtn(android);
    paintBtn(ios);
    return;
  }

  if (!releases.length) {
    meta.textContent = text("releaseLoading");
    paintBtn(win);
    paintBtn(mac);
    paintBtn(android);
    paintBtn(ios);
    return;
  }

  const latestRelease = releases[releaseIndex];
  const tag = latestRelease.tag_name || "latest";
  const date = latestRelease.published_at
    ? new Date(latestRelease.published_at).toLocaleDateString(locale)
    : "";
  const page = text("releasePage")
    .replace("{n}", String(releaseIndex + 1))
    .replace("{total}", String(releases.length));
  meta.textContent = date
    ? text("releaseVersion")
        .replace("{tag}", tag)
        .replace("{date}", date)
        .replace("{page}", page)
    : text("releaseVersionOnly").replace("{tag}", tag).replace("{page}", page);

  const assets = latestRelease.assets || [];
  const exe = pickWindows(assets);
  const pkg = pickMac(assets);
  const apk = pickAndroid(assets);
  const ipa = pickIos(assets);

  paintBtn(win, exe);
  paintBtn(mac, pkg);
  paintBtn(android, apk);
  paintBtn(ios, ipa);

  if (!exe && !pkg && !apk && !ipa) {
    meta.textContent = text("releaseNoAsset").replace("{tag}", tag);
  }
}

async function loadRelease() {
  try {
    const res = await fetch(
      `https://api.github.com/repos/${REPO}/releases?per_page=30`,
    );
    if (!res.ok) throw new Error(String(res.status));
    const list = await res.json();
    releases = (Array.isArray(list) ? list : []).filter((r) => r && !r.draft);
    releaseIndex = 0;
    if (!releases.length) throw new Error("empty");
  } catch {
    releaseFailed = true;
  }
  paintRelease();
}

let startLocale = "zh-CN";
try {
  const saved = localStorage.getItem("prism-lang");
  if (saved && messages[saved]) startLocale = saved;
} catch {
  /* ignore */
}
applyLocale(startLocale);

document.querySelectorAll("[data-lang]").forEach((btn) => {
  btn.addEventListener("click", () => {
    const next = btn.getAttribute("data-lang");
    try {
      localStorage.setItem("prism-lang", next);
    } catch {
      /* ignore */
    }
    applyLocale(next);
  });
});

document.getElementById("rel-newer")?.addEventListener("click", () => {
  if (releaseIndex > 0) {
    releaseIndex -= 1;
    paintRelease();
  }
});

document.getElementById("rel-older")?.addEventListener("click", () => {
  if (releaseIndex < releases.length - 1) {
    releaseIndex += 1;
    paintRelease();
  }
});

loadRelease();

const topBar = document.querySelector(".top");
if (topBar) {
  const syncTopOffset = () => {
    document.documentElement.style.setProperty(
      "--top-offset",
      Math.ceil(topBar.getBoundingClientRect().height) + "px",
    );
  };
  syncTopOffset();
  new ResizeObserver(syncTopOffset).observe(topBar);
}
