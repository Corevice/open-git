"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";

import { apiClient } from "./api-client";

export interface AuthContextValue {
  pat: string | null;
  setPat: (pat: string | null) => void;
}

const AuthContext = React.createContext<AuthContextValue>({
  pat: null,
  setPat: () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [pat, setPatState] = useState<string | null>(null);

  useEffect(() => {
    const stored = localStorage.getItem("pat");
    if (stored) {
      setPatState(stored);
      apiClient.setPat(stored);
    }
  }, []);

  const setPat = useCallback((value: string | null) => {
    setPatState(value);
    apiClient.setPat(value);
    if (value) {
      localStorage.setItem("pat", value);
    } else {
      localStorage.removeItem("pat");
    }
  }, []);

  const value = useMemo(() => ({ pat, setPat }), [pat, setPat]);

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  return React.useContext(AuthContext);
}
