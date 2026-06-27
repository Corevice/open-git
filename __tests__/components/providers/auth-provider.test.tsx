import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AuthProvider, useAuth } from "@/components/providers/auth-provider";

const mockPush = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}));

function TestConsumer() {
  const auth = useAuth();
  return (
    <div>
      <span data-testid="token">{auth.token ?? "null"}</span>
      <button type="button" onClick={() => auth.signOut()}>
        Sign out
      </button>
    </div>
  );
}

describe("AuthProvider", () => {
  beforeEach(() => {
    mockPush.mockClear();
    vi.spyOn(Storage.prototype, "getItem").mockReturnValue(null);
    vi.spyOn(Storage.prototype, "removeItem");
    vi.spyOn(Storage.prototype, "setItem");
  });

  it("defaults to token null", () => {
    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>,
    );

    expect(screen.getByTestId("token")).toHaveTextContent("null");
  });

  it("signOut removes pat from localStorage", async () => {
    const user = userEvent.setup();
    const removeItemSpy = vi.spyOn(Storage.prototype, "removeItem");

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Sign out" }));

    expect(removeItemSpy).toHaveBeenCalledWith("pat");
  });
});
