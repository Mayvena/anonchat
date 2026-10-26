# ANONCHAT User Manual (Beginner Edition)

## 1. What is ANONCHAT?

ANONCHAT is an anonymous, encrypted chat system. It is built around three ideas:

- **Anonymous:** your traffic is passed through three relay servers, so no single server sees both who you are and where you are going.
- **No history:** messages are never saved to disk. They live in memory and disappear after a short time.
- **Panic button:** if you need to, the app can wipe its secret keys and shut down immediately.

> **Project status:** this is an early development version. The desktop window shows the route, an activity log, and a chat panel where you can send and receive self-destructing messages through the chat router. Chat messages currently go straight to the router, not through the three relay servers, and are not yet end-to-end encrypted.

---

## 2. How it works (the simple version)

```
Your app  ->  Guard  ->  Middle  ->  Exit  ->  Chat Router  ->  Redis (memory)
```

| Part | Plain-English role |
|------|--------------------|
| **Desktop app** | The window you see. Shows a globe, chat panel and log. |
| **Guard node** | The first relay. It is the only one that talks to you directly. |
| **Middle node** | Passes data along. It knows neither you nor the destination. |
| **Exit node** | The last relay. It hands your message to the chat router. |
| **Chat Router** | Delivers messages between users. |
| **Redis** | Short-term memory only. If a recipient is offline, their message waits here for about 5 minutes, then is deleted. |

---

## 3. What you need

**To run the server (backend):**

- Docker with the `docker compose` plugin (Docker Desktop on Windows/macOS)
- A computer with an internet connection for the first build

**To build the desktop app (optional, advanced):**

- Rust (1.75 or newer), Node.js 20+, and the Tauri CLI
- On Windows: Visual Studio C++ Build Tools

Detailed server and client install steps are in [DEPLOYMENT.md](DEPLOYMENT.md). A deployment helper agent named "ANONCHAT Deployment" is available in VS Code Copilot chat.

---

## 4. Starting the server

1. Open a terminal in the ANONCHAT folder (the one containing `docker-compose.yml`).
2. Run:

   ```bash
   docker compose up -d --build
   ```

3. Wait for the build to finish (the first time takes a few minutes).
4. Check everything is running:

   ```bash
   docker compose ps
   ```

   You should see five services: `redis-ram`, `chat-router`, `proxy-guard`, `proxy-middle` and `proxy-exit`, all marked "Up".

### Stopping and restarting

| Goal | Command |
|------|---------|
| Stop everything | `docker compose down` |
| Clean restart | `docker compose down` then `docker compose up -d --build` |
| Watch logs | `docker compose logs -f` |
| Logs of one service | `docker compose logs -f chat-router` |

### Ports used

| Port | Service | Public? |
|------|---------|---------|
| 9001 | Guard proxy | Yes (clients connect here) |
| 8080 | Chat router | Local/testing only |
| 9002, 9003 | Middle, Exit proxies | Internal only |
| 6379 | Redis | Local only. Do not expose to the internet. |

---

## 5. Starting the desktop app

For developers running from source:

```bash
cd client/frontend
npm install
npm run dev
```

For a packaged app, build it as described in [DEPLOYMENT.md](DEPLOYMENT.md) (sections 9 and 10) and install the generated `.msi`, `.exe`, `.dmg`, `.deb` or `.AppImage` file.

---

## 6. Understanding the app window

The window has three areas.

### Left: the globe and route bar

- A 3D globe shows the three relay servers (Guard, Middle, Exit) and the path between them.
- The bar at the top of the globe shows the current route as country codes, for example `DE ──► NL ──► SE`. It says `CONNECTING…` until the first route is received.
- The route changes (rotates) from time to time. This is normal and improves anonymity.

### Right: the ephemeral chat panel

- **Connection status:** a label in the top bar shows the state of the link to the chat router:

  | Status | Meaning |
  |--------|---------|
  | CONNECTING | First connection attempt in progress |
  | CONNECTED | Ready. You can send and receive. |
  | DEGRADED | Connected, but sending is struggling. Try again shortly. |
  | RECONNECTING | The link dropped. The app retries automatically with growing pauses. |
  | FAILED | Automatic retries are used up. Click **RETRY CONNECTION**. |

- **Your user ID:** the first pill in the user row reads `You: xxxxxxxx`. This random 8-character ID is created each time you start the app and is not saved. Give it to the person you want to chat with.
- **Online users:** the other pills list the relay nodes, the system, and anyone who has messaged you. A green dot means online.
- **Messages:** every message has a countdown timer (for example `⏳ 18s`). Messages you send and receive last 30 seconds, then fade out and are permanently removed. Nothing is saved.
- **Sending a message:** type the recipient's user ID in the first field, type your message in the second, then press **Enter** or click **SEND**. Use **Shift+Enter** for a new line. The Send button is disabled until the status is CONNECTED or DEGRADED. Messages are limited to 2000 characters.
- **Sound button (speaker icon):** click to mute or unmute the short sound played when a message is deleted.
- Route changes and warnings appear here as short-lived system messages.

### Bottom: the core log

A terminal-style panel that shows what the app's security core is doing, with timestamps. Levels:

| Level | Meaning |
|-------|---------|
| info | Normal activity (for example "Proxy rotated") |
| warn | Something unusual but not fatal |
| error | Something failed. Also shown briefly in the chat panel. |

---

## 7. Trying out chat

### Option A: two app windows

1. Start the server (section 4).
2. Start the desktop app twice (section 5), for example in two terminals or on two computers that can reach the server.
3. Wait until both show **CONNECTED**, then note the `You: …` ID shown in each window.
4. In window 1, enter window 2's ID as the recipient, type a message and press Enter. It appears in window 2 right away.
5. **Offline test:** close window 2, send another message from window 1, then start the app again. The new window gets a new ID, so the waiting message will not arrive. To test offline delivery with a fixed ID, use Option B.

The app connects to `ws://127.0.0.1:8080/ws` by default. To use a different server, set `VITE_ROUTER_WS_URL` when building, and add that address to the connection allow-list in `client/src-tauri/tauri.conf.json`.

### Option B: command line (fixed user IDs)

This lets you see sending, online delivery and offline delivery working with names you choose. It needs Node.js.

1. Make sure the server is running (section 4).
2. Open a terminal and connect as **alice**:

   ```bash
   npx wscat -c "ws://127.0.0.1:8080/ws?user_id=alice"
   ```

   You will receive a confirmation message of type `session`.
3. Open a second terminal and connect as **bob**:

   ```bash
   npx wscat -c "ws://127.0.0.1:8080/ws?user_id=bob"
   ```

4. In alice's terminal, send a message to bob:

   ```json
   {"id":"m1","to":"bob","body":"hello bob"}
   ```

   Bob's terminal shows it immediately.
5. **Offline test:** close bob's terminal (Ctrl+C), then send from alice:

   ```json
   {"id":"m2","to":"bob","body":"you were offline"}
   ```

6. Reconnect bob with the same command as step 3. The waiting message is delivered once and then deleted. Messages not collected within about 5 minutes are discarded.

Tips: each message needs a unique `id`; the `to` field is the recipient's user ID; the user ID is just a name you choose in the web address.

---

## 8. Privacy and security features

- **No message history:** messages are not written to disk, and the server's Redis runs in memory-only mode.
- **Auto-expiry:** offline messages are deleted automatically after about 5 minutes.
- **Local keys:** the app creates its encryption keys on your device on first run. Keys are never stored unencrypted.
- **Fixed-size packets:** the security core builds onion-style packets of a fixed 4096 bytes so message size reveals less.
- **Panic button:** the security core wipes secret keys from memory, overwrites the local encrypted database three times with random data, and closes the app instantly. If the database cannot be overwritten completely, the app exits with failure instead of claiming the wipe succeeded. This cannot be undone. Only use it in an emergency. File-level overwrite cannot guarantee physical-sector destruction on SSDs or copy-on-write storage; use the storage vendor's secure-erase facility when that guarantee is required.
- **Minimal logs:** the servers do not log message contents, keys or user identities.

### Good habits

- Messages are not yet routed through the relay chain or end-to-end encrypted, so avoid sending anything sensitive until that is added.
- Use a user ID that does not reveal your real name.
- Do not share your user ID with people you do not trust.
- Keep the Redis port (6379) closed to the internet.
- For anything beyond local testing, use real TLS certificates instead of self-signed ones (see [DEPLOYMENT.md](DEPLOYMENT.md)).

---

## 9. Troubleshooting

| Problem | What to try |
|---------|-------------|
| `docker compose` not found | Install Docker (or Docker Desktop) and reopen your terminal. |
| A service is not "Up" | Run `docker compose logs <service-name>` and read the last lines. |
| Port already in use (8080, 9001, 6379) | Stop the program using it, or change the port mapping in `docker-compose.yml`. |
| Route bar stays on `CONNECTING…` | The app has not received a route yet. Check the log panel for errors and confirm the desktop app was started through Tauri, not just a plain browser. |
| `Failed to start event stream` in the log | The frontend is running outside the Tauri shell, or the Rust side failed to start. Restart the app. |
| Status shows FAILED | The app could not reach the chat router. Check `docker compose ps`, then click **RETRY CONNECTION**. |
| Send button is greyed out | The status is not CONNECTED yet. Wait, or see the status row above. |
| Message sent but never appears for the other person | Check the recipient ID for typos. If you restarted the app, your ID has changed, so tell the other person the new one. |
| App cannot connect to a non-local server | The server address must be on the allow-list in `client/src-tauri/tauri.conf.json`. |
| `wscat` cannot connect | Confirm `docker compose ps` shows `chat-router` as Up and that you included `?user_id=name` in the address. |
| Offline message never arrives | It may have passed its 5-minute limit, or you reconnected with a different user ID. The desktop app picks a new ID on every launch. |
| Messages vanish too quickly | This is by design. Messages are temporary. |
| Proxy container exits right away | Check its logs. Misconfigured TLS settings are the usual cause: certificate and key variables must be set together. |

---

## 10. Glossary

- **Anonymous routing:** sending traffic through several relays so no one can easily trace it back to you.
- **Ephemeral:** temporary. Deleted automatically.
- **Encryption:** scrambling data so only the intended reader can read it.
- **TLS:** the standard that protects connections between servers.
- **WebSocket:** a lasting two-way connection used for live chat.
- **Redis:** a very fast in-memory database.
- **Docker:** a tool that runs each service in its own isolated container.
- **Tauri:** the framework that turns the web interface into a desktop app.

---

## 11. Where to learn more

- [README.md](README.md) for local development and the message format
- [DEPLOYMENT.md](DEPLOYMENT.md) for server setup, firewall and hardening
- [AGENTS.md](AGENTS.md) for project rules and architecture boundaries
