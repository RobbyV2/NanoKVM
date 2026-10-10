import assert from 'node:assert/strict';
import { registerHooks } from 'node:module';
import test from 'node:test';

// Node has no Vite worklet loader; the camera does not use this URL.
const hooks = registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith('?worker&url')) {
      return { url: 'data:text/javascript,export default "worklet.js"', shortCircuit: true };
    }
    return nextResolve(specifier, context);
  }
});
const { CameraCapture } = await import('./capture.ts');
hooks.deregister();

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

function stream() {
  const track = {
    readyState: 'live',
    stops: 0,
    stop() {
      this.stops++;
      this.readyState = 'ended';
    }
  };
  return { track, getTracks: () => [track], getVideoTracks: () => [track] };
}

function setup(t: test.TestContext) {
  const capture = new CameraCapture();
  t.after(() => capture.stop());
  const requests: ReturnType<typeof deferred<MediaStream>>[] = [];
  const plays: ReturnType<typeof deferred<void>>[] = [];
  const videos: { srcObject: unknown; readyState: number; play: () => Promise<void> }[] = [];
  const workers: {
    terminated: boolean;
    messages: unknown[];
    onmessage?: (event: unknown) => void;
  }[] = [];
  const timers = new Map<number, () => void>();
  let timer = 0;
  const install = (name: string, value: unknown) => {
    const previous = Object.getOwnPropertyDescriptor(globalThis, name);
    Object.defineProperty(globalThis, name, { configurable: true, value });
    t.after(() => {
      if (previous) Object.defineProperty(globalThis, name, previous);
      else Reflect.deleteProperty(globalThis, name);
    });
  };
  class Worker {
    terminated = false;
    messages: unknown[] = [];
    onmessage?: (event: unknown) => void;
    constructor() {
      workers.push(this);
    }
    terminate() {
      this.terminated = true;
    }
    postMessage(message: unknown) {
      this.messages.push(message);
    }
  }
  install('Worker', Worker);
  install('OffscreenCanvas', class {});
  install('createImageBitmap', async () => ({ close() {} }));
  install('HTMLMediaElement', { HAVE_CURRENT_DATA: 2 });
  install('window', {
    Worker,
    OffscreenCanvas: globalThis.OffscreenCanvas,
    createImageBitmap: globalThis.createImageBitmap,
    setTimeout: (callback: () => void) => {
      timers.set(++timer, callback);
      return timer;
    },
    clearTimeout: (id: number) => timers.delete(id),
    setInterval: (callback: () => void) => {
      timers.set(++timer, callback);
      return timer;
    },
    clearInterval: (id: number) => timers.delete(id)
  });
  install('navigator', {
    mediaDevices: {
      getUserMedia: () => {
        const request = deferred<MediaStream>();
        requests.push(request);
        return request.promise;
      }
    }
  });
  install('document', {
    createElement: () => {
      const play = deferred<void>();
      plays.push(play);
      const video = { srcObject: null as unknown, readyState: 0, play: () => play.promise };
      videos.push(video);
      return video;
    }
  });
  const errors: string[] = [];
  const frames: ArrayBuffer[] = [];
  const start = () =>
    capture.start(
      'camera',
      { streaming: true, width: 320, height: 240, fps: 30 },
      (frame) => frames.push(frame),
      (message) => errors.push(message)
    );
  return { capture, start, requests, plays, videos, workers, timers, errors, frames, install };
}

test('stop while camera permission is pending releases the late stream', async (t) => {
  const h = setup(t);
  const starting = h.start();
  h.capture.stop();
  const late = stream();
  h.requests[0].resolve(late as unknown as MediaStream);
  await Promise.resolve();
  // Let the broken implementation finish rather than leave the test hanging.
  h.plays[0]?.resolve();
  await starting;
  assert.equal(late.track.stops, 1);
  assert.equal(h.videos.length, 0);
  assert.equal(h.workers.length, 0);
  assert.equal(h.timers.size, 0);
});

test('stop during video play does not report an intentional abort as a camera failure', async (t) => {
  const h = setup(t);
  const starting = h.start();
  h.requests[0].resolve(stream() as unknown as MediaStream);
  await Promise.resolve();
  h.capture.stop();
  h.plays[0].reject(new DOMException('Media resource aborted', 'AbortError'));
  await assert.doesNotReject(starting);
  assert.equal(h.workers.length, 0);
  assert.equal(h.timers.size, 0);
});

test('an old permission result cannot replace a newer camera stream', async (t) => {
  const h = setup(t);
  const oldStart = h.start();
  const newStart = h.start();
  const current = stream();
  h.requests[1].resolve(current as unknown as MediaStream);
  await Promise.resolve();
  h.plays[0].resolve();
  await newStart;
  const late = stream();
  h.requests[0].resolve(late as unknown as MediaStream);
  await Promise.resolve();
  h.plays[1]?.resolve();
  await oldStart;
  assert.equal(late.track.stops, 1);
  assert.equal(current.track.stops, 0);
  assert.equal(h.workers.length, 1);
  assert.equal(h.videos[0].srcObject, current);
});

test('a real play failure releases the camera and still reaches the caller', async (t) => {
  const h = setup(t);
  const starting = h.start();
  const camera = stream();
  h.requests[0].resolve(camera as unknown as MediaStream);
  await Promise.resolve();
  h.plays[0].reject(new DOMException('Camera unavailable', 'NotReadableError'));
  await assert.rejects(starting, { name: 'NotReadableError' });
  assert.equal(camera.track.stops, 1);
  assert.equal(h.videos[0].srcObject, null);
});

test('a retired play rejection cannot tear down a replacement already producing frames', async (t) => {
  const h = setup(t);
  const oldStart = h.start();
  h.requests[0].resolve(stream() as unknown as MediaStream);
  await Promise.resolve();
  const newStart = h.start();
  const current = stream();
  h.requests[1].resolve(current as unknown as MediaStream);
  await Promise.resolve();
  h.plays[1].resolve();
  await newStart;
  h.plays[0].reject(new DOMException('Media resource aborted', 'AbortError'));
  await assert.doesNotReject(oldStart);
  const payload = new ArrayBuffer(4);
  h.workers[0].onmessage?.({ data: { id: 1, payload } });
  assert.deepEqual(h.frames, [payload]);
  assert.equal(current.track.stops, 0);
  assert.equal(h.workers[0].terminated, false);
  assert.deepEqual(h.errors, []);
});

test('an in-flight bitmap from the old capture never reaches the new encoder', async (t) => {
  const h = setup(t);
  const bitmap = deferred<ImageBitmap>();
  h.install('createImageBitmap', () => bitmap.promise);
  const oldStart = h.start();
  h.requests[0].resolve(stream() as unknown as MediaStream);
  await Promise.resolve();
  h.videos[0].readyState = 2;
  h.plays[0].resolve();
  await Promise.resolve();
  const oldMessage = h.workers[0].onmessage;
  const newStart = h.start();
  h.requests[1].resolve(stream() as unknown as MediaStream);
  await Promise.resolve();
  h.plays[1].resolve();
  await newStart;
  let closed = false;
  bitmap.resolve({
    close: () => {
      closed = true;
    }
  } as ImageBitmap);
  await oldStart;
  oldMessage?.({ data: { payload: new ArrayBuffer(4) } });
  assert.equal(closed, true);
  assert.equal(h.workers[0].terminated, true);
  assert.deepEqual(h.workers[1].messages, []);
  assert.deepEqual(h.frames, []);
});
