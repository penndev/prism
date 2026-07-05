export function debounce(fn, delay = 1000) {
  let timer = null

  return (...args) => {
    clearTimeout(timer)

    timer = setTimeout(() => {
      fn(...args)
    }, delay)
  }
}

/**
 * 在 document 上监听 mousemove/mouseup，按轴向拖拽更新数值（与 BottomBar 纵向拖拽同一模式）。
 * @param {MouseEvent} e
 * @param {{ axis: 'x' | 'y', startValue: number, min: number, max: number, onChange: (next: number) => void }} opts
 */
export function startAxisResize(e, opts) {
  const { axis, startValue, min, max, onChange } = opts;
  e.preventDefault();
  const startPrimary = axis === "x" ? e.clientX : e.clientY;

  function onMove(ev) {
    const primary = axis === "x" ? ev.clientX : ev.clientY;
    const delta =
      axis === "x" ? primary - startPrimary : startPrimary - primary;
    const next = Math.round(startValue + delta);
    onChange(Math.min(max, Math.max(min, next)));
  }

  function onUp() {
    document.removeEventListener("mousemove", onMove);
    document.removeEventListener("mouseup", onUp);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }

  document.body.style.cursor = axis === "x" ? "ew-resize" : "ns-resize";
  document.body.style.userSelect = "none";
  document.addEventListener("mousemove", onMove);
  document.addEventListener("mouseup", onUp);
}

/** 写入 storage 前剥离所有 _ 开头的运行时字段（如 _id、_latency） */
export function stripForStorage(row) {
  if (!row || typeof row !== "object") return row;
  return Object.fromEntries(
    Object.entries(row).filter(([key]) => !key.startsWith("_")),
  );
}

/** 拓展服务器 item：补全 _id（代理 URL）并保留 _latency 等运行时字段 */
export function extendServerItem(row) {
  const persisted = stripForStorage(row);
  const protocol = persisted.protocol.toLowerCase();
  const username = persisted.username || "";
  const password = persisted.password || "";
  return {
    ...persisted,
    _id: `${protocol}://${username}:${password}@${persisted.host}`,
    ...(row._latency !== undefined ? { _latency: row._latency } : {}),
  };
}