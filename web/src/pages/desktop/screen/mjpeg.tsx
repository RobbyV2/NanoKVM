import { useEffect, useRef, useState } from 'react';
import clsx from 'clsx';
import { useAtomValue } from 'jotai';

import { stopFrameDetect } from '@/api/stream.ts';
import { getFrameDetect } from '@/lib/localstorage.ts';
import { getBaseUrl } from '@/lib/service.ts';
import { mouseStyleAtom } from '@/jotai/mouse.ts';
import { resolutionAtom } from '@/jotai/screen.ts';

import { ScreenViewport } from './viewport.tsx';

// An MJPEG stream that drops takes the <img> with it, and the element never
// tries again on its own. Left alone the operator is looking at a blank screen
// for the rest of the session with nothing to click, so the stream is reopened
// on a backoff until it comes back.
const retryDelaysMS = [500, 1000, 2000, 4000, 8000];

export const Mjpeg = () => {
  const resolution = useAtomValue(resolutionAtom);
  const mouseStyle = useAtomValue(mouseStyleAtom);
  const [hasError, setHasError] = useState(false);
  const [streamNonce, setStreamNonce] = useState(0);
  const retries = useRef(0);
  const streamURL = `${getBaseUrl('http')}/api/stream/mjpeg`;
  // The nonce defeats the cache so a retry is a new request rather than the
  // failed response served back from memory.
  const streamSrc = hasError ? undefined : `${streamURL}?v=${streamNonce}`;

  useEffect(() => {
    // stop frame detect for a while
    const enabled = getFrameDetect();
    if (enabled) {
      stopFrameDetect(10);
    }
    retries.current = 0;
    setHasError(false);
    setStreamNonce((current) => current + 1);
  }, [resolution]);

  useEffect(() => {
    if (!hasError) {
      return;
    }

    const delay = retryDelaysMS[Math.min(retries.current, retryDelaysMS.length - 1)];
    retries.current += 1;
    const timer = window.setTimeout(() => {
      setHasError(false);
      setStreamNonce((current) => current + 1);
    }, delay);

    return () => window.clearTimeout(timer);
  }, [hasError]);

  return (
    <ScreenViewport>
      <img
        id="screen"
        className={clsx('block touch-none select-none', mouseStyle)}
        style={{
          visibility: hasError ? 'hidden' : 'visible'
        }}
        src={streamSrc}
        onError={() => setHasError(true)}
        onLoad={() => {
          retries.current = 0;
        }}
        alt="screen"
      />
    </ScreenViewport>
  );
};
