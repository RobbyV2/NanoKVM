import type { ReactNode } from 'react';
import { CircleAlertIcon, InfoIcon, TriangleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { EdidPreflight, EdidSummary } from '@/api/edid.ts';

export type CheckLevel = 'error' | 'warning' | 'info';

export type Check = {
  level: CheckLevel;
  key?: string;
  text?: string;
};

type Mode = {
  width: number;
  height: number;
  refresh: number;
  interlaced: boolean;
};

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

// the tool writes modes as 1920x1080p60
const modePattern = /^(\d+)x(\d+)([pi])(\d+)$/;

export function parseMode(mode?: string): Mode | undefined {
  const matched = mode ? modePattern.exec(mode) : null;
  if (!matched) return undefined;

  return {
    width: Number(matched[1]),
    height: Number(matched[2]),
    interlaced: matched[3] === 'i',
    refresh: Number(matched[4])
  };
}

export function formatMode(mode?: string): string {
  const parsed = parseMode(mode);
  if (!parsed) return mode ?? '';

  return `${parsed.width} × ${parsed.height} @ ${parsed.refresh} Hz`;
}

// errors block the apply, warnings and information do not
export function buildChecks(summary?: Partial<EdidSummary>, preflight?: EdidPreflight): Check[] {
  const checks: Check[] = [];

  if (preflight && !preflight.supported) {
    checks.push({ level: 'error', key: 'unsupported', text: preflight.reason });
  }
  if (preflight && !preflight.toolAvailable) {
    checks.push({ level: 'error', key: 'toolMissing' });
  }

  if (summary) {
    const mode = parseMode(summary.preferredMode);

    if (mode?.interlaced) {
      checks.push({ level: 'warning', key: 'interlaced' });
    }
    if (mode && (mode.height > 1080 || mode.refresh > 60)) {
      checks.push({ level: 'warning', key: 'tooLarge' });
    }
    if (summary.audio === false) {
      checks.push({ level: 'warning', key: 'noAudio' });
    }
    if (summary.version && summary.version < '1.4') {
      checks.push({ level: 'warning', key: 'oldVersion' });
    }

    checks.push({ level: 'info', key: 'hdmiNotice' });
    if (preflight?.requiresPowerCycle) {
      checks.push({ level: 'info', key: 'powerCycleNotice' });
    }
  }

  return checks;
}

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
