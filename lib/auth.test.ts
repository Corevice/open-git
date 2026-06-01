import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { createElement, type ReactNode } from "react";

import { AUTH_TOKEN_KEY, AuthProvider, useAuth } from "./auth";

function wrapper({ children }: { children: ReactNode }) {
  return createElement(AuthProvider, null, children);
}

describe("auth", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("login() stores token in localStorage", () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.login("test-token-123");
    });

    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBe("test-token-123");
    expect(result.current.token).toBe("test-token-123");
    expect(result.current.isAuthenticated).toBe(true);
  });

  it("logout() clears token from localStorage", () => {
    localStorage.setItem(AUTH_TOKEN_KEY, "test-token-123");
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.logout();
    });

    expect(localStorage.getItem(AUTH_TOKEN_KEY)).toBeNull();
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it("reads token from localStorage on mount", async () => {
    localStorage.setItem(AUTH_TOKEN_KEY, "stored-token");
    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => {
      expect(result.current.token).toBe("stored-token");
    });
    expect(result.current.isAuthenticated).toBe(true);
  });
});
