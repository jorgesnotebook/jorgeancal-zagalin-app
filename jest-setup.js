import './.config/jest-setup';

// Polyfill crypto.randomUUID — not available in jsdom/Node < 19
if (typeof globalThis.crypto === 'undefined') {
  (globalThis as any).crypto = {};
}
if (typeof (globalThis.crypto as any).randomUUID !== 'function') {
  (globalThis.crypto as any).randomUUID = () =>
    'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
      const r = (Math.random() * 16) | 0;
      const v = c === 'x' ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
}

if (typeof window !== 'undefined') {
  window.HTMLElement.prototype.scrollIntoView = jest.fn();

  window.IntersectionObserver = jest.fn().mockImplementation(() => ({
    observe: jest.fn(),
    unobserve: jest.fn(),
    disconnect: jest.fn(),
  }));
}
