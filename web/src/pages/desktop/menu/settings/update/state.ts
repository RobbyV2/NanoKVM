import semver from 'semver';

export type KernelState = {
  slot?: string;
  installed?: string;
  rolledBack?: string;
};

export type VersionInfo = {
  current: string;
  latest?: string;
  latestKernel?: string;
};

// A restart only has to outlive the supervisor; a kernel install has to outlive
// a full reboot, so that reload polls instead of guessing a single delay.
export const restartWaitMs = 12000;
export const rebootGraceMs = 20000;
export const rebootWaitMs = 180000;
export const pollIntervalMs = 3000;

// A kernel package carries no application, so its offer is decided against the
// kernel the device last confirmed, not against /kvmapp/version.
export function updateStatus(version: VersionInfo, kernel: KernelState | null): string {
  if (version.latestKernel) {
    return version.latestKernel === (kernel?.installed || '') ? 'latest' : 'outdated';
  }
  if (!version.latest) return 'latest';
  return semver.gte(version.current, version.latest) ? 'latest' : 'outdated';
}

// A silent revert is its own bug: the operator has to be told which kernel they
// installed did not take.
export function rollbackWarning(kernel: KernelState | null): string {
  return kernel?.rolledBack || '';
}
