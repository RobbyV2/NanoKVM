export type Rect = { left: number; top: number; width: number; height: number };

// A selection in viewport CSS pixels -> fractions of the video frame, given
// where the frame is rendered. Null when it misses the picture.
export function selectionToCrop(sel: Rect, media: Rect) {
  if (media.width <= 0 || media.height <= 0) return null;
  const left = Math.max(sel.left, media.left);
  const top = Math.max(sel.top, media.top);
  const right = Math.min(sel.left + sel.width, media.left + media.width);
  const bottom = Math.min(sel.top + sel.height, media.top + media.height);
  if (right - left < 1 || bottom - top < 1) return null;
  return {
    x: (left - media.left) / media.width,
    y: (top - media.top) / media.height,
    w: (right - left) / media.width,
    h: (bottom - top) / media.height
  };
}

// L crop-lock: the transform (origin 0 0) that makes `sel` fill `area`,
// centred, as the extension did for the whole page. `element` is #screen's
// untransformed rect; `layoutScale` is its rendered/layout size ratio (the
// viewport's video scale), since translate is in the element's own units.
export function cropLockTransform(sel: Rect, element: Rect, area: Rect, layoutScale: number) {
  const w = Math.max(1, sel.width);
  const h = Math.max(1, sel.height);
  const scale = Math.min(area.width / w, area.height / h);
  if (!Number.isFinite(scale) || scale <= 0 || !(layoutScale > 0)) return null;
  const offX = area.left + (area.width - w * scale) / 2;
  const offY = area.top + (area.height - h * scale) / 2;
  const tx = (offX - element.left - (sel.left - element.left) * scale) / layoutScale;
  const ty = (offY - element.top - (sel.top - element.top) * scale) / layoutScale;
  return `translate(${tx}px, ${ty}px) scale(${scale})`;
}
