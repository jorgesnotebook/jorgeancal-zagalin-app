import './.config/jest-setup';

if (typeof window !== 'undefined') {
  window.HTMLElement.prototype.scrollIntoView = jest.fn();

  window.IntersectionObserver = jest.fn().mockImplementation(() => ({
    observe: jest.fn(),
    unobserve: jest.fn(),
    disconnect: jest.fn(),
  }));
}
