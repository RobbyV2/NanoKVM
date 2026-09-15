import { Button, Popconfirm, Tooltip } from 'antd';
import { UnplugIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { formatBytes, formatTime, formatUptime } from './state.ts';
import { downstreamKeys } from './types.ts';
import type { ExitDownstream, ExitStatus, ExitTunnelState } from './types.ts';

type ExitStatusCardProps = {
  status: ExitStatus;
  disconnecting: boolean;
  onDisconnect: () => void;
};

// the same palette as the tunnel page: idle grey, on the way blue, healthy green
const stateColors: Record<ExitTunnelState, string> = {
  disconnected: 'text-neutral-400',
  connecting: 'text-blue-500',
  connected: 'text-green-500'
};

// hev and the forwarder are always expected; the wstunnel daemon exists only in
// Mode B, and the server reports it true in native mode (proto.ExitDownstream)
function downstreamOk(downstream: ExitDownstream): boolean {
  return downstreamKeys.every((key) => downstream[key]);
}

export const ExitStatusCard = ({ status, disconnecting, onDisconnect }: ExitStatusCardProps) => {
  const { t } = useTranslation();

  const isConnected = status.tunnel === 'connected';
  const peer = status.peer;
  const previous = status.previousPeer;

  return (
    <div className="flex flex-col space-y-4">
      <div className="flex items-center justify-between">
        <span>{t('settings.exit.status.title')}</span>

        {peer && (
          <Popconfirm
            placement="bottomRight"
            title={t('settings.exit.status.disconnect')}
            description={
              <div className="max-w-[320px] text-xs text-neutral-400">
                {t('settings.exit.status.disconnectDesc')}
              </div>
            }
            okText={t('settings.exit.okBtn')}
            cancelText={t('settings.exit.cancelBtn')}
            onConfirm={onDisconnect}
          >
            <Button size="small" danger icon={<UnplugIcon size={13} />} loading={disconnecting}>
              {t('settings.exit.status.disconnect')}
            </Button>
          </Popconfirm>
        )}
      </div>

      <div className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 text-sm">
        {/* tunnel */}
        <span className="text-neutral-400">{t('settings.exit.status.tunnel')}</span>
        <div className="flex flex-wrap items-baseline gap-x-3">
          <span className={stateColors[status.tunnel]}>
            {t(`settings.exit.state.${status.tunnel}`)}
          </span>
          {isConnected ? (
            <span className="text-xs text-neutral-500">
              {t('settings.exit.status.uptime', { uptime: formatUptime(status.uptimeSeconds) })}
            </span>
          ) : (
            <span className="text-xs text-neutral-500">
              {t('settings.exit.status.lastConnected', {
                time: formatTime(status.lastConnectedAt) || t('settings.exit.status.never')
              })}
            </span>
          )}
        </div>

        {/* exit device */}
        <span className="text-neutral-400">{t('settings.exit.status.peer')}</span>
        <div className="flex flex-col space-y-1">
          {peer ? (
            <span className="break-all">
              <span className="font-mono">{peer.addr}</span>
              {peer.hostname && <span className="text-neutral-300"> · {peer.hostname}</span>}
              {peer.os && <span className="text-neutral-500"> · {peer.os}</span>}
              <span className="text-neutral-500">
                {' '}
                · {t(`settings.exit.mode.${peer.transport}`)}
              </span>
            </span>
          ) : (
            <span className="text-neutral-500">{t('settings.exit.status.noPeer')}</span>
          )}

          {previous && (
            <span className="text-xs text-amber-500">
              {t('settings.exit.status.peerChanged', {
                addr: previous.hostname ? `${previous.addr} (${previous.hostname})` : previous.addr,
                time: formatTime(status.peerChangedAt)
              })}
            </span>
          )}
        </div>

        {/* target nic */}
        <span className="text-neutral-400">{t('settings.exit.status.nic')}</span>
        <div className="flex flex-wrap items-baseline gap-x-2">
          {status.nic.ifname ? (
            <>
              <span className="font-mono">{status.nic.ifname}</span>
              <span className={status.nic.up ? 'text-green-500' : 'text-neutral-500'}>
                {t(status.nic.up ? 'settings.exit.status.nicUp' : 'settings.exit.status.nicDown')}
              </span>
              {status.nic.protocol && (
                <span className="text-xs uppercase text-neutral-500">{status.nic.protocol}</span>
              )}
              {status.nic.address && (
                <span className="font-mono text-xs text-neutral-500">{status.nic.address}</span>
              )}
            </>
          ) : (
            <span className="text-neutral-500">{t('settings.exit.status.nicMissing')}</span>
          )}
        </div>

        {/* internet through the exit */}
        <span className="text-neutral-400">{t('settings.exit.status.upstream')}</span>
        <div className="flex flex-wrap items-baseline gap-x-2">
          {isConnected && status.upstream.checkedAt ? (
            <>
              <span className={status.upstream.reachable ? 'text-green-500' : 'text-red-500'}>
                {t(
                  status.upstream.reachable
                    ? 'settings.exit.status.reachable'
                    : 'settings.exit.status.unreachable'
                )}
              </span>
              {status.upstream.reachable && (
                <span className="text-xs text-neutral-500">
                  {t('settings.exit.status.latency', { ms: status.upstream.latencyMs })}
                </span>
              )}
              <span className="text-xs text-neutral-600">
                {formatTime(status.upstream.checkedAt)}
              </span>
            </>
          ) : (
            <span className="text-neutral-500">{t('settings.exit.status.notProbed')}</span>
          )}
        </div>

        {/* downstream vector */}
        <span className="text-neutral-400">{t('settings.exit.status.downstream')}</span>
        <div className="flex flex-wrap gap-1.5">
          {status.enabled ? (
            downstreamKeys.map((key) => (
              <Tooltip key={key} title={t(`settings.exit.status.downstreamTip.${key}`)}>
                <span
                  className={`rounded px-1.5 py-0.5 font-mono text-xs ${
                    status.downstream[key]
                      ? 'bg-green-500/15 text-green-500'
                      : 'bg-red-500/15 text-red-500'
                  }`}
                >
                  {key}
                </span>
              </Tooltip>
            ))
          ) : (
            <span className="text-neutral-500">{t('settings.exit.status.downstreamOff')}</span>
          )}
          {status.enabled && !downstreamOk(status.downstream) && (
            <span className="w-full text-xs text-amber-500">
              {t('settings.exit.status.downstreamDegraded')}
            </span>
          )}
        </div>

        {/* dns */}
        <span className="text-neutral-400">{t('settings.exit.status.dns')}</span>
        <span className="text-neutral-300">
          {t('settings.exit.status.dnsStats', {
            queries: status.dns.queries,
            failures: status.dns.failures,
            redirected: status.dns.redirected
          })}
        </span>

        {/* traffic */}
        <span className="text-neutral-400">{t('settings.exit.status.traffic')}</span>
        <span className="text-neutral-300">
          {t('settings.exit.status.trafficUp', { bytes: formatBytes(status.bytes.up) })}
          <span className="text-neutral-600"> · </span>
          {t('settings.exit.status.trafficDown', { bytes: formatBytes(status.bytes.down) })}
        </span>
      </div>
    </div>
  );
};
