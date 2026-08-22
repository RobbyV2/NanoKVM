import { useState } from 'react';
import { Button, Modal } from 'antd';
import { LifeBuoyIcon, RotateCcwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/edid.ts';
import type { EdidBackup, EdidPreflight, EdidResult } from '@/api/edid.ts';

// the restore endpoint takes the factory image or one history entry
type Target = 'factory' | 'history';

type RecoveryProps = {
  needsRecovery: boolean;
  factoryAvailable: boolean;
  backup?: EdidBackup;
  preflight?: EdidPreflight;
  setIsLocked: (isLocked: boolean) => void;
  onSuccess: () => void;
};

export const Recovery = ({
  needsRecovery,
  factoryAvailable,
  backup,
  preflight,
  setIsLocked,
  onSuccess
}: RecoveryProps) => {
  const { t } = useTranslation();

  const [isRestoring, setIsRestoring] = useState(false);
  const [target, setTarget] = useState<Target>();
  const [result, setResult] = useState<EdidResult>();
  const [errMsg, setErrMsg] = useState('');

  function openModal(next: Target) {
    if (isRestoring) return;
    setErrMsg('');
    setResult(undefined);
    setTarget(next);
  }

  function closeModal() {
    if (isRestoring) return;
    setTarget(undefined);
  }

  function restore() {
    if (isRestoring || !target) return;
    setIsRestoring(true);
    setIsLocked(true);
    setErrMsg('');
    setResult(undefined);

    // an empty id means the most recent history entry
    api
      .restore(target, target === 'history' ? (backup?.id ?? '') : '')
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        const outcome: EdidResult = rsp.data;
        setResult(outcome);

        if (outcome.verified) {
          onSuccess();
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to restore edid');
      })
      .finally(() => {
        setTarget(undefined);
        setIsRestoring(false);
        setIsLocked(false);
      });
  }

  return (
    <>
      <div className="flex flex-col space-y-3">
        <span>{t('settings.display.recovery')}</span>

        {needsRecovery && (
          <span className="text-sm text-red-500">{t('settings.display.recoveryNeeded')}</span>
        )}

        <span className="text-xs text-neutral-500">{t('settings.display.recoveryDesc')}</span>

        <div className="flex items-center space-x-3">
          {factoryAvailable && (
            <Button
              type="primary"
              ghost={!needsRecovery}
              danger={needsRecovery}
              icon={<LifeBuoyIcon size={14} />}
              disabled={isRestoring}
              loading={isRestoring && target === 'factory'}
              onClick={() => openModal('factory')}
            >
              {t('settings.display.restoreFactory')}
            </Button>
          )}

          {backup && (
            <Button
              icon={<RotateCcwIcon size={14} />}
              disabled={isRestoring}
              loading={isRestoring && target === 'history'}
              onClick={() => openModal('history')}
            >
              {t('settings.display.restoreBackup')}
            </Button>
          )}

          {result?.verified && (
            <span className="text-xs text-green-500">{t('settings.display.restored')}</span>
          )}
          {errMsg && <span className="text-red-500">{errMsg}</span>}
        </div>

        {result?.verified && result.requiresPowerCycle && (
          <span className="text-xs text-amber-500">{t('settings.display.powerCycleNotice')}</span>
        )}

        {result && !result.verified && (
          <div className="flex flex-col space-y-1">
            <span className="text-red-500">{t('settings.display.restoreFailed')}</span>
            {result.message && <span className="text-xs text-neutral-500">{result.message}</span>}
          </div>
        )}
      </div>

      <Modal
        title={t('settings.display.restoreTitle')}
        open={!!target}
        centered={true}
        okText={t('settings.display.restoreOkBtn')}
        cancelText={t('settings.display.cancelBtn')}
        confirmLoading={isRestoring}
        onOk={restore}
        onCancel={closeModal}
      >
        <div className="flex flex-col space-y-4 py-4">
          <span className="text-sm">
            {target === 'factory'
              ? t('settings.display.restoreFactoryTarget')
              : t('settings.display.restoreBackupTarget', {
                  time: backup ? new Date(backup.appliedAt).toLocaleString() : ''
                })}
          </span>

          {/* a restore is a flash like any other, so it carries the same consequences */}
          <span className="text-sm">{t('settings.display.restoreNotice')}</span>

          <span className="text-sm">{t('settings.display.hdmiNotice')}</span>

          {preflight?.requiresPowerCycle && (
            <span className="text-sm text-amber-500">{t('settings.display.powerCycleNotice')}</span>
          )}
        </div>
      </Modal>
    </>
  );
};
