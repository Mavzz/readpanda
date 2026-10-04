import { useState } from "react";
import { useNavigate } from "react-router-dom";
import AuthShell from "../components/AuthShell";
import { Button, Field, TextAction } from "../components/ui";
import { api } from "../services/api";
import { startSession } from "../utils/session";

const SignUpPage = ({ setIsLoggedIn, onSwitchToLogin }) => {
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const handleSignUp = async () => {
    setError("");
    if (!username || !email || !password || !confirmPassword) {
      setError("Fill in every field.");
      return;
    }
    if (password !== confirmPassword) {
      setError("The passwords don't match.");
      return;
    }

    setLoading(true);
    try {
      const auth = await api.post(
        "/signup",
        { username, email, password },
        { token: "" }
      );
      if (!auth?.accessToken) throw new Error("No access token in response");
      await startSession(auth);
      setIsLoggedIn(true);
      navigate("/dashboard");
    } catch (err) {
      setError(err.status && err.status < 500 ? err.message : "Couldn't create the account. Try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell
      subtitle="Create your admin account"
      error={error}
      onSubmit={handleSignUp}
      footer={<>Already have an account? <TextAction onClick={onSwitchToLogin}>Sign in</TextAction></>}
    >
      <Field label="Username" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" autoFocus />
      <Field label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" />
      <Field label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
      <Field label="Confirm password" type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} autoComplete="new-password" />
      <Button type="submit" disabled={loading} className="w-full mt-1">
        {loading ? "Creating account…" : "Create account"}
      </Button>
    </AuthShell>
  );
};

export default SignUpPage;
