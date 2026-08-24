import { useRef, useState } from 'react';
import { Button, Input } from 'antd';
import { ExternalLinkIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';

import { reloadWhenBack } from './state.ts';

interface UpdateProps {
  status: string;
  setStatus: (status: string) => void;
  setIsLocked: (isClosable: boolean) => void;
  setErrMsg: (msg: string) => void;
}

export const Offline = ({ status, setStatus, setIsLocked, setErrMsg }: UpdateProps) => {
  const { t } = useTranslation();

  const inputRef = useRef<HTMLInputElement | null>(null);
  const [sha256Checksum, setSha256Checksum] = useState('');
  const [kernelFile, setKernelFile] = useState<File | null>(null);

  function handleClick() {
    inputRef.current?.click();
  }

  async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) {
      return;
    }
    e.target.value = '';
    setKernelFile(null);

    if (!validateFilename(file.name)) {
      setStatus('failed');
      setErrMsg(t('settings.update.offline.invalidName'));
      return;
    }

    // A kernel reboots the device onto a trial slot, so it is the one package
    // worth naming before it is sent rather than after it has been written.
    if (await isKernelPackage(file)) {
      setKernelFile(file);
      return;
    }

    upload(file);
  }

  function upload(file: File | null) {
    if (!file) return;

    const checksum = sha256Checksum.trim();
    if (checksum && !/^[a-fA-F0-9]{64}$/.test(checksum)) {
      setStatus('failed');
      setErrMsg(t('settings.update.offline.invalidChecksum'));
      return;
    }

    if (!validateFilename(file.name)) {
      setStatus('failed');
      setErrMsg(t('settings.update.offline.invalidName'));
      return;
    }

    if (status === 'loading' || status === 'updating') {
      return;
    }

    setIsLocked(true);
    setStatus('updating');
    setErrMsg('');

    const formData = new FormData();
    formData.append('file', file);

    api
      .offlineUpdate(formData, checksum)
      .then(async (rsp: Response) => {
        // The proxy may return 502 after the update stops the old server.
        if (rsp.status === 502) return;
        if (!rsp.ok) throw new Error(`HTTP error ${rsp.status}`);

        const rspj = await rsp.json();
        if (rspj.code !== 0) {
          const message = rspj.msg?.includes('sha256 checksum mismatch')
            ? t('settings.update.offline.checksumMismatch')
            : rspj.msg || t('settings.update.offline.updateFailed');
          throw new Error(message);
        }
        return !!rspj.data?.reboot;
      })
      .then((isRebooting) => {
        if (isRebooting) setStatus('rebooting');
        reloadWhenBack(!!isRebooting, api.getKernel, () => {
          setIsLocked(false);
          window.location.reload();
        });
      })
      .catch((error: unknown) => {
        setIsLocked(false);
        setStatus('failed');
        setErrMsg(
          error instanceof Error ? error.message : t('settings.update.offline.updateFailed')
        );
      });
  }

  function validateFilename(filename: string) {
    const regex: RegExp = /^nanokvm_\d+\.\d+\.\d+\.tar\.gz$/;
    return regex.test(filename);
  }

  function confirmKernel() {
    const file = kernelFile;
    setKernelFile(null);
    upload(file);
  }

  return (
    <>
      <div className="mt-8 flex flex-col gap-3">
        <div className="flex items-center justify-between gap-4">
          <div className="flex flex-col space-y-1">
            <div className="flex items-center space-x-2">
              <span>{t('settings.update.offline.title')}</span>

              <a
                className="flex items-center text-neutral-500 hover:text-blue-500"
                href="https://github.com/sipeed/NanoKVM/releases"
                target="_blank"
              >
                <ExternalLinkIcon size={15} />
              </a>
            </div>

            <span className="text-xs text-neutral-500">{t('settings.update.offline.desc')}</span>
          </div>

          <input
            id="file-upload"
            ref={inputRef}
            type="file"
            accept=".tar.gz"
            onChange={handleFileChange}
            className="hidden"
          />
          <Button disabled={status === 'loading' || status === 'updating'} onClick={handleClick}>
            {t('settings.update.offline.upload')}
          </Button>
        </div>

        <Input
          value={sha256Checksum}
          maxLength={64}
          disabled={status === 'updating'}
          placeholder={t('settings.update.offline.checksumPlaceholder')}
          onChange={(event) => setSha256Checksum(event.target.value)}
        />

        {kernelFile && (
          <div className="flex flex-col space-y-2">
            <span className="text-xs text-amber-500">
              {t('settings.update.offline.kernelNotice')}
            </span>

            <div className="flex gap-2">
              <Button size="small" type="primary" onClick={confirmKernel}>
                {t('settings.update.offline.kernelConfirm')}
              </Button>
              <Button size="small" onClick={() => setKernelFile(null)}>
                {t('settings.update.offline.kernelCancel')}
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  );
};

// An app release and a kernel release are both nanokvm_<version>.tar.gz, and the
// server only discovers which it has after the package is written to the trial
// slot. Reading the archive here is the only way to tell the operator what the
// upload will do while it can still be cancelled. A kernel package carries
// nothing but version and kernel/, so the walk stops at the first entry that
// settles it either way and never decompresses the payload.
async function isKernelPackage(file: File) {
  if (typeof DecompressionStream === 'undefined') return false;

  const reader = file.stream().pipeThrough(new DecompressionStream('gzip')).getReader();
  let buffer = new Uint8Array(0);

  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) return false;

      const merged = new Uint8Array(buffer.length + value.length);
      merged.set(buffer);
      merged.set(value, buffer.length);
      buffer = merged;

      while (buffer.length >= 512) {
        const name = tarString(buffer, 0, 100);
        if (name === '') return false;

        const size = parseInt(tarString(buffer, 124, 12).trim() || '0', 8) || 0;
        const entry = name.replace(/^[^/]*\//, '');
        if (entry !== '' && entry !== 'version') {
          return entry.startsWith('kernel/');
        }
        buffer = buffer.subarray(512 + Math.ceil(size / 512) * 512);
      }
    }
  } catch {
    return false;
  } finally {
    await reader.cancel().catch(() => {});
  }
}

function tarString(bytes: Uint8Array, offset: number, length: number) {
  const field = bytes.subarray(offset, offset + length);
  const end = field.indexOf(0);
  return new TextDecoder().decode(end < 0 ? field : field.subarray(0, end));
}
