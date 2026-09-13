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
  document.querySelector('meta[name="description"]')?.setAttribute(
    "content",
    text("description"),
  );

  document.querySelectorAll("[data-i18n]").forEach((el) => {
    const s = text(el.dataset.i18n);
    if (/<[a-z/]/i.test(s)) el.innerHTML = s;
    else el.textContent = s;
  });
  document.querySelectorAll("[data-lang]").forEach((btn) => {
    const on = btn.dataset.lang === locale;
    btn.classList.toggle("is-active", on);
    btn.setAttribute("aria-pressed", String(on));
  });
  paintRelease();
  document.documentElement.classList.remove("i18n-pending");
}

function findAsset(assets, ...tests) {
  for (const test of tests) {
    const hit = (assets || []).find((a) => test(a.name || ""));
    if (hit) return hit;
  }
}

function paintBtn(id, asset) {
  const el = document.getElementById(id);
  if (!el) return;
  el.href = asset ? asset.browser_download_url : RELEASES;
  el.textContent = asset
    ? text("downloadFile").replace("{name}", asset.name)
    : text("goReleases");
}

function paintRelease() {
  const meta = document.getElementById("release-meta");
  const newer = document.getElementById("rel-newer");
  const older = document.getElementById("rel-older");
  if (!meta) return;

  const rel = releases[releaseIndex];
  const assets = rel?.assets;
  const win = findAsset(
    assets,
    (n) => /windows/i.test(n) && n.endsWith("-installer.exe"),
    (n) => /installer/i.test(n) && n.endsWith(".exe"),
    (n) => /windows/i.test(n) && n.endsWith(".exe"),
  );
  const mac = findAsset(
    assets,
    (n) => /darwin/i.test(n) && n.endsWith(".pkg"),
    (n) => /darwin/i.test(n) && n.endsWith(".dmg"),
    (n) => /darwin/i.test(n) && n.endsWith(".zip"),
  );
  const apk = findAsset(assets, (n) => n.toLowerCase().endsWith(".apk"));
  const ipa = findAsset(assets, (n) => n.toLowerCase().endsWith(".ipa"));

  if (newer) newer.disabled = releaseFailed || releaseIndex <= 0;
  if (older) older.disabled = releaseFailed || releaseIndex >= releases.length - 1;
  paintBtn("dl-windows", win);
  paintBtn("dl-darwin", mac);
  paintBtn("dl-android", apk);
  paintBtn("dl-ios", ipa);

  if (releaseFailed) {
    meta.innerHTML = text("releaseFail");
    return;
  }
  if (!rel) {
    meta.textContent = text("releaseLoading");
    return;
  }

  const tag = rel.tag_name || "latest";
  if (!win && !mac && !apk && !ipa) {
    meta.textContent = text("releaseNoAsset").replace("{tag}", tag);
    return;
  }

  const page = text("releasePage")
    .replace("{n}", String(releaseIndex + 1))
    .replace("{total}", String(releases.length));
  const date = rel.published_at
    ? new Date(rel.published_at).toLocaleDateString(locale)
    : "";
  meta.textContent = date
    ? text("releaseVersion")
        .replace("{tag}", tag)
        .replace("{date}", date)
        .replace("{page}", page)
    : text("releaseVersionOnly").replace("{tag}", tag).replace("{page}", page);
}

async function loadRelease() {
  try {
    const res = await fetch(`https://api.github.com/repos/${REPO}/releases?per_page=30`);
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

try {
  const saved = localStorage.getItem("prism-lang");
  if (saved && messages[saved]) locale = saved;
} catch {
  /* ignore */
}
applyLocale(locale);

document.querySelector(".lang")?.addEventListener("click", (e) => {
  const next = e.target.closest("[data-lang]")?.dataset.lang;
  if (!next) return;
  try {
    localStorage.setItem("prism-lang", next);
  } catch {
    /* ignore */
  }
  applyLocale(next);
});

document.querySelector(".pager")?.addEventListener("click", (e) => {
  const id = e.target.closest("button")?.id;
  if (id === "rel-newer" && releaseIndex > 0) releaseIndex -= 1;
  else if (id === "rel-older" && releaseIndex < releases.length - 1) releaseIndex += 1;
  else return;
  paintRelease();
});

loadRelease();
