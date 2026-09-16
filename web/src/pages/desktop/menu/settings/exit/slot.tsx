import { useEffect, useRef, useState } from 'react';
import { Divider, Popconfirm, Segmented, Switch } from 'antd';
import { TriangleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/exit.ts';

import { ExitAdvanced } from './advanced.tsx';
import { ExitCommands } from './commands.tsx';
import { ExitLogs } from './logs.tsx';
import { isScriptServed } from './state.ts';
import { ExitStatusCard } from './status.tsx';
import { ExitToken } from './token.tsx';
import { exitModes } from './types.ts';
import type { ExitCommands as Commands, ExitConfig, ExitMode, ExitStatus } from './types.ts';

type ExitSlotProps = {
  initial: ExitStatus;
  showSlot: boolean;
  setIsLocked: (isLocked: boolean) => void;
};

type ExitAction = '' | 'enable' | 'disable' | 'regenerate' | 'disconnect' | 'mode';

const pollInterval = 3000;

export const ExitSlot = ({ initial, showSlot, setIsLocked }: ExitSlotProps) => {
  const { t } = useTranslation();
  const slot = initial.slot;

  const [status, setStatus] = useState<ExitStatus>(initial);
  const [config, setConfig] = useState<ExitConfig>();
  const [commands, setCommands] = useState<Commands>();
  const [action, setAction] = useState<ExitAction>('');
  const [isStale, setIsStale] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  // one status request in flight at a time: a slow device must not pile up
  // polls, and a poll that lands after unmount must not touch state
  const isPolling = useRef(false);
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;
    getConfig();
    getCommands();

    const timer = setInterval(getStatus, pollInterval);
    return () => {
      isMounted.current = false;
      clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slot]);

  function getStatus() {
    if (isPolling.current) return;
    isPolling.current = true;

    api
      .getStatus(slot)
      .then((rsp) => {
        if (!isMounted.current) return;
        if (rsp.code !== 0) {
          setIsStale(true);
          return;
        }

        setStatus(rsp.data);
        setIsStale(false);
      })
      .catch(() => {
        if (isMounted.current) setIsStale(true);
      })
      .finally(() => {
        isPolling.current = false;
      });
  }

  function getConfig() {
    api
      .getConfig(slot)
      .then((rsp) => {
        if (!isMounted.current) return;
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setConfig(rsp.data);
      })
      .catch((err) => {
        if (isMounted.current) setErrMsg(err?.message || 'Failed to get exit config');
      });
  }

  function getCommands() {
    api
      .getCommands(slot)
      .then((rsp) => {
        if (!isMounted.current) return;
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setCommands(rsp.data);
      })
      .catch((err) => {
        if (isMounted.current) setErrMsg(err?.message || 'Failed to get exit commands');
      });
  }

  // enable and disable re-enumerate the gadget (D5); the settings modal is
  // locked for the few seconds the transaction takes
  function toggle() {
    if (action !== '') return;
    const next: ExitAction = status.enabled ? 'disable' : 'enable';
    setAction(next);
    setErrMsg('');
    setIsLocked(true);

    (next === 'enable' ? api.enable(slot) : api.disable(slot))
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to update exit tunnel');
      })
      .finally(() => {
        setIsLocked(false);
        setAction('');
        getStatus();
      });
  }

  function regenerate() {
    if (action !== '') return;
    setAction('regenerate');
    setErrMsg('');

    api
      .regenerateToken(slot)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to regenerate token');
      })
      .finally(() => {
        setAction('');
        getStatus();
        // the commands carry the token
        getCommands();
      });
  }

  function disconnect() {
    if (action !== '') return;
    setAction('disconnect');
    setErrMsg('');

    api
      .disconnect(slot)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to disconnect exit');
      })
      .finally(() => {
        setAction('');
        getStatus();
      });
  }

  function selectMode(mode: ExitMode) {
    if (action !== '' || !config || config.mode === mode) return;
    setAction('mode');
    setErrMsg('');

    api
      .setConfig(slot, { mode })
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setConfig((current) => (current ? { ...current, mode } : current));
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to update exit config');
      })
      .finally(() => {
        setAction('');
        getStatus();
      });
  }

  const mode = config?.mode ?? status.mode;
  const isBusy = action !== '' || status.pending;
  // the manager switches mode live (restart of the daemons through S94exit,
  // restrict yaml rewritten, front door rewired); the price is that the
  // connected exit is dropped, which the hint under the selector says

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex items-baseline space-x-3">
          <span>{showSlot ? t('settings.exit.slot', { slot }) : t('settings.exit.enable')}</span>
          {status.pending && (
            <span className="text-xs text-blue-500">{t('settings.exit.pending')}</span>
          )}
          {isStale && (
            <span className="text-xs text-amber-500">{t('settings.exit.statusStale')}</span>
          )}
        </div>

        <Popconfirm
          placement="bottomRight"
          title={t(status.enabled ? 'settings.exit.disableConfirm' : 'settings.exit.enableConfirm')}
          description={
            <div className="max-w-[320px] text-xs text-neutral-400">
              {t('settings.exit.reenumerate')}
            </div>
          }
          okText={t('settings.exit.okBtn')}
          cancelText={t('settings.exit.cancelBtn')}
          disabled={isBusy}
          onConfirm={toggle}
        >
          <Switch
            checked={status.enabled}
            loading={action === 'enable' || action === 'disable' || status.pending}
          />
        </Popconfirm>
      </div>

      {status.message && <div className="pt-2 text-xs text-red-500">{status.message}</div>}
      {errMsg && <div className="pt-2 text-xs text-red-500">{errMsg}</div>}

      {status.enabled && !status.nic.protocol && (
        <div className="flex items-start space-x-1 pt-2 text-xs text-amber-500">
          <TriangleAlertIcon size={13} className="mt-[2px] shrink-0" />
          <span>{t('settings.exit.status.nicNone')}</span>
        </div>
      )}

      <Divider className="opacity-50" />

      <ExitStatusCard
        status={status}
        disconnecting={action === 'disconnect'}
        onDisconnect={disconnect}
      />

      <Divider className="opacity-50" />

      <ExitToken
        token={status.token}
        regenerating={action === 'regenerate'}
        onRegenerate={regenerate}
      />

      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-3">
        <div className="flex items-center justify-between space-x-10">
          <div className="flex flex-col space-y-1">
            <span>{t('settings.exit.mode.title')}</span>
            <span className="text-xs text-neutral-500">
              {mode === 'wstunnel'
                ? t('settings.exit.mode.wstunnelDesc', {
                    version: commands?.wstunnelVersion || 'wstunnel'
                  })
                : t('settings.exit.mode.nativeDesc')}
            </span>
          </div>

          <Segmented
            disabled={isBusy || !config}
            value={mode}
            onChange={(value) => selectMode(value as ExitMode)}
            options={exitModes.map((value) => ({
              label: t(`settings.exit.mode.${value}`),
              value
            }))}
          />
        </div>

        {status.enabled && (
          <span className="text-xs text-neutral-500">{t('settings.exit.mode.whileEnabled')}</span>
        )}
      </div>

      <Divider className="opacity-50" />

      <ExitCommands
        slot={slot}
        mode={mode}
        token={status.token}
        commands={commands}
        hasConnected={!!status.lastConnectedAt}
        scriptServed={isScriptServed(status)}
      />

      <Divider className="opacity-50" />

      <ExitAdvanced slot={slot} config={config} disabled={isBusy} onSaved={setConfig} />

      <Divider className="opacity-50" />

      <ExitLogs slot={slot} mode={mode} />
    </>
  );
};
