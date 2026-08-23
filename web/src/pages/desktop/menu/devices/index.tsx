import { useMemo, useState } from 'react';
import { useAuth } from '@/contexts/auth.ts';
import { Alert, Button, Divider, Select } from 'antd';
import clsx from 'clsx';
import { CameraIcon, MicIcon, RadioTowerIcon, UsbIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { SourceSink } from '@/api/sources.ts';
import { MenuItem } from '@/components/menu-item.tsx';

import { useDevices } from './context.ts';

export const Devices = () => {
  const { t } = useTranslation();
  const { account } = useAuth();
  const state = useDevices();
  const [selected, setSelected] = useState<Record<string, string>>({});
  const activeCount = state.snapshot.sinks.filter((sink) => sink.binding).length;

  const content = (
    <div className="w-[min(88vw,390px)] space-y-3 p-1">
      <div className="flex items-center justify-between gap-3">
        <div className="text-sm font-medium">{t('devices.title')}</div>
        <div
          className={clsx(
            'text-xs',
            state.eventsConnection === 'connected' ? 'text-neutral-500' : 'text-amber-400'
          )}
        >
          {t(`devices.connection.${state.eventsConnection}`)}
        </div>
      </div>

      {state.eventsConnection !== 'connected' && (
        <Alert type="warning" showIcon message={t('devices.stale')} />
      )}
      {state.errors.connection && <Alert type="error" showIcon message={state.errors.connection} />}

      {state.snapshot.sinks.length === 0 ? (
        <div className="py-5 text-center text-sm text-neutral-500">{t('devices.empty')}</div>
      ) : (
        <div className="space-y-2">
          {state.snapshot.sinks.map((sink) => (
            <SinkRow
              key={sink.id}
              sink={sink}
              username={account.username}
              isAdmin={account.role === 'admin'}
              selected={selected[sink.id]}
              setSelected={(deviceID) => {
                setSelected((current) => ({ ...current, [sink.id]: deviceID }));
                state.choose(sink.id, deviceID);
              }}
            />
          ))}
        </div>
      )}

      {state.snapshot.sources.length > 0 && (
        <>
          <Divider className="my-2 opacity-50" />
          <div className="space-y-1 text-xs text-neutral-500">
            <div>{t('devices.connectedSources', { count: state.snapshot.sources.length })}</div>
            {state.snapshot.sources.map((source) => (
              <div key={source.id} className="flex justify-between gap-4">
                <span className="truncate">{source.owner}</span>
                <span className="truncate">{source.label}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );

  return (
    <MenuItem
      title={t('devices.title')}
      icon={
        <div className="relative">
          <RadioTowerIcon size={18} />
          {activeCount > 0 && (
            <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-emerald-400" />
          )}
        </div>
      }
      content={content}
      fresh
    />
  );
};

type SinkRowProps = {
  sink: SourceSink;
  username: string;
  isAdmin: boolean;
  selected?: string;
  setSelected: (deviceID: string) => void;
};

const SinkRow = ({ sink, username, isAdmin, selected, setSelected }: SinkRowProps) => {
  const { t } = useTranslation();
  const state = useDevices();
  const options = state.devices[sink.kind];
  const sameOwner = sink.binding?.owner === username;
  const canRelease = sameOwner || (isAdmin && !!sink.binding);
  const busy = state.busy.has(sink.id);
  const demand = sink.demand.streaming;
  const active = state.active.has(sink.id);
  const selectedID = selected || options[0]?.deviceID;
  const detail = useMemo(() => {
    if (sink.binding) {
      return `${sink.binding.owner} · ${sink.binding.stream_label || sink.binding.source_label}`;
    }
    if (demand) return t('devices.waiting');
    return t('devices.available');
  }, [demand, sink.binding, t]);

  return (
    <div className="rounded-md border border-neutral-700/70 bg-neutral-800/60 p-3">
      <div className="flex items-start gap-3">
        <div className="pt-0.5 text-neutral-400">
          {sink.kind === 'camera' ? (
            <CameraIcon size={17} />
          ) : sink.kind === 'microphone' ? (
            <MicIcon size={17} />
          ) : (
            <UsbIcon size={17} />
          )}
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate text-sm">{sink.label}</div>
              <div className="truncate text-xs text-neutral-500">{detail}</div>
            </div>
            <div
              className={clsx('shrink-0 text-xs', demand ? 'text-emerald-400' : 'text-neutral-500')}
            >
              {demand ? t('devices.hostOpen') : t('devices.hostIdle')}
            </div>
          </div>

          {!sink.binding && options.length > 0 && (
            <Select
              size="small"
              className="w-full"
              value={selectedID}
              options={options.map((device) => ({ value: device.deviceID, label: device.label }))}
              onChange={setSelected}
            />
          )}

          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-neutral-500">
              {active
                ? t('devices.sending')
                : sink.output === 'black'
                  ? t('devices.black')
                  : sink.output === 'silence'
                    ? t('devices.silence')
                    : sink.binding?.state === 'orphaned'
                      ? t('devices.resuming')
                      : ''}
            </span>
            {!sink.binding && (sink.kind !== 'usb_device' || isAdmin) ? (
              <Button
                size="small"
                type={demand ? 'primary' : 'default'}
                loading={busy}
                onClick={() => state.share(sink, selectedID)}
              >
                {sink.kind === 'usb_device'
                  ? t('devices.share.usbDevice', { defaultValue: 'Share USB' })
                  : t(`devices.share.${sink.kind}`)}
              </Button>
            ) : canRelease ? (
              <Button danger size="small" loading={busy} onClick={() => state.release(sink.id)}>
                {sameOwner ? t('devices.stop') : t('devices.disconnect')}
              </Button>
            ) : null}
          </div>

          {state.errors[sink.id] && (
            <div className="text-xs text-red-400">{state.errors[sink.id]}</div>
          )}
        </div>
      </div>
    </div>
  );
};
