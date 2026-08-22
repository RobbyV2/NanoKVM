import { ChangeEvent, useRef, useState } from 'react';
import { Button, Popconfirm } from 'antd';
import { RotateCcwIcon, UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/tunnel.ts';

import type { TunnelService } from './types.ts';

type BinarySourceProps = {
  service: TunnelService;
  custom: boolean;
  running: boolean;
  setIsLocked: (isLocked: boolean) => void;
  onSuccess: () => void;
};

export const BinarySource = ({
  service,
  custom,
  running,
  setIsLocked,
  onSuccess
}: BinarySourceProps) => {
  const { t } = useTranslation();

  const inputRef = useRef<HTMLInputElement>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function select() {
    if (isLoading) return;
    inputRef.current?.click();
  }

  function upload(e: ChangeEvent<HTMLInputElement>) {
    if (isLoading) return;

    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;

    setIsLoading(true);
    setIsLocked(true);
    setErrMsg('');

    api
      .uploadBinary(service, file)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        onSuccess();
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to upload binary');
      })
      .finally(() => {
        setIsLoading(false);
        setIsLocked(false);
      });
  }

  function revert() {
    if (isLoading) return;
    setIsLoading(true);
    setErrMsg('');

    api
      .deleteBinary(service)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        onSuccess();
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to delete binary');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  return (
    <div className="flex flex-col space-y-2">
      <div className="flex items-center justify-between space-x-4">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.tunnel.binary')}</span>
          <span className="text-xs text-neutral-500">
            {custom ? t('settings.tunnel.binaryCustom') : t('settings.tunnel.binaryShipped')}
          </span>
        </div>

        <div className="flex shrink-0 items-center space-x-2">
          <input ref={inputRef} type="file" className="hidden" onChange={upload} />

          <Button
            ghost
            type="primary"
            size="small"
            icon={<UploadIcon size={14} />}
            loading={isLoading}
            disabled={running}
            onClick={select}
          >
            {t('settings.tunnel.binaryUpload')}
          </Button>

          {custom && (
            <Popconfirm
              title={t('settings.tunnel.binaryRevert')}
              description={t('settings.tunnel.binaryRevertDesc')}
              okText={t('settings.tunnel.okBtn')}
              cancelText={t('settings.tunnel.cancelBtn')}
              placement="bottom"
              disabled={running || isLoading}
              onConfirm={revert}
            >
              <Button
                danger
                size="small"
                icon={<RotateCcwIcon size={14} />}
                disabled={running || isLoading}
              >
                {t('settings.tunnel.binaryRevert')}
              </Button>
            </Popconfirm>
          )}
        </div>
      </div>

      {errMsg && <span className="text-red-500">{errMsg}</span>}
    </div>
  );
};
