import '@testing-library/jest-dom/vitest';
import Prism from 'prismjs';

// prismjs language component files (e.g. prismjs/components/prism-bash) augment a
// global `Prism`. When a test mocks the "prismjs" module, those side-effect
// imports still expect the global to exist, so expose the real core globally.
(globalThis as unknown as { Prism: typeof Prism }).Prism = Prism;

// In-memory Storage implementation used to back localStorage/sessionStorage in
// the jsdom test environment. It is installed as the global `Storage` class so
// that `vi.spyOn(Storage.prototype, ...)` continues to work in tests.
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  clear(): void {
    this.store.clear();
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null;
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }
}

// Install the class globally so `Storage.prototype` spies resolve to the same
// prototype that backs the storage instances below.
Object.defineProperty(globalThis, 'Storage', {
  value: MemoryStorage,
  configurable: true,
  writable: true,
});

function installStorage(name: 'localStorage' | 'sessionStorage') {
  const instance = new MemoryStorage();
  Object.defineProperty(globalThis, name, {
    value: instance,
    configurable: true,
    writable: true,
  });
  if (typeof window !== 'undefined') {
    Object.defineProperty(window, name, {
      value: instance,
      configurable: true,
      writable: true,
    });
  }
}

installStorage('localStorage');
installStorage('sessionStorage');

// jsdom does not implement EventSource, but components that stream job logs
// construct one in an effect. Install an inert stub so those effects don't throw
// "EventSource is not defined" during render-only tests. Tests that exercise
// streaming still install their own richer mock via vi.stubGlobal, which
// overrides this.
if (typeof (globalThis as { EventSource?: unknown }).EventSource === 'undefined') {
  class NoopEventSource {
    static readonly CONNECTING = 0;
    static readonly OPEN = 1;
    static readonly CLOSED = 2;
    onopen: ((this: unknown, ev: unknown) => unknown) | null = null;
    onmessage: ((this: unknown, ev: unknown) => unknown) | null = null;
    onerror: ((this: unknown, ev: unknown) => unknown) | null = null;
    readyState = NoopEventSource.CONNECTING;
    constructor(public url: string) {}
    addEventListener(): void {}
    removeEventListener(): void {}
    close(): void {
      this.readyState = NoopEventSource.CLOSED;
    }
  }
  Object.defineProperty(globalThis, 'EventSource', {
    value: NoopEventSource,
    configurable: true,
    writable: true,
  });
}
