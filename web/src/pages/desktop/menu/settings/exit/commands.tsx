import { useMemo, useState } from 'react';
import { Button, Input, Modal, Tabs, Tooltip } from 'antd';
import { CheckIcon, CopyIcon, FileCodeIcon, LoaderCircleIcon, ShieldAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { copyText } from '@/lib/clipboard.ts';
import { getBaseUrl } from '@/lib/service.ts';
import { ScrollArea } from '@/components/ui/scroll-area.tsx';

import { detectPlatform, parseOrigin, presentCommand, scriptErrorKey } from './state.ts';
import { exitPlatforms } from './types.ts';
import type { ExitCommands as Commands, ExitMode, ExitPlatform } from './types.ts';

type ExitCommandsProps = {
  slot: string;
  mode: ExitMode;
  token: string;
  commands?: Commands;
  hasConnected: boolean;
  // the gate serves the script only while the slot is enabled and settled
  scriptServed: boolean;
};

// what each platform's one-liner fetches, and therefore what "view script" shows
const scriptNames: Record<ExitPlatform, string> = {
  windows: 'client.ps1',
  macos: 'client.sh',
  linux: 'client.sh'
};

export const ExitCommands = ({
  slot,
  mode,
  token,
  commands,
  hasConnected,
  scriptServed
}: ExitCommandsProps) => {
  const { t } = useTranslation();

  const [platform, setPlatform] = useState<ExitPlatform>(() => detectPlatform(navigator.userAgent));
  const [copiedKey, setCopiedKey] = useState('');
  const [script, setScript] = useState<{ name: string; body: string }>();
  const [isScriptLoading, setIsScriptLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  // the server templated from the request it saw; the browser knows better
  // what host it actually reached (D15), but the scheme decides flags the
  // panel cannot re-template, so a scheme mismatch is shown, not fixed
  const local = useMemo(() => parseOrigin(getBaseUrl('http')), []);
  const server = commands ? { scheme: commands.scheme, host: commands.host } : local;

  const list = commands ? commands[mode] : [];
  const current = list.find((command) => command.platform === platform);
  const view = presentCommand(mode, current, server, local, commands?.fingerprint ?? '');

  // the latest-release variant of the wstunnel command, one per platform; an
  // older server does not send the list at all, and then the block is not shown
  const latestList = commands?.wstunnelLatest;
  const latest = latestList?.find((command) => command.platform === platform);
  const latestView = presentCommand(mode, latest, server, local, commands?.fingerprint ?? '');
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

  // the script route is token-gated outside /api, so a plain link would 404;
  // the panel fetches it with the header the one-liner would send. Every
  // rejection there counts against this address (D10), so the button is dead
  // while the gate would refuse anyway
  function viewScript() {
    if (isScriptLoading || !scriptServed) return;
    setIsScriptLoading(true);
    setErrMsg('');

    const name = scriptNames[platform];

    fetch(`${getBaseUrl('http')}/exit/${slot}/${name}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
      .then((rsp) => {
        if (!rsp.ok) {
          throw new Error(String(rsp.status));
        }
        return rsp.text();
      })
      .then((body) => {
        setScript({ name, body });
      })
      .catch((err) => {
        const httpStatus = Number((err as Error)?.message);
        setErrMsg(t(`settings.exit.commands.${scriptErrorKey(httpStatus)}`));
      })
      .finally(() => {
        setIsScriptLoading(false);
      });
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
          {view.pinsFingerprint && (
            <li className="break-all">
              {t('settings.exit.commands.warnFingerprint', { fingerprint: commands?.fingerprint })}
            </li>
          )}
          {view.pinUncertain && (
            <li className="break-all">
              {t('settings.exit.commands.warnFingerprintReaddressed', {
                fingerprint: commands?.fingerprint,
                host: localAddress
              })}
            </li>
          )}
          {view.cleartext && <li>{t('settings.exit.commands.warnCleartext')}</li>}
          {view.wstunnelUnverified && <li>{t('settings.exit.commands.warnWstunnelUnverified')}</li>}
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
          activeKey={platform}
          onChange={(key) => setPlatform(key as ExitPlatform)}
          items={exitPlatforms.map((value) => ({
            key: value,
            label: t(`settings.exit.commands.${value}`),
            children: current ? (
              <div className="flex flex-col space-y-2">
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

                  {mode === 'native' && (
                    <Tooltip
                      title={scriptServed ? '' : t('settings.exit.commands.viewScriptDisabled')}
                      placement="bottom"
                    >
                      <Button
                        size="small"
                        type="text"
                        icon={<FileCodeIcon size={14} />}
                        loading={isScriptLoading}
                        disabled={!scriptServed}
                        onClick={viewScript}
                      >
                        {t('settings.exit.commands.viewScript')}
                      </Button>
                    </Tooltip>
                  )}
                </div>

                {current.notes && <span className="text-xs text-neutral-500">{current.notes}</span>}

                {view.rewritten && (
                  <span className="text-xs text-neutral-500">
                    {t('settings.exit.commands.rewritten', { host: localAddress })}
                  </span>
                )}

                {mode === 'wstunnel' && latestList && (
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

      {hasConnected && (
        <span className="text-xs text-neutral-500">
          {t('settings.exit.commands.regenerateHint')}
        </span>
      )}

      <span className="text-xs text-neutral-500">{t('settings.exit.commands.security')}</span>

      {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}

      <Modal
        title={t('settings.exit.commands.scriptTitle', { name: script?.name ?? '' })}
        open={!!script}
        centered={true}
        width={'80%'}
        style={{ maxWidth: '900px' }}
        footer={null}
        onCancel={() => setScript(undefined)}
      >
        <ScrollArea className="h-[60vh] rounded bg-neutral-800/60 p-2">
          <pre className="whitespace-pre font-mono text-xs text-neutral-300">{script?.body}</pre>
        </ScrollArea>
      </Modal>
    </div>
  );
};
