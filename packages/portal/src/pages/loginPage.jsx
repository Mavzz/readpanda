import { useState } from "react";
import { GoogleLogin } from "@react-oauth/google";
import { useNavigate } from "react-router-dom";
import AuthShell from "../components/AuthShell";
import { Button, Field, TextAction } from "../components/ui";
import { api } from "../services/api";
import { encryptedPassword } from "../utils/Helper";
import { startSession } from "../utils/session";

const LOGIN_TIMEOUT_MS = 10000;

const loginErrorMessage = (err) => {
  if (err.name === "AbortError") return "Sign in timed out. Check the API is running and try again.";
  if (err.status === 401 || err.status === 404) return "That username and password don't match.";
  if (err.status === 403) return "This account isn't an admin.";
  return "Couldn't reach the server. Check your connection and try again.";
};

const LoginPage = ({ setIsLoggedIn, onSwitchToSignUp }) => {
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const finish = async (auth) => {
    await startSession(auth);
    setIsLoggedIn(true);
    navigate("/dashboard");
  };

  const handleLogin = async () => {
    setError("");
    if (!username || !password) {
      setError("Enter your username and password.");
      return;
    }

    setLoading(true);
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), LOGIN_TIMEOUT_MS);
    try {
      const auth = await api.post(
        "/auth/login",
        { username, password: encryptedPassword(password) },
        { signal: controller.signal, token: "" }
      );
      if (!auth?.accessToken) throw new Error("No access token in response");
      await finish(auth);
    } catch (err) {
      console.error("Login failed:", err);
      setError(loginErrorMessage(err));
    } finally {
      clearTimeout(timeoutId);
      setLoading(false);
    }
  };

  const handleGoogleSuccess = async ({ credential }) => {
    setLoading(true);
    setError("");
    try {
      // The API reads the Google ID token from the Authorization header, as the app sends it.
      const auth = await api.post("/auth/google", { token: credential }, { token: credential });
      await finish(auth);
    } catch (err) {
      console.error("Google sign-in failed:", err);
      setError(err.status === 403 ? "This account isn't an admin." : `Google sign-in didn't go through: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell
      subtitle="Sign in with your admin account"
      error={error}
      onSubmit={handleLogin}
      footer={<>Don&apos;t have an account? <TextAction onClick={onSwitchToSignUp}>Sign up</TextAction></>}
    >
      <Field label="Username" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" autoFocus />
      <Field label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
      <Button type="submit" disabled={loading} className="w-full mt-1">
        {loading ? "Signing in…" : "Sign in"}
      </Button>
      <div className="flex items-center gap-3 text-[11px] font-bold tracking-[1px] uppercase text-ink-holder">
        <span className="flex-1 h-px bg-hairline" />or<span className="flex-1 h-px bg-hairline" />
      </div>
      {/* Google's iframe is light; a matching color-scheme stops Chrome painting a white box behind it. */}
      <div className="flex justify-center" style={{ colorScheme: "light" }}>
        <GoogleLogin
          onSuccess={handleGoogleSuccess}
          onError={() => setError("Google sign-in didn't go through. Try again.")}
          theme="filled_black"
          shape="pill"
          size="large"
          width="352"
        />
      </div>
    </AuthShell>
  );
};

export default LoginPage;
