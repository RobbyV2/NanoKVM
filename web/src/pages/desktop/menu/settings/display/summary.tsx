import type { ReactNode } from 'react';
import { CircleAlertIcon, InfoIcon, TriangleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { EdidSummary } from '@/api/edid.ts';

import type { Check, CheckLevel } from './utils.ts';
import { formatMode } from './utils.ts';

type SummaryProps = {
  // a shipped profile only knows the monitor identity and its preferred mode,
  // a decoded upload knows everything, so every field is rendered on demand
  summary: Partial<EdidSummary>;
};

type ChecksProps = {
  checks: Check[];
};

const groups: { level: CheckLevel; label: string; color: string; icon: ReactNode }[] = [
  {
    level: 'error',
    label: 'errors',
    color: 'text-red-500',
    icon: <CircleAlertIcon size={13} />
  },
  {
    level: 'warning',
    label: 'warnings',
    color: 'text-amber-500',
    icon: <TriangleAlertIcon size={13} />
  },
  {
    level: 'info',
    label: 'info',
    color: 'text-neutral-400',
    icon: <InfoIcon size={13} />
  }
];

export const Summary = ({ summary }: SummaryProps) => {
  const { t } = useTranslation();

  const identity = [summary.manufacturer, summary.model].filter(Boolean).join(' ');
  const mode = formatMode(summary.preferredMode);

  const details: string[] = [];
  if (summary.version) {
    details.push(t('settings.display.edidVersion', { version: summary.version }));
  }
  if (summary.audio !== undefined) {
    details.push(summary.audio ? t('settings.display.audioYes') : t('settings.display.audioNo'));
  }
  if (summary.extensions) {
    details.push(t('settings.display.extensionBlocks', { blocks: summary.extensions }));
  }

  return (
    <div className="flex flex-col">
      <span className="text-sm">{identity || t('settings.display.unknownMonitor')}</span>
      {mode && <span className="text-xs text-neutral-400">{mode}</span>}
      {details.length > 0 && (
        <span className="text-xs text-neutral-500">{details.join(' · ')}</span>
      )}
    </div>
  );
};

export const Checks = ({ checks }: ChecksProps) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col space-y-3">
      {groups.map((group) => {
        const items = checks.filter((check) => check.level === group.level);
        if (items.length === 0) return null;

        return (
          <div key={group.level} className="flex flex-col space-y-1">
            <span className="text-xs text-neutral-500">{t(`settings.display.${group.label}`)}</span>

            {items.map((item, index) => (
              <div key={index} className={`flex items-start space-x-1.5 text-xs ${group.color}`}>
                <div className="pt-[1px]">{group.icon}</div>
                <span>{item.text || t(`settings.display.${item.key}`)}</span>
              </div>
            ))}
          </div>
        );
      })}
    </div>
  );
};
