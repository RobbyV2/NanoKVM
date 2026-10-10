import { useEffect, useRef } from 'react';
import { notification } from 'antd';
import clsx from 'clsx';
import { useAtomValue } from 'jotai';
import { useTranslation } from 'react-i18next';

import { getBaseUrl } from '@/lib/service.ts';
import { mouseStyleAtom } from '@/jotai/mouse';

import { directDecoderAcceleration } from './decoder-compatibility.ts';
import DirectWorker from './direct.worker.ts?worker';
import { ScreenViewport } from './viewport.tsx';

export const H264Direct = () => {
  const { t } = useTranslation();
  const mouseStyle = useAtomValue(mouseStyleAtom);
  const [notificationApi, contextHolder] = notification.useNotification();

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const workerRef = useRef<Worker | null>(null);

  useEffect(() => {
    if (directDecoderAcceleration(navigator.userAgent) !== 'prefer-software') {
      return;
    }

    notificationApi.info({
      key: 'direct-software-decoder',
      message: t('screen.directSoftwareDecoder.title'),
      description: t('screen.directSoftwareDecoder.description'),
      duration: 10
    });
    return () => notificationApi.destroy('direct-software-decoder');
  }, [notificationApi, t]);

  useEffect(() => {
    if (!window.VideoDecoder) {
      console.log('Error: WebCodecs API not supported.');
      return;
    }
    if (!canvasRef.current) {
      return;
    }

    const worker = new DirectWorker();
    workerRef.current = worker;

    const offscreen = canvasRef.current.transferControlToOffscreen();
    const url = `${getBaseUrl('ws')}/api/stream/h264/direct`;
    worker.onmessage = (
      event: MessageEvent<{ type?: string; width?: number; height?: number }>
    ) => {
      const { type, width, height } = event.data;
      if (type !== 'frame-size' || !width || !height || !canvasRef.current) {
        return;
      }

      canvasRef.current.dataset.mediaWidth = String(width);
      canvasRef.current.dataset.mediaHeight = String(height);
    };
    worker.postMessage({ type: 'h264', canvas: offscreen, url }, [offscreen]);

    return () => {
      worker.postMessage({ type: 'stop' });
      worker.terminate();
    };
  }, []);

  return (
    <>
      {contextHolder}
      <ScreenViewport>
        <canvas
          id="screen"
          ref={canvasRef}
          className={clsx('block touch-none select-none', mouseStyle)}
        ></canvas>
      </ScreenViewport>
    </>
  );
};
