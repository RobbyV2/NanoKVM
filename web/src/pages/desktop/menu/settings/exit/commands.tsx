import { useMemo, useState } from 'react';
import { Button, Input, Segmented, Tabs } from 'antd';
import {
  CheckIcon,
  CopyIcon,
  DownloadIcon,
  LoaderCircleIcon,
  ShieldAlertIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { downloadNexit } from '@/api/extensions/exit.ts';
import { copyText } from '@/lib/clipboard.ts';
import { getBaseUrl } from '@/lib/service.ts';

import { detectPlatform, nexitDownloads, parseOrigin, presentCommand } from './state.ts';
import type { NexitDownload } from './state.ts';
import { exitPlatforms } from './types.ts';
import type { ExitCommands as Commands, ExitPlatform, ExitWay } from './types.ts';

type ExitCommandsProps = {
  slot: string;
  // the slot's token, which the nexit downloads present as a bearer
  token: string;
  // the gate serves nothing for a disabled slot
  enabled: boolean;
  way: ExitWay;
  commands?: Commands;
  hasConnected: boolean;
};

export const ExitCommands = ({
  slot,
  token,
  enabled,
  way,
  commands,
  hasConnected,
}: ExitCommandsProps) => {
  const { t } = useTranslation();

  const [platform, setPlatform] = useState<ExitPlatform>(() => detectPlatform(navigator.userAgent));
  const [shell, setShell] = useState('');
  const [copiedKey, setCopiedKey] = useState('');
  const [errMsg, setErrMsg] = useState('');
  const [downloading, setDownloading] = useState('');

  // the server templated from the request it saw; the browser knows better
  // what host it actually reached (D15), but the scheme decides flags the
  // panel cannot re-template, so a scheme mismatch is shown, not fixed
  const local = useMemo(() => parseOrigin(getBaseUrl('http')), []);
  const server = commands ? { scheme: commands.scheme, host: commands.host } : local;

  const list = commands ? commands[way] : [];
  // A mode offers what it offers: Mode A is the nexit binary and Windows only,
  // Mode B carries all three. Showing a tab the server sent no command for is
  // how the panel ends up saying "no command for this platform".
  const platforms = exitPlatforms.filter((value) =>
    list.some((command) => command.platform === value)
  );
  // a platform can offer more than one shell (windows has powershell and cmd),
  // so the shell is part of the selection and falls back to the first on offer
  const activePlatform = platforms.includes(platform) ? platform : (platforms[0] ?? platform);
  const shells = list.filter((command) => command.platform === activePlatform).map((c) => c.shell);
  const activeShell = shells.includes(shell) ? shell : (shells[0] ?? '');
  const current = list.find(
    (command) => command.platform === activePlatform && command.shell === activeShell
  );
  const view = presentCommand(way, current, server, local);

  // the latest-release variant of the wstunnel command, one per platform; an
  // older server does not send the list at all, and then the block is not shown.
  // It is only rendered for shells it was built for, so fall back by platform.
  const latestList = way === 'wstunnel' ? commands?.wstunnelLatest : undefined;
  const latest =
    latestList?.find(
      (command) => command.platform === activePlatform && command.shell === activeShell
    ) ?? latestList?.find((command) => command.platform === activePlatform);
  const latestView = presentCommand(way, latest, server, local);
  const repo = commands?.wstunnelRepo ?? '';

  const serverAddress = `${server.scheme}://${server.host}`;
  const localAddress = `${local.scheme}://${local.host}`;

  function copy(key: string, text: string) {
    copyText(text)
      .then(() => {
        setCopiedKey(key);
        setErrMsg('');
        setTimeout(() => setCopiedKey(''), 1500);
      })
      .catch(() => {
        setErrMsg(t('settings.exit.copyFailed'));
      });
  }

  function download(item: NexitDownload) {
    setDownloading(item.key);
    setErrMsg('');
    downloadNexit(slot, token, item)
      .catch((err) => {
        setErrMsg(
          t('settings.exit.commands.nexitFiles.failed', { error: err?.message || String(err) })
        );
      })
      .finally(() => setDownloading(''));
  }

  return (
    <div className="flex flex-col space-y-3">
      <div className="flex flex-col space-y-1">
        <span>{t('settings.exit.commands.title')}</span>
        <span className="text-xs text-neutral-500">{t('settings.exit.commands.description')}</span>
      </div>

      <div className="flex items-start space-x-2 rounded-lg bg-amber-500/10 px-3 py-2 text-xs text-amber-500">
        <ShieldAlertIcon size={14} className="mt-[1px] shrink-0" />
        <ul className="list-disc space-y-1 pl-4">
          <li>{t('settings.exit.commands.warnDownload')}</li>
          <li>{t('settings.exit.commands.warnSecret')}</li>
          <li>{t('settings.exit.commands.warnReach')}</li>
          {view.cleartext && <li>{t('settings.exit.commands.warnCleartext')}</li>}
          {view.trustsTransport && <li>{t('settings.exit.commands.warnTransportTrusted')}</li>}
          {way === 'wstunnel' && activePlatform === 'windows' && (
            <li>{t('settings.exit.commands.warnWstunnelDefender')}</li>
          )}
          {view.schemeMismatch && (
            <li className="break-all">
              {t('settings.exit.commands.warnSchemeMismatch', {
                server: serverAddress,
                local: localAddress
              })}
            </li>
          )}
        </ul>
      </div>

      {!commands ? (
        <div className="flex items-center space-x-2 text-xs text-neutral-500">
          <LoaderCircleIcon className="animate-spin" size={14} />
          <span>{t('settings.exit.loading')}</span>
        </div>
      ) : (
        <Tabs
          size="small"
          activeKey={activePlatform}
          onChange={(key) => setPlatform(key as ExitPlatform)}
          items={platforms.map((value) => ({
            key: value,
            label: t(`settings.exit.commands.${value}`),
            children: current ? (
              <div className="flex flex-col space-y-2">
                {shells.length > 1 && (
                  <Segmented
                    size="small"
                    value={activeShell}
                    onChange={(value) => setShell(value as string)}
                    options={shells.map((value) => ({
                      value,
                      label: t(`settings.exit.commands.shell.${value}`, value)
                    }))}
                  />
                )}
                <Input.TextArea
                  value={view.text}
                  readOnly
                  autoSize={{ minRows: 2, maxRows: 8 }}
                  spellCheck={false}
                  className="font-mono text-xs"
                  onFocus={(e) => e.target.select()}
                />

                <div className="flex items-center space-x-2">
                  <Button
                    size="small"
                    type="primary"
                    icon={copiedKey === value ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
                    onClick={() => copy(value, view.text)}
                  >
                    {t(copiedKey === value ? 'settings.exit.copied' : 'settings.exit.copy')}
                  </Button>

                </div>

                {current.notes && <span className="text-xs text-neutral-500">{current.notes}</span>}

                {view.rewritten && (
                  <span className="text-xs text-neutral-500">
                    {t('settings.exit.commands.rewritten', { host: localAddress })}
                  </span>
                )}

                {way === 'wstunnel' && latestList && (
                  <div className="flex flex-col space-y-2 border-t border-neutral-700/60 pt-2">
                    <span className="text-xs">{t('settings.exit.commands.latestTitle')}</span>

                    {latest && (
                      <>
                        <span className="text-xs text-neutral-500">
                          {t('settings.exit.commands.latestDesc', {
                            version: commands?.wstunnelVersion
                          })}
                        </span>

                        <Input.TextArea
                          value={latestView.text}
                          readOnly
                          autoSize={{ minRows: 2, maxRows: 8 }}
                          spellCheck={false}
                          className="font-mono text-xs"
                          onFocus={(e) => e.target.select()}
                        />

                        <div className="flex items-center space-x-2">
                          <Button
                            size="small"
                            type="primary"
                            icon={
                              copiedKey === `${value}-latest` ? (
                                <CheckIcon size={14} />
                              ) : (
                                <CopyIcon size={14} />
                              )
                            }
                            onClick={() => copy(`${value}-latest`, latestView.text)}
                          >
                            {t(
                              copiedKey === `${value}-latest`
                                ? 'settings.exit.copied'
                                : 'settings.exit.copy'
                            )}
                          </Button>
                        </div>

                        {latest.notes && (
                          <span className="text-xs text-neutral-500">{latest.notes}</span>
                        )}

                        {latestView.rewritten && (
                          <span className="text-xs text-neutral-500">
                            {t('settings.exit.commands.rewritten', { host: localAddress })}
                          </span>
                        )}
                      </>
                    )}

                    <a
                      className="text-xs text-neutral-500 hover:text-blue-500"
                      href={repo}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {t('settings.exit.commands.latestFallback')}
                    </a>
                  </div>
                )}
              </div>
            ) : (
              <span className="text-xs text-neutral-500">
                {t('settings.exit.commands.unavailable')}
              </span>
            )
          }))}
        />
      )}

      {way === 'nexit' && commands && (
        <div className="flex flex-col space-y-2 border-t border-neutral-700/60 pt-2">
          <span className="text-xs">{t('settings.exit.commands.nexitFiles.title')}</span>

          <div className="flex flex-wrap items-center gap-2">
            {nexitDownloads.map((item) => (
              <Button
                key={item.key}
                size="small"
                icon={<DownloadIcon size={14} />}
                loading={downloading === item.key}
                disabled={!enabled || !token || (downloading !== '' && downloading !== item.key)}
                onClick={() => download(item)}
              >
                {t(`settings.exit.commands.nexitFiles.${item.key}`)}
              </Button>
            ))}
          </div>

          {!enabled && (
            <span className="text-xs text-neutral-500">
              {t('settings.exit.commands.nexitFiles.needsEnabled')}
            </span>
          )}

          <span className="text-xs text-neutral-500">
            {t('settings.exit.commands.nexitFiles.instructions')}
          </span>
          <span className="text-xs text-amber-500">
            {t('settings.exit.commands.nexitFiles.secret')}
          </span>
        </div>
      )}

      {hasConnected && (
        <span className="text-xs text-neutral-500">
          {t('settings.exit.commands.regenerateHint')}
        </span>
      )}

      <span className="text-xs text-neutral-500">{t('settings.exit.commands.security')}</span>

      {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}

    </div>
  );
};
