type EncodeRequest = {
  id: number;
  bitmap: ImageBitmap;
  width: number;
  height: number;
};

type EncodeResponse = { id: number; payload: ArrayBuffer } | { id: number; error: string };

const scope = self as unknown as DedicatedWorkerGlobalScope;
let canvas: OffscreenCanvas | undefined;

scope.onmessage = async ({ data }: MessageEvent<EncodeRequest>) => {
  const { id, bitmap, width, height } = data;
  try {
    canvas ||= new OffscreenCanvas(width, height);
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext('2d', { alpha: false });
    if (!context) throw new Error('camera encoder unavailable');

    const scale = Math.max(width / bitmap.width, height / bitmap.height);
    const sourceWidth = width / scale;
    const sourceHeight = height / scale;
    const sourceX = (bitmap.width - sourceWidth) / 2;
    const sourceY = (bitmap.height - sourceHeight) / 2;
    context.drawImage(bitmap, sourceX, sourceY, sourceWidth, sourceHeight, 0, 0, width, height);
    bitmap.close();

    let blob = await canvas.convertToBlob({ type: 'image/jpeg', quality: 0.82 });
    if (blob.size > 2 << 20) {
      blob = await canvas.convertToBlob({ type: 'image/jpeg', quality: 0.62 });
    }
    if (blob.size > 2 << 20) throw new Error('camera frame exceeds 2 MiB');
    const payload = await blob.arrayBuffer();
    scope.postMessage({ id, payload } satisfies EncodeResponse, [payload]);
  } catch (error) {
    bitmap.close();
    const message = error instanceof Error ? error.message : 'camera encoding failed';
    scope.postMessage({ id, error: message } satisfies EncodeResponse);
  }
};

export {};
