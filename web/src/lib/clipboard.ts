// The page is served over plain http on most deployments, where the async
// clipboard is unavailable, so the textarea path is the one that usually runs.
function legacyCopy(text: string): Promise<void> {
  const textArea = document.createElement('textarea');
  textArea.value = text;
  textArea.setAttribute('readonly', '');
  textArea.style.position = 'fixed';
  textArea.style.left = '-9999px';
  textArea.style.top = '0';

  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();
  textArea.setSelectionRange(0, text.length);

  const copied = document.execCommand('copy');
  document.body.removeChild(textArea);

  return copied ? Promise.resolve() : Promise.reject(new Error('copy command was rejected'));
}

export function copyText(text: string): Promise<void> {
  if (window.isSecureContext === true && navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text).catch(() => legacyCopy(text));
  }

  return legacyCopy(text);
}
