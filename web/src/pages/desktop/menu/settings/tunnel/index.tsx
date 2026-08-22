import { useEffect, useState } from 'react';
import { Button, Divider, Input, Tooltip } from 'antd';
import {
  CirclePlayIcon,
  CircleStopIcon,
  LoaderCircleIcon,
  LoaderIcon,
  RefreshCcwIcon,
  RotateCwIcon,
  TriangleAlertIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/tunnel.ts';
import { ScrollArea } from '@/components/ui/scroll-area.tsx';

import { BinarySource } from './binary.tsx';
import { EnvTable } from './env.tsx';
import { reportsConnection } from './types.ts';
import type { EnvEntry, TunnelService, TunnelState, TunnelStatus } from './types.ts';

type TunnelProps = {
  service: TunnelService;
  setIsLocked: (isLocked: boolean) => void;
};

type Action = '' | 'start' | 'stop' | 'restart' | 'save';

const lifecycle = {
  start: api.start,
  stop: api.stop,
  restart: api.restart
};

const stateColors: Record<TunnelState, string> = {
  notInstall: 'text-neutral-500',
  notConfigured: 'text-neutral-500',
  stopped: 'text-neutral-400',
  running: 'text-blue-500',
  connected: 'text-green-500',
  error: 'text-red-500'
};

export const Tunnel = ({ service, setIsLocked }: TunnelProps) => {
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const [action, setAction] = useState<Action>('');
  const [status, setStatus] = useState<TunnelStatus>();
  const [args, setArgs] = useState('');
  const [env, setEnv] = useState<EnvEntry[]>([]);
  const [logs, setLogs] = useState<string[]>([]);
  const [isSaved, setIsSaved] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  const state = status?.state;
  const isRunning = state === 'running' || state === 'connected';

  // running is the last state a service with no health signal can reach, so its
  // blue is the end of the road rather than a way station to a green it can
  // never show
  const isBlindRunning = state === 'running' && !reportsConnection[service];
  const isOpenServer =
    service === 'wstunnel' &&
    /^server(\s|$)/.test(args.trim()) &&
    !/(^|\s)(--restrict-config|-r)(=|\s|$)/.test(args);

  useEffect(() => {
    load();
  }, [service]);

  function load() {
    if (isLoading) return;
    setIsLoading(true);

    Promise.all([api.getStatus(service), api.getConfig(service), api.getLogs(service)])
      .then(([statusRsp, configRsp, logsRsp]) => {
        if (statusRsp.code !== 0) {
          setErrMsg(statusRsp.msg);
          return;
        }

        // the server seeds the keys a service expects from its own spec, so
        // the client does not carry a second copy of that list
        setStatus(statusRsp.data);
        setArgs(configRsp.data?.args ?? '');
        setEnv(configRsp.data?.env ?? []);
        setLogs(logsRsp.data?.lines ?? []);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get tunnel status');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function getStatus() {
    if (isLoading) return;
    setIsLoading(true);

    Promise.all([api.getStatus(service), api.getLogs(service)])
      .then(([statusRsp, logsRsp]) => {
        if (statusRsp.code !== 0) {
          setErrMsg(statusRsp.msg);
          return;
        }

        setStatus(statusRsp.data);
        setLogs(logsRsp.data?.lines ?? []);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get tunnel status');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function save() {
    if (isLoading) return;
    setIsLoading(true);
    setAction('save');
    setIsSaved(false);
    setErrMsg('');

    const values: Record<string, string> = {};
    env.forEach(({ key, value }) => {
      const name = key.trim();
      if (name) {
        values[name] = value;
      }
    });

    api
      .setConfig(service, args, values)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setIsSaved(true);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to save config');
      })
      .finally(() => {
        setAction('');
        setIsLoading(false);
        load();
      });
  }

  function run(next: 'start' | 'stop' | 'restart') {
    if (isLoading) return;
    setIsLoading(true);
    setAction(next);
    setErrMsg('');

    lifecycle[next](service)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to update tunnel');
      })
      .finally(() => {
        setAction('');
        setIsLoading(false);
        getStatus();
      });
  }

  if (isLoading && !status) {
    return (
      <div className="flex w-full items-center justify-center space-x-2 pt-5 text-neutral-500">
        <LoaderCircleIcon className="animate-spin" size={18} />
        <span>{t('settings.tunnel.loading')}</span>
      </div>
    );
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex items-baseline space-x-3">
          <span className="text-base">{t(`settings.${service}.title`)}</span>
          {state && (
            <span className={`text-xs ${stateColors[state]}`}>{t(`settings.tunnel.${state}`)}</span>
          )}
        </div>

        <div className="flex items-center space-x-2">
          <Tooltip title={t('settings.tunnel.start')} placement="bottom">
            <div
              className="flex cursor-pointer rounded p-1 text-green-500 hover:bg-neutral-600 hover:text-green-500/80"
              onClick={() => run('start')}
            >
              {action === 'start' ? (
                <LoaderIcon className="animate-spin" size={18} />
              ) : (
                <CirclePlayIcon size={18} />
              )}
            </div>
          </Tooltip>

          <Tooltip title={t('settings.tunnel.restart')} placement="bottom">
            <div
              className="flex cursor-pointer rounded p-1 text-blue-500 hover:bg-neutral-600 hover:text-blue-500/80"
              onClick={() => run('restart')}
            >
              {action === 'restart' ? (
                <LoaderIcon className="animate-spin" size={18} />
              ) : (
                <RotateCwIcon size={18} />
              )}
            </div>
          </Tooltip>

          <Tooltip title={t('settings.tunnel.stop')} placement="bottom">
            <div
              className="flex cursor-pointer rounded p-1 text-red-500 hover:bg-neutral-600 hover:text-red-500/80"
              onClick={() => run('stop')}
            >
              {action === 'stop' ? (
                <LoaderIcon className="animate-spin" size={18} />
              ) : (
                <CircleStopIcon size={18} />
              )}
            </div>
          </Tooltip>
        </div>
      </div>

      {status?.message && <div className="pt-2 text-xs text-red-500">{status.message}</div>}

      {isBlindRunning && (
        <div className="flex items-start space-x-1 pt-2 text-xs text-neutral-500">
          <TriangleAlertIcon size={13} className="mt-[2px] shrink-0" />
          <span>{t('settings.tunnel.noHealthSignal')}</span>
        </div>
      )}

      {isRunning && (
        <div className="flex items-center space-x-1 pt-2 text-xs text-neutral-500">
          <TriangleAlertIcon size={13} />
          <span>{t('settings.tunnel.memoryWarning')}</span>
        </div>
      )}

      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-2">
        <div className="flex items-baseline space-x-2">
          <span>{t('settings.tunnel.arguments')}</span>
          <span className="text-xs text-neutral-500">{t('settings.tunnel.argumentsTip')}</span>
        </div>

        <Input.TextArea
          value={args}
          rows={3}
          spellCheck={false}
          onChange={(e) => {
            setArgs(e.target.value);
            setIsSaved(false);
          }}
        />

        {isOpenServer && (
          <div className="flex items-center space-x-1 text-xs text-amber-500">
            <TriangleAlertIcon size={13} />
            <span>{t('settings.tunnel.serverWarning')}</span>
          </div>
        )}
      </div>

      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-2">
        <span>{t('settings.tunnel.env')}</span>

        <EnvTable
          entries={env}
          onChange={(entries) => {
            setEnv(entries);
            setIsSaved(false);
          }}
        />
      </div>

      <div className="flex items-center space-x-3 pt-5">
        <Button type="primary" loading={action === 'save'} onClick={save}>
          {t('settings.tunnel.save')}
        </Button>

        {isSaved && <span className="text-xs text-green-500">{t('settings.tunnel.saved')}</span>}
        {errMsg && <span className="text-red-500">{errMsg}</span>}
      </div>

      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-2">
        <div className="flex items-center justify-between">
          <span>{t('settings.tunnel.logs')}</span>

          <Tooltip title={t('settings.tunnel.refresh')} placement="bottom">
            <div
              className="flex cursor-pointer rounded p-1 text-neutral-400 hover:bg-neutral-700/50 hover:text-white"
              onClick={getStatus}
            >
              <RefreshCcwIcon size={15} />
            </div>
          </Tooltip>
        </div>

        {logs.length === 0 ? (
          <span className="text-xs text-neutral-500">{t('settings.tunnel.logsEmpty')}</span>
        ) : (
          <ScrollArea className="h-[220px] rounded bg-neutral-800/60 p-2">
            <div className="flex flex-col font-mono text-xs text-neutral-300">
              {logs.map((line, index) => (
                <span key={index} className="whitespace-pre-wrap break-all">
                  {line}
                </span>
              ))}
            </div>
          </ScrollArea>
        )}
      </div>

      <Divider className="opacity-50" />

      <BinarySource
        service={service}
        custom={status?.custom === true}
        running={isRunning}
        setIsLocked={setIsLocked}
        onSuccess={getStatus}
      />
    </>
  );
};
