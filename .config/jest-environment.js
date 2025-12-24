/*
 * Custom Jest environment that extends jsdom and ensures window is available
 * before any modules are imported, including those loaded in jest.config.js
 */

const JSDOMEnvironment = require('jest-environment-jsdom').TestEnvironment;

class CustomJestEnvironment extends JSDOMEnvironment {
  constructor(config, context) {
    // Set up minimal window object before calling super
    // This ensures window exists before any modules are loaded
    if (typeof global.window === 'undefined') {
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
        document: global.document || {},
        getComputedStyle: () => ({}),
        requestAnimationFrame: (cb) => setTimeout(cb, 0),
        cancelAnimationFrame: (id) => clearTimeout(id),
      };
      global.self = global.self || global;
    }

    super(config, context);
  }

  async setup() {
    await super.setup();

    // Ensure window and self are also available in the test environment
    if (this.global.window) {
      this.global.self = this.global.self || this.global.window;
    }
  }
}

module.exports = CustomJestEnvironment;
