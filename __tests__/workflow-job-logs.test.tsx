import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { JobLogsPageContent } from "@/app/12-actions-job/page";

type MockEventSourceInstance = {
  url: string;
  onmessage: ((event: MessageEvent) => void) | null;
  addEventListener: (type: string, handler: (event: MessageEvent) => void) => void;
  close: ReturnType<typeof vi.fn>;
  emitMessage: (data: string) => void;
  emitDone: () => void;
};

let mockInstances: MockEventSourceInstance[] = [];

function createMockEventSource(url: string): MockEventSourceInstance {
  const listeners = new Map<string, ((event: MessageEvent) => void)[]>();
  const instance: MockEventSourceInstance = {
    url,
    onmessage: null,
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
    emitDone() {
      const handlers = listeners.get("done") ?? [];
      for (const handler of handlers) {
        handler({ data: "done" } as MessageEvent);
      }
    },
  };

  mockInstances.push(instance);
  return instance;
}

const jobResponse = {
  id: 42,
  name: "build",
  status: "in_progress",
  conclusion: null,
  started_at: "2026-06-28T10:00:00Z",
  completed_at: null,
  steps: [
    { number: 1, name: "Set up job", status: "completed", conclusion: "success" },
    { number: 2, name: "Run tests", status: "in_progress", conclusion: null },
  ],
};

vi.mock("next/navigation", () => ({
  useSearchParams: () =>
    new URLSearchParams("jobId=42&owner=octocat&repo=hello-world"),
}));

describe("workflow job logs page", () => {
  beforeEach(() => {
    mockInstances = [];
    vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "http://localhost:8080");

    vi.stubGlobal(
      "EventSource",
      vi.fn((url: string) => createMockEventSource(url)),
    );

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);

        if (url.endsWith("/actions/jobs/42")) {
          return {
            ok: true,
            status: 200,
            json: async () => jobResponse,
          };
        }

        if (url.endsWith("/actions/jobs/42/logs")) {
          return {
            ok: true,
            status: 200,
            text: async () => "existing line\n",
          };
        }

        return {
          ok: false,
          status: 404,
          json: async () => ({ message: "Not Found" }),
          text: async () => "Not Found",
        };
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it("opens EventSource with offset from initial logs and appends SSE lines", async () => {
    render(<JobLogsPageContent />);

    expect(await screen.findByText("build")).toBeInTheDocument();
    expect(screen.getByText("existing line")).toBeInTheDocument();
    expect(screen.getByText("Set up job")).toBeInTheDocument();
    expect(screen.getByText("Run tests")).toBeInTheDocument();

    await waitFor(() => {
      expect(mockInstances).toHaveLength(1);
    });

    expect(mockInstances[0].url).toBe(
      "http://localhost:8080/repos/octocat/hello-world/actions/jobs/42/logs/stream?offset=14",
    );

    act(() => {
      mockInstances[0].emitMessage("streamed line");
    });

    expect(await screen.findByText("streamed line")).toBeInTheDocument();
  });

  it("disables auto-scroll after the user scrolls up", async () => {
    render(<JobLogsPageContent />);

    const pre = await screen.findByText("existing line");
    const preElement = pre.closest("pre");
    expect(preElement).not.toBeNull();

    Object.defineProperty(preElement!, "scrollHeight", {
      configurable: true,
      value: 1000,
    });
    Object.defineProperty(preElement!, "clientHeight", {
      configurable: true,
      value: 200,
    });
    Object.defineProperty(preElement!, "scrollTop", {
      configurable: true,
      writable: true,
      value: 0,
    });

    fireEvent.scroll(preElement!);

    expect(screen.getByRole("button", { name: "Paused" })).toBeInTheDocument();
  });

  it("closes EventSource when a done event is received", async () => {
    render(<JobLogsPageContent />);

    await waitFor(() => {
      expect(mockInstances).toHaveLength(1);
    });

    act(() => {
      mockInstances[0].emitDone();
    });

    expect(mockInstances[0].close).toHaveBeenCalled();
  });

  it("toggles Live/Paused auto-scroll state", async () => {
    const user = userEvent.setup();

    render(<JobLogsPageContent />);

    const toggle = await screen.findByRole("button", { name: "Live" });
    await user.click(toggle);

    expect(screen.getByRole("button", { name: "Paused" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Paused" }));

    expect(screen.getByRole("button", { name: "Live" })).toBeInTheDocument();
  });
});
