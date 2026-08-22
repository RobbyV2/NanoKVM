import { useEffect, useState } from 'react';
import { Segmented, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { getHidMode } from '@/api/hid.ts';
import * as api from '@/api/virtual-device.ts';
import type { NetworkProtocol } from '@/api/virtual-device.ts';

// the gadget layer builds f_ncm and f_rndis and nothing else, so these are the
// only two the selector offers
const protocols: NetworkProtocol[] = ['ncm', 'rndis'];

export const VirtualDevices = () => {
  const { t } = useTranslation();

  const [isHidOnlyMode, setIsHidOnlyMode] = useState(false);
  const [isDiskEnabled, setIsDiskEnabled] = useState(false);
  const [isNetworkEnabled, setIsNetworkEnabled] = useState(false);
  const [protocol, setProtocol] = useState<NetworkProtocol>('rndis');
  const [loading, setLoading] = useState<'' | 'disk' | 'network' | 'protocol'>('');

  useEffect(() => {
    getHidOnlyMode();
    getVirtualDevice();
  }, []);

  async function getHidOnlyMode() {
    try {
      const rsp = await getHidMode();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }
      setIsHidOnlyMode(rsp.data.mode === 'hid-only');
    } catch (err) {
      console.log(err);
    }
  }

  async function getVirtualDevice() {
    try {
      const rsp = await api.getVirtualDevice();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setIsDiskEnabled(rsp.data.disk);
      setIsNetworkEnabled(rsp.data.network);

      // an unmounted gadget reports no protocol, so the last one stands
      if (protocols.includes(rsp.data.protocol)) {
        setProtocol(rsp.data.protocol);
      }
    } catch (err) {
      console.log(err);
    }
  }

  async function update(device: 'disk' | 'network') {
    if (loading) return;
    setLoading(device);

    try {
      const rsp = await api.updateVirtualDevice(device);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      await getVirtualDevice();
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

      await getVirtualDevice();
    } catch (err) {
      console.log(err);
    } finally {
      setLoading('');
    }
  }

  if (isHidOnlyMode) {
    return (
      <div className="flex items-center justify-between space-x-10">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.hidOnly')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.hidOnlyDesc')}</span>
        </div>

        <Switch checked={true} disabled={true} />
      </div>
    );
  }

  return (
    <>
      {/* Virtual Disk */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.disk')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.diskDesc')}</span>
        </div>

        <Switch
          checked={isDiskEnabled}
          loading={loading === 'disk'}
          onChange={() => update('disk')}
        />
      </div>

      {/* Virtual Network */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.network')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.networkDesc')}</span>
        </div>

        <Switch
          checked={isNetworkEnabled}
          loading={loading === 'network'}
          onChange={() => update('network')}
        />
      </div>

      {/* The protocol the gadget presents. It decides what the attached host
          binds whether or not a bridge exists, so it lives with the USB profile
          rather than on the bridge panel, which only names it. */}
      {isNetworkEnabled && (
        <div className="flex items-center justify-between space-x-10">
          <div className="flex flex-col space-y-1">
            <span>{t('settings.device.networkProtocol')}</span>
            <span className="text-xs text-neutral-500">
              {t('settings.device.networkProtocolDesc')}
            </span>
          </div>

          <Segmented
            disabled={loading !== ''}
            value={protocol}
            onChange={(value) => selectProtocol(value as NetworkProtocol)}
            options={protocols.map((value) => ({ label: value.toUpperCase(), value }))}
          />
        </div>
      )}
    </>
  );
};
