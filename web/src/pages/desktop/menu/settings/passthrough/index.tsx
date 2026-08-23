import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Divider, Input, Modal, Segmented, Switch } from 'antd';
import { CircleStopIcon, LoaderCircleIcon, PlayIcon, TriangleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/passthrough.ts';
import type { PassthroughMode, PassthroughStatus } from '@/api/passthrough.ts';

import { Instructions } from './instructions.tsx';
import {
  busIdExample,
  defaultExporter,
  deviceId,
  formatTime,
  isIsochronous,
  isValidBusId,
  isValidExporter
} from './utils.ts';

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

type StartProps = {
  exporter: string;
  busId: string;
  mode: PassthroughMode;
  setExporter: (exporter: string) => void;
  setBusId: (busId: string) => void;
  disabled: boolean;
  onSuccess: () => void;
};

const Start = ({
  exporter,
  busId,
  mode,
  setExporter,
  setBusId,
  disabled,
  onSuccess
}: StartProps) => {
  const { t } = useTranslation();

  const [isStarting, setIsStarting] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [allowIso, setAllowIso] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  const isReady = isValidExporter(exporter) && isValidBusId(busId);

  function openModal() {
    if (isStarting || disabled || !isReady) return;
    setErrMsg('');
    if (mode === 'hybrid') {
      start();
    } else {
      setIsModalOpen(true);
    }
  }

  function closeModal() {
    if (isStarting) return;
    setIsModalOpen(false);
  }

  function start() {
    if (isStarting || !isReady) return;
    setIsStarting(true);
    setErrMsg('');

    api
      .startPassthrough(exporter.trim(), busId.trim(), mode, mode === 'exact' && allowIso)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to start passthrough');
      })
      .finally(() => {
        setIsModalOpen(false);
        setIsStarting(false);
        onSuccess();
      });
  }

  return (
    <>
      <div className="flex flex-col space-y-4">
        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.passthrough.exporterLabel')}</span>
          <Input
            value={exporter}
            placeholder={defaultExporter}
            disabled={isStarting || disabled}
            onChange={(e) => setExporter(e.target.value)}
          />
          <span className="text-xs text-neutral-500">
            {t('settings.passthrough.exporterHint', { exporter: defaultExporter })}
          </span>
        </div>

        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.passthrough.busIdLabel')}</span>
          <Input
            value={busId}
            placeholder={busIdExample}
            disabled={isStarting || disabled}
            onChange={(e) => setBusId(e.target.value)}
          />
          <span className="text-xs text-neutral-500">
            {t('settings.passthrough.busIdHint', { example: busIdExample })}
          </span>
        </div>

        {mode === 'exact' && (
          <div className="flex flex-col space-y-1">
            <div className="flex items-center justify-between space-x-4">
              <span className="text-sm">{t('settings.passthrough.isoLabel')}</span>
              <Switch
                checked={allowIso}
                disabled={isStarting || disabled}
                onChange={(checked) => setAllowIso(checked)}
              />
            </div>
            <span className="text-xs text-neutral-500">{t('settings.passthrough.isoHint')}</span>
            {allowIso && (
              <div className="flex items-start space-x-1 text-xs text-amber-500">
                <TriangleAlertIcon size={13} className="mt-[2px] shrink-0" />
                <span>{t('settings.passthrough.isoWarning')}</span>
              </div>
            )}
          </div>
        )}

        <div className="flex items-center space-x-3">
          <Button
            type="primary"
            icon={<PlayIcon size={14} />}
            disabled={disabled || !isReady}
            loading={isStarting}
            onClick={openModal}
          >
            {t('settings.passthrough.start')}
          </Button>

          {errMsg && <span className="text-red-500">{errMsg}</span>}
        </div>
      </div>

      <Modal
        title={t('settings.passthrough.startTitle')}
        open={isModalOpen}
        centered={true}
        okText={t('settings.passthrough.okBtn')}
        cancelText={t('settings.passthrough.cancelBtn')}
        confirmLoading={isStarting}
        onOk={start}
        onCancel={closeModal}
      >
        <div className="flex flex-col space-y-4 py-4">
          <span className="text-sm text-neutral-400">
            {t('settings.passthrough.startDevice', {
              busId: busId.trim(),
              exporter: exporter.trim()
            })}
          </span>

          <span className="text-sm">{t('settings.passthrough.startHid')}</span>
          <span className="text-sm">{t('settings.passthrough.startIso')}</span>
          <span className="text-sm text-neutral-400">{t('settings.passthrough.startNetwork')}</span>
        </div>
      </Modal>
    </>
  );
};

type StopProps = {
  onSuccess: () => void;
};

const Stop = ({ onSuccess }: StopProps) => {
  const { t } = useTranslation();

  const [isStopping, setIsStopping] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function stop() {
    if (isStopping) return;
    setIsStopping(true);
    setErrMsg('');

    api
      .stopPassthrough()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
        }
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to stop passthrough');
      })
      .finally(() => {
        setIsStopping(false);
        onSuccess();
      });
  }

  return (
    <div className="flex shrink-0 flex-col items-end space-y-1">
      <Button
        danger
        type="primary"
        icon={<CircleStopIcon size={14} />}
        loading={isStopping}
        onClick={stop}
      >
        {t('settings.passthrough.stop')}
      </Button>

      {errMsg && <span className="text-red-500">{errMsg}</span>}
    </div>
  );
};

export const Passthrough = () => {
  const { t } = useTranslation();

  const isLoadingRef = useRef(false);
  const [isLoading, setIsLoading] = useState(false);
  const [status, setStatus] = useState<PassthroughStatus>();
  const [exporter, setExporter] = useState(defaultExporter);
  const [busId, setBusId] = useState('');
  const [mode, setMode] = useState<PassthroughMode>('hybrid');
  const [errMsg, setErrMsg] = useState('');

  const getStatus = useCallback(() => {
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;
    setIsLoading(true);

    api
      .getPassthrough()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setErrMsg('');
        setStatus(rsp.data);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get passthrough status');
      })
      .finally(() => {
        isLoadingRef.current = false;
        setIsLoading(false);
      });
  }, []);

  useEffect(() => {
    getStatus();
  }, [getStatus]);

  const isActive = !!status?.active;
  const device = status?.device;

  // a running session already knows what it imported, so the commands below
  // repeat that rather than whatever is left in the form
  const shownExporter = isActive && status?.exporter ? status.exporter : exporter;
  const shownBusId = isActive && device?.busId ? device.busId : busId;

  return (
    <>
      <div className="text-base">{t('settings.passthrough.title')}</div>
      <Divider className="opacity-50" />

      {isLoading && !status ? (
        <div className="flex w-full items-center justify-center space-x-2 pt-5 text-neutral-500">
          <LoaderCircleIcon className="animate-spin" size={18} />
          <span>{t('settings.passthrough.loading')}</span>
        </div>
      ) : (
        <div className="flex flex-col space-y-5">
          <Alert
            type="info"
            showIcon
            message={t('settings.passthrough.info.title')}
            description={
              <ul className="list-disc space-y-1 pl-4 text-xs">
                <li>
                  {(isActive ? status?.mode : mode) === 'exact'
                    ? t('settings.passthrough.info.exact')
                    : t('settings.passthrough.info.hybrid')}
                </li>
                <li>{t('settings.passthrough.info.udc')}</li>
                <li>{t('settings.passthrough.info.web')}</li>
                <li>{t('settings.passthrough.info.network')}</li>
                <li>{t('settings.passthrough.info.iso')}</li>
                <li>{t('settings.passthrough.info.camera')}</li>
              </ul>
            }
          />

          {!isActive && (
            <div className="flex flex-col space-y-2">
              <span className="text-sm">{t('settings.passthrough.mode')}</span>
              <Segmented
                value={mode}
                options={[
                  { label: t('settings.passthrough.hybrid'), value: 'hybrid' },
                  { label: t('settings.passthrough.exact'), value: 'exact' }
                ]}
                onChange={(value) => setMode(value as PassthroughMode)}
              />
              <span className="text-xs text-neutral-500">
                {t(`settings.passthrough.${mode}Desc`)}
              </span>
            </div>
          )}

          <div className="flex flex-col space-y-4">
            <div className="flex items-start justify-between space-x-4">
              <div className="flex flex-col space-y-1">
                <span>{t('settings.passthrough.session')}</span>
                <span className="text-xs text-neutral-500">
                  {isActive
                    ? t('settings.passthrough.activeDesc')
                    : t('settings.passthrough.inactiveDesc')}
                </span>
              </div>

              {isActive && <Stop onSuccess={getStatus} />}
            </div>

            {isActive ? (
              <>
                <div className="overflow-hidden rounded-xl bg-neutral-800/50">
                  <InfoRow label={t('settings.passthrough.device')} value={deviceId(device)} />
                  <InfoRow
                    label={t('settings.passthrough.mode')}
                    value={t(`settings.passthrough.${status?.mode || 'hybrid'}`)}
                  />
                  <InfoRow label={t('settings.passthrough.busId')} value={device?.busId} />
                  <InfoRow label={t('settings.passthrough.speed')} value={device?.speed} />
                  <InfoRow label={t('settings.passthrough.exporter')} value={status?.exporter} />
                  <InfoRow
                    label={t('settings.passthrough.local')}
                    value={t('settings.passthrough.localValue', {
                      bus: status?.bus,
                      address: status?.address
                    })}
                  />
                  <InfoRow label={t('settings.passthrough.udc')} value={status?.udc} />
                  {status?.mode === 'exact' && (
                    <InfoRow
                      label={t('settings.passthrough.pid')}
                      value={String(status?.pid ?? '')}
                    />
                  )}
                  <InfoRow
                    label={t('settings.passthrough.startedAt')}
                    value={formatTime(status?.startedAt)}
                    isLast
                  />
                </div>

                {isIsochronous(device) && (
                  <div className="flex items-start space-x-1 text-xs text-amber-500">
                    <TriangleAlertIcon size={13} className="mt-[2px] shrink-0" />
                    <span>{t('settings.passthrough.isoDevice')}</span>
                  </div>
                )}
              </>
            ) : (
              <Start
                exporter={exporter}
                busId={busId}
                mode={mode}
                setExporter={setExporter}
                setBusId={setBusId}
                disabled={isLoading}
                onSuccess={getStatus}
              />
            )}

            {errMsg && <div className="text-red-500">{errMsg}</div>}
          </div>

          <Divider className="opacity-50" />

          <Instructions exporter={shownExporter} busId={shownBusId} />
        </div>
      )}
    </>
  );
};
