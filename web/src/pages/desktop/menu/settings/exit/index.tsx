import { useEffect, useState } from 'react';
import { Divider } from 'antd';
import { LoaderCircleIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/exit.ts';

import { ExitSlot } from './slot.tsx';
import type { ExitStatus } from './types.ts';

type ExitProps = {
  setIsLocked: (isLocked: boolean) => void;
};

// One panel per slot from /slots. Only slot 0 exists today, but everything
// below is keyed by the slot id so a second tunnel is additive (D9).
export const Exit = ({ setIsLocked }: ExitProps) => {
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const [slots, setSlots] = useState<ExitStatus[]>();
  const [errMsg, setErrMsg] = useState('');

  useEffect(() => {
    getSlots();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function getSlots() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .getSlots()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setSlots(rsp.data?.slots ?? []);
        setErrMsg('');
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get exit slots');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  if (isLoading && !slots) {
    return (
      <div className="flex w-full items-center justify-center space-x-2 pt-5 text-neutral-500">
        <LoaderCircleIcon className="animate-spin" size={18} />
        <span>{t('settings.exit.loading')}</span>
      </div>
    );
  }

  return (
    <>
      <div className="flex flex-col space-y-1">
        <span className="text-base">{t('settings.exit.title')}</span>
        <span className="text-xs text-neutral-500">{t('settings.exit.description')}</span>
      </div>

      {errMsg && <div className="pt-2 text-red-500">{errMsg}</div>}

      {slots?.length === 0 && (
        <div className="pt-2 text-xs text-neutral-500">{t('settings.exit.noSlots')}</div>
      )}

      {slots?.map((slot, index) => (
        <div key={slot.slot}>
          {index > 0 && <Divider className="opacity-50" />}
          <ExitSlot initial={slot} showSlot={slots.length > 1} setIsLocked={setIsLocked} />
        </div>
      ))}
    </>
  );
};
