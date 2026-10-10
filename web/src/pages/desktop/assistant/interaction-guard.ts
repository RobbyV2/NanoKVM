// Assistant gestures belong to the remote picture. Local controls (including
// portals) must retain their normal keyboard and mouse behavior.
const LOCAL_OVERLAYS = [
  '[role="dialog"]',
  '.ant-modal-wrap',
  '.ant-drawer-content',
  '.ant-popover',
  '.ant-dropdown',
  '.ant-select-dropdown',
  '.ant-picker-dropdown',
  '.chaice-overlay'
].join(', ');

type Target = { closest: (selector: string) => unknown };
type Overlay = { getClientRects: () => { length: number } };
type Page = {
  body: unknown;
  documentElement: unknown;
  activeElement: unknown;
  querySelectorAll: (selector: string) => ArrayLike<Overlay>;
};

export function isRemoteScreen(target: unknown): boolean {
  return !!(target as Target | null)?.closest?.('#screen');
}

export function hasLocalOverlay(page: Page): boolean {
  return Array.from(page.querySelectorAll(LOCAL_OVERLAYS)).some(
    (element) => element.getClientRects().length > 0
  );
}

export function allowAssistantKeyboard(page: Page, target: unknown, localInteraction: boolean) {
  const isDesktop = (element: unknown) =>
    element == null ||
    element === page.body ||
    element === page.documentElement ||
    isRemoteScreen(element);
  return (
    !localInteraction &&
    isDesktop(target) &&
    isDesktop(page.activeElement) &&
    !hasLocalOverlay(page)
  );
}
