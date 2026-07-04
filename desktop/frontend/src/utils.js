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

/** 为列表项生成稳定 __id，用于选择与延迟映射 */
export function withRuntimeIds(serverList) {
  const list = Array.isArray(serverList) ? serverList : [];
  return list.map((server) => {
    const __id = `${server.host}|${server.protocol}|${server.username}|${server.password}`;
    return { ...server, __id };
  });
}

/** 写入 storage 前去掉 __id 等运行时字段 */
export function stripForStorage(row) {
  if (!row) return row;
  const { id: _i, __id: _x, ...rest } = /** @type {any} */ (row);
  return rest;
}

/** 从 storage 读出的列表去掉运行时字段 */
export function stripServerListForStorage(raw) {
  return (Array.isArray(raw) ? raw : []).map(stripForStorage);
}