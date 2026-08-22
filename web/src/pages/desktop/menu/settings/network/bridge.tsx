import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Modal, Switch } from 'antd';
import { LoaderCircleIcon, RotateCcwIcon, TriangleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/bridge.ts';
import type { BridgeChecks, BridgePort, BridgeStatus } from '@/api/bridge.ts';

// the verification gates, in the order the transaction runs them
type Gate = 'address' | 'gateway' | 'inbound';

const gates: Gate[] = ['address', 'gateway', 'inbound'];

// the gates that did not pass, so the notice can name the leg that failed
function failedGates(checks?: BridgeChecks): Gate[] {
  if (!checks) return [];
  return gates.filter((gate) => !checks[gate]);
}

const InfoRow = ({
  label,
  value,
  isLast = false
}: {
  label: string;
  value?: string;
  isLast?: boolean;
}) => {
  return (
    <div className="px-4">
      <div
        className={`flex min-h-[44px] items-center justify-between ${
          isLast ? '' : 'border-b border-neutral-700/50'
        }`}
      >
        <span className="text-sm text-neutral-300">{label}</span>
        <span className="max-w-[330px] break-all text-right text-sm text-neutral-500">
          {value || '-'}
        </span>
      </div>
    </div>
  );
};

type ToggleProps = {
  enabled: boolean;
  disabled: boolean;
  onSuccess: () => void;
};

const Toggle = ({ enabled, disabled, onSuccess }: ToggleProps) => {
  const { t } = useTranslation();

  const [isApplying, setIsApplying] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isInterrupted, setIsInterrupted] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function openModal() {
    if (isApplying || disabled) return;
    setErrMsg('');
    setIsInterrupted(false);
    setIsModalOpen(true);
  }

  function closeModal() {
    if (isApplying) return;
    setIsModalOpen(false);
  }

  function apply() {
    if (isApplying) return;
    setIsApplying(true);
    setErrMsg('');
    setIsInterrupted(false);

    api
      .setBridge(!enabled)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch(() => {
        // moving the management address cuts this very request, so a transport
        // failure is not an outcome. The refetch below is what reports one.
        setIsInterrupted(true);
      })
      .finally(() => {
        setIsModalOpen(false);
        setIsApplying(false);
        onSuccess();
      });
  }

  return (
    <>
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.network.bridge.title')}</span>
          <span className="text-xs text-neutral-500">
            {t('settings.network.bridge.twoDevices')}
          </span>
        </div>

        <Switch checked={enabled} loading={isApplying} disabled={disabled} onChange={openModal} />
      </div>

      {isInterrupted && (
        <div className="flex items-center space-x-1 text-xs text-amber-500">
          <TriangleAlertIcon size={13} />
          <span>{t('settings.network.bridge.interrupted')}</span>
        </div>
      )}

      {errMsg && <div className="text-red-500">{errMsg}</div>}

      <Modal
        title={
          enabled
            ? t('settings.network.bridge.disableTitle')
            : t('settings.network.bridge.enableTitle')
        }
        open={isModalOpen}
        centered={true}
        okText={
          enabled ? t('settings.network.bridge.disableBtn') : t('settings.network.bridge.enableBtn')
        }
        cancelText={t('settings.network.bridge.cancelBtn')}
        confirmLoading={isApplying}
        onOk={apply}
        onCancel={closeModal}
      >
        <div className="flex flex-col space-y-4 py-4">
          <span className="text-sm">{t('settings.network.bridge.twoDevices')}</span>
          <span className="text-sm">{t('settings.network.bridge.reconnect')}</span>
          <span className="text-sm">{t('settings.network.bridge.rollback')}</span>
        </div>
      </Modal>
    </>
  );
};

type RevertProps = {
  onSuccess: () => void;
};

const Revert = ({ onSuccess }: RevertProps) => {
  const { t } = useTranslation();

  const [isReverting, setIsReverting] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function revert() {
    if (isReverting) return;
    setIsReverting(true);
    setErrMsg('');

    api
      .revertBridge()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to revert bridge');
      })
      .finally(() => {
        setIsReverting(false);
        onSuccess();
      });
  }

  return (
    <div className="flex flex-col space-y-2">
      <div className="flex items-center space-x-1 text-xs text-amber-500">
        <TriangleAlertIcon size={13} />
        <span>{t('settings.network.bridge.pendingNotice')}</span>
      </div>

      <div className="flex items-center space-x-3">
        <Button
          size="small"
          icon={<RotateCcwIcon size={14} />}
          loading={isReverting}
          onClick={revert}
        >
          {t('settings.network.bridge.revert')}
        </Button>

        {errMsg && <span className="text-red-500">{errMsg}</span>}
      </div>
    </div>
  );
};

export const Bridge = () => {
  const { t } = useTranslation();

  const isLoadingRef = useRef(false);
  const [isLoading, setIsLoading] = useState(false);
  const [status, setStatus] = useState<BridgeStatus>();
  const [errMsg, setErrMsg] = useState('');

  const getStatus = useCallback(() => {
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;
    setIsLoading(true);

    api
      .getBridge()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setErrMsg('');
        setStatus(rsp.data);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get bridge status');
      })
      .finally(() => {
        isLoadingRef.current = false;
        setIsLoading(false);
      });
  }, []);

  useEffect(() => {
    getStatus();
  }, [getStatus]);

  if (isLoading && !status) {
    return (
      <div className="flex w-full items-center justify-center space-x-2 pt-5 text-neutral-500">
        <LoaderCircleIcon className="animate-spin" size={18} />
        <span>{t('settings.network.bridge.loading')}</span>
      </div>
    );
  }

  const state = status?.state ?? 'disabled';
  const isPending = state === 'pending';
  const lastApply = status?.lastApply;
  const failed = failedGates(lastApply?.checks);

  const ports = (status?.ports ?? [])
    .map(
      (port: BridgePort) =>
        `${port.name} · ${port.up ? t('settings.network.bridge.up') : t('settings.network.bridge.down')}`
    )
    .join(', ');

  return (
    <div className="flex flex-col space-y-4">
      <Toggle
        enabled={state === 'enabled'}
        disabled={isLoading || isPending}
        onSuccess={getStatus}
      />

      <div className="overflow-hidden rounded-xl bg-neutral-800/50">
        <InfoRow
          label={t('settings.network.bridge.state')}
          value={t(`settings.network.bridge.states.${state}`)}
        />
        <InfoRow label={t('settings.network.bridge.uplink')} value={status?.uplink} />
        <InfoRow label={t('settings.network.bridge.ports')} value={ports} />
        {/* named, not offered: the control for it is the Virtual Network one
            under Settings, Device, since the protocol decides what the gadget
            presents whether or not a bridge exists */}
        <InfoRow
          label={t('settings.network.bridge.protocol')}
          value={status?.protocol ? status.protocol.toUpperCase() : ''}
          isLast
        />
      </div>

      {isPending && <Revert onSuccess={getStatus} />}

      {/* the dead-man was disarmed on a proof that never crossed the wire, so
          the gate list saying "inbound" overstates what was actually shown */}
      {lastApply?.checks?.inboundWeak && (
        <div className="flex items-start space-x-1 text-xs text-amber-500">
          <TriangleAlertIcon size={13} className="mt-[2px] shrink-0" />
          <span>{t('settings.network.bridge.inboundWeak')}</span>
        </div>
      )}

      {lastApply?.state === 'rolledBack' && (
        <div className="flex flex-col space-y-1">
          <span className="text-xs text-amber-500">
            {t('settings.network.bridge.rolledBackNotice')}
          </span>

          {failed.length > 0 && (
            <span className="text-xs text-neutral-500">
              {t('settings.network.bridge.verifyFailed', {
                gates: failed.map((gate) => t(`settings.network.bridge.gates.${gate}`)).join(', ')
              })}
            </span>
          )}

          {lastApply.message && (
            <span className="text-xs text-neutral-500">{lastApply.message}</span>
          )}
        </div>
      )}

      {lastApply?.state === 'failed' && (
        <div className="flex flex-col space-y-1">
          <span className="text-red-500">{t('settings.network.bridge.failedNotice')}</span>
          {lastApply.message && (
            <span className="text-xs text-neutral-500">{lastApply.message}</span>
          )}
        </div>
      )}

      {errMsg && <div className="text-red-500">{errMsg}</div>}
    </div>
  );
};
