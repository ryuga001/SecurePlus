"use client";

import { createContext, useCallback, useContext, useEffect, useState } from "react";

import { api, setCsrfToken, type Identity } from "@/lib/api";

type AuthState = {
  identity: Identity | null;
  loading: boolean;
  setIdentity: (identity: Identity | null) => void;
  signOut: () => Promise<void>;
  reload: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const current = await api.me();
        if (!cancelled) setIdentity(current);
      } catch {
        if (!cancelled) {
          setIdentity(null);
          setCsrfToken(null);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, []);

  const reload = useCallback(async () => {
    try {
      setIdentity(await api.me());
    } catch {
      setIdentity(null);
      setCsrfToken(null);
    }
  }, []);

  const signOut = useCallback(async () => {
    try {
      await api.logout();
    } catch {
      setIdentity(null);
    } finally {
      setIdentity(null);
      setCsrfToken(null);
    }
  }, []);

  return (
    <AuthContext.Provider value={{ identity, loading, setIdentity, signOut, reload }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used inside AuthProvider");

  return context;
}
