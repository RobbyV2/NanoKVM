import { useEffect, useState } from 'react';
import { Button, Collapse, Input, InputNumber, Switch } from 'antd';
import { PlusIcon, XIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/exit.ts';

import { advancedValuesKey, isValidDNS } from './state.ts';
import type { ExitAdvancedValues } from './state.ts';
import { dnsMax, mtuMax, mtuMin } from './types.ts';
import type { ExitConfig } from './types.ts';

type ExitAdvancedProps = {
  slot: string;
  config?: ExitConfig;
  disabled: boolean;
  // only the fields the form saved; the parent merges them into its config
  onSaved: (saved: ExitAdvancedValues) => void;
  // the parent holds the mode selector still while a save is out, so the two
  // config writes cannot race
  onSavingChange: (isSaving: boolean) => void;
};

export const ExitAdvanced = ({
  slot,
  config,
  disabled,
  onSaved,
  onSavingChange
}: ExitAdvancedProps) => {
  const { t } = useTranslation();

  const [dns, setDns] = useState<string[]>([]);
  const [mtu, setMtu] = useState<number | null>(null);
  const [allowPrivate, setAllowPrivate] = useState(false);
  const [pinPeer, setPinPeer] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isSaved, setIsSaved] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  // the form follows the server until the operator edits it; a save writes
  // back and the parent's config moves on to the saved values. It is keyed on
  // those values, not on the config object: a mode switch swaps the object
  // and must not throw away an unsaved edit
  const valuesKey = advancedValuesKey(config);
  useEffect(() => {
    if (!config) return;
    setDns(config.dns ?? []);
    setMtu(config.mtu);
    setAllowPrivate(config.allowPrivate);
    setPinPeer(config.pinPeer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [valuesKey]);

  useEffect(() => {
    onSavingChange(isSaving);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isSaving]);

  const isDnsValid = dns.every(isValidDNS);
  const isMtuValid = mtu !== null && mtu >= mtuMin && mtu <= mtuMax;
  const isDirty =
    !!config &&
    (dns.join(',') !== (config.dns ?? []).join(',') ||
      mtu !== config.mtu ||
      allowPrivate !== config.allowPrivate ||
      pinPeer !== config.pinPeer);

  function save() {
    if (isSaving || !config || !isDnsValid || !isMtuValid || mtu === null) return;
    setIsSaving(true);
    setIsSaved(false);
    setErrMsg('');

    const next: ExitAdvancedValues = {
      dns: dns.map((address) => address.trim()),
      mtu,
      allowPrivate,
      pinPeer
    };

    api
      .setConfig(slot, next)
      .then((rsp) => {
        if (rsp.code !== 0) {
          setErrMsg(rsp.msg);
          return;
        }

        setIsSaved(true);
        onSaved(next);
      })
      .catch((err) => {
        setErrMsg(err?.message || 'Failed to save exit config');
      })
      .finally(() => {
        setIsSaving(false);
      });
  }

  const body = (
    <div className="flex flex-col space-y-6 pt-3">
      {/* dns servers */}
      <div className="flex flex-col space-y-2">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.exit.advanced.dns')}</span>
          <span className="text-xs text-neutral-500">{t('settings.exit.advanced.dnsTip')}</span>
        </div>

        <div className="flex flex-col space-y-2">
          {dns.map((address, index) => (
            <div key={index} className="flex items-center space-x-2">
              <Input
                value={address}
                status={address && !isValidDNS(address) ? 'error' : ''}
                placeholder="1.1.1.1"
                spellCheck={false}
                className="font-mono"
                disabled={disabled || !config}
                onChange={(e) => {
                  const value = e.target.value;
                  setDns((current) => current.map((item, i) => (i === index ? value : item)));
                  setIsSaved(false);
                }}
              />
              <Button
                type="text"
                size="small"
                disabled={disabled || dns.length <= 1}
                icon={<XIcon size={14} />}
                onClick={() => {
                  setDns((current) => current.filter((_, i) => i !== index));
                  setIsSaved(false);
                }}
              />
            </div>
          ))}

          {dns.length < dnsMax && (
            <Button
              type="text"
              size="small"
              className="self-start"
              disabled={disabled || !config}
              icon={<PlusIcon size={14} />}
              onClick={() => {
                setDns((current) => [...current, '']);
                setIsSaved(false);
              }}
            >
              {t('settings.exit.advanced.dnsAdd')}
            </Button>
          )}
        </div>

        {!isDnsValid && (
          <span className="text-xs text-red-500">{t('settings.exit.advanced.dnsInvalid')}</span>
        )}
      </div>

      {/* mtu */}
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.exit.advanced.mtu')}</span>
          <span className="text-xs text-neutral-500">
            {t('settings.exit.advanced.mtuTip', { min: mtuMin, max: mtuMax })}
          </span>
        </div>

        <InputNumber
          value={mtu}
          min={mtuMin}
          max={mtuMax}
          status={isMtuValid ? '' : 'error'}
          disabled={disabled || !config}
          onChange={(value) => {
            setMtu(value);
            setIsSaved(false);
          }}
        />
      </div>

      {/* private destinations */}
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.exit.advanced.allowPrivate')}</span>
          <span className="text-xs text-neutral-500">
            {t('settings.exit.advanced.allowPrivateTip')}
          </span>
        </div>

        <Switch
          className="shrink-0"
          checked={allowPrivate}
          disabled={disabled || !config}
          onChange={(value) => {
            setAllowPrivate(value);
            setIsSaved(false);
          }}
        />
      </div>

      {/* pin peer */}
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.exit.advanced.pinPeer')}</span>
          <span className="text-xs text-neutral-500">{t('settings.exit.advanced.pinPeerTip')}</span>
        </div>

        <Switch
          className="shrink-0"
          checked={pinPeer}
          disabled={disabled || !config}
          onChange={(value) => {
            setPinPeer(value);
            setIsSaved(false);
          }}
        />
      </div>

      <div className="flex items-center space-x-3">
        <Button
          type="primary"
          size="small"
          loading={isSaving}
          disabled={disabled || !config || !isDirty || !isDnsValid || !isMtuValid}
          onClick={save}
        >
          {t('settings.exit.advanced.save')}
        </Button>

        {isSaved && (
          <span className="text-xs text-green-500">{t('settings.exit.advanced.saved')}</span>
        )}
        {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}
      </div>
    </div>
  );

  return (
    <Collapse
      ghost
      items={[
        {
          key: 'advanced',
          label: <span className="text-neutral-200">{t('settings.exit.advanced.title')}</span>,
          children: body
        }
      ]}
    />
  );
};
