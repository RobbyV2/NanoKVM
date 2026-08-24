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
const restartWaitMs = 12000;
const rebootGraceMs = 20000;
const rebootWaitMs = 180000;
const pollIntervalMs = 3000;

// The grace period matters: the server answers for another second before it
// reboots, and a probe that early would reload the page into the old kernel.
export function reloadWhenBack(
  isRebooting: boolean,
  probe: () => Promise<unknown>,
  done: () => void
) {
  const deadline = Date.now() + rebootWaitMs;

  function poll() {
    if (!isRebooting || Date.now() >= deadline) {
      done();
      return;
    }
    probe()
      .then(done)
      .catch(() => setTimeout(poll, pollIntervalMs));
  }

  setTimeout(poll, isRebooting ? rebootGraceMs : restartWaitMs);
}

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
