import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useJobLogStream } from "@/hooks/useJobLogStream";

type MockEventSourceInstance = {
  url: string;
  onopen: ((event: Event) => void) | null;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  addEventListener: (type: string, handler: (event: MessageEvent) => void) => void;
  close: ReturnType<typeof vi.fn>;
  emitMessage: (data: string) => void;
  emitError: () => void;
};

let mockInstances: MockEventSourceInstance[] = [];

function createMockEventSource(url: string): MockEventSourceInstance {
  const listeners = new Map<string, ((event: MessageEvent) => void)[]>();
  const instance: MockEventSourceInstance = {
    url,
    onopen: null,
    onmessage: null,
    onerror: null,
    addEventListener(type: string, handler: (event: MessageEvent) => void) {
      const existing = listeners.get(type) ?? [];
      existing.push(handler);
      listeners.set(type, existing);
    },
    close: vi.fn(),
    emitMessage(data: string) {
      const event = { data } as MessageEvent;
      instance.onmessage?.(event);
    },
    emitError() {
      instance.onerror?.({} as Event);
    },
  };

  mockInstances.push(instance);
  return instance;
}

describe("useJobLogStream", () => {
  beforeEach(() => {
    mockInstances = [];
    vi.stubGlobal(
      "EventSource",
      vi.fn((url: string) => createMockEventSource(url)),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("appends lines to state on each message event", async () => {
    const { result } = renderHook(() =>
      useJobLogStream({
        owner: "octocat",
        repo: "hello-world",
        jobId: 42,
        enabled: true,
      }),
    );

    await waitFor(() => {
      expect(mockInstances).toHaveLength(1);
    });

    const source = mockInstances[0];
    expect(source.url).toBe(
      "/api/v3/repos/octocat/hello-world/actions/jobs/42/logs",
    );

    act(() => {
      source.emitMessage("line one");
      source.emitMessage("line two");
    });

    await waitFor(() => {
      expect(result.current.lines).toEqual(["line one", "line two"]);
    });
  });

  it("sets status to done when error event fires", async () => {
    const { result } = renderHook(() =>
      useJobLogStream({
        owner: "octocat",
        repo: "hello-world",
        jobId: 42,
        enabled: true,
      }),
    );

    await waitFor(() => {
      expect(mockInstances).toHaveLength(1);
    });

    act(() => {
      mockInstances[0].emitError();
    });

    await waitFor(() => {
      expect(result.current.status).toBe("done");
    });
    expect(mockInstances[0].close).toHaveBeenCalled();
  });

  it("does not create EventSource when enabled=false", () => {
    renderHook(() =>
      useJobLogStream({
        owner: "octocat",
        repo: "hello-world",
        jobId: 42,
        enabled: false,
      }),
    );

    expect(mockInstances).toHaveLength(0);
    expect(EventSource).not.toHaveBeenCalled();
  });
});
