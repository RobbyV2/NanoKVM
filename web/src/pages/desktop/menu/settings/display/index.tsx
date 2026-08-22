import { useEffect, useState } from 'react';
import { Button, Divider } from 'antd';
import { DownloadIcon, HistoryIcon, LoaderCircleIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/edid.ts';
import type { EdidResult, EdidStatus } from '@/api/edid.ts';

import { Preset } from './preset.tsx';
import { Recovery } from './recovery.tsx';
import { Summary } from './summary.tsx';

type DisplayProps = {
  setIsLocked: (isLocked: boolean) => void;
};

export const Display = ({ setIsLocked }: DisplayProps) => {
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [status, setStatus] = useState<EdidStatus>();
  const [result, setResult] = useState<EdidResult>();
  const [errMsg, setErrMsg] = useState('');

  const backup = status?.backups?.[0];

  // the flash left the edid region half written, and only a restore leaves it known again
  const needsRecovery = result?.state === 'needs_recovery';
  const canRestore = needsRecovery || !!backup || !!status?.factoryAvailable;

  function restored() {
    setResult(undefined);
    getStatus();
  }

  useEffect(() => {
    getStatus();
  }, []);

  function getStatus() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .getEdid()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setStatus(rsp.data);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get edid');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function download(id?: string) {
    if (isDownloading) return;
    setIsDownloading(true);
    setErrMsg('');

    (id ? api.downloadBackup(id) : api.downloadEdid())
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to download edid');
      })
      .finally(() => {
        setIsDownloading(false);
      });
  }

  return (
    <>
      <div className="text-base">{t('settings.display.title')}</div>
      <Divider className="opacity-50" />

      {isLoading && !status ? (
        <div className="flex w-full items-center justify-center space-x-2 pt-5 text-neutral-500">
          <LoaderCircleIcon className="animate-spin" size={18} />
          <span>{t('settings.display.loading')}</span>
        </div>
      ) : (
        <>
          <div className="flex items-start justify-between space-x-4">
            <div className="flex flex-col space-y-1">
              <span>{t('settings.display.active')}</span>

              {status?.active ? (
                <Summary summary={status.active} />
              ) : (
                <span className="text-xs text-neutral-500">
                  {t('settings.display.activeUnknown')}
                </span>
              )}

              {status?.appliedAt && (
                <span className="text-xs text-neutral-500">
                  {t('settings.display.appliedAt', {
                    time: new Date(status.appliedAt).toLocaleString()
                  })}
                </span>
              )}
            </div>

            <div className="flex shrink-0 flex-col items-end space-y-2">
              <Button
                ghost
                type="primary"
                size="small"
                icon={<DownloadIcon size={14} />}
                disabled={!status?.active || isDownloading}
                onClick={() => download()}
              >
                {t('settings.display.download')}
              </Button>

              {backup && (
                <Button
                  size="small"
                  icon={<HistoryIcon size={14} />}
                  disabled={isDownloading}
                  onClick={() => download(backup.id)}
                >
                  {t('settings.display.downloadBackup')}
                </Button>
              )}
            </div>
          </div>

          {errMsg && <div className="pt-2 text-red-500">{errMsg}</div>}

          <Divider className="opacity-50" />

          <Preset
            active={status?.active}
            preflight={status?.preflight}
            result={result}
            setResult={setResult}
            setIsLocked={setIsLocked}
            onSuccess={getStatus}
          />

          {canRestore && (
            <>
              <Divider className="opacity-50" />

              <Recovery
                needsRecovery={needsRecovery}
                factoryAvailable={!!status?.factoryAvailable}
                backup={backup}
                preflight={status?.preflight}
                setIsLocked={setIsLocked}
                onSuccess={restored}
              />
            </>
          )}
        </>
      )}
    </>
  );
};
