// Utility to get or create a persistent guest ID
export function getOrCreateGuestId(): string {
  let guestId = localStorage.getItem('guestId');
  if (!guestId) {
    // Use crypto.randomUUID if available, else fallback to random string
    guestId = (window.crypto?.randomUUID?.().substring(4, 10) ?? Math.random().toString(36).substring(4, 10));
    localStorage.setItem('guestId', guestId);
  }
  return guestId;
}
