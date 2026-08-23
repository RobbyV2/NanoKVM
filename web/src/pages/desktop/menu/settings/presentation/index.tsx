import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Divider, Input, Modal, Popconfirm, Select } from 'antd';
import { useAtomValue } from 'jotai';
import {
  CopyIcon,
  DownloadIcon,
  LoaderCircleIcon,
  SaveIcon,
  Trash2Icon,
  Undo2Icon,
  UploadIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/presentation.ts';
import type {
  PresentationPreview,
  PresentationProfile,
  PresentationStatus,
  ProfileSummary
} from '@/api/presentation.ts';
import { client } from '@/lib/websocket.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import {
  CAPTURE_STATUS_EVENT,
  getCaptureStatusMessageKey,
  parseCaptureStatusMessage,
  type CaptureStatus
} from '@/pages/desktop/capture-status/model.ts';

import {
  descriptorCount,
  editIdentity,
  formatFIFOs,
  identityChanged,
  identityFields,
  isProfileName,
  recoveryKey,
  type IdentityFields
} from './editor.ts';

type PresentationProps = {
  setIsLocked: (locked: boolean) => void;
};

export const Presentation = ({ setIsLocked }: PresentationProps) => {
  const { t } = useTranslation();
  const videoMode = useAtomValue(videoModeAtom);
  const fileInput = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<PresentationStatus>();
  const [profiles, setProfiles] = useState<ProfileSummary[]>([]);
  const [selected, setSelected] = useState('');
  const [profile, setProfile] = useState<PresentationProfile>();
  const [fields, setFields] = useState<IdentityFields>();
  const [preview, setPreview] = useState<PresentationPreview>();
  const [cloneName, setCloneName] = useState('');
  const [cloneOpen, setCloneOpen] = useState(false);
  const [capture, setCapture] = useState<CaptureStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');

  const loadProfile = useCallback(async (name: string) => {
    setSelected(name);
    setPreview(undefined);
    setError('');
    const rsp = await api.getProfile(name);
    if (rsp.code !== 0) throw new Error(rsp.msg);
    const next = rsp.data as PresentationProfile;
    setProfile(next);
    setFields(identityFields(next));
    const previewRsp = await api.previewProfile(next);
    if (previewRsp.code === 0) setPreview(previewRsp.data as PresentationPreview);
  }, []);

  const refresh = useCallback(
    async (preferred?: string) => {
      setLoading(true);
      setError('');
      try {
        const [statusRsp, profilesRsp] = await Promise.all([api.getStatus(), api.getProfiles()]);
        if (statusRsp.code !== 0) throw new Error(statusRsp.msg);
        if (profilesRsp.code !== 0) throw new Error(profilesRsp.msg);
        const nextProfiles = profilesRsp.data.profiles as ProfileSummary[];
        const nextStatus = statusRsp.data as PresentationStatus;
        setStatus(nextStatus);
        setProfiles(nextProfiles);
        const next = preferred || nextStatus.snapshot.active || nextProfiles[0]?.name;
        if (next) await loadProfile(next);
      } catch (err: any) {
        setError(err?.message || t('settings.presentation.loadFailed'));
      } finally {
        setLoading(false);
      }
    },
    [loadProfile, t]
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    return client.on(CAPTURE_STATUS_EVENT, (message) => {
      const status = parseCaptureStatusMessage(message);
      if (status && status.mode === videoMode) setCapture(status);
    });
  }, [videoMode]);

  function candidate() {
    return profile && fields ? editIdentity(profile, fields) : undefined;
  }

  async function inspect() {
    const next = candidate();
    if (!next) return;
    await act(async () => {
      const rsp = await api.previewProfile(next);
      if (rsp.code !== 0) throw new Error(rsp.msg);
      setPreview(rsp.data as PresentationPreview);
    });
  }

  async function save() {
    const next = candidate();
    if (!next || next.built_in) return;
    await act(async () => {
      const rsp = await api.updateProfile(next);
      if (rsp.code !== 0) throw new Error(rsp.msg);
      setProfile(rsp.data as PresentationProfile);
      setFields(identityFields(rsp.data as PresentationProfile));
      await refresh(next.name);
    });
  }

  async function apply() {
    const next = candidate();
    if (!next) return;
    const rsp = await api.previewProfile(next);
    if (rsp.code !== 0) {
      setError(rsp.msg);
      return;
    }
    const result = rsp.data as PresentationPreview;
    setPreview(result);
    if (!result.valid) return;

    Modal.confirm({
      title: t('settings.presentation.applyTitle'),
      content: (
        <div className="space-y-2 text-sm text-neutral-300">
          <div>{t('settings.presentation.applyDesc', { profile: next.name })}</div>
          <div className="break-words text-neutral-500">
            {result.device.vendor_id}:{result.device.product_id} · {result.device.manufacturer}{' '}
            {result.device.product}
            {result.device.serial ? ` · ${result.device.serial}` : ''}
          </div>
          <div className="break-words text-neutral-500">
            {t('settings.presentation.applyLinks', {
              functions: (result.apply?.linked || result.functions).join(', ')
            })}
          </div>
          {result.apply && result.apply.removes.length > 0 && (
            <div className="break-words text-neutral-500">
              {t('settings.presentation.applyRemoves', {
                functions: result.apply.removes.join(', ')
              })}
            </div>
          )}
          {result.apply && !result.apply.hid && <div>{t('settings.presentation.applyNoHid')}</div>}
          <div>
            {result.apply
              ? t(recoveryKey(result.apply.recovery))
              : t('settings.presentation.reconnect')}
          </div>
          {result.rollback && (
            <div className="text-neutral-500">
              {t('settings.presentation.applyRollback', { profile: result.rollback.profile })}
            </div>
          )}
        </div>
      ),
      okText: t('settings.presentation.apply'),
      cancelText: t('settings.presentation.cancel'),
      onOk: () =>
        act(async () => {
          if (!next.built_in) {
            const saveRsp = await api.updateProfile(next);
            if (saveRsp.code !== 0) throw new Error(saveRsp.msg);
          }
          const applyRsp = await api.applyProfile(next.name);
          if (applyRsp.code !== 0) throw new Error(applyRsp.msg);
          await refresh(next.name);
        })
    });
  }

  async function rollback() {
    await act(async () => {
      const rsp = await api.rollbackProfile();
      if (rsp.code !== 0) throw new Error(rsp.msg);
      await refresh((rsp.data as { profile: string }).profile);
    });
  }

  async function clone() {
    if (!selected || !cloneName) return;
    await act(async () => {
      const rsp = await api.cloneProfile(selected, cloneName.trim());
      if (rsp.code !== 0) throw new Error(rsp.msg);
      setCloneOpen(false);
      setCloneName('');
      await refresh((rsp.data as PresentationProfile).name);
    });
  }

  function remove() {
    if (!profile || profile.built_in) return;
    Modal.confirm({
      title: t('settings.presentation.deleteTitle'),
      content: t('settings.presentation.deleteDesc', { profile: profile.name }),
      okText: t('settings.presentation.delete'),
      okButtonProps: { danger: true },
      cancelText: t('settings.presentation.cancel'),
      onOk: () =>
        act(async () => {
          const rsp = await api.deleteProfile(profile.name);
          if (rsp.code !== 0) throw new Error(rsp.msg);
          await refresh(status?.snapshot.active);
        })
    });
  }

  async function importPackage(file?: File) {
    if (!file) return;
    await act(async () => {
      const rsp = await api.importProfile(file);
      if (rsp.code !== 0) throw new Error(rsp.msg);
      await refresh((rsp.data as PresentationProfile).name);
    });
    if (fileInput.current) fileInput.current.value = '';
  }

  async function act(fn: () => Promise<void>) {
    if (working) return;
    setWorking(true);
    setIsLocked(true);
    setError('');
    try {
      await fn();
    } catch (err: any) {
      setError(err?.message || t('settings.presentation.operationFailed'));
    } finally {
      setWorking(false);
      setIsLocked(false);
    }
  }

  const readOnly = profile?.built_in !== false;
  const assets = profile ? descriptorCount(profile) : 0;
  const snapshot = status?.snapshot;
  const udc = snapshot?.udc;
  const failure = snapshot?.last_error;
  const pending =
    profile && fields && identityChanged(profile, fields)
      ? t('settings.presentation.pendingEdits')
      : selected && selected !== snapshot?.active
        ? t('settings.presentation.pendingProfile', { profile: selected })
        : t('settings.presentation.pendingNone');
  const host = !udc
    ? undefined
    : udc.bound
      ? [udc.state, udc.speed, udc.name].filter(Boolean).join(' · ')
      : t('settings.presentation.hostUnbound');
  const hdmi = !capture
    ? t('settings.presentation.hdmiUnreported')
    : capture.ok
      ? t('settings.presentation.hdmiSignal')
      : t(getCaptureStatusMessageKey(capture.result));

  return (
    <>
      <div className="text-base">{t('settings.presentation.title')}</div>
      <Divider className="opacity-50" />

      {loading && !profile ? (
        <div className="flex items-center justify-center space-x-2 pt-5 text-neutral-500">
          <LoaderCircleIcon className="animate-spin" size={18} />
          <span>{t('settings.presentation.loading')}</span>
        </div>
      ) : (
        <div className="space-y-5">
          <div className="space-y-1">
            <div className="text-sm text-neutral-400">{t('settings.presentation.current')}</div>
            <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
              <span>{snapshot?.active || t('settings.presentation.noProfile')}</span>
              <span className="text-xs text-neutral-500">
                {status?.profile?.manufacturer} {status?.profile?.product}
              </span>
              <span className="text-xs text-neutral-500">{snapshot?.mode}</span>
            </div>
            <div className="space-y-1 pt-2">
              <Readout
                label={t('settings.presentation.linked')}
                value={
                  snapshot &&
                  (snapshot.linked?.join(', ') || t('settings.presentation.noFunctions'))
                }
              />
              <Readout label={t('settings.presentation.hostState')} value={host} />
              <Readout label={t('settings.presentation.hdmiState')} value={hdmi} />
              <Readout
                label={t('settings.presentation.endpoints')}
                value={
                  snapshot &&
                  t('settings.presentation.endpointUse', {
                    inUse: snapshot.endpoints.in,
                    inFree: snapshot.headroom.in,
                    outUse: snapshot.endpoints.out,
                    outFree: snapshot.headroom.out
                  })
                }
              />
              <Readout
                label={t('settings.presentation.fifos')}
                value={formatFIFOs(snapshot?.fifos)}
              />
              <Readout label={t('settings.presentation.pending')} value={pending} />
              <Readout
                label={t('settings.presentation.lastApply')}
                value={
                  failure
                    ? t('settings.presentation.applyFailed', {
                        profile: failure.profile,
                        time: new Date(failure.at).toLocaleString()
                      })
                    : t('settings.presentation.applyClean')
                }
              />
              {failure && (
                <div className="break-words text-xs text-neutral-500">{failure.message}</div>
              )}
              <Readout
                label={t('settings.presentation.lastKnownGood')}
                value={status?.last_known_good}
              />
              <Readout
                label={t('settings.presentation.rollbackTarget')}
                value={status?.rollback_target || t('settings.presentation.rollbackNone')}
              />
            </div>
            {snapshot?.pending_power_cycle && (
              <Alert
                type="warning"
                showIcon
                message={t('settings.presentation.powerCyclePending')}
              />
            )}
            {status?.rollback_target && (
              <div className="flex justify-end pt-1">
                <Popconfirm
                  title={t('settings.presentation.rollbackTitle', {
                    profile: status.rollback_target
                  })}
                  description={t('settings.presentation.rollbackDesc')}
                  okText={t('settings.presentation.rollback')}
                  cancelText={t('settings.presentation.cancel')}
                  onConfirm={rollback}
                >
                  <Button size="small" icon={<Undo2Icon size={14} />} disabled={working}>
                    {t('settings.presentation.rollback')}
                  </Button>
                </Popconfirm>
              </div>
            )}
          </div>

          <Divider className="my-0 opacity-50" />

          <div className="space-y-3">
            <div className="text-sm text-neutral-400">{t('settings.presentation.profile')}</div>
            <Select
              className="w-full"
              value={selected || undefined}
              options={profiles.map((item) => ({
                value: item.name,
                label: `${item.name} · ${item.manufacturer} ${item.product}${item.built_in ? ` (${t('settings.presentation.builtIn')})` : ''}${item.provenance.descriptors ? ` · ${t('settings.presentation.descriptors')}` : ''}`
              }))}
              onChange={(name) => loadProfile(name).catch((err) => setError(err.message))}
            />
            <div className="flex flex-wrap gap-2">
              <Button size="small" icon={<CopyIcon size={14} />} onClick={() => setCloneOpen(true)}>
                {t('settings.presentation.clone')}
              </Button>
              <Button
                size="small"
                icon={<DownloadIcon size={14} />}
                disabled={!selected}
                onClick={() => act(() => api.exportProfile(selected))}
              >
                {t('settings.presentation.export')}
              </Button>
              <Button
                size="small"
                icon={<UploadIcon size={14} />}
                onClick={() => fileInput.current?.click()}
              >
                {t('settings.presentation.import')}
              </Button>
              <input
                ref={fileInput}
                type="file"
                accept=".zip,application/zip,application/vnd.nanokvm.presentation+zip"
                className="hidden"
                onChange={(event) => importPackage(event.target.files?.[0])}
              />
              {!readOnly && (
                <Button danger size="small" icon={<Trash2Icon size={14} />} onClick={remove}>
                  {t('settings.presentation.delete')}
                </Button>
              )}
            </div>
          </div>

          {fields && profile && (
            <>
              <Divider className="my-0 opacity-50" />
              <div className="space-y-3">
                <div>
                  <div className="text-sm text-neutral-400">
                    {t('settings.presentation.identity')}
                  </div>
                  {readOnly && (
                    <div className="pt-1 text-xs text-neutral-500">
                      {t('settings.presentation.cloneToEdit')}
                    </div>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <Field
                    label={t('settings.presentation.vendorId')}
                    value={fields.vendorId}
                    disabled={readOnly}
                    onChange={(vendorId) => setFields({ ...fields, vendorId })}
                  />
                  <Field
                    label={t('settings.presentation.productId')}
                    value={fields.productId}
                    disabled={readOnly}
                    onChange={(productId) => setFields({ ...fields, productId })}
                  />
                  <Field
                    label={t('settings.presentation.bcdUSB')}
                    value={fields.bcdUSB}
                    disabled={readOnly}
                    onChange={(bcdUSB) => setFields({ ...fields, bcdUSB })}
                  />
                  <Field
                    label={t('settings.presentation.bcdDevice')}
                    value={fields.bcdDevice}
                    disabled={readOnly}
                    onChange={(bcdDevice) => setFields({ ...fields, bcdDevice })}
                  />
                </div>
                <Field
                  label={t('settings.presentation.manufacturer')}
                  value={fields.manufacturer}
                  disabled={readOnly}
                  onChange={(manufacturer) => setFields({ ...fields, manufacturer })}
                />
                <Field
                  label={t('settings.presentation.product')}
                  value={fields.product}
                  disabled={readOnly}
                  onChange={(product) => setFields({ ...fields, product })}
                />
                <Field
                  label={t('settings.presentation.serial')}
                  value={fields.serial}
                  disabled={readOnly}
                  onChange={(serial) => setFields({ ...fields, serial })}
                />
                <Field
                  label={t('settings.presentation.configuration')}
                  value={fields.configuration}
                  disabled={readOnly}
                  onChange={(configuration) => setFields({ ...fields, configuration })}
                />
              </div>

              <div className="space-y-1 text-xs text-neutral-500">
                <div>
                  {t('settings.presentation.functions')}:{' '}
                  {profile.functions.map((item) => `${item.kind}.${item.instance}`).join(', ')}
                </div>
                <div>{t('settings.presentation.descriptorAssets', { count: assets })}</div>
              </div>

              {preview?.errors.map((message) => (
                <Alert key={message} type="error" showIcon message={message} />
              ))}
              {preview?.warnings.map((message) => (
                <Alert key={message} type="warning" showIcon message={message} />
              ))}
              {preview?.valid && (
                <div className="space-y-1 text-xs text-neutral-500">
                  <div>
                    {t('settings.presentation.endpointUse', {
                      inUse: preview.endpoints.in,
                      inFree: preview.headroom.in,
                      outUse: preview.endpoints.out,
                      outFree: preview.headroom.out
                    })}
                  </div>
                  {formatFIFOs(preview.fifos) && (
                    <div>
                      {t('settings.presentation.fifos')}: {formatFIFOs(preview.fifos)}
                    </div>
                  )}
                </div>
              )}
              {error && <Alert type="error" showIcon message={error} />}

              <div className="flex justify-end gap-2">
                <Button disabled={working} onClick={inspect}>
                  {t('settings.presentation.preview')}
                </Button>
                {!readOnly && (
                  <Button icon={<SaveIcon size={14} />} disabled={working} onClick={save}>
                    {t('settings.presentation.save')}
                  </Button>
                )}
                <Button type="primary" loading={working} onClick={apply}>
                  {t('settings.presentation.apply')}
                </Button>
              </div>
            </>
          )}
        </div>
      )}

      <Modal
        open={cloneOpen}
        title={t('settings.presentation.cloneTitle')}
        okText={t('settings.presentation.clone')}
        cancelText={t('settings.presentation.cancel')}
        okButtonProps={{ disabled: !isProfileName(cloneName), loading: working }}
        onOk={clone}
        onCancel={() => setCloneOpen(false)}
      >
        <div className="space-y-2">
          <div className="text-sm text-neutral-400">{t('settings.presentation.profileName')}</div>
          <Input
            value={cloneName}
            maxLength={64}
            status={cloneName && !isProfileName(cloneName) ? 'error' : undefined}
            onChange={(event) => setCloneName(event.target.value)}
          />
          <div className="text-xs text-neutral-500">
            {t('settings.presentation.profileNameHint')}
          </div>
        </div>
      </Modal>
    </>
  );
};

const Readout = ({ label, value }: { label: string; value?: string }) => (
  <div className="flex items-baseline justify-between gap-3 text-xs">
    <span className="shrink-0 text-neutral-400">{label}</span>
    <span className="break-words text-right text-neutral-300">{value || '-'}</span>
  </div>
);

type FieldProps = {
  label: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
};

const Field = ({ label, value, disabled, onChange }: FieldProps) => (
  <label className="block space-y-1">
    <span className="text-xs text-neutral-400">{label}</span>
    <Input
      size="small"
      value={value}
      disabled={disabled}
      onChange={(event) => onChange(event.target.value)}
    />
  </label>
);
