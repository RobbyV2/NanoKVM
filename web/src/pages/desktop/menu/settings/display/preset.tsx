import { ChangeEvent, useEffect, useRef, useState } from 'react';
import { Button, Input, Modal, Select } from 'antd';
import { UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/edid.ts';
import type { EdidPreflight, EdidProfile, EdidResult, EdidSummary } from '@/api/edid.ts';

import { Mismatch } from './mismatch.tsx';
import { Checks, Summary } from './summary.tsx';
import { buildChecks, formatMode, hardwareName } from './utils.ts';

type PresetProps = {
  active?: EdidSummary;
  preflight?: EdidPreflight;
  result?: EdidResult;
  setResult: (result?: EdidResult) => void;
  setIsLocked: (isLocked: boolean) => void;
  onSuccess: () => void;
};

// exactly one of profile and data is sent, the other stays empty
type Selection = {
  profile: string;
  data: string;
  summary: Partial<EdidSummary>;
};

export const Preset = ({
  active,
  preflight,
  result,
  setResult,
  setIsLocked,
  onSuccess
}: PresetProps) => {
  const { t } = useTranslation();

  const inputRef = useRef<HTMLInputElement>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isApplying, setIsApplying] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [profiles, setProfiles] = useState<EdidProfile[]>([]);
  const [selection, setSelection] = useState<Selection>();
  const [confirmation, setConfirmation] = useState('');
  const [errMsg, setErrMsg] = useState('');

  const checks = buildChecks(selection?.summary, preflight);
  const hasError = checks.some((check) => check.level === 'error');

  // on cube the power cycle after a bad flash is a trip to the device, so the
  // confirmation is typed there; on pcie the tool resets hdmi over gpio itself
  const needsTypedConfirm = !!preflight?.requiresPowerCycle;
  const confirmWord = t('settings.display.confirmWord');
  const isConfirmed =
    !needsTypedConfirm || confirmation.trim().toLowerCase() === confirmWord.toLowerCase();

  useEffect(() => {
    getProfiles();
  }, []);

  function getProfiles() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .getProfiles()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setProfiles(rsp.data?.profiles ?? []);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to get edid profiles');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function selectProfile(id: string) {
    if (isApplying) return;

    const profile = profiles.find((item) => item.id === id);
    if (!profile) return;

    setErrMsg('');
    setResult(undefined);
    setSelection({
      profile: profile.id,
      data: '',
      summary: {
        manufacturer: profile.manufacturer,
        model: profile.model,
        preferredMode: profile.preferredMode
      }
    });
  }

  function select() {
    if (isLoading || isApplying) return;
    inputRef.current?.click();
  }

  function upload(e: ChangeEvent<HTMLInputElement>) {
    if (isLoading || isApplying) return;

    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;

    setIsLoading(true);
    setErrMsg('');
    setResult(undefined);

    api
      .toBase64(file)
      .then((data) =>
        api.decode(data).then((rsp) => {
          if (rsp.code !== 0) {
            setSelection(undefined);
            setErrMsg(rsp.msg);
            return;
          }

          setSelection({ profile: '', data, summary: rsp.data?.summary ?? {} });
        })
      )
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to read edid file');
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function openModal() {
    if (isLoading || isApplying || hasError || !selection) return;
    setConfirmation('');
    setIsModalOpen(true);
  }

  function closeModal() {
    if (isApplying) return;
    setIsModalOpen(false);
  }

  function apply() {
    if (isApplying || !selection || !isConfirmed) return;
    setIsApplying(true);
    setIsLocked(true);
    setErrMsg('');
    setResult(undefined);

    api
      .apply(selection.profile, selection.data)
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
        setErrMsg(err?.message || 'Failed to apply edid');
      })
      .finally(() => {
        setIsModalOpen(false);
        setIsApplying(false);
        setIsLocked(false);
      });
  }

  return (
    <>
      <div className="flex flex-col space-y-4">
        <div className="flex flex-col space-y-2">
          <span>{t('settings.display.preset')}</span>

          <div className="flex items-center space-x-2">
            <Select
              className="flex-1"
              value={selection?.profile || undefined}
              placeholder={t('settings.display.presetPlaceholder')}
              options={profiles.map((profile) => ({
                value: profile.id,
                label: `${profile.manufacturer} ${profile.model} · ${formatMode(profile.preferredMode)}`
              }))}
              showSearch
              optionFilterProp="label"
              loading={isLoading}
              disabled={isApplying}
              onChange={selectProfile}
            />

            <input ref={inputRef} type="file" className="hidden" onChange={upload} />

            <Button
              ghost
              type="primary"
              icon={<UploadIcon size={14} />}
              disabled={isApplying}
              onClick={select}
            >
              {t('settings.display.upload')}
            </Button>
          </div>
        </div>

        {selection && (
          <div className="flex flex-col space-y-1">
            <span className="text-xs text-neutral-500">{t('settings.display.selected')}</span>
            <Summary summary={selection.summary} />
          </div>
        )}

        {checks.length > 0 && <Checks checks={checks} />}

        <div className="flex items-center space-x-3">
          <Button
            type="primary"
            disabled={!selection || hasError}
            loading={isApplying}
            onClick={openModal}
          >
            {t('settings.display.apply')}
          </Button>

          {result?.verified && (
            <span className="text-xs text-green-500">{t('settings.display.applied')}</span>
          )}
          {errMsg && <span className="text-red-500">{errMsg}</span>}
        </div>

        {result?.verified && result.requiresPowerCycle && (
          <span className="text-xs text-amber-500">{t('settings.display.powerCycleNotice')}</span>
        )}

        {result && !result.verified && (
          <div className="flex flex-col space-y-2">
            <span className="text-red-500">{t('settings.display.applyFailed')}</span>
            {result.retryable && (
              <span className="text-xs text-amber-500">{t('settings.display.busy')}</span>
            )}
            {result.message && <span className="text-xs text-neutral-500">{result.message}</span>}
            <Mismatch result={result} />
          </div>
        )}
      </div>

      <Modal
        title={t('settings.display.applyTitle')}
        open={isModalOpen}
        centered={true}
        okText={t('settings.display.okBtn')}
        cancelText={t('settings.display.cancelBtn')}
        confirmLoading={isApplying}
        okButtonProps={{ disabled: !isConfirmed }}
        onOk={apply}
        onCancel={closeModal}
      >
        <div className="flex flex-col space-y-4 py-4">
          <div className="flex flex-col space-y-1">
            <span className="text-xs text-neutral-500">{t('settings.display.before')}</span>
            {active ? (
              <Summary summary={active} />
            ) : (
              <span className="text-sm text-neutral-400">
                {t('settings.display.activeUnknown')}
              </span>
            )}
          </div>

          <div className="flex flex-col space-y-1">
            <span className="text-xs text-neutral-500">{t('settings.display.after')}</span>
            {selection && <Summary summary={selection.summary} />}
          </div>

          <span className="text-sm text-neutral-400">
            {t('settings.display.hardware', {
              hardware: hardwareName(preflight) || t('settings.display.hardwareUnknown')
            })}
          </span>

          <span className="text-sm">{t('settings.display.hdmiNotice')}</span>

          {preflight?.requiresPowerCycle && (
            <span className="text-sm text-amber-500">{t('settings.display.powerCycleNotice')}</span>
          )}

          {needsTypedConfirm && (
            <div className="flex flex-col space-y-2">
              <span className="text-sm">
                {t('settings.display.confirmPrompt', { word: confirmWord })}
              </span>

              <Input
                value={confirmation}
                placeholder={confirmWord}
                disabled={isApplying}
                onChange={(e) => setConfirmation(e.target.value)}
              />
            </div>
          )}
        </div>
      </Modal>
    </>
  );
};
