import { useCallback, useEffect, useRef, useState } from "react";
import {
  Button,
  InlineLoading,
  InlineNotification,
  Modal,
  PasswordInput,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  TextInput,
} from "@carbon/react";
import { Add } from "@carbon/icons-react";
import { createUser, listUsers, type UserSummary } from "../api/auth";
import { ApiError } from "../api/client";

const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "The server could not be reached.";
}

export default function AdminPage() {
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState("");
  const [addOpen, setAddOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [usernameError, setUsernameError] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [addError, setAddError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [added, setAdded] = useState("");
  const usernameRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  const refresh = useCallback(async () => {
    try {
      setUsers(await listUsers());
      setListError("");
    } catch (error) {
      setListError(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // Focus the invalid field once it is rendered with its error and enabled again.
  useEffect(() => {
    if (submitting) return;
    if (usernameError) usernameRef.current?.focus();
    else if (passwordError) passwordRef.current?.focus();
  }, [submitting, usernameError, passwordError]);

  function closeAdd() {
    if (submitting) return;
    setAddOpen(false);
    setUsername("");
    setPassword("");
    setUsernameError("");
    setPasswordError("");
    setAddError("");
  }

  async function submitAdd() {
    setSubmitting(true);
    setUsernameError("");
    setPasswordError("");
    setAddError("");
    try {
      const user = await createUser(username.trim(), password);
      setSubmitting(false);
      setAddOpen(false);
      setUsername("");
      setPassword("");
      setAdded(user.username);
      await refresh();
    } catch (error) {
      setSubmitting(false);
      const code = error instanceof ApiError ? error.code : "";
      if (code === "INVALID_USERNAME" || code === "USERNAME_TAKEN") {
        setUsernameError(errorMessage(error));
      } else if (code === "PASSWORD_TOO_SHORT" || code === "PASSWORD_TOO_LONG") {
        setPasswordError(errorMessage(error));
      } else {
        setAddError(errorMessage(error));
      }
    }
  }

  return (
    <section aria-labelledby="users-heading">
      <div className="page-heading">
        <h1 id="users-heading">Users</h1>
      </div>
      {added && (
        <InlineNotification
          kind="success"
          title="User added"
          subtitle={`${added} can now log in.`}
          lowContrast
          className="detail-notification"
          onCloseButtonClick={() => setAdded("")}
        />
      )}
      {listError && (
        <div className="page-feedback">
          <InlineNotification kind="error" title="Could not load users" subtitle={listError} lowContrast className="detail-notification" />
          <Button kind="tertiary" size="sm" onClick={() => void refresh()}>Retry</Button>
        </div>
      )}
      <TableContainer>
        <TableToolbar aria-label="Users toolbar">
          <TableToolbarContent>
            <Button renderIcon={Add} onClick={() => setAddOpen(true)}>Add user</Button>
          </TableToolbarContent>
        </TableToolbar>
        {loading ? (
          <div className="empty-state"><InlineLoading description="Loading users…" /></div>
        ) : !listError && users.length === 0 ? (
          <div className="empty-state">
            <p>No users yet. Adding the first user turns on login and gives them the existing items and settings.</p>
          </div>
        ) : (
          <Table size="lg" aria-label="Users">
            <TableHead>
              <TableRow>
                <TableHeader>Username</TableHeader>
                <TableHeader>Last login</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.username}>
                  <TableCell>{user.username}</TableCell>
                  <TableCell>{user.lastLoginAt ? dateTime.format(new Date(user.lastLoginAt)) : "Never"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </TableContainer>

      <Modal
        open={addOpen}
        modalHeading="Add user"
        primaryButtonText={submitting ? "Adding user…" : "Add user"}
        secondaryButtonText="Cancel"
        primaryButtonDisabled={submitting}
        onRequestSubmit={() => void submitAdd()}
        onRequestClose={closeAdd}
        size="sm"
      >
        {addError && (
          <InlineNotification kind="error" title="Could not add user" subtitle={addError} lowContrast className="modal-notification" />
        )}
        <TextInput
          id="add-user-username"
          ref={usernameRef}
          labelText="Username"
          helperText="3 to 32 letters, digits, dots, dashes or underscores"
          autoComplete="off"
          value={username}
          onChange={(event) => { setUsername(event.target.value); setUsernameError(""); }}
          disabled={submitting}
          invalid={Boolean(usernameError)}
          invalidText={usernameError}
        />
        <PasswordInput
          id="add-user-password"
          ref={passwordRef}
          labelText="Password"
          helperText="At least 12 characters"
          autoComplete="new-password"
          value={password}
          onChange={(event) => { setPassword(event.target.value); setPasswordError(""); }}
          disabled={submitting}
          invalid={Boolean(passwordError)}
          invalidText={passwordError}
        />
      </Modal>
    </section>
  );
}
