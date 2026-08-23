import { useEffect, useState } from 'react';
import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/tunnel.ts';
import * as vm from '@/api/vm.ts';

import type { TunnelService } from './types.ts';

const swapSize = 256;

export const Resources = ({ service }: { service: TunnelService }) => {
  const { t } = useTranslation();

  const [isSupported, setIsSupported] = useState(false);
  const [limit, setLimit] = useState(0);
  const [isLimited, setIsLimited] = useState(false);
  const [isMemoryLoading, setIsMemoryLoading] = useState(false);
  const [hasSwap, setHasSwap] = useState(false);
  const [isSwapLoading, setIsSwapLoading] = useState(false);

  useEffect(() => {
    setIsSupported(false);
    setIsLimited(false);

    api.getMemory(service).then((rsp) => {
      if (rsp.code !== 0) return;

      setIsSupported(!!rsp.data?.supported);
      setLimit(rsp.data?.limit ?? 0);
      setIsLimited(!!rsp.data?.enabled);
    });
  }, [service]);

  useEffect(() => {
    vm.getSwap().then((rsp) => {
      if (rsp.code !== 0) return;
      setHasSwap((rsp.data?.size ?? 0) > 0);
    });
  }, []);

  function updateMemory(enabled: boolean) {
    if (isMemoryLoading) return;
    setIsMemoryLoading(true);

    api
      .setMemory(service, enabled)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }

        setIsLimited(enabled);
      })
      .finally(() => {
        setIsMemoryLoading(false);
      });
  }

  function updateSwap(enabled: boolean) {
    if (isSwapLoading) return;
    setIsSwapLoading(true);

    vm.setSwap(enabled ? swapSize : 0)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }

        setHasSwap(enabled);
      })
      .finally(() => {
        setIsSwapLoading(false);
      });
  }

  return (
    <div className="flex flex-col space-y-4">
      <span>{t('settings.tunnel.resources')}</span>

      <div className="flex flex-col space-y-8">
        <div className="flex items-center justify-between gap-4">
          <div className="flex min-w-0 flex-col space-y-1">
            <span>{t('settings.tunnel.memory.title')}</span>
            <span className="text-xs text-neutral-500">
              {isSupported
                ? t('settings.tunnel.memory.description', { limit })
                : t('settings.tunnel.memory.noRuntime')}
            </span>
          </div>

          {isSupported ? (
            <Switch
              className="shrink-0"
              checked={isLimited}
              loading={isMemoryLoading}
              onChange={updateMemory}
            />
          ) : (
            <span className="shrink-0 text-sm text-neutral-500">
              {t('settings.tunnel.memory.notApplicable')}
            </span>
          )}
        </div>

        <div className="flex items-center justify-between gap-4">
          <div className="flex min-w-0 flex-col space-y-1">
            <span>{t('settings.tunnel.swap.title')}</span>
            <span className="text-xs text-neutral-500">
              {t('settings.tunnel.swap.description')}
            </span>
          </div>

          <Switch
            className="shrink-0"
            checked={hasSwap}
            loading={isSwapLoading}
            onChange={updateSwap}
          />
        </div>
      </div>
    </div>
  );
};
