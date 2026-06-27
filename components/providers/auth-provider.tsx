"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import type { Viewer } from "@/types/viewer";

export interface SessionContext {
  viewer: Viewer | null;
  token: string | null;
  setToken(t: string): void;
  signOut(): void;
}

const AuthContext = createContext<SessionContext | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [token, setTokenState] = useState<string | null>(null);
  const [viewer, setViewer] = useState<Viewer | null>(null);

  useEffect(() => {
    const stored = localStorage.getItem("pat");
    if (stored) {
      setTokenState(stored);
    }
  }, []);

  const setToken = (t: string) => {
    localStorage.setItem("pat", t);
    setTokenState(t);
  };

  const signOut = () => {
    localStorage.removeItem("pat");
    setTokenState(null);
    setViewer(null);
    router.push("/sign-in");
  };

  return (
    <AuthContext.Provider value={{ viewer, token, setToken, signOut }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): SessionContext {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
