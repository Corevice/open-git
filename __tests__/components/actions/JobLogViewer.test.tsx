import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

import JobLogViewer, { stripAnsi } from "@/components/actions/JobLogViewer";

class MockEventSource {
  static instances: MockEventSource[] = [];

  url: string;
  listeners: Record<string, Array<(event: MessageEvent) => void>> = {};
  close = vi.fn();
  onerror: ((event: Event) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    this.listeners[type] ??= [];
    this.listeners[type].push(listener);
  }

  removeEventListener() {}

  emit(type: string, data: string) {
    for (const listener of this.listeners[type] ?? []) {
      listener({ data } as MessageEvent);
    }
  }

  triggerError() {
    this.onerror?.(new Event("error"));
  }
}

const defaultProps = {
  runId: "run-1",
  jobId: "job-1",
  repoOwner: "acme",
  repoName: "demo",
};

describe("JobLogViewer", () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.stubGlobal("EventSource", MockEventSource);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders waiting message initially", () => {
    render(<JobLogViewer {...defaultProps} />);

    expect(screen.getByText("Waiting for log output…")).toBeInTheDocument();
  });

  it("renders log text after a log event", async () => {
    render(<JobLogViewer {...defaultProps} />);

    const source = MockEventSource.instances[0];
    source.emit(
      "log",
      JSON.stringify({
        step: 0,
        line: 1,
        ts: "2026-06-28T00:00:00Z",
        stream: "stdout",
        text: "hello world",
      }),
    );

    await waitFor(() => {
      expect(screen.getByText("hello world")).toBeInTheDocument();
    });
  });

  it("closes the source and shows completed status after done event", async () => {
    render(<JobLogViewer {...defaultProps} />);

    const source = MockEventSource.instances[0];
    source.emit("done", JSON.stringify({ status: "success" }));

    await waitFor(() => {
      expect(source.close).toHaveBeenCalled();
      expect(
        screen.getByRole("status", { name: "Completed" }),
      ).toBeInTheDocument();
    });
  });

  it("strips ANSI escape codes", () => {
    expect(stripAnsi("\x1b[32mgreen\x1b[0m")).toBe("green");
  });
});
