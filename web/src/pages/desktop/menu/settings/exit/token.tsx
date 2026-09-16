import { useState } from 'react';
import { Button, Popconfirm, Tooltip } from 'antd';
import { CheckIcon, CopyIcon, RefreshCcwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { copyText } from '@/lib/clipboard.ts';

type ExitTokenProps = {
  token: string;
  regenerating: boolean;
  onRegenerate: () => void;
};

export const ExitToken = ({ token, regenerating, onRegenerate }: ExitTokenProps) => {
  const { t } = useTranslation();

  const [isCopied, setIsCopied] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  function copy() {
    copyText(token)
      .then(() => {
        setIsCopied(true);
        setErrMsg('');
        setTimeout(() => setIsCopied(false), 1500);
      })
      .catch(() => {
        setErrMsg(t('settings.exit.copyFailed'));
      });
  }

  return (
    <div className="flex flex-col space-y-2">
      <div className="flex items-center justify-between space-x-4">
        <div className="flex min-w-0 flex-col space-y-1">
          <span>{t('settings.exit.token.title')}</span>
          <span className="text-xs text-neutral-500">{t('settings.exit.token.history')}</span>
        </div>

        <div className="flex shrink-0 items-center space-x-1">
          <code className="rounded bg-neutral-800/60 px-2 py-1 font-mono text-sm tracking-wider text-neutral-200">
            {token || '--------'}
          </code>

          <Tooltip title={t('settings.exit.copy')} placement="bottom">
            <Button
              type="text"
              size="small"
              aria-label={t('settings.exit.copy')}
              disabled={!token}
              icon={
                isCopied ? (
                  <CheckIcon size={15} className="text-green-500" />
                ) : (
                  <CopyIcon size={15} />
                )
              }
              onClick={copy}
            />
          </Tooltip>

          <Popconfirm
            placement="bottomRight"
            title={t('settings.exit.token.regenerateConfirm')}
            description={
              <div className="max-w-[320px] text-xs text-neutral-400">
                {t('settings.exit.token.regenerateDesc')}
              </div>
            }
            okText={t('settings.exit.okBtn')}
            cancelText={t('settings.exit.cancelBtn')}
            disabled={regenerating}
            onConfirm={onRegenerate}
          >
            {/* the confirm binds its click to this span, the tooltip its hover to
                the button, so neither has to pass handlers through the other */}
            <span>
              <Tooltip title={t('settings.exit.token.regenerate')} placement="bottom">
                <Button
                  type="text"
                  size="small"
                  aria-label={t('settings.exit.token.regenerate')}
                  loading={regenerating}
                  icon={<RefreshCcwIcon size={15} />}
                />
              </Tooltip>
            </span>
          </Popconfirm>
        </div>
      </div>

      {errMsg && <span className="text-xs text-red-500">{errMsg}</span>}
    </div>
  );
};
