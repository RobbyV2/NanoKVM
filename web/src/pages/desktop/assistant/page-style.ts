import { useEffect, useRef } from 'react';
import { useSetAtom } from 'jotai';

import { menuCloseSignalAtom } from '@/jotai/settings.ts';

const PAGE_CSS = `
    * {
      scrollbar-width: none !important;
      -ms-overflow-style: none !important;
    }
    *::-webkit-scrollbar {
      display: none !important;
      width: 0 !important;
      height: 0 !important;
    }
    div.flex.h-screen.w-screen.items-start.justify-center.xl\\:items-center {
      background-color: white !important;
    }
    #screen-viewport[data-cropped="false"] #screen {
      height: 100vh;
      object-fit: contain !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable {
      background-color: white !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable .bg-neutral-800\\/80 {
      background-color: #fafafa !important;
    }
    div.fixed.left-1\\/2.top-\\[10px\\].z-\\[1000\\].-translate-x-1\\/2.react-draggable .bg-neutral-800\\/50 {
      background-color: #fafafa !important;
    }
  `;

// UI on: the extension's page CSS, and the menu bar closed once, 2 s after load.
export function usePageStyle(ui: boolean) {
  const closeMenu = useSetAtom(menuCloseSignalAtom);
  const closedOnce = useRef(false);

  useEffect(() => {
    if (!ui) return;
    const style = document.createElement('style');
    style.textContent = PAGE_CSS;
    document.head.appendChild(style);
    return () => style.remove();
  }, [ui]);

  useEffect(() => {
    if (!ui || closedOnce.current) return;
    const timer = setTimeout(() => {
      closedOnce.current = true;
      closeMenu((n) => n + 1);
    }, 2000);
    return () => clearTimeout(timer);
  }, [ui, closeMenu]);
}
