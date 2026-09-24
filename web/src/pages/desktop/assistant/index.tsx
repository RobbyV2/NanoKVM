import { useEffect } from 'react';
import { useAuth } from '@/contexts/auth.ts';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { createPortal } from 'react-dom';

import * as api from '@/api/assistant.ts';
import type { AssistantConfig } from '@/api/assistant.ts';
import { assistantConfigAtom, assistantContextCountAtom } from '@/jotai/assistant.ts';
import { keyboardLockSourcesAtom } from '@/jotai/keyboard.ts';

import { useAssistantActions } from './actions.ts';
import { FreezeOverlay } from './freeze.tsx';
import { FRQ_LOCK_SOURCE, FrqBox } from './frq-box.tsx';
import { usePageStyle } from './page-style.ts';
import { useQuadClick } from './quad-click.ts';
import { Selection, useRequestSelection } from './selection.tsx';
import { StatusBar } from './status.tsx';
import { Toast } from './toast.tsx';
import { useHotkeys } from './use-hotkeys.ts';

export const Assistant = () => {
  const { account } = useAuth();
  const isAdmin = account.role === 'admin';
  const [config, setConfig] = useAtom(assistantConfigAtom);
  const setContextCount = useSetAtom(assistantContextCountAtom);

  useEffect(() => {
    if (!isAdmin) return;
    api
      .getAssistantConfig()
      .then((rsp) => {
        if (rsp.code === 0) setConfig(rsp.data);
      })
      .catch(() => undefined);
  }, [isAdmin, setConfig]);

  useEffect(() => {
    if (!config?.enabled) return;
    api
      .getContextCount()
      .then((rsp) => {
        if (rsp.code === 0) setContextCount(rsp.data.count);
      })
      .catch(() => undefined);
  }, [config?.enabled, setContextCount]);

  if (!isAdmin || !config?.enabled) return null;
  return <AssistantRuntime config={config} />;
};

const AssistantRuntime = ({ config }: { config: AssistantConfig }) => {
  const requestSelection = useRequestSelection();
  const actions = useAssistantActions(requestSelection);
  const lockSources = useAtomValue(keyboardLockSourcesAtom);
  const hotkeysAllowed = [...lockSources].every((source) => source === FRQ_LOCK_SOURCE);

  useHotkeys(hotkeysAllowed, actions);
  useQuadClick(config.quadClickMCQ, actions.quadClick);
  usePageStyle(config.ui);
  useEffect(() => () => actions.teardown(), [actions]);

  return createPortal(
    <>
      {config.ui && <StatusBar />}
      <Toast />
      <FreezeOverlay />
      <FrqBox />
      <Selection />
    </>,
    document.body
  );
};
