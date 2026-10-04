import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Form, InlineNotification, Link, PasswordInput, Stack, TextInput } from "@carbon/react";
import { login } from "../api/auth";

interface Props {
  openMode: boolean;
}

export default function LoginPage({ openMode }: Props) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const errorRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (error) errorRef.current?.focus();
  }, [error]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const user = await login(username.trim(), password);
      window.location.assign(user.role === "admin" ? "/admin" : "/");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The server could not be reached.");
      setSubmitting(false);
    }
  }

  return (
    <main id="main-content" className="login-page">
      <Form className="login-form" onSubmit={(event) => void submit(event)} aria-labelledby="login-heading">
        <Stack gap={6}>
          <h1 id="login-heading">Log in to Price follower</h1>
          {error && (
            <div ref={errorRef} tabIndex={-1}>
              <InlineNotification kind="error" title="Could not log in" subtitle={error} lowContrast hideCloseButton />
            </div>
          )}
          <TextInput
            id="login-username"
            labelText="Username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            disabled={submitting}
            required
          />
          <PasswordInput
            id="login-password"
            labelText="Password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            disabled={submitting}
            required
          />
          <Button type="submit" disabled={submitting}>{submitting ? "Logging in…" : "Log in"}</Button>
          {openMode && <Link href="/">Back to tracked items</Link>}
        </Stack>
      </Form>
    </main>
  );
}
