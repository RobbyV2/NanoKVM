// Browser key events are not a reliable stream of matched pairs. The window can
// lose focus without a keyup (window-manager shortcuts, Alt menus, OS dialogs),
// and an input-method composition can end without a compositionend reaching the
// document. These helpers derive the true state from each new event so a lost
// event cannot leave a key held on the host or suppress keyboard input.

export interface ModifierState {
  readonly ctrlKey: boolean;
  readonly shiftKey: boolean;
  readonly altKey: boolean;
  readonly metaKey: boolean;
  getModifierState(key: string): boolean;
}

const MODIFIER_FAMILIES: { codes: readonly string[]; held: (event: ModifierState) => boolean }[] = [
  { codes: ['ControlLeft', 'ControlRight'], held: (event) => event.ctrlKey },
  { codes: ['ShiftLeft', 'ShiftRight'], held: (event) => event.shiftKey },
  {
    codes: ['AltLeft', 'AltRight'],
    // Linux and Windows report AltGr as AltGraph rather than Alt.
    held: (event) => event.altKey || event.getModifierState('AltGraph')
  },
  { codes: ['MetaLeft', 'MetaRight'], held: (event) => event.metaKey }
];

/**
 * Modifiers recorded as pressed that the event reports as released. The family
 * of the key that produced the event is skipped, because browsers disagree on
 * whether a modifier's own keydown/keyup already includes its new state.
 */
export function releasedModifiers(
  pressed: Iterable<string>,
  event: ModifierState,
  eventCode: string
): string[] {
  const pressedSet = new Set(pressed);
  const released: string[] = [];

  for (const family of MODIFIER_FAMILIES) {
    if (family.codes.includes(eventCode) || family.held(event)) continue;
    for (const code of family.codes) {
      if (pressedSet.has(code)) released.push(code);
    }
  }

  return released;
}

export interface CompositionEvent {
  readonly isComposing: boolean;
  readonly keyCode: number;
  readonly key: string;
}

// keyCode 229 / key "Process" marks a key the input method is consuming.
const IME_KEY_CODE = 229;

/**
 * Whether a key event belongs to an input-method composition. A keydown that is
 * neither flagged as composing nor addressed to the input method proves the
 * composition is over even if its compositionend was never delivered.
 */
export function isCompositionKey(compositionStarted: boolean, event: CompositionEvent): boolean {
  if (event.isComposing) return true;
  if (!compositionStarted) return false;
  return event.keyCode === IME_KEY_CODE || event.key === 'Process';
}
