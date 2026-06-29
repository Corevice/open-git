// Provides a Web Storage (localStorage/sessionStorage) implementation for the
// test environment. jsdom does not reliably expose localStorage, and Node's
// experimental localStorage is unavailable without a backing file, so we install
// a small in-memory polyfill that satisfies the Storage interface used by the app.
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

function install(name: 'localStorage' | 'sessionStorage') {
  const storage = new MemoryStorage();
  Object.defineProperty(globalThis, name, { value: storage, writable: true, configurable: true });
  if (typeof window !== 'undefined') {
    Object.defineProperty(window, name, { value: storage, writable: true, configurable: true });
  }
}

install('localStorage');
install('sessionStorage');
