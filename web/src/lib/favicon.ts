// The device serves the icon; the browser never fetches the operator's URL
// itself. See server/service/vm/favicon for why.
export const FAVICON_ENDPOINT = '/api/vm/favicon';

// The server hands back a stable numeric code for each failure so the message
// the operator reads is translated rather than an untranslated Go string.
export function faviconErrorKey(code: number): string {
  switch (code) {
    case -2:
      return 'settings.appearance.faviconErrUrl';
    case -3:
      return 'settings.appearance.faviconErrFetch';
    case -4:
      return 'settings.appearance.faviconErrLarge';
    case -5:
      return 'settings.appearance.faviconErrType';
    default:
      return 'settings.appearance.faviconErrSave';
  }
}

export function faviconSourceKey(source: string): string {
  switch (source) {
    case 'custom':
      return 'settings.appearance.faviconCustom';
    case 'boot':
      return 'settings.appearance.faviconBoot';
    default:
      return 'settings.appearance.faviconDefault';
  }
}

// The version is a digest of the stored bytes. Putting it in the query string
// is what makes a change visible in the live tab: browsers key the tab icon on
// the URL, so a new URL is a new icon.
export function faviconHref(version?: string): string {
  return version ? `${FAVICON_ENDPOINT}?v=${encodeURIComponent(version)}` : FAVICON_ENDPOINT;
}

// Repoints every <link rel="icon"> at the new URL. The element is removed and
// re-appended rather than merely edited because Chrome re-reads the icon on
// insertion, not on an href mutation, and the whole point of this is that the
// tab updates without a reload.
export function applyFavicon(doc: Document, version?: string): string {
  const href = faviconHref(version);
  const links = Array.from(doc.querySelectorAll('link[rel~="icon"]'));

  if (links.length === 0) {
    const created = doc.createElement('link');
    created.setAttribute('rel', 'icon');
    created.setAttribute('href', href);
    doc.head.appendChild(created);
    return href;
  }

  for (const link of links) {
    // The response declares its own content type, and a type attribute left
    // over from a previous icon makes some browsers ignore the new one.
    link.removeAttribute('type');
    link.setAttribute('href', href);
    link.remove();
    doc.head.appendChild(link);
  }

  return href;
}

// A cheap pre-flight so an obvious typo does not cost a 15 second device-side
// download attempt. The server validates for real; this only avoids the trip.
export function isFaviconUrl(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;

  try {
    const parsed = new URL(trimmed);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}
