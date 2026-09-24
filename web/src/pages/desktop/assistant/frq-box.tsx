import { useEffect, useRef, useState } from 'react';
import { atom, useAtom, useAtomValue, useSetAtom } from 'jotai';

import * as ls from '@/lib/localstorage.ts';
import { keyboardLockAtom } from '@/jotai/keyboard.ts';

export const FRQ_LOCK_SOURCE = 'assistant-frq';

export const frqVisibleAtom = atom(false);
// rev bumps when the text is set from outside (an answer), so the editable is rewritten.
export const frqContentAtom = atom({ text: ls.getAssistantFrqText(), rev: 0 });

export const FrqBox = () => {
  const visible = useAtomValue(frqVisibleAtom);
  const [content, setContent] = useAtom(frqContentAtom);
  const setKeyboardLock = useSetAtom(keyboardLockAtom);
  const boxRef = useRef<HTMLDivElement>(null);
  const editableRef = useRef<HTMLDivElement>(null);
  const [initialRect] = useState(ls.getAssistantFrqRect);

  useEffect(() => {
    if (editableRef.current) editableRef.current.innerText = content.text;
    // Only external writes (rev) rewrite the editable; typing must keep the caret.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [content.rev]);

  useEffect(() => {
    const timer = setTimeout(() => ls.setAssistantFrqText(content.text), 500);
    return () => clearTimeout(timer);
  }, [content.text]);

  useEffect(() => {
    return () => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: false });
  }, [setKeyboardLock]);

  // Clicks on the remote screen preventDefault on mousedown, so focus would stay in
  // the editable and the keyboard lock would never be released. Blur on any
  // mousedown outside the box; the event itself is left untouched.
  useEffect(() => {
    const onDocDown = (e: MouseEvent) => {
      const editable = editableRef.current;
      if (!editable || document.activeElement !== editable) return;
      const box = boxRef.current;
      if (box && e.target instanceof Node && box.contains(e.target)) return;
      editable.blur();
    };
    document.addEventListener('mousedown', onDocDown, true);
    return () => document.removeEventListener('mousedown', onDocDown, true);
  }, []);

  // Hiding the box (display:none) does not reliably fire blur, so release the lock.
  useEffect(() => {
    if (visible) return;
    const editable = editableRef.current;
    if (editable && document.activeElement === editable) editable.blur();
    setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: false });
  }, [visible, setKeyboardLock]);

  // Drag on the box background (not the text, not the 8 px resize border).
  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    let dragging = false;
    let dx = 0;
    let dy = 0;
    const saveRect = () => {
      const r = el.getBoundingClientRect();
      ls.setAssistantFrqRect({
        left: r.left,
        top: r.top,
        width: el.offsetWidth,
        height: el.offsetHeight
      });
    };
    const onDown = (e: MouseEvent) => {
      if (e.target !== el || e.button !== 0) return;
      const r = el.getBoundingClientRect();
      const border = 8;
      if (
        e.clientX - r.left < border ||
        r.right - e.clientX < border ||
        e.clientY - r.top < border ||
        r.bottom - e.clientY < border
      ) {
        return;
      }
      dragging = true;
      dx = e.clientX - r.left;
      dy = e.clientY - r.top;
      document.body.style.userSelect = 'none';
      e.preventDefault();
    };
    const onMove = (e: MouseEvent) => {
      if (!dragging) return;
      el.style.left = `${e.clientX - dx}px`;
      el.style.top = `${e.clientY - dy}px`;
      el.style.transform = 'none';
    };
    const onUp = () => {
      if (!dragging) return;
      dragging = false;
      document.body.style.userSelect = '';
      saveRect();
    };
    const resize = new ResizeObserver(() => {
      if (el.offsetWidth > 0) saveRect();
    });
    el.addEventListener('mousedown', onDown);
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    resize.observe(el);
    return () => {
      el.removeEventListener('mousedown', onDown);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      resize.disconnect();
    };
  }, []);

  return (
    <div
      id="frq-box"
      ref={boxRef}
      style={{
        position: 'fixed',
        left: initialRect ? `${initialRect.left}px` : '50%',
        top: initialRect ? `${initialRect.top}px` : '10%',
        transform: initialRect ? 'none' : 'translateX(-50%)',
        width: `${initialRect?.width ?? 340}px`,
        height: `${initialRect?.height ?? 120}px`,
        maxWidth: '80vw',
        maxHeight: '70vh',
        overflow: 'auto',
        resize: 'both',
        border: '2px solid rgba(40,62,159,0.25)',
        background: 'none',
        color: '#111',
        fontSize: '15px',
        fontFamily: 'sans-serif',
        zIndex: 10004,
        padding: '18px 18px 18px 18px',
        borderRadius: '10px',
        boxShadow: '0 2px 16px 0 rgba(40,62,159,0.07)',
        backdropFilter: 'none',
        pointerEvents: 'auto',
        userSelect: 'text',
        boxSizing: 'content-box',
        display: visible ? 'block' : 'none'
      }}
    >
      <div
        id="frq-editable"
        ref={editableRef}
        contentEditable
        suppressContentEditableWarning
        style={{
          width: '100%',
          height: '100%',
          outline: 'none',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          fontFamily: 'sans-serif'
        }}
        onFocus={() => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: true })}
        onBlur={() => setKeyboardLock({ source: FRQ_LOCK_SOURCE, locked: false })}
        onInput={(e) => {
          const text = e.currentTarget.innerText || '';
          setContent((c) => ({ text, rev: c.rev }));
        }}
      />
    </div>
  );
};
