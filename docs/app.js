const REPO = "penndev/socks5";
const RELEASES = `https://github.com/${REPO}/releases`;

let locale = "zh-CN";
let latestRelease = null;
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

function paintRelease() {
  const meta = document.getElementById("release-meta");
  const win = document.getElementById("dl-windows");
  const mac = document.getElementById("dl-darwin");
  if (!meta || !win || !mac) return;

  if (releaseFailed) {
    meta.innerHTML = text("releaseFail");
    win.textContent = text("goReleases");
    mac.textContent = text("goReleases");
    win.href = RELEASES;
    mac.href = RELEASES;
    return;
  }

  if (!latestRelease) {
    meta.textContent = text("releaseLoading");
    win.textContent = text("goReleases");
    mac.textContent = text("goReleases");
    return;
  }

  const tag = latestRelease.tag_name || "latest";
  const date = latestRelease.published_at
    ? new Date(latestRelease.published_at).toLocaleDateString(locale)
    : "";
  meta.textContent = date
    ? text("releaseVersion").replace("{tag}", tag).replace("{date}", date)
    : text("releaseVersionOnly").replace("{tag}", tag);

  const assets = latestRelease.assets || [];
  const exe = pickAsset(
    assets,
    (name) => name.includes("windows") && name.endsWith(".exe"),
  );
  const dmg = pickAsset(
    assets,
    (name) => name.includes("darwin") && name.endsWith(".dmg"),
  );

  if (exe) {
    win.href = exe.browser_download_url;
    win.textContent = text("downloadFile").replace("{name}", exe.name);
  } else {
    win.href = RELEASES;
    win.textContent = text("goReleases");
  }
  if (dmg) {
    mac.href = dmg.browser_download_url;
    mac.textContent = text("downloadFile").replace("{name}", dmg.name);
  } else {
    mac.href = RELEASES;
    mac.textContent = text("goReleases");
  }
  if (!exe && !dmg) {
    meta.textContent = text("releaseNoAsset").replace("{tag}", tag);
  }
}

async function loadRelease() {
  try {
    const res = await fetch(
      `https://api.github.com/repos/${REPO}/releases/latest`,
    );
    if (!res.ok) throw new Error(String(res.status));
    latestRelease = await res.json();
  } catch {
    releaseFailed = true;
  }
  paintRelease();
}

applyLocale("zh-CN");

document.querySelectorAll("[data-lang]").forEach((btn) => {
  btn.addEventListener("click", () => applyLocale(btn.getAttribute("data-lang")));
});

loadRelease();
