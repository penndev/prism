import { ref, nextTick } from "vue";

// 贴在底部时跟随新日志。滚离底部，或面板里有选中文字时，停在原地。
export function useStickyLog() {
  const el = ref(null);
  const stick = ref(true);

  function onScroll() {
    const node = el.value;
    if (!node) return;
    stick.value = node.scrollHeight - node.scrollTop - node.clientHeight < 8;
  }

  function follow() {
    if (!stick.value) return;
    nextTick(() => {
      const node = el.value;
      if (!node || !stick.value) return;
      const sel = window.getSelection();
      if (sel && !sel.isCollapsed && node.contains(sel.anchorNode)) return;
      node.scrollTop = node.scrollHeight;
    });
  }

  function reset() {
    stick.value = true;
  }

  return { el, onScroll, follow, reset };
}
