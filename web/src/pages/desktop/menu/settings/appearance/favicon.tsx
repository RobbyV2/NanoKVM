import { useEffect, useRef, useState } from 'react';
import { Button, Input } from 'antd';
import { useAtom } from 'jotai';
import { RotateCcwIcon, UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import {
  applyFavicon,
  faviconErrorKey,
  faviconHref,
  faviconSourceKey,
  isFaviconUrl
} from '@/lib/favicon.ts';
import { faviconVersionAtom } from '@/jotai/settings.ts';

type FaviconState = {
  source: string;
  version?: string;
  bootLogo?: boolean;
};

export const Favicon = () => {
  const { t } = useTranslation();
  const [version, setVersion] = useAtom(faviconVersionAtom);

  const [source, setSource] = useState('default');
  const [bootLogo, setBootLogo] = useState(false);
  const [url, setUrl] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  const fileRef = useRef<HTMLInputElement>(null);
  // The URL box commits on blur like the web title beside it, so it has to
  // remember what it last sent: blurring an unchanged field must not re-run a
  // download on the device.
  const submittedRef = useRef('');

  useEffect(() => {
    api
      .getFavicon()
      .then((rsp) => {
        if (rsp.code !== 0) return;
        adopt(rsp.data as FaviconState);
      })
      .catch(() => {
        // A panel that cannot read the state still renders; the tab keeps
        // whatever icon the document already asked for.
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function adopt(state: FaviconState) {
    setSource(state?.source || 'default');
    setBootLogo(Boolean(state?.bootLogo));
    setVersion(state?.version || '');
    // Repoint the live tab. Nothing here reloads the page, so this is what
    // makes the change visible immediately.
    applyFavicon(document, state?.version);
  }

  function handle(rsp: { code: number; data: unknown }) {
    if (rsp.code !== 0) {
      setErrMsg(t(faviconErrorKey(rsp.code)));
      return;
    }
    setErrMsg('');
    adopt(rsp.data as FaviconState);
  }

  function submitUrl() {
    const trimmed = url.trim();
    if (isLoading || trimmed === submittedRef.current) return;

    if (!trimmed) {
      // Clearing the box is not a reset: resetting is an explicit button, so
      // that tabbing through an empty field cannot throw the icon away.
      submittedRef.current = '';
      setErrMsg('');
      return;
    }

    if (!isFaviconUrl(trimmed)) {
      setErrMsg(t('settings.appearance.faviconErrUrl'));
      return;
    }

    submittedRef.current = trimmed;
    setIsLoading(true);

    api
      .setFavicon(trimmed)
      .then((rsp) => {
        handle(rsp);
        if (rsp.code === 0) {
          // The device stored the bytes, not the address. Leaving the URL in
          // the box would show a value that is no longer what is live.
          setUrl('');
          submittedRef.current = '';
        }
      })
      .catch(() => setErrMsg(t('settings.appearance.faviconErrFetch')))
      .finally(() => setIsLoading(false));
  }

  function upload(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file || isLoading) return;

    setIsLoading(true);

    api
      .uploadFavicon(file)
      .then(handle)
      .catch(() => setErrMsg(t('settings.appearance.faviconErrSave')))
      .finally(() => setIsLoading(false));
  }

  function reset() {
    if (isLoading) return;
    setIsLoading(true);
    setUrl('');
    submittedRef.current = '';

    api
      .setFavicon('')
      .then(handle)
      .catch(() => setErrMsg(t('settings.appearance.faviconErrSave')))
      .finally(() => setIsLoading(false));
  }

  return (
    <div className="mt-8 flex flex-col space-y-2">
      <div className="flex items-center justify-between space-x-5">
        <div className="flex flex-col">
          <span>{t('settings.appearance.favicon')}</span>
          <span className="text-xs text-neutral-500">{t('settings.appearance.faviconDesc')}</span>
        </div>

        <div className="flex shrink-0 items-center space-x-2">
          <input
            ref={fileRef}
            type="file"
            accept="image/*,.ico"
            className="hidden"
            onChange={upload}
          />

          <Input
            disabled={isLoading}
            style={{ width: 180 }}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onPressEnter={submitUrl}
            onBlur={submitUrl}
            placeholder={t('settings.appearance.faviconPlaceholder')}
          />

          <Button
            ghost
            type="primary"
            size="small"
            icon={<UploadIcon size={14} />}
            loading={isLoading}
            onClick={() => fileRef.current?.click()}
          >
            {t('settings.appearance.faviconUpload')}
          </Button>

          {source === 'custom' && (
            <Button
              danger
              size="small"
              icon={<RotateCcwIcon size={14} />}
              disabled={isLoading}
              onClick={reset}
            >
              {t('settings.appearance.faviconReset')}
            </Button>
          )}
        </div>
      </div>

      <div className="flex items-center space-x-2 text-xs text-neutral-500">
        <img
          src={faviconHref(version)}
          alt=""
          className="h-4 w-4 shrink-0 rounded-sm object-contain"
        />
        <span>{t(faviconSourceKey(source))}</span>
        {source === 'custom' && bootLogo && (
          <span className="text-amber-500">{t('settings.appearance.faviconOverridesBoot')}</span>
        )}
      </div>

      {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}
    </div>
  );
};
