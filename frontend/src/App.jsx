import { useState } from "react";

const API_URL = (import.meta.env.VITE_API_URL ?? "").replace(/\/+$/, "");
const MODE = API_URL ? "cross-site" : "same-origin";

const ACTIONS = [
  { label: "Login", method: "POST", path: "/api/login" },
  { label: "Me", method: "GET", path: "/api/me" },
  { label: "Logout", method: "POST", path: "/api/logout" },
];

async function call({ method, path }) {
  const init = { method, credentials: "include" };
  if (method === "POST") {
    // A JSON body is not a CORS "simple request", so cross-site it triggers
    // the OPTIONS preflight on purpose.
    init.headers = { "Content-Type": "application/json" };
    init.body = "{}";
  }
  try {
    const res = await fetch(API_URL + path, init);
    const text = await res.text();
    let body;
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
    return { status: res.status, body };
  } catch (err) {
    // The browser hides CORS failures from JS: details are only in the console.
    return { error: String(err) };
  }
}

export default function App() {
  const [result, setResult] = useState(null);
  const [pending, setPending] = useState(false);

  async function run(action) {
    const label = `${action.method} ${action.path}`;
    setPending(true);
    setResult({ action: label, pending: true });
    setResult({ action: label, ...(await call(action)) });
    setPending(false);
  }

  return (
    <main
      style={{
        fontFamily: "system-ui, sans-serif",
        maxWidth: 640,
        margin: "2rem auto",
        padding: "0 1rem",
      }}
    >
      <h1>Cookie Lab</h1>
      <p>
        Modo: <strong>{MODE}</strong> · API:{" "}
        <code>{API_URL || window.location.origin}</code>
      </p>
      <div style={{ display: "flex", gap: "0.5rem" }}>
        {ACTIONS.map((a) => (
          <button key={a.label} onClick={() => run(a)} disabled={pending}>
            {a.label}
          </button>
        ))}
      </div>
      <pre
        style={{
          background: "#f4f4f4",
          padding: "1rem",
          overflowX: "auto",
          whiteSpace: "pre-wrap",
        }}
      >
        {result ? JSON.stringify(result, null, 2) : "Apretá un botón."}
      </pre>
    </main>
  );
}
