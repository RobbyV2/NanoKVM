import { useEffect, useRef, useState } from 'react';
import { LoadingOutlined, RocketOutlined, SmileOutlined } from '@ant-design/icons';
import { Alert, Button, Divider, Result, Spin } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';

import { CustomServer } from './custom-server.tsx';
import { Offline } from './offline.tsx';
import { Preview } from './preview.tsx';
import {
  pollIntervalMs,
  rebootGraceMs,
  rebootWaitMs,
  restartWaitMs,
  rollbackWarning,
  updateStatus,
  type KernelState
} from './state.ts';

type UpdateProps = {
  setIsLocked: (isClosable: boolean) => void;
};

export const Update = ({ setIsLocked }: UpdateProps) => {
  const { t } = useTranslation();

  const [status, setStatus] = useState('');
  const [currentVersion, setCurrentVersion] = useState('');
  const [latestVersion, setLatestVersion] = useState('');
  const [latestKernel, setLatestKernel] = useState('');
  const [rolledBack, setRolledBack] = useState('');
  const [errMsg, setErrMsg] = useState('');
  const [isCustomServerEnabled, setIsCustomServerEnabled] = useState(false);
  const [isCustomServerPending, setIsCustomServerPending] = useState(false);
  const versionRequestRef = useRef(0);

  useEffect(() => {
    checkForUpdates();
  }, []);

  async function checkForUpdates() {
    const requestId = ++versionRequestRef.current;
    setStatus('loading');

    let kernel: KernelState | null = null;
    try {
      const rsp: any = await api.getKernel();
      if (rsp.code === 0 && rsp.data) kernel = rsp.data;
    } catch {
      kernel = null;
    }
    if (requestId !== versionRequestRef.current) return;
    setRolledBack(rollbackWarning(kernel));

    try {
      const rsp: any = await api.getVersion();
      if (requestId !== versionRequestRef.current) return;
      if (rsp.code !== 0 || !rsp.data) {
        setStatus('failed');
        setErrMsg(t('settings.update.queryFailed'));
        return;
      }

      setCurrentVersion(rsp.data.current);
      setLatestVersion(rsp.data.latest || '');
      setLatestKernel(rsp.data.latestKernel || '');
      setStatus(updateStatus(rsp.data, kernel));
    } catch {
      if (requestId !== versionRequestRef.current) return;
      setStatus('failed');
      setErrMsg(t('settings.update.queryFailed'));
    }
  }

  function dismissRollback() {
    setRolledBack('');
    api.dismissKernelRollback().catch(() => {});
  }

  function reloadWhenBack(isRebooting: boolean) {
    const deadline = Date.now() + rebootWaitMs;

    function finish() {
      setIsLocked(false);
      setErrMsg('');
      window.location.reload();
    }

    function poll() {
      if (!isRebooting || Date.now() >= deadline) {
        finish();
        return;
      }
      api
        .getKernel()
        .then(finish)
        .catch(() => setTimeout(poll, pollIntervalMs));
    }

    setTimeout(poll, isRebooting ? rebootGraceMs : restartWaitMs);
  }

  function update() {
    if (status !== 'outdated' || isCustomServerPending) return;

    setIsLocked(true);
    setStatus('updating');

    api
      .update()
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          setStatus('failed');
          setErrMsg(t('settings.update.updateFailed'));
          reloadWhenBack(false);
          return;
        }
        const isRebooting = !!rsp.data?.reboot;
        if (isRebooting) setStatus('rebooting');
        reloadWhenBack(isRebooting);
      })
      .catch(() => reloadWhenBack(false));
  }

  return (
    <>
      <div className="text-base">{t('settings.update.title')}</div>
      <Divider className="opacity-50" />

      {rolledBack && (
        <Alert
          className="mb-3"
          type="warning"
          showIcon
          closable
          onClose={dismissRollback}
          message={t('settings.update.rolledBack', { version: rolledBack })}
        />
      )}

      <Preview
        checkForUpdates={checkForUpdates}
        disabled={isCustomServerEnabled || isCustomServerPending}
      />
      <CustomServer
        checkForUpdates={checkForUpdates}
        onEnabledChange={setIsCustomServerEnabled}
        onPendingChange={setIsCustomServerPending}
      />
      <Offline
        status={status}
        setStatus={setStatus}
        setIsLocked={setIsLocked}
        setErrMsg={setErrMsg}
      />
      <Divider className="opacity-50" />

      <div className="flex min-h-[320px] flex-col justify-between">
        {status === 'loading' && (
          <div className="flex justify-center pt-24">
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          </div>
        )}

        {status === 'updating' && (
          <div className="flex flex-col items-center justify-center space-y-10 pb-10 pt-24">
            <Spin size="large" />
            <span className="text-neutral-500">{t('settings.update.updating')}</span>
          </div>
        )}

        {status === 'rebooting' && (
          <div className="flex flex-col items-center justify-center space-y-10 pb-10 pt-24">
            <Spin size="large" />
            <span className="px-6 text-center text-neutral-500">
              {t('settings.update.rebooting')}
            </span>
          </div>
        )}

        {status === 'latest' && (
          <Result
            status="success"
            icon={<SmileOutlined />}
            title={currentVersion}
            subTitle={t('settings.update.isLatest')}
            extra={[
              <Button key="confirm" onClick={checkForUpdates}>
                {t('settings.update.title')}
              </Button>
            ]}
          />
        )}

        {status === 'outdated' && (
          <Result
            status="warning"
            icon={<RocketOutlined />}
            title={`${currentVersion} -> ${latestVersion}`}
            subTitle={
              latestKernel
                ? t('settings.update.kernelUpdate', { version: latestKernel })
                : t('settings.update.available')
            }
            extra={[
              <Button
                key="confirm"
                type="primary"
                disabled={isCustomServerPending}
                onClick={update}
              >
                {t('settings.update.confirm')}
              </Button>
            ]}
          />
        )}

        {status === 'failed' && <Result subTitle={errMsg} />}

        <div className="flex justify-center">
          <Button
            type="link"
            size="small"
            href="https://github.com/sipeed/NanoKVM/blob/main/CHANGELOG.md"
            target="_blank"
          >
            CHANGELOG
          </Button>
        </div>
      </div>
    </>
  );
};
