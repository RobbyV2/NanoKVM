import { http } from '@/lib/http.ts';

// the two network functions the gadget layer can build; ECM has no f_ecm branch
// anywhere below this, so it is not offered
export type NetworkProtocol = 'ncm' | 'rndis';

// get virtual devices status
export function getVirtualDevice() {
  return http.get('/api/vm/device/virtual');
}

// mount/unmount virtual device, or, with a protocol, select which network
// function the gadget presents and leave it mounted
export function updateVirtualDevice(device: string, protocol?: NetworkProtocol) {
  const data = protocol ? { device, protocol } : { device };

  return http.post('/api/vm/device/virtual', data);
}
