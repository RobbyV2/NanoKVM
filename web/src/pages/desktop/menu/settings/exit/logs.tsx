import { useState } from 'react';
import { Button, Collapse, Tooltip } from 'antd';
import { RefreshCcwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/exit.ts';
import { ScrollArea } from '@/components/ui/scroll-area.tsx';

import type { ExitMode, ExitLogs as Logs } from './types.ts';

type ExitLogsProps = {
  slot: string;
  mode: ExitMode;
};

const Tail = ({ title, lines }: { title: string; lines: string[] }) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col space-y-1">
      <span className="text-xs text-neutral-400">{title}</span>

      {lines.length === 0 ? (
        <span className="text-xs text-neutral-500">{t('settings.exit.logs.empty')}</span>
      ) : (
        <ScrollArea className="h-[160px] rounded bg-neutral-800/60 p-2">
          <div className="flex flex-col font-mono text-xs text-neutral-300">
            {lines.map((line, index) => (
              <span key={index} className="whitespace-pre-wrap break-all">
                {line}
              </span>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
};

// fetched on expand and on demand, not polled: the tails are the daemons'
// files, and the token-shaped values in them are already redacted server-side
export const ExitLogs = ({ slot, mode }: ExitLogsProps) => {
  const { t } = useTranslation();

  const [logs, setLogs] = useState<Logs>();
  const [isLoading, setIsLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function getLogs() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .getLogs(slot)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setLogs({ hev: rsp.data?.hev ?? [], wstunnel: rsp.data?.wstunnel ?? [] });
        setErrMsg('');
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get exit logs');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  const body = (
    <div className="flex flex-col space-y-3 pt-3">
      <div className="flex justify-end">
        <Tooltip title={t('settings.exit.logs.refresh')} placement="bottom">
          {/* a button, not a div: reachable by keyboard and announced by name */}
          <Button
            type="text"
            size="small"
            aria-label={t('settings.exit.logs.refresh')}
            loading={isLoading}
            icon={<RefreshCcwIcon size={15} />}
            onClick={getLogs}
          />
        </Tooltip>
      </div>

      <Tail title={t('settings.exit.logs.hev')} lines={logs?.hev ?? []} />
      {mode === 'wstunnel' && (
        <Tail title={t('settings.exit.logs.wstunnel')} lines={logs?.wstunnel ?? []} />
      )}

      {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}
    </div>
  );

  return (
    <Collapse
      ghost
      onChange={(keys) => {
        if (keys.length > 0 && !logs) getLogs();
      }}
      items={[
        {
          key: 'logs',
          label: <span className="text-neutral-200">{t('settings.exit.logs.title')}</span>,
          children: body
        }
      ]}
    />
  );
};
