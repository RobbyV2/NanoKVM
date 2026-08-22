import { Button, Input, Tooltip } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { EnvEntry } from './types.ts';

type EnvTableProps = {
  entries: EnvEntry[];
  onChange: (entries: EnvEntry[]) => void;
};

const envKeyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;

export const EnvTable = ({ entries, onChange }: EnvTableProps) => {
  const { t } = useTranslation();

  function update(index: number, entry: Partial<EnvEntry>) {
    onChange(entries.map((item, i) => (i === index ? { ...item, ...entry } : item)));
  }

  function remove(index: number) {
    onChange(entries.filter((_, i) => i !== index));
  }

  function add() {
    onChange([...entries, { key: '', value: '', secret: false, configured: false }]);
  }

  return (
    <div className="flex flex-col space-y-2">
      {entries.length > 0 && (
        <div className="flex items-center space-x-2 text-xs text-neutral-500">
          <span className="flex-1">{t('settings.tunnel.envKey')}</span>
          <span className="flex-1">{t('settings.tunnel.envValue')}</span>
          <span className="w-[24px]" />
        </div>
      )}

      {entries.map((entry, index) => {
        const isInvalidKey = entry.key !== '' && !envKeyPattern.test(entry.key);

        return (
          <div key={index} className="flex items-center space-x-2">
            <Input
              className="flex-1"
              value={entry.key}
              placeholder={t('settings.tunnel.envKey')}
              spellCheck={false}
              status={isInvalidKey ? 'error' : undefined}
              onChange={(e) => update(index, { key: e.target.value })}
            />

            {entry.secret ? (
              <Input.Password
                className="flex-1"
                value={entry.value}
                placeholder={
                  entry.configured ? t('settings.tunnel.configured') : t('settings.tunnel.envValue')
                }
                autoComplete="new-password"
                onChange={(e) => update(index, { value: e.target.value })}
              />
            ) : (
              <Input
                className="flex-1"
                value={entry.value}
                placeholder={t('settings.tunnel.envValue')}
                spellCheck={false}
                onChange={(e) => update(index, { value: e.target.value })}
              />
            )}

            <Tooltip title={t('settings.tunnel.envRemove')} placement="top">
              <div
                className="flex w-[24px] cursor-pointer justify-center rounded p-1 text-neutral-400 hover:bg-neutral-700/50 hover:text-red-500"
                onClick={() => remove(index)}
              >
                <Trash2Icon size={15} />
              </div>
            </Tooltip>
          </div>
        );
      })}

      <div className="flex">
        <Button type="dashed" size="small" icon={<PlusIcon size={14} />} onClick={add}>
          {t('settings.tunnel.envAdd')}
        </Button>
      </div>
    </div>
  );
};
