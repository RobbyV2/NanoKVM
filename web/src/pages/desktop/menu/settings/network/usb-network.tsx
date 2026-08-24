import { useCallback, useEffect, useState } from 'react';
import { Segmented, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { getHidMode } from '@/api/hid.ts';
import * as api from '@/api/virtual-device.ts';
import type { NetworkProtocol } from '@/api/virtual-device.ts';

import { Bridge } from './bridge.tsx';

// the gadget layer builds f_ncm and f_rndis and nothing else, so these are the
// only two the selector offers
const protocols: NetworkProtocol[] = ['ncm', 'rndis'];

// The adapter switch and everything that configures it, as one group. The two
// children below are settings of the adapter, not peers of it: the class the
// host binds, and where the far end of the link goes. Both are meaningless
// with no adapter, so both are inert while the switch is off.
export const UsbNetwork = () => {
  const { t } = useTranslation();

  const [isHidOnlyMode, setIsHidOnlyMode] = useState(false);
  const [isEnabled, setIsEnabled] = useState(false);
  const [protocol, setProtocol] = useState<NetworkProtocol>('rndis');
  const [loading, setLoading] = useState<'' | 'adapter' | 'protocol'>('');

  const read = useCallback(async () => {
    try {
      const rsp = await api.getVirtualDevice();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setIsEnabled(rsp.data.network);

      // an unmounted gadget reports no protocol, so the last one stands
      if (protocols.includes(rsp.data.protocol)) {
        setProtocol(rsp.data.protocol);
      }
    } catch (err) {
      console.log(err);
    }
  }, []);

  useEffect(() => {
    getHidMode()
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }
        setIsHidOnlyMode(rsp.data.mode === 'hid-only');
      })
      .catch((err) => console.log(err));

    void read();
  }, [read]);

  async function toggle() {
    if (loading) return;
    setLoading('adapter');

    try {
      const rsp = await api.updateVirtualDevice('network');
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      await read();
    } catch (err) {
      console.log(err);
    } finally {
      setLoading('');
    }
  }

  // a protocol is a switch rather than a remount: the network function is
  // replaced in place, so the host sees one adapter change class
  async function selectProtocol(next: NetworkProtocol) {
    if (loading || next === protocol) return;
    setLoading('protocol');

    try {
      const rsp = await api.updateVirtualDevice('network', next);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      await read();
    } catch (err) {
      console.log(err);
    } finally {
      setLoading('');
    }
  }

  // HID-only mode builds no network function at all, so the adapter cannot be
  // turned on and nothing under it applies
  const isOff = isHidOnlyMode || !isEnabled;

  return (
    <div className="flex flex-col space-y-4">
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.network.usb.title')}</span>
          <span className="text-xs text-neutral-500">
            {isHidOnlyMode ? t('settings.device.hidOnlyDesc') : t('settings.network.usb.desc')}
          </span>
        </div>

        <Switch
          checked={isEnabled && !isHidOnlyMode}
          loading={loading === 'adapter'}
          disabled={isHidOnlyMode}
          onChange={toggle}
        />
      </div>

      {!isHidOnlyMode && (
        <div className="text-xs text-amber-500">{t('settings.device.rebindNotice')}</div>
      )}

      {/* the rule and the indent are the dependency: what is inside belongs to
          the switch above it, and greys out with it */}
      <div
        className={`ml-1 flex flex-col space-y-6 border-l border-neutral-700/60 pl-4 ${
          isOff ? 'opacity-40' : ''
        }`}
      >
        <div className="flex items-center justify-between space-x-10">
          <div className="flex flex-col space-y-1">
            <span>{t('settings.network.usb.type')}</span>
            <span className="text-xs text-neutral-500">{t('settings.network.usb.typeDesc')}</span>
          </div>

          <Segmented
            disabled={isOff || loading !== ''}
            value={protocol}
            onChange={(value) => selectProtocol(value as NetworkProtocol)}
            options={protocols.map((value) => ({ label: value.toUpperCase(), value }))}
          />
        </div>

        <Bridge disabled={isOff} />
      </div>
    </div>
  );
};
