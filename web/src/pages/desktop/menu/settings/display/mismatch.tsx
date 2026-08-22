import { useTranslation } from 'react-i18next';

import type { EdidResult } from '@/api/edid.ts';

type MismatchProps = {
  result?: EdidResult;
};

const Dump = ({ label, bytes, other }: { label: string; bytes: string; other: string }) => {
  const values = bytes.split(' ');
  const compare = other.split(' ');

  return (
    <div className="flex flex-col space-y-1">
      <span className="text-xs text-neutral-500">{label}</span>

      <div className="max-h-[140px] overflow-auto rounded bg-neutral-800/60 p-2 font-mono text-xs leading-5">
        {values.map((value, index) => (
          <span
            key={index}
            className={`mr-1 ${value === compare[index] ? 'text-neutral-400' : 'text-red-500'}`}
          >
            {value}
          </span>
        ))}
      </div>
    </div>
  );
};

// The tool has no read primitive of its own, so the pair of dumps it prints when
// the readback disagrees is the only evidence that exists on hardware of what
// the chip actually took. It reaches the operator on the one state that means
// the region is half written.
export const Mismatch = ({ result }: MismatchProps) => {
  const { t } = useTranslation();

  if (result?.state !== 'needs_recovery' || !result.writtenHex || !result.readHex) {
    return null;
  }

  return (
    <div className="flex flex-col space-y-2">
      <span className="text-xs text-neutral-400">{t('settings.display.mismatchTitle')}</span>

      <Dump
        label={t('settings.display.mismatchWritten')}
        bytes={result.writtenHex}
        other={result.readHex}
      />
      <Dump
        label={t('settings.display.mismatchRead')}
        bytes={result.readHex}
        other={result.writtenHex}
      />
    </div>
  );
};
