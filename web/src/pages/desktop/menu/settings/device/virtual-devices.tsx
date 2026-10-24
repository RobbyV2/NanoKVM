import { useEffect, useState } from 'react';
import { useAuth } from '@/contexts/auth.ts';
import { App, Button, Modal, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { cancelDownloadImage, statusImage } from '@/api/download.ts';
import { getHidMode } from '@/api/hid.ts';
import * as api from '@/api/virtual-device.ts';

const imageUpdatedEvent = 'nanokvm:image-updated';

// The disk half of the gadget. The network half, and everything that
// configures it, lives under Settings, Network, next to the rest of the
// networking it belongs to.
export const VirtualDevices = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const { hasPermission } = useAuth();
  // The switch is a device setting. The transfer enabling the disk can
  // interrupt belongs to the media library.
  const canManageMedia = hasPermission('library.write');

  const [isHidOnlyMode, setIsHidOnlyMode] = useState(false);
  const [isDiskEnabled, setIsDiskEnabled] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [loading, setLoading] = useState(false);

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
    } catch (err) {
      console.log(err);
    }
  }

  async function update() {
    if (loading) return;
    setLoading(true);

    try {
      const rsp = await api.updateVirtualDevice('disk');
      if (rsp.code !== 0) {
        // Enabling the disk is refused while a transfer or the file manager holds
        // /data. An account without library.write cannot cancel the transfer and
        // has to see why the switch did not move.
        message.error(rsp.msg || t('settings.device.updateFailed'));
        return;
      }

      await getVirtualDevice();

      // the desktop shows the mounted image and the CD-ROM flag too
      window.dispatchEvent(new Event(imageUpdatedEvent));
    } catch (err) {
      console.log(err);
      message.error(t('settings.device.updateFailed'));
    } finally {
      setLoading(false);
    }
  }

  // Enabling the disk serves the raw data partition to the target machine, which
  // takes /data away from the image downloader that writes there.
  async function toggleDisk() {
    if (isDiskEnabled) {
      await update();
      return;
    }

    setIsConfirmOpen(true);
  }

  // A transfer writes to /data, which the host takes over. Cancelling one needs
  // library.write, so an account without that grant cannot stop it and leaves the
  // toggle to the server, which refuses the change while a transfer holds /data.
  async function stopTransfer() {
    if (!canManageMedia) return;

    try {
      const rsp = await statusImage();
      if (rsp.code !== 0 || rsp.data?.status !== 'in_progress') return;

      const cancel = await cancelDownloadImage();
      if (cancel.code !== 0) {
        console.log(cancel.msg);
      }
    } catch (err) {
      console.log(err);
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

        <Switch checked={isDiskEnabled} loading={loading} onChange={toggleDisk} />
      </div>

      {/* configfs only adds or removes a function with the UDC unbound, so the
          switch re-enumerates the whole composite device. Turning it off also
          releases a mounted image. */}
      <div className="text-xs text-amber-500">{t('settings.device.rebindNotice')}</div>

      <Modal
        title={t('settings.device.attention')}
        open={isConfirmOpen}
        width={520}
        footer={null}
        onCancel={() => setIsConfirmOpen(false)}
      >
        <div className="flex flex-col items-center space-y-1 pb-10">
          <p>{t('settings.device.diskBreakDownload')}</p>
        </div>

        <div className="flex justify-center space-x-3 pb-3">
          <Button
            type="primary"
            loading={loading}
            onClick={async () => {
              setIsConfirmOpen(false);
              await stopTransfer();
              await update();
            }}
          >
            {t('settings.device.okBtn')}
          </Button>
          <Button onClick={() => setIsConfirmOpen(false)}>{t('settings.device.cancelBtn')}</Button>
        </div>
      </Modal>
    </>
  );
};
