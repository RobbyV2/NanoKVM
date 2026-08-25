import { useEffect, useState } from 'react';
import { Button, Input } from 'antd';
import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useDevices } from '../../devices/context.ts';
import { mediaSlotRows, mediaSlots, type MediaSlotRow } from '../../devices/state.ts';

export const MediaSlots = () => {
  const { t } = useTranslation();
  const state = useDevices();
  // Bindings change on every claim and release, so the panel resyncs off the
  // slots alone; anything wider would wipe a half-typed name.
  const live = mediaSlotRows(state.snapshot.sinks);
  const slots = JSON.stringify(live);
  const [rows, setRows] = useState<MediaSlotRow[]>(JSON.parse(slots));
  const [added, setAdded] = useState(0);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setRows(JSON.parse(slots));
  }, [slots]);

  // A live slot with no host name is a kernel that cannot carry one; a row the
  // operator just added has none yet either, and says nothing about the kernel.
  const unnamed = live.some((row) => row.hostName === '');
  const bound = new Map(state.snapshot.bindings.map((b) => [b.sink_id, b.stream_label]));

  const defaultKey = {
    camera: 'settings.device.media.cameraDefault',
    microphone: 'settings.device.media.microphoneDefault',
    speaker: 'settings.device.media.speakerDefault'
  } as const;
  const kindKey = {
    camera: 'settings.device.media.cameras',
    microphone: 'settings.device.media.microphones',
    speaker: 'settings.device.media.speakers'
  } as const;

  function add(kind: MediaSlotRow['kind']) {
    const index = rows.filter((row) => row.kind === kind).length + 1;
    setRows([
      ...rows,
      { key: `new-${added}`, kind, label: t(defaultKey[kind], { index }), hostName: '' }
    ]);
    setAdded(added + 1);
  }

  function rename(key: string, label: string) {
    setRows(rows.map((row) => (row.key === key ? { ...row, label } : row)));
  }

  async function save() {
    if (rows.length > 8) {
      setError(t('settings.device.media.limit'));
      return;
    }
    if (rows.some((row) => row.label.trim() === '')) {
      setError(t('settings.device.media.nameRequired'));
      return;
    }
    setWorking(true);
    setError('');
    try {
      await state.setSlots(mediaSlots(rows.map((row) => ({ ...row, label: row.label.trim() }))));
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
      <div className="space-y-2">
        {rows.map((row) => (
          <div key={row.key} className="flex items-center gap-2">
            <span className="w-24 shrink-0 text-xs text-neutral-400">{t(kindKey[row.kind])}</span>
            <div className="min-w-0 flex-1 space-y-1">
              <Input
                size="small"
                value={row.label}
                maxLength={80}
                placeholder={t('settings.device.media.namePlaceholder')}
                onChange={(event) => rename(row.key, event.target.value)}
              />
              {(row.hostName || bound.get(row.key)) && (
                <div className="flex flex-wrap items-center gap-2 text-xs text-neutral-500">
                  {row.hostName && <span className="min-w-0 truncate">{row.hostName}</span>}
                  {bound.get(row.key) && (
                    <Button
                      type="link"
                      size="small"
                      className="h-auto shrink-0 p-0 text-xs"
                      onClick={() => rename(row.key, bound.get(row.key) as string)}
                    >
                      {bound.get(row.key)}
                    </Button>
                  )}
                </div>
              )}
            </div>
            <Button
              type="text"
              size="small"
              className="shrink-0"
              title={t('settings.device.media.remove')}
              icon={<Trash2 size={14} />}
              onClick={() => setRows(rows.filter((other) => other.key !== row.key))}
            />
          </div>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button size="small" disabled={working} onClick={() => add('camera')}>
          {t('settings.device.media.addCamera')}
        </Button>
        <Button size="small" disabled={working} onClick={() => add('microphone')}>
          {t('settings.device.media.addMicrophone')}
        </Button>
        <Button size="small" disabled={working} onClick={() => add('speaker')}>
          {t('settings.device.media.addSpeaker')}
        </Button>
        <Button type="primary" size="small" loading={working} onClick={save}>
          {t('settings.device.media.save')}
        </Button>
        {state.snapshot.bindings.length > 0 && (
          <Button danger size="small" disabled={working} onClick={disconnect}>
            {t('settings.device.media.disconnectAll')}
          </Button>
        )}
      </div>
      {unnamed && (
        <div className="text-xs text-neutral-500">{t('settings.device.media.unsupported')}</div>
      )}
      {state.snapshot.bindings.map((binding) => (
        <div key={binding.sink_id} className="flex items-center justify-between gap-4 text-xs">
          <span className="min-w-0 truncate text-neutral-500">
            {binding.sink_id} · {binding.owner} · {binding.stream_label}
          </span>
          <Button
            danger
            type="text"
            size="small"
            className="shrink-0"
            onClick={() => state.release(binding.sink_id)}
          >
            {t('settings.device.media.disconnect')}
          </Button>
        </div>
      ))}
      {error && <div className="text-xs text-red-400">{error}</div>}
      {error.includes('endpoint budget') && (
        <div className="text-xs text-neutral-500">{t('settings.device.media.budgetHint')}</div>
      )}
    </div>
  );
};
