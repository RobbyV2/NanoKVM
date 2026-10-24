import { useAuth } from '@/contexts/auth.ts';
import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import { Advanced } from './advanced';
import { Hdmi } from './hdmi.tsx';
import { Mdns } from './mdns.tsx';
import { MediaSlots } from './media-slots.tsx';
import { MouseJiggler } from './mouse-jiggler.tsx';
import { Oled } from './oled.tsx';
import { Reboot } from './reboot.tsx';
import { Ssh } from './ssh.tsx';
import { VirtualDevices } from './virtual-devices.tsx';

export const Device = () => {
  const { t } = useTranslation();
  const { account, hasPermission } = useAuth();
  // The media slots reshape the USB presentation, which stays with the
  // administrator role.
  const isAdmin = account.role === 'admin';
  // SSH logs in as root, so its switch needs the terminal grant.
  const canTerminal = hasPermission('terminal');

  return (
    <>
      <div className="text-base">{t('settings.device.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-8">
        {canTerminal && <Ssh />}
        <Mdns />
        <Hdmi />
        <Divider className="opacity-50" />

        {/* The emulated USB devices are device settings, like the rest of the tab */}
        <VirtualDevices />
        <Divider className="opacity-50" />

        {isAdmin && (
          <>
            <MediaSlots />
            <Divider className="opacity-50" />
          </>
        )}

        <Oled />
        <MouseJiggler />
        <Divider className="opacity-50" />

        <Advanced />
      </div>

      <Divider className="opacity-50" />

      <Reboot />
    </>
  );
};
