import { http, HttpResponse } from 'msw';
import { setupWorker } from 'msw/browser';

let isLoggedIn = false;
let activeProfile = 'standard';

const standardProfile = {
  schema_version: 1,
  name: 'standard',
  built_in: true,
  device: {
    vendor_id: '0x3346',
    product_id: '0x1009',
    bcd_device: '0x0510',
    class: 239,
    subclass: 2,
    protocol: 1,
    serial: '0123456789ABCDEF',
    manufacturer: 'sipeed',
    product: 'NanoKVM'
  },
  config: { bm_attributes: 224, max_power: 120, configuration: 'NanoKVM' },
  functions: [
    { kind: 'hid', instance: 'GS0' },
    { kind: 'hid', instance: 'GS1' },
    { kind: 'hid', instance: 'GS2' }
  ]
};

const presentationProfiles = new Map([[standardProfile.name, standardProfile]]);

function profileSummary(profile: typeof standardProfile) {
  return {
    name: profile.name,
    built_in: profile.built_in,
    active: profile.name === activeProfile,
    manufacturer: profile.device.manufacturer,
    product: profile.device.product,
    functions: profile.functions.map((item) => `${item.kind}.${item.instance}`),
    has_descriptors: false
  };
}

export const handlers = [
  http.post('/api/auth/login', () => {
    isLoggedIn = true;
    return HttpResponse.json({
      code: 0,
      data: {}
    });
  }),
  http.get('/api/auth/account', () => {
    if (!isLoggedIn) {
      return HttpResponse.json('unauthorized', { status: 401 });
    }
    return HttpResponse.json({
      code: 0,
      data: { username: 'admin', role: 'admin' }
    });
  }),
  http.post('/api/auth/logout', () => {
    isLoggedIn = false;
    return HttpResponse.json({ code: 0 });
  }),
  http.get('/api/presentation/status', () => {
    const profile = presentationProfiles.get(activeProfile)!;
    return HttpResponse.json({
      code: 0,
      data: {
        snapshot: {
          active: activeProfile,
          mode: 'normal',
          linked: profile.functions.map((item) => `${item.kind}.${item.instance}`),
          endpoints: { in: 3, out: 3 },
          headroom: { in: 3, out: 2 }
        },
        profile: profileSummary(profile),
        last_known_good: activeProfile
      }
    });
  }),
  http.get('/api/presentation/profiles', () =>
    HttpResponse.json({
      code: 0,
      data: { profiles: [...presentationProfiles.values()].map(profileSummary) }
    })
  ),
  http.post('/api/presentation/profiles/import', () => {
    const profile = structuredClone(standardProfile);
    profile.name = 'imported-profile';
    profile.built_in = false;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.get('/api/presentation/profiles/:name', ({ params }) => {
    const profile = presentationProfiles.get(String(params.name));
    return HttpResponse.json(
      profile ? { code: 0, data: profile } : { code: -2, msg: 'profile not found' }
    );
  }),
  http.post('/api/presentation/profiles/:name/clone', async ({ params, request }) => {
    const source = presentationProfiles.get(String(params.name));
    const body = (await request.json()) as { name: string };
    if (!source) return HttpResponse.json({ code: -2, msg: 'profile not found' });
    const profile = structuredClone(source);
    profile.name = body.name;
    profile.built_in = false;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.put('/api/presentation/profiles/:name', async ({ request }) => {
    const profile = (await request.json()) as typeof standardProfile;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.delete('/api/presentation/profiles/:name', ({ params }) => {
    presentationProfiles.delete(String(params.name));
    return HttpResponse.json({ code: 0 });
  }),
  http.get(
    '/api/presentation/profiles/:name/export',
    () =>
      new HttpResponse(new Uint8Array([0x50, 0x4b, 0x05, 0x06]), {
        headers: { 'Content-Type': 'application/vnd.nanokvm.presentation+zip' }
      })
  ),
  http.put('/api/presentation/config/preview', async ({ request }) => {
    const profile = (await request.json()) as typeof standardProfile;
    return HttpResponse.json({
      code: 0,
      data: {
        valid: true,
        errors: [],
        warnings: [],
        profile: profile.name,
        functions: profile.functions.map((item) => `${item.kind}.${item.instance}`),
        endpoints: { in: 3, out: 3 },
        headroom: { in: 3, out: 2 },
        operations: 24
      }
    });
  }),
  http.put('/api/presentation/config/apply', async ({ request }) => {
    const body = (await request.json()) as { name: string };
    activeProfile = body.name;
    return HttpResponse.json({ code: 0 });
  })
];
export const worker = setupWorker(...handlers);
