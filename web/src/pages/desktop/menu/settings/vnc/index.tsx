import { useEffect, useState } from 'react';
import { Divider, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

export const Vnc = () => {
  const { t } = useTranslation();

  const [isEnabled, setIsEnabled] = useState(false);
  const [port, setPort] = useState(5900);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    setIsLoading(true);

    api
      .getVNC()
      .then((rsp) => {
        if (rsp.code !== 0) return;

        setIsEnabled(!!rsp.data?.enabled);
        if (rsp.data?.port) {
          setPort(rsp.data.port);
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  async function update() {
    if (isLoading) return;
    setIsLoading(true);

    const rsp = await api.setVNC(!isEnabled);
    setIsLoading(false);

    if (rsp.code !== 0) {
      console.log(rsp.msg);
      return;
    }

    setIsEnabled(!isEnabled);
  }

  return (
    <>
      <div className="text-base">{t('settings.vnc.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-8">
        <div className="flex items-center justify-between">
          <div className="flex flex-col space-y-1">
            <span>{t('settings.vnc.server')}</span>
            <span className="text-xs text-neutral-500">{t('settings.vnc.description')}</span>
          </div>

          <Switch checked={isEnabled} loading={isLoading} onChange={update} />
        </div>

        <div className="flex items-center justify-between">
          <div className="flex flex-col space-y-1">
            <span>{t('settings.vnc.port')}</span>
            <span className="text-xs text-neutral-500">{t('settings.vnc.portDescription')}</span>
          </div>

          <span className="text-sm text-neutral-400">{port}</span>
        </div>
      </div>
    </>
  );
};
