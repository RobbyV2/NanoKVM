// Firefox's Linux VA-API readback scaled the entire padded decode surface into
// the visible frame. For 1080p this exposes eight uninitialized rows at the bottom
// even though codedSize, visibleRect and displaySize all report 1920x1080.
// Cropping drawImage() cannot repair the already-resampled picture.
// https://bugzilla.mozilla.org/show_bug.cgi?id=2054811 (fixed in Firefox 157)
// Keep acceleration on fixed Firefox versions and other browsers. Remove this
// compatibility policy once the oldest supported Linux Firefox includes the fix.
export function directDecoderAcceleration(userAgent: string): HardwareAcceleration {
  const firefoxVersion = /\bFirefox\/(\d+)(?:\.|$)/.exec(userAgent);
  if (
    /\bLinux\b/.test(userAgent) &&
    !/\bAndroid\b/.test(userAgent) &&
    firefoxVersion &&
    Number(firefoxVersion[1]) < 157
  ) {
    return 'prefer-software';
  }

  return 'prefer-hardware';
}
