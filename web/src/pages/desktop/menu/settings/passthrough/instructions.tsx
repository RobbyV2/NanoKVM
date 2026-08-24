import { useState } from 'react';
import { Button } from 'antd';
import { CheckIcon, CopyIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { buildCommands, busIdExample, copyText, deviceHost, exporterPortOf } from './utils.ts';

type InstructionsProps = {
  // the live session's values while one runs, the form's while none does, so
  // every command already carries the busid and the port that will be used
  exporter: string;
  busId: string;
};

export const Instructions = ({ exporter, busId }: InstructionsProps) => {
  const { t } = useTranslation();

  const [isCopying, setIsCopying] = useState(false);
  const [copiedKey, setCopiedKey] = useState('');
  const [errMsg, setErrMsg] = useState('');

  const port = exporterPortOf(exporter);
  const commands = buildCommands(exporter, busId, deviceHost());

  const steps = [
    { key: 'modprobe', command: commands.modprobe, notice: false },
    { key: 'list', command: commands.list, notice: false },
    { key: 'bind', command: commands.bind, notice: false },
    // stock usbipd takes no listen address, so the confinement is the tunnel's
    { key: 'serve', command: commands.serve, notice: true },
    { key: 'tunnel', command: commands.tunnel, notice: false },
    { key: 'exporter', command: commands.exporter, notice: false },
    { key: 'unbind', command: commands.unbind, notice: false }
  ];

  function copy(key: string, command: string) {
    if (isCopying) return;
    setIsCopying(true);
    setErrMsg('');

    copyText(command)
      .then(() => {
        setCopiedKey(key);
        window.setTimeout(() => setCopiedKey(''), 2000);
      })
      .catch(() => {
        setErrMsg(
          t(
            window.isSecureContext === false
              ? 'settings.passthrough.copyInsecure'
              : 'settings.passthrough.copyFailed'
          )
        );
      })
      .finally(() => {
        setIsCopying(false);
      });
  }

  return (
    <div className="flex flex-col space-y-4">
      <div className="flex flex-col space-y-1">
        <span>{t('settings.passthrough.instructions')}</span>
        <span className="text-xs text-neutral-500">
          {t('settings.passthrough.instructionsDesc')}
        </span>
      </div>

      {steps.map((step, index) => (
        <div key={step.key} className="flex flex-col space-y-1">
          <span className="text-sm">
            {index + 1}. {t(`settings.passthrough.steps.${step.key}.title`)}
          </span>

          <span className="text-xs text-neutral-500">
            {t(`settings.passthrough.steps.${step.key}.desc`, {
              port,
              host: deviceHost(),
              example: busIdExample
            })}
          </span>

          <div className="flex items-center space-x-2 rounded-lg bg-neutral-800/50 px-3 py-2">
            <code className="flex-1 overflow-x-auto whitespace-pre text-xs text-neutral-300">
              {step.command}
            </code>

            <Button
              type="text"
              size="small"
              icon={
                copiedKey === step.key ? (
                  <CheckIcon size={14} className="text-green-500" />
                ) : (
                  <CopyIcon size={14} />
                )
              }
              onClick={() => copy(step.key, step.command)}
            />
          </div>

          {step.notice && (
            <span className="text-xs text-amber-500">
              {t(`settings.passthrough.steps.${step.key}.notice`, { port })}
            </span>
          )}
        </div>
      ))}

      <span className="text-xs text-neutral-500">
        {t('settings.passthrough.directNote', { port })}
      </span>

      {errMsg && <div className="text-red-500">{errMsg}</div>}
    </div>
  );
};
