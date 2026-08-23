import { useEffect, useState } from 'react';
import { Button, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';

import { useDevices } from '../../devices/context.ts';

export const MediaSlots = () => {
  const { t } = useTranslation();
  const state = useDevices();
  const liveCameras = state.snapshot.sinks.filter((sink) => sink.kind === 'camera').length;
  const liveMicrophones = state.snapshot.sinks.filter((sink) => sink.kind === 'microphone').length;
  const [cameras, setCameras] = useState(liveCameras);
  const [microphones, setMicrophones] = useState(liveMicrophones);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setCameras(liveCameras);
    setMicrophones(liveMicrophones);
  }, [liveCameras, liveMicrophones]);

  async function save() {
    if (cameras + microphones > 8) {
      setError(t('settings.device.media.limit'));
      return;
    }
    setWorking(true);
    setError('');
    try {
      await state.setCounts(cameras, microphones);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('settings.device.media.failed'));
    } finally {
      setWorking(false);
    }
  }

  async function disconnect() {
    setWorking(true);
    setError('');
    try {
      await state.disconnectAll();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('settings.device.media.failed'));
    } finally {
      setWorking(false);
    }
  }

  return (
    <div className="space-y-3">
      <div>
        <div>{t('settings.device.media.title')}</div>
        <div className="text-xs text-neutral-500">{t('settings.device.media.desc')}</div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <label className="space-y-1 text-xs text-neutral-400">
          <span>{t('settings.device.media.cameras')}</span>
          <InputNumber
            className="w-full"
            min={0}
            max={8}
            value={cameras}
            onChange={(value) => setCameras(value || 0)}
          />
        </label>
        <label className="space-y-1 text-xs text-neutral-400">
          <span>{t('settings.device.media.microphones')}</span>
          <InputNumber
            className="w-full"
            min={0}
            max={8}
            value={microphones}
            onChange={(value) => setMicrophones(value || 0)}
          />
        </label>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button type="primary" size="small" loading={working} onClick={save}>
          {t('settings.device.media.save')}
        </Button>
        {state.snapshot.bindings.length > 0 && (
          <Button danger size="small" disabled={working} onClick={disconnect}>
            {t('settings.device.media.disconnectAll')}
          </Button>
        )}
      </div>
      {state.snapshot.bindings.map((binding) => (
        <div key={binding.sink_id} className="flex items-center justify-between gap-4 text-xs">
          <span className="truncate text-neutral-500">
            {binding.sink_id} · {binding.owner} · {binding.stream_label}
          </span>
          <Button danger type="text" size="small" onClick={() => state.release(binding.sink_id)}>
            {t('settings.device.media.disconnect')}
          </Button>
        </div>
      ))}
      {error && <div className="text-xs text-red-400">{error}</div>}
    </div>
  );
};
