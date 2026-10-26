let locale = preferredLocale();

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
  document.documentElement.classList.remove("i18n-pending");
}

applyLocale(locale);

document.querySelector(".lang")?.addEventListener("click", (e) => {
  const next = e.target.closest("[data-lang]")?.dataset.lang;
  if (next !== "zh-CN" && next !== "en") return;
  saveLocale(next);
  applyLocale(next);
});

document.querySelectorAll(".copy-url").forEach((btn) => {
  btn.addEventListener("click", async () => {
    const url = btn.previousElementSibling?.href;
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
    } catch {
      const input = document.createElement("textarea");
      input.value = url;
      document.body.appendChild(input);
      input.select();
      document.execCommand("copy");
      input.remove();
    }
    const label = btn.querySelector("span");
    label.textContent = text("assetCopied");
    window.setTimeout(() => {
      label.textContent = text("assetCopy");
    }, 1200);
  });
});
