/**
 * Returns a promise that is already settled and pre-tagged with React's
 * thenable tracking fields (`status`/`value`). React's `use()` hook reads these
 * synchronously, so a component calling `use(params)` resolves on its first
 * render instead of suspending.
 *
 * This mirrors how Next.js delivers already-resolved `params`/`searchParams`
 * promises during client rendering and avoids relying on Suspense-retry
 * flushing, which is unreliable in the jsdom + React 19 test environment.
 */
export function resolvedParams<T>(value: T): Promise<T> {
  const promise = Promise.resolve(value) as Promise<T> & {
    status: "fulfilled";
    value: T;
  };
  promise.status = "fulfilled";
  promise.value = value;
  return promise;
}
