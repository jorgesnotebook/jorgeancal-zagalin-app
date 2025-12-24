/*
 * This file runs BEFORE the test framework is installed.
 * It ensures global objects like window are available when modules are imported.
 */

// Ensure window is defined with all necessary properties for @grafana/data
if (typeof window === 'undefined') {
  global.window = {
    location: {
      href: 'http://localhost:3000',
      protocol: 'http:',
      host: 'localhost:3000',
      hostname: 'localhost',
      port: '3000',
      pathname: '/',
      search: '',
      hash: '',
    },
    localStorage: {
      getItem: () => null,
      setItem: () => {},
      removeItem: () => {},
      clear: () => {},
      length: 0,
      key: () => null,
    },
    sessionStorage: {
      getItem: () => null,
      setItem: () => {},
      removeItem: () => {},
      clear: () => {},
      length: 0,
      key: () => null,
    },
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => true,
    navigator: {
      userAgent: 'jest',
    },
    document: {},
    getComputedStyle: () => ({}),
    requestAnimationFrame: (cb) => setTimeout(cb, 0),
    cancelAnimationFrame: (id) => clearTimeout(id),
  };
}

// Ensure self is defined
global.self = global.self || global;
