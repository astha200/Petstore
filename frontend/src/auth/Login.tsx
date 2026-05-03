import { useState, type FormEvent } from "react";
import { useAuth } from "./AuthProvider";

interface Props {
  storeSlug: string;
}

export function Login({ storeSlug }: Props) {
  const { signIn } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!username || !password) {
      setError("Please enter a username and password.");
      return;
    }
    signIn({ username, password });
  };

  return (
    <div className="login-shell">
      <form className="login-card" onSubmit={onSubmit}>
        <div>
          <h1>Welcome to {storeSlug}</h1>
          <p>Sign in to browse and purchase pets.</p>
        </div>

        {error && <div className="alert alert--error">{error}</div>}

        <div className="field">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </div>

        <div className="field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>

        <button type="submit" className="btn btn--primary btn--block">
          Sign in
        </button>

        <p style={{ fontSize: 12, color: "var(--muted)", margin: 0 }}>
          Demo customer: <code>alice / alicepass</code>
        </p>
      </form>
    </div>
  );
}
