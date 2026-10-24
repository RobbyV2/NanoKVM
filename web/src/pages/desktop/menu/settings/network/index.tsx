import { useAuth } from '@/contexts/auth.ts';
import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import { DNS } from './dns.tsx';
import { Tls } from './tls.tsx';
import { UsbNetwork } from './usb-network.tsx';
import { Wifi } from './wifi.tsx';

export const Network = () => {
  const { t } = useTranslation();
  const { hasPermission } = useAuth();
  // The USB network adapter is an emulated device, so its switch follows the
  // device grant like the virtual disk's; Wi-Fi, DNS and TLS are this tab's own.
  const canDevice = hasPermission('device');

  return (
    <>
      <div className="text-base">{t('settings.network.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-8">
        <Tls />
        <Wifi />
      </div>

      <Divider className="opacity-50" style={{ margin: '32px 0' }} />

      <DNS />

      {canDevice && (
        <>
          <Divider className="opacity-50" style={{ margin: '32px 0' }} />
          <UsbNetwork />
        </>
      )}
    </>
  );
};
