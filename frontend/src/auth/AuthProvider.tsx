import { createContext, useContext, useState, useMemo, useCallback, type ReactNode } from "react";

export interface Credentials {
  username: string;
  password: string;
}

interface AuthContextValue {
  credentials: Credentials | null;
  signIn: (creds: Credentials) => void;
  signOut: () => void;
  basicHeader: string | null;
}

const STORAGE_KEY = "petstore.credentials";

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function loadFromStorage(): Credentials | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Credentials;
    if (typeof parsed.username !== "string" || typeof parsed.password !== "string") return null;
    return parsed;
  } catch {
    return null;
  }
}

function encodeBasic({ username, password }: Credentials): string {
  return `Basic ${btoa(`${username}:${password}`)}`;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [credentials, setCredentials] = useState<Credentials | null>(loadFromStorage);

  const signIn = useCallback((creds: Credentials) => {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(creds));
    setCredentials(creds);
  }, []);

  const signOut = useCallback(() => {
    sessionStorage.removeItem(STORAGE_KEY);
    setCredentials(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      credentials,
      signIn,
      signOut,
      basicHeader: credentials ? encodeBasic(credentials) : null,
    }),
    [credentials, signIn, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}

// Reads the latest credentials directly from session storage. Used by the
// Apollo link so it never needs to re-render to pick up new auth.
export function readCredentialsHeader(): string | null {
  const c = loadFromStorage();
  return c ? encodeBasic(c) : null;
}
